package formatter

import (
	"fmt"
	"strings"

	"github.com/kataras/figma-extractor/pkg/extractor"
)

// ToMarkdown transforms extracted design specifications into a well-formatted markdown document.
// The output includes CSS variable definitions for colors, typography, spacing, shadows, border radii,
// and layout specifications, ready to be integrated into a design system or CSS framework.
func ToMarkdown(specs *extractor.DesignSpecs, fileName string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# Figma Design Specifications - %s\n\n", fileName))
	sb.WriteString("This document contains the complete design specifications extracted from the Figma file.\n\n")

	// Colors
	sb.WriteString("## Design System\n\n")
	sb.WriteString("### Color Palette\n\n")
	sb.WriteString("```css\n")

	if len(specs.Colors.Primary) > 0 {
		sb.WriteString("/* Primary Colors */\n")
		for name, color := range specs.Colors.Primary {
			cssName := toKebabCase(name)
			sb.WriteString(fmt.Sprintf("--color-primary-%s: %s;\n", cssName, color))
		}
		sb.WriteString("\n")
	}

	if len(specs.Colors.Secondary) > 0 {
		sb.WriteString("/* Secondary Colors */\n")
		for name, color := range specs.Colors.Secondary {
			cssName := toKebabCase(name)
			sb.WriteString(fmt.Sprintf("--color-secondary-%s: %s;\n", cssName, color))
		}
		sb.WriteString("\n")
	}

	if len(specs.Colors.Background) > 0 {
		sb.WriteString("/* Background Colors */\n")
		for name, color := range specs.Colors.Background {
			cssName := toKebabCase(name)
			sb.WriteString(fmt.Sprintf("--color-bg-%s: %s;\n", cssName, color))
		}
		sb.WriteString("\n")
	}

	if len(specs.Colors.Text) > 0 {
		sb.WriteString("/* Text Colors */\n")
		for name, color := range specs.Colors.Text {
			cssName := toKebabCase(name)
			sb.WriteString(fmt.Sprintf("--color-text-%s: %s;\n", cssName, color))
		}
		sb.WriteString("\n")
	}

	if len(specs.Colors.Status) > 0 {
		sb.WriteString("/* Status Colors */\n")
		for name, color := range specs.Colors.Status {
			cssName := toKebabCase(name)
			sb.WriteString(fmt.Sprintf("--color-%s: %s;\n", cssName, color))
		}
		sb.WriteString("\n")
	}

	if len(specs.Colors.Border) > 0 {
		sb.WriteString("/* Border Colors */\n")
		for name, color := range specs.Colors.Border {
			cssName := toKebabCase(name)
			sb.WriteString(fmt.Sprintf("--color-border-%s: %s;\n", cssName, color))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("```\n\n")

	// Typography
	sb.WriteString("### Typography\n\n")
	sb.WriteString("```css\n")

	if specs.Typography.FontFamily != "" {
		sb.WriteString(fmt.Sprintf("/* Font Family */\n--font-primary: '%s', system-ui, -apple-system, sans-serif;\n\n", specs.Typography.FontFamily))
	}

	if len(specs.Typography.FontSizes) > 0 {
		sb.WriteString("/* Font Sizes */\n")
		for name, size := range specs.Typography.FontSizes {
			sb.WriteString(fmt.Sprintf("--text-%s: %.0fpx;\n", name, size))
		}
		sb.WriteString("\n")
	}

	if len(specs.Typography.FontWeights) > 0 {
		sb.WriteString("/* Font Weights */\n")
		for name, weight := range specs.Typography.FontWeights {
			sb.WriteString(fmt.Sprintf("--font-%s: %.0f;\n", toKebabCase(name), weight))
		}
		sb.WriteString("\n")
	}

	if len(specs.Typography.LineHeights) > 0 {
		sb.WriteString("/* Line Heights */\n")
		for name, height := range specs.Typography.LineHeights {
			sb.WriteString(fmt.Sprintf("--leading-%s: %.0fpx;\n", toKebabCase(name), height))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("```\n\n")

	// Spacing
	if len(specs.Spacing.Values) > 0 {
		sb.WriteString("### Spacing\n\n")
		sb.WriteString("```css\n")
		sb.WriteString("/* Spacing Scale */\n")
		for name, value := range specs.Spacing.Values {
			sb.WriteString(fmt.Sprintf("--space-%s: %.0fpx;\n", name, value))
		}
		sb.WriteString("```\n\n")
	}

	// Border Radii
	if len(specs.Radii.Values) > 0 {
		sb.WriteString("### Border Radius\n\n")
		sb.WriteString("```css\n")
		for name, radius := range specs.Radii.Values {
			sb.WriteString(fmt.Sprintf("--radius-%s: %.0fpx;\n", name, radius))
		}
		sb.WriteString("--radius-full: 9999px; /* Full radius (circles) */\n")
		sb.WriteString("```\n\n")
	}

	// Shadows
	if len(specs.Shadows) > 0 {
		sb.WriteString("### Shadows\n\n")
		sb.WriteString("```css\n")
		for i, shadow := range specs.Shadows {
			shadowName := toKebabCase(shadow.Name)
			if shadowName == "" {
				shadowName = fmt.Sprintf("shadow-%d", i+1)
			}

			shadowValue := fmt.Sprintf("%.0fpx %.0fpx %.0fpx", shadow.X, shadow.Y, shadow.Blur)
			if shadow.Spread > 0 {
				shadowValue += fmt.Sprintf(" %.0fpx", shadow.Spread)
			}
			shadowValue += fmt.Sprintf(" %s", shadow.Color)

			sb.WriteString(fmt.Sprintf("--shadow-%s: %s;\n", shadowName, shadowValue))
		}
		sb.WriteString("```\n\n")
	}

	// Layout
	sb.WriteString("## Layout Specifications\n\n")
	sb.WriteString("### Main Layout\n\n")

	if specs.Layout.HeaderHeight > 0 {
		sb.WriteString(fmt.Sprintf("- **Header Height**: %.0fpx\n", specs.Layout.HeaderHeight))
	}

	if specs.Layout.SidebarWidth > 0 {
		sb.WriteString(fmt.Sprintf("- **Sidebar Width**: %.0fpx\n", specs.Layout.SidebarWidth))
	}

	if specs.Layout.ContentPadding > 0 {
		sb.WriteString(fmt.Sprintf("- **Content Padding**: %.0fpx\n", specs.Layout.ContentPadding))
	}

	sb.WriteString("\n")

	// Exported Assets
	if len(specs.ExportedAssets) > 0 {
		sb.WriteString("## Exported Assets\n\n")
		sb.WriteString("| Asset | File | Format | Scale |\n")
		sb.WriteString("|-------|------|--------|-------|\n")
		for _, asset := range specs.ExportedAssets {
			name := asset.NodeName
			if name == "" {
				name = asset.FileName
			}
			sb.WriteString(fmt.Sprintf("| %s | `%s` | %s | %gx |\n", name, asset.FileName, strings.ToUpper(asset.Format), asset.Scale))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// toKebabCase converts a string to kebab-case format (lowercase with hyphens).
// This is used for generating CSS variable names from Figma node names.
// Special characters are removed, and spaces/underscores are replaced with hyphens.
func toKebabCase(s string) string {
	// Remove special characters and replace spaces with hyphens
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")

	// Remove any non-alphanumeric characters except hyphens
	var result strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}

	return result.String()
}
