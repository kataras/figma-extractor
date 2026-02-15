package imager

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/kataras/figma-extractor/pkg/figma"
)

// ExportConfig holds configuration for image export.
type ExportConfig struct {
	Format    string    // "png", "svg", "jpg", "pdf"
	Scales    []float64 // e.g., [1, 2] for raster; ignored for svg/pdf
	OutputDir string    // local directory, default "figma-assets"
}

// ExportedAsset represents a single exported image asset.
type ExportedAsset struct {
	NodeID   string
	NodeName string
	FileName string
	Format   string
	Scale    float64
}

// ExportResult holds the results of an image export operation.
type ExportResult struct {
	Assets []ExportedAsset
	Errors []error // non-fatal per-image download failures
}

const maxNodesPerRequest = 100
const maxParallelDownloads = 5

// CollectExportableNodes walks the Figma node tree and returns a map of nodeID -> nodeName
// for nodes that have ExportSettings defined by the designer.
func CollectExportableNodes(root *figma.Node) map[string]string {
	nodes := make(map[string]string)
	collectExportable(root, nodes)
	return nodes
}

func collectExportable(node *figma.Node, nodes map[string]string) {
	if len(node.ExportSettings) > 0 {
		nodes[node.ID] = node.Name
	}
	for i := range node.Children {
		collectExportable(&node.Children[i], nodes)
	}
}

// ExportImages orchestrates the full image export pipeline:
// creates output directory, batches API requests, downloads images concurrently.
func ExportImages(client *figma.Client, fileKey string, nodes map[string]string, config ExportConfig) (*ExportResult, error) {
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory %q: %w", config.OutputDir, err)
	}

	result := &ExportResult{}
	usedNames := make(map[string]int) // track filename collisions

	// Collect node IDs into a slice for batching.
	nodeIDs := make([]string, 0, len(nodes))
	for id := range nodes {
		nodeIDs = append(nodeIDs, id)
	}

	// Determine effective scales: for SVG/PDF, always use scale 1.
	scales := config.Scales
	if config.Format == "svg" || config.Format == "pdf" {
		scales = []float64{1}
	}

	for _, scale := range scales {
		// Batch node IDs (max 100 per API request).
		for i := 0; i < len(nodeIDs); i += maxNodesPerRequest {
			end := i + maxNodesPerRequest
			if end > len(nodeIDs) {
				end = len(nodeIDs)
			}
			batch := nodeIDs[i:end]

			imgResp, err := client.GetImages(fileKey, batch, config.Format, scale)
			if err != nil {
				return nil, fmt.Errorf("failed to get images from Figma API: %w", err)
			}

			// Download images concurrently with a semaphore.
			var wg sync.WaitGroup
			sem := make(chan struct{}, maxParallelDownloads)
			var mu sync.Mutex

			for nodeID, imageURL := range imgResp.Images {
				if imageURL == "" {
					mu.Lock()
					result.Errors = append(result.Errors, fmt.Errorf("no image URL returned for node %s", nodeID))
					mu.Unlock()
					continue
				}

				wg.Add(1)
				go func(nID, url string) {
					defer wg.Done()
					sem <- struct{}{}
					defer func() { <-sem }()

					nodeName := nodes[nID]
					fileName := buildFileName(nodeName, nID, config.Format, scale)

					// Deduplicate filenames.
					mu.Lock()
					if count, exists := usedNames[fileName]; exists {
						ext := filepath.Ext(fileName)
						base := strings.TrimSuffix(fileName, ext)
						fileName = fmt.Sprintf("%s-%d%s", base, count+1, ext)
						usedNames[fileName] = count + 1
					} else {
						usedNames[fileName] = 1
					}
					mu.Unlock()

					destPath := filepath.Join(config.OutputDir, fileName)
					if err := downloadFile(url, destPath); err != nil {
						mu.Lock()
						result.Errors = append(result.Errors, fmt.Errorf("failed to download %s: %w", nodeName, err))
						mu.Unlock()
						return
					}

					mu.Lock()
					result.Assets = append(result.Assets, ExportedAsset{
						NodeID:   nID,
						NodeName: nodeName,
						FileName: fileName,
						Format:   config.Format,
						Scale:    scale,
					})
					mu.Unlock()
				}(nodeID, imageURL)
			}

			wg.Wait()
		}
	}

	return result, nil
}

// downloadFile performs an HTTP GET and saves the response body to destPath.
func downloadFile(url, destPath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("HTTP GET failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d downloading image", resp.StatusCode)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file %q: %w", destPath, err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("failed to write file %q: %w", destPath, err)
	}

	return nil
}

// buildFileName creates a sanitized filename from a node name.
// Uses kebab-case, adds @2x/@3x suffix for raster scales > 1,
// falls back to sanitized node ID if name is empty.
func buildFileName(nodeName, nodeID, format string, scale float64) string {
	name := nodeName
	if name == "" {
		name = nodeID
	}

	name = toKebabCase(name)
	if name == "" {
		name = "asset"
	}

	// Add scale suffix for raster formats with scale > 1.
	scaleSuffix := ""
	if scale > 1 && format != "svg" && format != "pdf" {
		scaleSuffix = fmt.Sprintf("@%gx", scale)
	}

	return fmt.Sprintf("%s%s.%s", name, scaleSuffix, format)
}

// toKebabCase converts a string to kebab-case format (lowercase with hyphens).
func toKebabCase(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")

	var result strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}

	return result.String()
}
