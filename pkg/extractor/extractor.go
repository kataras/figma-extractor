package extractor

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/hellenic-development/figma-extractor/pkg/figma"
)

// DesignSpecs represents the complete set of design specifications extracted from a Figma file.
// It includes color palettes, typography settings, spacing values, shadows, border radii, layout measurements,
// and optionally exported image assets.
type DesignSpecs struct {
	Colors         ColorPalette
	Typography     Typography
	Spacing        Spacing
	Shadows        []Shadow
	Radii          BorderRadii
	Layout         LayoutSpecs
	ExportedAssets []ExportedAssetInfo
	NodeTree       []*NodeDescription
}

// ExportedAssetInfo represents metadata about an exported image asset.
type ExportedAssetInfo struct {
	NodeID       string // Figma node ID this asset was exported from
	NodeName     string
	FileName     string
	Format       string
	Scale        float64
	IsScreenshot bool // true for the complete design screenshot of the target node(s)
}

// NodeDescription describes a single node in the Figma design hierarchy with its visual properties.
type NodeDescription struct {
	ID   string
	Name string
	Type string // FRAME, TEXT, RECTANGLE, COMPONENT, INSTANCE, GROUP, etc.

	// Dimensions
	Width, Height float64

	// Visual
	FillColors   []string // hex from SOLID fills
	ImageFills   []string // imageRef values from IMAGE fills
	StrokeColors []string
	StrokeWeight float64
	CornerRadius float64

	// Text (TEXT nodes only)
	TextContent         string
	FontFamily          string
	FontSize            float64
	FontWeight          float64
	LineHeightPx        float64
	TextAlignHorizontal string

	// Layout (auto-layout)
	LayoutMode                                         string // "HORIZONTAL", "VERTICAL", ""
	PaddingTop, PaddingRight, PaddingBottom, PaddingLeft float64
	ItemSpacing                                        float64

	// Effects
	Shadows []Shadow

	// Linked exported assets (populated after image export)
	ExportedAssets []ExportedAssetInfo

	// Recursive children
	Children []*NodeDescription
}

// ColorPalette organizes colors into semantic categories for easier reference and usage.
// Colors are categorized as Primary, Secondary, Background, Text, Status (success/error/warning), and Border colors.
type ColorPalette struct {
	Primary    map[string]string
	Secondary  map[string]string
	Background map[string]string
	Text       map[string]string
	Status     map[string]string
	Border     map[string]string
}

// Typography holds all font-related specifications including font family, sizes, weights, and line heights.
// Font sizes and other values are normalized to a standard scale for consistency across the design system.
type Typography struct {
	FontFamily  string
	FontSizes   map[string]float64
	FontWeights map[string]float64
	LineHeights map[string]float64
}

// Spacing defines the spacing scale used throughout the design.
// Values are normalized to a standard scale, typically in multiples of 4 pixels for consistency.
type Spacing struct {
	Values map[string]float64
}

// Shadow represents a visual shadow effect with its positioning, blur, spread, and color properties.
// Supports both DROP_SHADOW and INNER_SHADOW types from Figma.
type Shadow struct {
	Name   string
	Type   string
	X      float64
	Y      float64
	Blur   float64
	Spread float64
	Color  string
}

// BorderRadii defines the border radius values used in the design system.
// Values are normalized to standard sizes (sm, md, lg, xl, 2xl) for consistent rounded corners.
type BorderRadii struct {
	Values map[string]float64
}

// LayoutSpecs captures common layout dimensions such as header heights, sidebar widths, and content padding.
// These measurements are automatically detected from nodes with relevant names in the Figma file.
type LayoutSpecs struct {
	HeaderHeight   float64
	SidebarWidth   float64
	ContentPadding float64
}

// Extract analyzes a Figma file response and extracts all design specifications including colors,
// typography, spacing, shadows, border radii, and layout measurements. The extracted values are
// normalized and deduplicated for consistency in the final design system.
func Extract(fileResp *figma.FileResponse) *DesignSpecs {
	specs := &DesignSpecs{
		Colors: ColorPalette{
			Primary:    make(map[string]string),
			Secondary:  make(map[string]string),
			Background: make(map[string]string),
			Text:       make(map[string]string),
			Status:     make(map[string]string),
			Border:     make(map[string]string),
		},
		Typography: Typography{
			FontSizes:   make(map[string]float64),
			FontWeights: make(map[string]float64),
			LineHeights: make(map[string]float64),
		},
		Spacing: Spacing{
			Values: make(map[string]float64),
		},
		Radii: BorderRadii{
			Values: make(map[string]float64),
		},
		Shadows: []Shadow{},
		Layout:  LayoutSpecs{},
	}

	// Extract colors, typography, and other specs
	extractFromNode(&fileResp.Document, specs)

	// Build hierarchical node tree
	specs.NodeTree = []*NodeDescription{buildNodeTree(&fileResp.Document)}

	// Normalize and categorize extracted values
	normalizeSpecs(specs)

	return specs
}

// ExtractNodes analyzes specific nodes from a Figma file and extracts their design specifications.
// This is more efficient than extracting the entire file when you only need specific elements.
//
// Parameters:
//   - fileResp: The complete file response for accessing file-level metadata and styles
//   - nodesResp: The nodes response containing the specific nodes to extract
//   - nodeIDs: List of node IDs being extracted (for validation and reporting)
//   - inheritFileContext: If true, includes file-level context (document root colors, styles);
//     if false, extracts only from the specified nodes
//
// Returns a DesignSpecs containing specifications from the target nodes, optionally merged with file-level context.
func ExtractNodes(fileResp *figma.FileResponse, nodesResp *figma.NodesResponse, nodeIDs []string, inheritFileContext bool) *DesignSpecs {
	specs := &DesignSpecs{
		Colors: ColorPalette{
			Primary:    make(map[string]string),
			Secondary:  make(map[string]string),
			Background: make(map[string]string),
			Text:       make(map[string]string),
			Status:     make(map[string]string),
			Border:     make(map[string]string),
		},
		Typography: Typography{
			FontSizes:   make(map[string]float64),
			FontWeights: make(map[string]float64),
			LineHeights: make(map[string]float64),
		},
		Spacing: Spacing{
			Values: make(map[string]float64),
		},
		Radii: BorderRadii{
			Values: make(map[string]float64),
		},
		Shadows: []Shadow{},
		Layout:  LayoutSpecs{},
	}

	// Optionally extract file-level context from the document root
	// This includes published styles, global colors, and typography definitions
	if inheritFileContext {
		extractFileContext(&fileResp.Document, specs)
	}

	// Extract specifications from each target node
	for _, nodeID := range nodeIDs {
		if nodeData, exists := nodesResp.Nodes[nodeID]; exists {
			extractFromNode(&nodeData.Document, specs)
		}
	}

	// Build hierarchical node tree for each target node
	for _, nodeID := range nodeIDs {
		if nodeData, exists := nodesResp.Nodes[nodeID]; exists {
			specs.NodeTree = append(specs.NodeTree, buildNodeTree(&nodeData.Document))
		}
	}

	// Normalize and categorize extracted values (deduplicates automatically)
	normalizeSpecs(specs)

	return specs
}

// extractFileContext extracts file-level design system context from the document root and its immediate children.
// This includes document-level colors, styles, and typography that should be preserved even when
// extracting specific nodes. It processes the root node and its direct children (typically pages/frames
// that contain design system definitions), but doesn't recurse deeper to avoid extracting the entire file.
func extractFileContext(node *figma.Node, specs *DesignSpecs) {
	// Extract properties from the document root itself
	extractNodeProperties(node, specs)

	// Also process immediate children (one level deep)
	// These often contain style pages, color palettes, or design system definitions
	for _, child := range node.Children {
		extractNodeProperties(&child, specs)
	}
}

// extractNodeProperties extracts design properties from a single node without recursing.
// This is used by extractFileContext to gather file-level context without processing entire subtrees.
func extractNodeProperties(node *figma.Node, specs *DesignSpecs) {
	// Extract background colors
	if node.BackgroundColor != nil {
		colorHex := colorToHex(node.BackgroundColor)
		specs.Colors.Background[node.Name] = colorHex
	}

	// Extract colors from fills
	for _, fill := range node.Fills {
		if fill.Type == "SOLID" && fill.Color != nil && fill.Visible {
			colorHex := colorToHex(fill.Color)
			categorizeColor(node.Name, colorHex, specs)
		}
	}

	// Extract colors from strokes
	for _, stroke := range node.Strokes {
		if stroke.Type == "SOLID" && stroke.Color != nil && stroke.Visible {
			colorHex := colorToHex(stroke.Color)
			specs.Colors.Border[node.Name] = colorHex
		}
	}

	// Extract typography
	if node.Style != nil {
		if node.Style.FontFamily != "" && specs.Typography.FontFamily == "" {
			specs.Typography.FontFamily = node.Style.FontFamily
		}
		if node.Style.FontSize > 0 {
			specs.Typography.FontSizes[node.Name] = node.Style.FontSize
		}
		if node.Style.FontWeight > 0 {
			specs.Typography.FontWeights[node.Name] = node.Style.FontWeight
		}
		if node.Style.LineHeightPx > 0 {
			specs.Typography.LineHeights[node.Name] = node.Style.LineHeightPx
		}
	}

	// Extract shadows
	for _, effect := range node.Effects {
		if (effect.Type == "DROP_SHADOW" || effect.Type == "INNER_SHADOW") && effect.Visible {
			shadow := Shadow{
				Name:   node.Name,
				Type:   effect.Type,
				X:      effect.Offset.X,
				Y:      effect.Offset.Y,
				Blur:   effect.Radius,
				Spread: effect.Spread,
				Color:  colorToHex(effect.Color),
			}
			specs.Shadows = append(specs.Shadows, shadow)
		}
	}

	// Extract border radii
	if node.CornerRadius > 0 {
		specs.Radii.Values[node.Name] = node.CornerRadius
	}
}

// extractFromNode recursively traverses the Figma document tree and extracts design specifications
// from each node. It processes fills, strokes, background colors, typography, shadows, border radii,
// spacing from layout properties, and layout dimensions.
func extractFromNode(node *figma.Node, specs *DesignSpecs) {
	// Extract colors from fills
	for _, fill := range node.Fills {
		if fill.Type == "SOLID" && fill.Color != nil && fill.Visible {
			colorHex := colorToHex(fill.Color)
			categorizeColor(node.Name, colorHex, specs)
		}
	}

	// Extract colors from strokes
	for _, stroke := range node.Strokes {
		if stroke.Type == "SOLID" && stroke.Color != nil && stroke.Visible {
			colorHex := colorToHex(stroke.Color)
			specs.Colors.Border[node.Name] = colorHex
		}
	}

	// Extract background colors
	if node.BackgroundColor != nil {
		colorHex := colorToHex(node.BackgroundColor)
		specs.Colors.Background[node.Name] = colorHex
	}

	// Extract typography
	if node.Style != nil {
		if node.Style.FontFamily != "" && specs.Typography.FontFamily == "" {
			specs.Typography.FontFamily = node.Style.FontFamily
		}
		if node.Style.FontSize > 0 {
			specs.Typography.FontSizes[node.Name] = node.Style.FontSize
		}
		if node.Style.FontWeight > 0 {
			specs.Typography.FontWeights[node.Name] = node.Style.FontWeight
		}
		if node.Style.LineHeightPx > 0 {
			specs.Typography.LineHeights[node.Name] = node.Style.LineHeightPx
		}
	}

	// Extract shadows
	for _, effect := range node.Effects {
		if (effect.Type == "DROP_SHADOW" || effect.Type == "INNER_SHADOW") && effect.Visible {
			shadow := Shadow{
				Name:   node.Name,
				Type:   effect.Type,
				X:      effect.Offset.X,
				Y:      effect.Offset.Y,
				Blur:   effect.Radius,
				Spread: effect.Spread,
				Color:  colorToHex(effect.Color),
			}
			specs.Shadows = append(specs.Shadows, shadow)
		}
	}

	// Extract border radii
	if node.CornerRadius > 0 {
		specs.Radii.Values[node.Name] = node.CornerRadius
	}

	// Extract spacing from layout properties
	if node.PaddingLeft > 0 || node.PaddingRight > 0 || node.PaddingTop > 0 || node.PaddingBottom > 0 {
		specs.Spacing.Values[node.Name+"-paddingLeft"] = node.PaddingLeft
		specs.Spacing.Values[node.Name+"-paddingRight"] = node.PaddingRight
		specs.Spacing.Values[node.Name+"-paddingTop"] = node.PaddingTop
		specs.Spacing.Values[node.Name+"-paddingBottom"] = node.PaddingBottom
	}

	if node.ItemSpacing > 0 {
		specs.Spacing.Values[node.Name+"-itemSpacing"] = node.ItemSpacing
	}

	// Extract layout dimensions
	if node.AbsoluteBoundingBox != nil {
		name := strings.ToLower(node.Name)
		if strings.Contains(name, "header") {
			specs.Layout.HeaderHeight = node.AbsoluteBoundingBox.Height
		}
		if strings.Contains(name, "sidebar") {
			specs.Layout.SidebarWidth = node.AbsoluteBoundingBox.Width
		}
	}

	// Recursively process children
	for _, child := range node.Children {
		extractFromNode(&child, specs)
	}
}

// categorizeColor intelligently categorizes a color into the appropriate palette category
// (Primary, Secondary, Background, Text, Status, or Border) based on keywords in the node name.
func categorizeColor(nodeName, colorHex string, specs *DesignSpecs) {
	name := strings.ToLower(nodeName)

	if strings.Contains(name, "primary") {
		specs.Colors.Primary[nodeName] = colorHex
	} else if strings.Contains(name, "secondary") {
		specs.Colors.Secondary[nodeName] = colorHex
	} else if strings.Contains(name, "background") || strings.Contains(name, "bg") {
		specs.Colors.Background[nodeName] = colorHex
	} else if strings.Contains(name, "text") {
		specs.Colors.Text[nodeName] = colorHex
	} else if strings.Contains(name, "success") || strings.Contains(name, "error") ||
		strings.Contains(name, "warning") || strings.Contains(name, "info") {
		specs.Colors.Status[nodeName] = colorHex
	} else if strings.Contains(name, "border") {
		specs.Colors.Border[nodeName] = colorHex
	}
}

// colorToHex converts a Figma RGBA color (with 0-1 float values) to standard hexadecimal format (#RRGGBB).
// Returns "#000000" if the color is nil.
func colorToHex(color *figma.Color) string {
	if color == nil {
		return "#000000"
	}

	r := int(math.Round(color.R * 255))
	g := int(math.Round(color.G * 255))
	b := int(math.Round(color.B * 255))

	return fmt.Sprintf("#%02X%02X%02X", r, g, b)
}

// normalizeSpecs applies normalization and deduplication to all extracted specifications.
// This ensures colors are unique, font sizes follow a standard scale (xs, sm, base, lg, xl, etc.),
// spacing values align to multiples of 4, and border radii use consistent naming.
func normalizeSpecs(specs *DesignSpecs) {
	// Deduplicate colors
	specs.Colors.Primary = deduplicateColors(specs.Colors.Primary)
	specs.Colors.Secondary = deduplicateColors(specs.Colors.Secondary)
	specs.Colors.Background = deduplicateColors(specs.Colors.Background)
	specs.Colors.Text = deduplicateColors(specs.Colors.Text)
	specs.Colors.Status = deduplicateColors(specs.Colors.Status)
	specs.Colors.Border = deduplicateColors(specs.Colors.Border)

	// Normalize font sizes to a standard scale
	specs.Typography.FontSizes = normalizeFontSizes(specs.Typography.FontSizes)

	// Normalize spacing to a standard scale
	specs.Spacing.Values = normalizeSpacing(specs.Spacing.Values)

	// Normalize border radii
	specs.Radii.Values = normalizeBorderRadii(specs.Radii.Values)
}

// deduplicateColors removes duplicate color values from a color map, keeping only the first
// occurrence of each unique color. This prevents redundancy in the final color palette.
func deduplicateColors(colors map[string]string) map[string]string {
	seen := make(map[string]bool)
	result := make(map[string]string)

	for name, color := range colors {
		if !seen[color] {
			result[name] = color
			seen[color] = true
		}
	}

	return result
}

// normalizeFontSizes converts extracted font sizes to a standardized naming scale (xs, sm, base, lg, xl, 2xl, 3xl, 4xl).
// Sizes are sorted and mapped to scale names, making them easier to reference in CSS and design tokens.
func normalizeFontSizes(sizes map[string]float64) map[string]float64 {
	if len(sizes) == 0 {
		return sizes
	}

	// Get unique sizes and sort them
	uniqueSizes := make([]float64, 0)
	seen := make(map[float64]bool)

	for _, size := range sizes {
		if !seen[size] {
			uniqueSizes = append(uniqueSizes, size)
			seen[size] = true
		}
	}

	sort.Float64s(uniqueSizes)

	// Map to standard size names
	result := make(map[string]float64)
	sizeNames := []string{"xs", "sm", "base", "lg", "xl", "2xl", "3xl", "4xl"}

	for i, size := range uniqueSizes {
		if i < len(sizeNames) {
			result[sizeNames[i]] = size
		}
	}

	return result
}

// normalizeSpacing converts spacing values to a standard scale using numeric names (1, 2, 3, 4, 5, 6, 8, 10, 12, 16, 20, 24).
// This creates a consistent spacing system typically based on multiples of 4 pixels.
func normalizeSpacing(spacing map[string]float64) map[string]float64 {
	if len(spacing) == 0 {
		return spacing
	}

	// Get unique spacing values
	uniqueSpacing := make([]float64, 0)
	seen := make(map[float64]bool)

	for _, space := range spacing {
		if !seen[space] && space > 0 {
			uniqueSpacing = append(uniqueSpacing, space)
			seen[space] = true
		}
	}

	sort.Float64s(uniqueSpacing)

	// Map to standard spacing scale (multiples of 4)
	result := make(map[string]float64)
	scaleNames := []string{"1", "2", "3", "4", "5", "6", "8", "10", "12", "16", "20", "24"}

	for i, space := range uniqueSpacing {
		if i < len(scaleNames) {
			result[scaleNames[i]] = space
		}
	}

	return result
}

// normalizeBorderRadii converts border radius values to a standard scale (sm, md, lg, xl, 2xl).
// This ensures consistent rounded corner styling across the design system.
func normalizeBorderRadii(radii map[string]float64) map[string]float64 {
	if len(radii) == 0 {
		return radii
	}

	// Get unique radii values
	uniqueRadii := make([]float64, 0)
	seen := make(map[float64]bool)

	for _, radius := range radii {
		if !seen[radius] && radius > 0 {
			uniqueRadii = append(uniqueRadii, radius)
			seen[radius] = true
		}
	}

	sort.Float64s(uniqueRadii)

	// Map to standard radius names
	result := make(map[string]float64)
	radiusNames := []string{"sm", "md", "lg", "xl", "2xl"}

	for i, radius := range uniqueRadii {
		if i < len(radiusNames) {
			result[radiusNames[i]] = radius
		}
	}

	return result
}

// buildNodeTree recursively walks the Figma Node tree and builds a parallel NodeDescription tree
// containing all visual properties for each node.
func buildNodeTree(node *figma.Node) *NodeDescription {
	nd := &NodeDescription{
		ID:   node.ID,
		Name: node.Name,
		Type: node.Type,
	}

	// Dimensions
	if node.AbsoluteBoundingBox != nil {
		nd.Width = node.AbsoluteBoundingBox.Width
		nd.Height = node.AbsoluteBoundingBox.Height
	}

	// Fills
	for _, fill := range node.Fills {
		if !fill.Visible {
			continue
		}
		if fill.Type == "SOLID" && fill.Color != nil {
			nd.FillColors = append(nd.FillColors, colorToHex(fill.Color))
		}
		if fill.Type == "IMAGE" && fill.ImageRef != "" {
			nd.ImageFills = append(nd.ImageFills, fill.ImageRef)
		}
	}

	// Strokes
	for _, stroke := range node.Strokes {
		if stroke.Type == "SOLID" && stroke.Color != nil && stroke.Visible {
			nd.StrokeColors = append(nd.StrokeColors, colorToHex(stroke.Color))
		}
	}
	nd.StrokeWeight = node.StrokeWeight
	nd.CornerRadius = node.CornerRadius

	// Text properties
	if node.Type == "TEXT" {
		nd.TextContent = node.Characters
	}
	if node.Style != nil {
		nd.FontFamily = node.Style.FontFamily
		nd.FontSize = node.Style.FontSize
		nd.FontWeight = node.Style.FontWeight
		nd.LineHeightPx = node.Style.LineHeightPx
		nd.TextAlignHorizontal = node.Style.TextAlignHorizontal
	}

	// Layout
	nd.LayoutMode = node.LayoutMode
	nd.PaddingTop = node.PaddingTop
	nd.PaddingRight = node.PaddingRight
	nd.PaddingBottom = node.PaddingBottom
	nd.PaddingLeft = node.PaddingLeft
	nd.ItemSpacing = node.ItemSpacing

	// Effects (shadows)
	for _, effect := range node.Effects {
		if (effect.Type == "DROP_SHADOW" || effect.Type == "INNER_SHADOW") && effect.Visible {
			nd.Shadows = append(nd.Shadows, Shadow{
				Name:   node.Name,
				Type:   effect.Type,
				X:      effect.Offset.X,
				Y:      effect.Offset.Y,
				Blur:   effect.Radius,
				Spread: effect.Spread,
				Color:  colorToHex(effect.Color),
			})
		}
	}

	// Recurse into children
	for i := range node.Children {
		nd.Children = append(nd.Children, buildNodeTree(&node.Children[i]))
	}

	return nd
}

// AttachAssetsToNodeTree walks the NodeDescription tree and attaches exported assets
// to the nodes they were exported from, matching by NodeID.
func AttachAssetsToNodeTree(roots []*NodeDescription, assets []ExportedAssetInfo) {
	// Build nodeID -> []asset map
	assetMap := make(map[string][]ExportedAssetInfo)
	for _, a := range assets {
		if a.NodeID != "" && !a.IsScreenshot {
			assetMap[a.NodeID] = append(assetMap[a.NodeID], a)
		}
	}

	if len(assetMap) == 0 {
		return
	}

	var walk func(nd *NodeDescription)
	walk = func(nd *NodeDescription) {
		if matched, ok := assetMap[nd.ID]; ok {
			nd.ExportedAssets = append(nd.ExportedAssets, matched...)
		}
		for _, child := range nd.Children {
			walk(child)
		}
	}

	for _, root := range roots {
		walk(root)
	}
}
