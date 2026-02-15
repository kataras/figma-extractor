package main

import (
	"fmt"
	"os"
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

		// Determine which nodes to export
		exportNodes := make(map[string]string) // nodeID -> nodeName

		if len(targetNodeIDs) > 0 {
			// Use the target node IDs; get names from nodesResp
			for _, id := range targetNodeIDs {
				name := id
				if nd, ok := nodesResp.Nodes[id]; ok {
					name = nd.Document.Name
				}
				exportNodes[id] = name
			}
		} else {
			// Full-file mode: discover nodes with exportSettings
			yellow.Print("\n🖼️  Discovering exportable nodes... ")
			exportNodes = imager.CollectExportableNodes(&fileResp.Document)
			if len(exportNodes) == 0 {
				yellow.Println("No nodes with export settings found, skipping image export")
			} else {
				green.Printf("✓ Found %d exportable node(s)\n", len(exportNodes))
			}
		}

		if len(exportNodes) > 0 {
			yellow.Printf("🖼️  Exporting images to %s... ", imageDir)
			config := imager.ExportConfig{
				Format:    imageFormat,
				Scales:    scales,
				OutputDir: imageDir,
			}

			result, err := imager.ExportImages(client, fileKey, exportNodes, config)
			if err != nil {
				red.Printf("✗\n")
				red.Printf("Error: %v\n", err)
				os.Exit(1)
			}
			green.Printf("✓ Exported %d image(s)\n", len(result.Assets))

			// Log warnings for failed downloads
			for _, dlErr := range result.Errors {
				yellow.Printf("  ⚠ %v\n", dlErr)
			}

			// Populate specs.ExportedAssets
			for _, asset := range result.Assets {
				specs.ExportedAssets = append(specs.ExportedAssets, extractor.ExportedAssetInfo{
					NodeName: asset.NodeName,
					FileName: asset.FileName,
					Format:   asset.Format,
					Scale:    asset.Scale,
				})
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
	markdown := formatter.ToMarkdown(specs, fileName)
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
