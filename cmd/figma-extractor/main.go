package main

import (
	"fmt"
	"os"

	"github.com/kataras/figma-extractor/pkg/extractor"
	"github.com/kataras/figma-extractor/pkg/figma"
	"github.com/kataras/figma-extractor/pkg/formatter"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

const version = "1.0.0"

var (
	figmaURL    string
	accessToken string
	outputFile  string
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

	// Create Figma client
	yellow.Print("🔑 Authenticating with Figma API... ")
	client := figma.NewClient(accessToken)
	green.Println("✓")

	// Fetch file data
	yellow.Print("📥 Fetching file data from Figma... ")
	fileResp, err := client.GetFile(fileKey)
	if err != nil {
		red.Printf("✗\n")
		red.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	green.Printf("✓ File: %s\n", fileResp.Name)

	// Extract design specifications
	yellow.Print("🔍 Extracting design specifications... ")
	specs := extractor.Extract(fileResp)
	green.Println("✓")

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

	// Format as markdown
	yellow.Printf("\n📝 Generating markdown documentation... ")
	markdown := formatter.ToMarkdown(specs, fileResp.Name)
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
