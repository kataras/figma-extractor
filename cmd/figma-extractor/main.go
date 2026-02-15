package main

import (
	"fmt"
	"os"

	figmaextractor "github.com/kataras/figma-extractor"
	"github.com/kataras/figma-extractor/pkg/figma"

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
	componentTree      bool
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
	rootCmd.Flags().BoolVar(&componentTree, "component-tree", false, "Include hierarchical component tree in output")

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
	red := color.New(color.FgRed)
	cyan := color.New(color.FgCyan)

	cyan.Println("\n🎨 Figma Design Extractor")
	cyan.Println("==========================")
	cyan.Println()

	// Parse scales from CLI string.
	scales, err := figmaextractor.ParseScales(imageScales)
	if err != nil {
		red.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Parse node IDs from CLI string.
	var parsedNodeIDs []string
	if nodeIDs != "" {
		parsedNodeIDs = figmaextractor.ParseNodeIDs(nodeIDs)
	}

	opts := figmaextractor.Options{
		AccessToken:        accessToken,
		FileURL:            figmaURL,
		NodeIDs:            parsedNodeIDs,
		InheritFileContext: inheritFileContext,
		ExportImages:       exportImages,
		ImageFormat:        imageFormat,
		ImageScales:        scales,
		ImageDir:           imageDir,
		ComponentTree:      componentTree,
		Logger:             &cliLogger{},
	}

	result, err := figmaextractor.Run(opts)
	if err != nil {
		red.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Display extracted stats.
	specs := result.Specs
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

	// Write markdown to file.
	green.Printf("\n💾 Writing to %s... ", outputFile)
	err = os.WriteFile(outputFile, []byte(result.Markdown), 0644)
	if err != nil {
		red.Printf("✗\n")
		red.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	green.Println("✓")

	green.Printf("\n✨ Successfully extracted design specifications to %s\n\n", outputFile)
}

// cliLogger implements figmaextractor.Logger with colored terminal output.
type cliLogger struct{}

func (l *cliLogger) Infof(format string, args ...any) {
	color.New(color.FgYellow).Printf(format+"\n", args...)
}

func (l *cliLogger) Warnf(format string, args ...any) {
	color.New(color.FgYellow).Printf("⚠ "+format+"\n", args...)
}

func (l *cliLogger) Errorf(format string, args ...any) {
	color.New(color.FgRed).Printf("✗ "+format+"\n", args...)
}
