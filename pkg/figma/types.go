package figma

// FileResponse represents the complete response from the Figma file API endpoint.
// It contains the file metadata, document structure, published styles, and schema version information.
type FileResponse struct {
	Name          string           `json:"name"`
	LastModified  string           `json:"lastModified"`
	ThumbnailURL  string           `json:"thumbnailUrl"`
	Version       string           `json:"version"`
	Document      Node             `json:"document"`
	Styles        map[string]Style `json:"styles"`
	SchemaVersion int              `json:"schemaVersion"`
}

// NodesResponse represents the response from the Figma nodes API endpoint when fetching specific nodes.
// It contains file metadata and a map of node IDs to their corresponding NodeData.
type NodesResponse struct {
	Name         string              `json:"name"`
	LastModified string              `json:"lastModified"`
	Version      string              `json:"version"`
	Nodes        map[string]NodeData `json:"nodes"`
}

// NodeData wraps a node with its document structure and optional component/style information.
// This is the structure returned for each requested node in a NodesResponse.
type NodeData struct {
	Document   Node                 `json:"document"`
	Components map[string]Component `json:"components,omitempty"`
	Styles     map[string]Style     `json:"styles,omitempty"`
}

// Component represents a Figma component definition with its metadata.
// Components are reusable design elements that can be instantiated throughout the file.
type Component struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// StylesResponse represents the response from the Figma styles API endpoint.
// It includes metadata about all published styles in the file and their detailed information.
type StylesResponse struct {
	Meta   Meta             `json:"meta"`
	Styles map[string]Style `json:"styles"`
}

// Meta contains metadata about published styles in a Figma file.
// This includes a list of all style metadata entries with their keys, names, and types.
type Meta struct {
	Styles []StyleMetadata `json:"styles"`
}

// StyleMetadata contains metadata for a single published style in Figma.
// It includes the unique key, file reference, node ID, style type (FILL, TEXT, EFFECT, or GRID), name, and description.
type StyleMetadata struct {
	Key         string `json:"key"`
	FileKey     string `json:"file_key"`
	NodeID      string `json:"node_id"`
	StyleType   string `json:"style_type"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Style represents a published Figma style with its basic properties.
// Styles can be colors (FILL), text styles (TEXT), effects (EFFECT), or layout grids (GRID).
type Style struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	StyleType   string `json:"style_type"`
}

// Node represents a single element in the Figma document tree hierarchy.
// Nodes can be frames, groups, text, shapes, or other Figma elements, each with their own properties
// such as fills, strokes, effects, layout settings, and children nodes.
type Node struct {
	ID                    string            `json:"id"`
	Name                  string            `json:"name"`
	Type                  string            `json:"type"`
	Children              []Node            `json:"children,omitempty"`
	BackgroundColor       *Color            `json:"backgroundColor,omitempty"`
	Fills                 []Paint           `json:"fills,omitempty"`
	Strokes               []Paint           `json:"strokes,omitempty"`
	StrokeWeight          float64           `json:"strokeWeight,omitempty"`
	CornerRadius          float64           `json:"cornerRadius,omitempty"`
	Effects               []Effect          `json:"effects,omitempty"`
	Characters            string            `json:"characters,omitempty"`
	Style                 *TypeStyle        `json:"style,omitempty"`
	AbsoluteBoundingBox   *Rectangle        `json:"absoluteBoundingBox,omitempty"`
	Constraints           *LayoutConstraint `json:"constraints,omitempty"`
	LayoutMode            string            `json:"layoutMode,omitempty"`
	PrimaryAxisSizingMode string            `json:"primaryAxisSizingMode,omitempty"`
	CounterAxisSizingMode string            `json:"counterAxisSizingMode,omitempty"`
	PaddingLeft           float64           `json:"paddingLeft,omitempty"`
	PaddingRight          float64           `json:"paddingRight,omitempty"`
	PaddingTop            float64           `json:"paddingTop,omitempty"`
	PaddingBottom         float64           `json:"paddingBottom,omitempty"`
	ItemSpacing           float64           `json:"itemSpacing,omitempty"`
}

// Color represents an RGBA color with float values ranging from 0 to 1.
// The R, G, B, and A (alpha/opacity) values must be converted to 0-255 range for standard use.
type Color struct {
	R float64 `json:"r"`
	G float64 `json:"g"`
	B float64 `json:"b"`
	A float64 `json:"a"`
}

// Paint represents a fill or stroke applied to a Figma node.
// It includes the paint type (SOLID, GRADIENT_LINEAR, etc.), visibility, opacity, and color information.
type Paint struct {
	Type    string  `json:"type"`
	Visible bool    `json:"visible"`
	Opacity float64 `json:"opacity"`
	Color   *Color  `json:"color,omitempty"`
}

// Effect represents a visual effect applied to a Figma node such as drop shadows, inner shadows, or blur effects.
// It includes positioning (offset), blur radius, spread, color, and blend mode settings.
type Effect struct {
	Type      string  `json:"type"`
	Visible   bool    `json:"visible"`
	Radius    float64 `json:"radius,omitempty"`
	Color     *Color  `json:"color,omitempty"`
	Offset    *Vector `json:"offset,omitempty"`
	Spread    float64 `json:"spread,omitempty"`
	BlendMode string  `json:"blendMode,omitempty"`
}

// Vector represents a 2D coordinate or offset with X and Y values.
// Used for positioning effects like shadows and other spatial properties.
type Vector struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// TypeStyle represents comprehensive text styling properties from Figma.
// It includes font family, weight, size, line height, letter spacing, and text alignment settings.
type TypeStyle struct {
	FontFamily          string  `json:"fontFamily"`
	FontPostScriptName  string  `json:"fontPostScriptName"`
	FontWeight          float64 `json:"fontWeight"`
	FontSize            float64 `json:"fontSize"`
	LineHeightPx        float64 `json:"lineHeightPx"`
	LineHeightPercent   float64 `json:"lineHeightPercent"`
	LetterSpacing       float64 `json:"letterSpacing"`
	TextAlignHorizontal string  `json:"textAlignHorizontal"`
	TextAlignVertical   string  `json:"textAlignVertical"`
}

// Rectangle represents a bounding box with position (X, Y) and dimensions (Width, Height).
// Used to define the absolute position and size of nodes in the Figma canvas.
type Rectangle struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// LayoutConstraint defines how a node's position and size behave when its parent is resized.
// Constraints can be set for both vertical (TOP, BOTTOM, CENTER, etc.) and horizontal directions.
type LayoutConstraint struct {
	Vertical   string `json:"vertical"`
	Horizontal string `json:"horizontal"`
}
