// Package figmaextractor extracts design specifications from Figma files
// via the Figma API and produces structured output (design tokens, colors,
// typography, spacing, shadows, assets, and a full markdown report).
//
// The CLI lives in cmd/figma-extractor; this root package exposes the same
// pipeline as a Go API so that callers can embed extraction in their own
// tools without shelling out.
//
// # Import
//
// The module path contains a hyphen but Go package names cannot, so the
// package is named figmaextractor:
//
//	import "github.com/hellenic-development/figma-extractor" // package figmaextractor
//
// # Quick start
//
//	result, err := figmaextractor.Run(figmaextractor.Options{
//	    AccessToken: os.Getenv("FIGMA_TOKEN"),
//	    FileURL:     "https://www.figma.com/design/ABC123/My-Design",
//	    ExportImages: true,
//	    ImageFormat:  "png",
//	    ImageDir:     "assets",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	os.WriteFile("design.md", []byte(result.Markdown), 0644)
//
// # Logging
//
// Pass a [Logger] implementation in [Options.Logger] to receive progress
// messages. A nil Logger silences all output.
//
//	type myLogger struct{}
//	func (l *myLogger) Infof(f string, a ...any)  { log.Printf("[INFO]  "+f, a...) }
//	func (l *myLogger) Warnf(f string, a ...any)  { log.Printf("[WARN]  "+f, a...) }
//	func (l *myLogger) Errorf(f string, a ...any) { log.Printf("[ERROR] "+f, a...) }
//
// # Node-scoped extraction
//
// To extract specific frames or components rather than the entire file,
// populate [Options.NodeIDs] or include node-id query parameters in the
// Figma URL. Set [Options.InheritFileContext] to true to include
// file-level colors and styles alongside the targeted nodes.
//
// # Image export
//
// When [Options.ExportImages] is true the pipeline captures a full design
// screenshot, exports nodes that have Figma ExportSettings, downloads
// embedded IMAGE fills, and falls back to the render API for any
// unresolved images. Duplicates are automatically removed.
package figmaextractor
