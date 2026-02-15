package figmaextractor

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kataras/figma-extractor/pkg/extractor"
	"github.com/kataras/figma-extractor/pkg/figma"
	"github.com/kataras/figma-extractor/pkg/formatter"
	"github.com/kataras/figma-extractor/pkg/imager"
)

// Options configures the extraction.
type Options struct {
	AccessToken        string
	FileURL            string    // Figma file URL
	NodeIDs            []string  // empty = entire file
	InheritFileContext bool
	ExportImages       bool
	ImageFormat        string    // "png", "svg", "jpg", "pdf"
	ImageScales        []float64
	ImageDir           string
	ComponentTree      bool
	Logger             Logger // nil = no logging
}

// Logger receives progress messages. A nil Logger means silent operation.
type Logger interface {
	Infof(format string, args ...any)
	Warnf(format string, args ...any)
	Errorf(format string, args ...any)
}

// Result contains the extraction output.
type Result struct {
	Specs    *extractor.DesignSpecs
	FileName string // Figma file name
	Markdown string // formatted markdown output
}

func (o *Options) logInfo(f string, a ...any) {
	if o.Logger != nil {
		o.Logger.Infof(f, a...)
	}
}

func (o *Options) logWarn(f string, a ...any) {
	if o.Logger != nil {
		o.Logger.Warnf(f, a...)
	}
}

func (o *Options) logError(f string, a ...any) {
	if o.Logger != nil {
		o.Logger.Errorf(f, a...)
	}
}

// Run executes the Figma extraction pipeline and returns the result.
func Run(opts Options) (*Result, error) {
	// Apply defaults.
	if opts.ImageFormat == "" {
		opts.ImageFormat = "png"
	}
	if opts.ImageDir == "" {
		opts.ImageDir = "figma-assets"
	}
	if len(opts.ImageScales) == 0 {
		opts.ImageScales = []float64{1}
	}

	// Extract file key from URL.
	opts.logInfo("Extracting file key from URL...")
	fileKey, err := figma.ExtractFileKey(opts.FileURL)
	if err != nil {
		return nil, fmt.Errorf("extract file key: %w", err)
	}
	opts.logInfo("File key: %s", fileKey)

	// Extract node IDs from URL or merge with explicit ones.
	var targetNodeIDs []string
	if len(opts.NodeIDs) > 0 {
		opts.logInfo("Using %d explicit node ID(s)", len(opts.NodeIDs))
		targetNodeIDs = opts.NodeIDs
	} else {
		opts.logInfo("Checking URL for node IDs...")
		urlNodeIDs, err := figma.ExtractNodeIDs(opts.FileURL)
		if err != nil {
			return nil, fmt.Errorf("extract node IDs from URL: %w", err)
		}
		if len(urlNodeIDs) > 0 {
			targetNodeIDs = urlNodeIDs
			opts.logInfo("Found %d node(s) in URL", len(targetNodeIDs))
		} else {
			opts.logInfo("No node IDs found, will extract entire file")
		}
	}

	// Create Figma client.
	opts.logInfo("Authenticating with Figma API...")
	client := figma.NewClient(opts.AccessToken)

	var specs *extractor.DesignSpecs
	var fileName string
	var fileResp *figma.FileResponse
	var nodesResp *figma.NodesResponse

	// Choose extraction strategy based on whether node IDs are provided.
	if len(targetNodeIDs) > 0 {
		opts.logInfo("Extracting %d specific node(s)...", len(targetNodeIDs))

		opts.logInfo("Fetching nodes from Figma...")
		nodesResp, err = client.GetFileNodes(fileKey, targetNodeIDs)
		if err != nil {
			return nil, fmt.Errorf("fetch nodes: %w", err)
		}
		opts.logInfo("Retrieved %d node(s)", len(nodesResp.Nodes))

		opts.logInfo("Fetching file metadata...")
		fileResp, err = client.GetFile(fileKey)
		if err != nil {
			return nil, fmt.Errorf("fetch file metadata: %w", err)
		}
		opts.logInfo("File: %s", fileResp.Name)
		fileName = fileResp.Name

		opts.logInfo("Extracting design specifications from nodes...")
		specs = extractor.ExtractNodes(fileResp, nodesResp, targetNodeIDs, opts.InheritFileContext)
	} else {
		opts.logInfo("Extracting entire file...")

		opts.logInfo("Fetching file data from Figma...")
		fileResp, err = client.GetFile(fileKey)
		if err != nil {
			return nil, fmt.Errorf("fetch file: %w", err)
		}
		opts.logInfo("File: %s", fileResp.Name)
		fileName = fileResp.Name

		opts.logInfo("Extracting design specifications...")
		specs = extractor.Extract(fileResp)
	}

	// Image export (opt-in).
	if opts.ExportImages {
		if err := exportImages(&opts, client, fileKey, specs, fileResp, nodesResp, targetNodeIDs); err != nil {
			return nil, err
		}
	}

	// Component tree is opt-in.
	if opts.ComponentTree {
		extractor.AttachAssetsToNodeTree(specs.NodeTree, specs.ExportedAssets)
	} else {
		specs.NodeTree = nil
	}

	// Format as markdown.
	opts.logInfo("Generating markdown documentation...")
	markdown := formatter.ToMarkdown(specs, fileName, opts.ImageDir)

	return &Result{
		Specs:    specs,
		FileName: fileName,
		Markdown: markdown,
	}, nil
}

// exportImages handles the full image export pipeline: screenshot, ExportSettings nodes,
// IMAGE fills, render fallback, and deduplication.
func exportImages(opts *Options, client *figma.Client, fileKey string, specs *extractor.DesignSpecs, fileResp *figma.FileResponse, nodesResp *figma.NodesResponse, targetNodeIDs []string) error {
	// Validate format.
	validFormats := map[string]bool{"png": true, "svg": true, "jpg": true, "pdf": true}
	if !validFormats[opts.ImageFormat] {
		return fmt.Errorf("invalid image format %q (must be png, svg, jpg, or pdf)", opts.ImageFormat)
	}

	// Validate scales.
	for _, s := range opts.ImageScales {
		if s <= 0 {
			return fmt.Errorf("scale value must be positive, got %g", s)
		}
	}

	config := imager.ExportConfig{
		Format:    opts.ImageFormat,
		Scales:    opts.ImageScales,
		OutputDir: opts.ImageDir,
	}

	// Screenshot: render the target node(s) (or full document) as a complete design screenshot.
	screenshotName := "complete_design_screenshot." + config.Format
	screenshotNodes := make(map[string]string) // nodeID -> nodeName

	if len(targetNodeIDs) > 0 {
		for _, id := range targetNodeIDs {
			if nd, ok := nodesResp.Nodes[id]; ok {
				screenshotNodes[id] = nd.Document.Name
				for _, child := range nd.Document.Children {
					screenshotNodes[child.ID] = child.Name
				}
			}
		}
	} else {
		screenshotNodes[fileResp.Document.ID] = fileResp.Document.Name
		for _, child := range fileResp.Document.Children {
			screenshotNodes[child.ID] = child.Name
		}
	}

	opts.logInfo("Capturing design screenshot to %s...", screenshotName)
	screenshotResult, err := imager.ExportImages(client, fileKey, screenshotNodes, imager.ExportConfig{
		Format:    config.Format,
		Scales:    []float64{1},
		OutputDir: config.OutputDir,
	})
	if err != nil {
		opts.logWarn("Screenshot failed: %v", err)
	} else {
		for _, asset := range screenshotResult.Assets {
			oldPath := filepath.Join(config.OutputDir, asset.FileName)
			newPath := filepath.Join(config.OutputDir, screenshotName)
			if err := os.Rename(oldPath, newPath); err != nil {
				opts.logWarn("Could not rename screenshot: %v", err)
				specs.ExportedAssets = append(specs.ExportedAssets, extractor.ExportedAssetInfo{
					NodeID:       asset.NodeID,
					NodeName:     asset.NodeName,
					FileName:     asset.FileName,
					Format:       asset.Format,
					Scale:        asset.Scale,
					IsScreenshot: true,
				})
			} else {
				specs.ExportedAssets = append(specs.ExportedAssets, extractor.ExportedAssetInfo{
					NodeID:       asset.NodeID,
					NodeName:     asset.NodeName,
					FileName:     screenshotName,
					Format:       asset.Format,
					Scale:        asset.Scale,
					IsScreenshot: true,
				})
			}
		}
	}

	// Phase 1: Collect and export nodes with ExportSettings via render API.
	exportNodes := make(map[string]string)

	if len(targetNodeIDs) > 0 {
		opts.logInfo("Discovering exportable child nodes...")
		for _, id := range targetNodeIDs {
			if nd, ok := nodesResp.Nodes[id]; ok {
				childExport := imager.CollectExportableNodes(&nd.Document)
				for cID, cName := range childExport {
					if _, isRoot := screenshotNodes[cID]; isRoot {
						continue
					}
					exportNodes[cID] = cName
				}
			}
		}
		if len(exportNodes) == 0 {
			opts.logInfo("No additional exportable child nodes")
		} else {
			opts.logInfo("Found %d exportable child node(s)", len(exportNodes))
		}
	} else {
		opts.logInfo("Discovering exportable nodes...")
		exportNodes = imager.CollectExportableNodes(&fileResp.Document)
		delete(exportNodes, fileResp.Document.ID)
		if len(exportNodes) == 0 {
			opts.logInfo("No additional exportable nodes")
		} else {
			opts.logInfo("Found %d exportable node(s)", len(exportNodes))
		}
	}

	if len(exportNodes) > 0 {
		opts.logInfo("Exporting rendered images to %s...", opts.ImageDir)
		result, err := imager.ExportImages(client, fileKey, exportNodes, config)
		if err != nil {
			return fmt.Errorf("export images: %w", err)
		}
		opts.logInfo("Exported %d image(s)", len(result.Assets))

		for _, dlErr := range result.Errors {
			opts.logWarn("%v", dlErr)
		}

		for _, asset := range result.Assets {
			specs.ExportedAssets = append(specs.ExportedAssets, extractor.ExportedAssetInfo{
				NodeID:   asset.NodeID,
				NodeName: asset.NodeName,
				FileName: asset.FileName,
				Format:   asset.Format,
				Scale:    asset.Scale,
			})
		}
	}

	// Phase 2: Collect and export embedded IMAGE fill nodes via file images API.
	var roots []*figma.Node
	if len(targetNodeIDs) > 0 {
		for _, id := range targetNodeIDs {
			if nd, ok := nodesResp.Nodes[id]; ok {
				doc := nd.Document // copy
				roots = append(roots, &doc)
			}
		}
	} else {
		roots = append(roots, &fileResp.Document)
	}

	var allImageFills []imager.ImageFillNode
	for _, root := range roots {
		for _, fill := range imager.CollectImageFillNodes(root) {
			if _, isScreenshot := screenshotNodes[fill.NodeID]; isScreenshot {
				continue
			}
			allImageFills = append(allImageFills, fill)
		}
	}

	if len(allImageFills) > 0 {
		opts.logInfo("Found %d embedded image(s), fetching download URLs...", len(allImageFills))
		var unresolvedNodes []imager.ImageFillNode

		fileImagesResp, err := client.GetFileImages(fileKey)
		if err != nil {
			opts.logWarn("File images API failed: %v", err)
			unresolvedNodes = allImageFills
		} else {
			opts.logInfo("Downloading embedded images to %s...", opts.ImageDir)
			fillResult, err := imager.ExportImageFills(fileImagesResp, allImageFills, config)
			if err != nil {
				return fmt.Errorf("export image fills: %w", err)
			}

			if len(fillResult.Assets) > 0 {
				opts.logInfo("Exported %d embedded image(s)", len(fillResult.Assets))
			}

			for _, dlErr := range fillResult.Errors {
				opts.logWarn("%v", dlErr)
			}

			for _, asset := range fillResult.Assets {
				specs.ExportedAssets = append(specs.ExportedAssets, extractor.ExportedAssetInfo{
					NodeID:   asset.NodeID,
					NodeName: asset.NodeName,
					FileName: asset.FileName,
					Format:   asset.Format,
					Scale:    asset.Scale,
				})
			}

			unresolvedNodes = fillResult.UnresolvedNodes
		}

		// Fallback: render unresolved IMAGE fill nodes via the render API.
		if len(unresolvedNodes) > 0 {
			opts.logInfo("Rendering %d image(s) via render API (no file image URLs)...", len(unresolvedNodes))
			renderNodes := imager.ImageFillNodesToMap(unresolvedNodes)
			for id := range screenshotNodes {
				delete(renderNodes, id)
			}
			renderResult, err := imager.ExportImages(client, fileKey, renderNodes, config)
			if err != nil {
				opts.logError("Rendering images failed: %v", err)
				// Non-fatal: continue.
			} else {
				opts.logInfo("Rendered %d image(s)", len(renderResult.Assets))

				for _, dlErr := range renderResult.Errors {
					opts.logWarn("%v", dlErr)
				}

				for _, asset := range renderResult.Assets {
					specs.ExportedAssets = append(specs.ExportedAssets, extractor.ExportedAssetInfo{
						NodeID:   asset.NodeID,
						NodeName: asset.NodeName,
						FileName: asset.FileName,
						Format:   asset.Format,
						Scale:    asset.Scale,
					})
				}
			}
		}
	}

	// Remove non-screenshot assets that duplicate a screenshot node.
	if len(screenshotNodes) > 0 {
		excludeIDs := make(map[string]bool, len(screenshotNodes))
		excludeNames := make(map[string]bool, len(screenshotNodes))
		for id, name := range screenshotNodes {
			excludeIDs[id] = true
			excludeNames[name] = true
		}
		filtered := specs.ExportedAssets[:0]
		for _, a := range specs.ExportedAssets {
			if !a.IsScreenshot && (excludeIDs[a.NodeID] || excludeNames[a.NodeName]) {
				os.Remove(filepath.Join(opts.ImageDir, a.FileName))
				continue
			}
			filtered = append(filtered, a)
		}
		specs.ExportedAssets = filtered
	}

	return nil
}

// ParseScales parses a comma-separated string of scale factors into a float64 slice.
func ParseScales(scalesStr string) ([]float64, error) {
	parts := strings.Split(scalesStr, ",")
	scales := make([]float64, 0, len(parts))

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}

		s, err := strconv.ParseFloat(trimmed, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid scale value %q: %w", trimmed, err)
		}
		if s <= 0 {
			return nil, fmt.Errorf("scale value must be positive, got %g", s)
		}

		scales = append(scales, s)
	}

	if len(scales) == 0 {
		return []float64{1}, nil
	}

	return scales, nil
}

// ParseNodeIDs parses a comma-separated string of node IDs and returns a slice.
func ParseNodeIDs(nodeIDsStr string) []string {
	parts := strings.Split(nodeIDsStr, ",")
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}
