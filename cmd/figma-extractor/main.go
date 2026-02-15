package main

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

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

const version = figma.Version

var (
	figmaURL           string
	accessToken        string
	outputFile         string
	nodeIDs            string
	inheritFileContext bool
	exportImages       bool
	imageFormat        string
	imageScales        string
	imageDir           string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "figma-extractor",
		Short: "Extract design specifications from Figma files",
		Long:  "A tool to extract design tokens, colors, typography, and other specifications from Figma files via the Figma API",
		Run:   run,
	}

	rootCmd.Flags().StringVarP(&figmaURL, "url", "u", "", "Figma file URL (required)")
	rootCmd.Flags().StringVarP(&accessToken, "token", "t", "", "Figma Personal Access Token (required)")
	rootCmd.Flags().StringVarP(&outputFile, "output", "o", "FIGMA_DESIGN_SPECIFICATIONS.md", "Output markdown file")
	rootCmd.Flags().StringVarP(&nodeIDs, "node-ids", "n", "", "Comma-separated node IDs to extract (optional, extracts specific nodes instead of entire file)")
	rootCmd.Flags().BoolVarP(&inheritFileContext, "inherit-context", "i", false, "Inherit file-level context (colors, styles) when extracting specific nodes")
	rootCmd.Flags().BoolVar(&exportImages, "export-images", false, "Export images/assets from Figma")
	rootCmd.Flags().StringVar(&imageFormat, "image-format", "png", "Image format: png, svg, jpg, pdf")
	rootCmd.Flags().StringVar(&imageScales, "image-scales", "1", "Comma-separated scale factors (e.g. \"1,2,3\")")
	rootCmd.Flags().StringVar(&imageDir, "image-dir", "figma-assets", "Output directory for exported images")

	rootCmd.MarkFlagRequired("url")
	rootCmd.MarkFlagRequired("token")

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version number",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("figma-extractor version %s\n", version)
		},
	}

	rootCmd.AddCommand(versionCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) {
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)
	red := color.New(color.FgRed)
	cyan := color.New(color.FgCyan)

	cyan.Println("\n🎨 Figma Design Extractor")
	cyan.Println("==========================")
	cyan.Println()

	// Extract file key from URL
	yellow.Print("📋 Extracting file key from URL... ")
	fileKey, err := figma.ExtractFileKey(figmaURL)
	if err != nil {
		red.Printf("✗\n")
		red.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	green.Printf("✓ File key: %s\n", fileKey)

	// Extract node IDs from URL or flag
	var targetNodeIDs []string
	if nodeIDs != "" {
		// Use node IDs from flag
		yellow.Print("🎯 Parsing node IDs from flag... ")
		targetNodeIDs = parseNodeIDsFromString(nodeIDs)
		green.Printf("✓ Found %d node(s)\n", len(targetNodeIDs))
	} else {
		// Try to extract node IDs from URL
		yellow.Print("🔍 Checking URL for node IDs... ")
		urlNodeIDs, err := figma.ExtractNodeIDs(figmaURL)
		if err != nil {
			red.Printf("✗\n")
			red.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		if len(urlNodeIDs) > 0 {
			targetNodeIDs = urlNodeIDs
			green.Printf("✓ Found %d node(s) in URL\n", len(targetNodeIDs))
		} else {
			yellow.Println("✓ No node IDs found, will extract entire file")
		}
	}

	// Create Figma client
	yellow.Print("🔑 Authenticating with Figma API... ")
	client := figma.NewClient(accessToken)
	green.Println("✓")

	var specs *extractor.DesignSpecs
	var fileName string
	var fileResp *figma.FileResponse
	var nodesResp *figma.NodesResponse

	// Choose extraction strategy based on whether node IDs are provided
	if len(targetNodeIDs) > 0 {
		// Node-specific extraction
		cyan.Printf("\n📦 Extracting %d specific node(s)...\n", len(targetNodeIDs))

		// Fetch specific nodes
		yellow.Print("📥 Fetching nodes from Figma... ")
		var err error
		nodesResp, err = client.GetFileNodes(fileKey, targetNodeIDs)
		if err != nil {
			red.Printf("✗\n")
			red.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		green.Printf("✓ Retrieved %d node(s)\n", len(nodesResp.Nodes))

		// Fetch file metadata for context
		yellow.Print("📥 Fetching file metadata... ")
		fileResp, err = client.GetFile(fileKey)
		if err != nil {
			red.Printf("✗\n")
			red.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		green.Printf("✓ File: %s\n", fileResp.Name)
		fileName = fileResp.Name

		// Extract design specifications from nodes
		yellow.Print("🔍 Extracting design specifications from nodes... ")
		specs = extractor.ExtractNodes(fileResp, nodesResp, targetNodeIDs, inheritFileContext)
		green.Println("✓")
	} else {
		// Full file extraction
		cyan.Println("\n📄 Extracting entire file...")

		// Fetch file data
		yellow.Print("📥 Fetching file data from Figma... ")
		var err error
		fileResp, err = client.GetFile(fileKey)
		if err != nil {
			red.Printf("✗\n")
			red.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		green.Printf("✓ File: %s\n", fileResp.Name)
		fileName = fileResp.Name

		// Extract design specifications
		yellow.Print("🔍 Extracting design specifications... ")
		specs = extractor.Extract(fileResp)
		green.Println("✓")
	}

	// Image export (opt-in via --export-images)
	if exportImages {
		// Validate format
		validFormats := map[string]bool{"png": true, "svg": true, "jpg": true, "pdf": true}
		if !validFormats[imageFormat] {
			red.Printf("\nError: invalid image format %q (must be png, svg, jpg, or pdf)\n", imageFormat)
			os.Exit(1)
		}

		// Parse scales
		scales, err := parseScales(imageScales)
		if err != nil {
			red.Printf("\nError: %v\n", err)
			os.Exit(1)
		}

		config := imager.ExportConfig{
			Format:    imageFormat,
			Scales:    scales,
			OutputDir: imageDir,
		}

		// Screenshot: render the target node(s) (or full document) as a complete design screenshot.
		screenshotName := "complete_design_screenshot." + config.Format
		screenshotNodes := make(map[string]string) // nodeID -> nodeName

		if len(targetNodeIDs) > 0 {
			for _, id := range targetNodeIDs {
				if nd, ok := nodesResp.Nodes[id]; ok {
					screenshotNodes[id] = nd.Document.Name
				}
			}
		} else {
			// Full-file: use the document root's first-level pages/frames.
			screenshotNodes[fileResp.Document.ID] = fileResp.Document.Name
		}

		yellow.Printf("\n🖼️  Capturing design screenshot to %s... ", screenshotName)
		screenshotResult, err := imager.ExportImages(client, fileKey, screenshotNodes, imager.ExportConfig{
			Format:    config.Format,
			Scales:    []float64{1},
			OutputDir: config.OutputDir,
		})
		if err != nil {
			red.Printf("✗\n")
			yellow.Printf("  ⚠ Screenshot failed: %v\n", err)
		} else {
			green.Printf("✓\n")
			// Rename the exported file to the fixed screenshot name.
			for _, asset := range screenshotResult.Assets {
				oldPath := filepath.Join(config.OutputDir, asset.FileName)
				newPath := filepath.Join(config.OutputDir, screenshotName)
				if err := os.Rename(oldPath, newPath); err != nil {
					yellow.Printf("  ⚠ Could not rename screenshot: %v\n", err)
					// Keep the original name.
					specs.ExportedAssets = append(specs.ExportedAssets, extractor.ExportedAssetInfo{
						NodeName:     asset.NodeName,
						FileName:     asset.FileName,
						Format:       asset.Format,
						Scale:        asset.Scale,
						IsScreenshot: true,
					})
				} else {
					specs.ExportedAssets = append(specs.ExportedAssets, extractor.ExportedAssetInfo{
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
		// Exclude the target root nodes since they were already rendered as screenshots.
		exportNodes := make(map[string]string) // nodeID -> nodeName

		if len(targetNodeIDs) > 0 {
			// Node-specific mode: walk children to find nodes with ExportSettings.
			yellow.Print("🖼️  Discovering exportable child nodes... ")
			for _, id := range targetNodeIDs {
				if nd, ok := nodesResp.Nodes[id]; ok {
					childExport := imager.CollectExportableNodes(&nd.Document)
					for cID, cName := range childExport {
						// Skip the root node(s) — already captured as screenshot.
						if _, isRoot := screenshotNodes[cID]; isRoot {
							continue
						}
						exportNodes[cID] = cName
					}
				}
			}
			if len(exportNodes) == 0 {
				yellow.Println("no additional exportable child nodes")
			} else {
				green.Printf("✓ Found %d exportable child node(s)\n", len(exportNodes))
			}
		} else {
			// Full-file mode: discover nodes with exportSettings.
			yellow.Print("🖼️  Discovering exportable nodes... ")
			exportNodes = imager.CollectExportableNodes(&fileResp.Document)
			// Remove root if present.
			delete(exportNodes, fileResp.Document.ID)
			if len(exportNodes) == 0 {
				yellow.Println("no additional exportable nodes")
			} else {
				green.Printf("✓ Found %d exportable node(s)\n", len(exportNodes))
			}
		}

		if len(exportNodes) > 0 {
			yellow.Printf("🖼️  Exporting rendered images to %s... ", imageDir)
			result, err := imager.ExportImages(client, fileKey, exportNodes, config)
			if err != nil {
				red.Printf("✗\n")
				red.Printf("Error: %v\n", err)
				os.Exit(1)
			}
			green.Printf("✓ Exported %d image(s)\n", len(result.Assets))

			for _, dlErr := range result.Errors {
				yellow.Printf("  ⚠ %v\n", dlErr)
			}

			for _, asset := range result.Assets {
				specs.ExportedAssets = append(specs.ExportedAssets, extractor.ExportedAssetInfo{
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
			allImageFills = append(allImageFills, imager.CollectImageFillNodes(root)...)
		}

		if len(allImageFills) > 0 {
			// Try file images API first for embedded image download URLs.
			yellow.Printf("🖼️  Found %d embedded image(s), fetching download URLs... ", len(allImageFills))
			var unresolvedNodes []imager.ImageFillNode

			fileImagesResp, err := client.GetFileImages(fileKey)
			if err != nil {
				red.Printf("✗\n")
				yellow.Printf("  ⚠ File images API failed: %v\n", err)
				// All nodes are unresolved; will fall back to render API.
				unresolvedNodes = allImageFills
			} else {
				green.Println("✓")
				yellow.Printf("🖼️  Downloading embedded images to %s... ", imageDir)
				fillResult, err := imager.ExportImageFills(fileImagesResp, allImageFills, config)
				if err != nil {
					red.Printf("✗\n")
					red.Printf("Error: %v\n", err)
					os.Exit(1)
				}

				if len(fillResult.Assets) > 0 {
					green.Printf("✓ Exported %d embedded image(s)\n", len(fillResult.Assets))
				}

				for _, dlErr := range fillResult.Errors {
					yellow.Printf("  ⚠ %v\n", dlErr)
				}

				for _, asset := range fillResult.Assets {
					specs.ExportedAssets = append(specs.ExportedAssets, extractor.ExportedAssetInfo{
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
				yellow.Printf("🖼️  Rendering %d image(s) via render API (no file image URLs)... ", len(unresolvedNodes))
				renderNodes := imager.ImageFillNodesToMap(unresolvedNodes)
				renderResult, err := imager.ExportImages(client, fileKey, renderNodes, config)
				if err != nil {
					red.Printf("✗\n")
					red.Printf("Error rendering images: %v\n", err)
					// Non-fatal: continue.
				} else {
					green.Printf("✓ Rendered %d image(s)\n", len(renderResult.Assets))

					for _, dlErr := range renderResult.Errors {
						yellow.Printf("  ⚠ %v\n", dlErr)
					}

					for _, asset := range renderResult.Assets {
						specs.ExportedAssets = append(specs.ExportedAssets, extractor.ExportedAssetInfo{
							NodeName: asset.NodeName,
							FileName: asset.FileName,
							Format:   asset.Format,
							Scale:    asset.Scale,
						})
					}
				}
			}
		}
	}

	// Display extracted stats
	cyan.Println("\n📊 Extraction Summary:")
	fmt.Printf("  • Colors: %d primary, %d background, %d text, %d status\n",
		len(specs.Colors.Primary),
		len(specs.Colors.Background),
		len(specs.Colors.Text),
		len(specs.Colors.Status))

	if specs.Typography.FontFamily != "" {
		fmt.Printf("  • Font Family: %s\n", specs.Typography.FontFamily)
	}

	fmt.Printf("  • Font Sizes: %d\n", len(specs.Typography.FontSizes))
	fmt.Printf("  • Spacing Values: %d\n", len(specs.Spacing.Values))
	fmt.Printf("  • Border Radii: %d\n", len(specs.Radii.Values))
	fmt.Printf("  • Shadows: %d\n", len(specs.Shadows))

	if specs.Layout.HeaderHeight > 0 {
		fmt.Printf("  • Header Height: %.0fpx\n", specs.Layout.HeaderHeight)
	}
	if specs.Layout.SidebarWidth > 0 {
		fmt.Printf("  • Sidebar Width: %.0fpx\n", specs.Layout.SidebarWidth)
	}
	if len(specs.ExportedAssets) > 0 {
		fmt.Printf("  • Exported Assets: %d\n", len(specs.ExportedAssets))
	}

	// Format as markdown
	yellow.Printf("\n📝 Generating markdown documentation... ")
	markdown := formatter.ToMarkdown(specs, fileName, imageDir)
	green.Println("✓")

	// Write to file
	yellow.Printf("💾 Writing to %s... ", outputFile)
	err = os.WriteFile(outputFile, []byte(markdown), 0644)
	if err != nil {
		red.Printf("✗\n")
		red.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	green.Println("✓")

	green.Printf("\n✨ Successfully extracted design specifications to %s\n\n", outputFile)
}

// parseNodeIDsFromString parses a comma-separated string of node IDs and returns a slice.
// Trims whitespace and filters out empty strings.
func parseNodeIDsFromString(nodeIDsStr string) []string {
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

// parseScales parses a comma-separated string of scale factors into a float64 slice.
func parseScales(scalesStr string) ([]float64, error) {
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
