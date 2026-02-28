# Figma Design Extractor

A Go tool to extract design specifications from Figma files using the Figma REST API. This tool automatically extracts colors, typography, spacing, shadows, border radii, and layout specifications from any Figma file and outputs them in a markdown format that can be used to implement the design with AI.

## Features

- 🎨 **Color Extraction**: Automatically categorizes colors into primary, secondary, background, text, status, and border colors
- 📝 **Typography**: Extracts font families, sizes, weights, and line heights
- 📏 **Spacing**: Identifies spacing patterns and normalizes them to a standard scale
- 🌈 **Visual Effects**: Extracts shadows and border radii
- 📐 **Layout Specs**: Captures layout dimensions like header height and sidebar width
- 🎯 **Node-Specific Extraction**: Extract specific elements or components instead of the entire file
- 📦 **Multi-Node Support**: Extract multiple nodes in a single operation
- 🖼️ **Image/Asset Export**: Export images and assets directly from Figma (PNG, SVG, JPG, PDF) with multi-scale support
- 📄 **Markdown Output**: Generates a comprehensive markdown file with all specifications

## Installation

The only requirement is the [Go Programming Language](https://go.dev/dl/).

```bash
go install github.com/hellenic-development/figma-extractor/cmd/figma-extractor@latest
```

## Usage

### Basic Usage (Extract Entire File)

```bash
figma-extractor --url "https://www.figma.com/file/YOUR_FILE_KEY/Design-Name" --token "YOUR_ACCESS_TOKEN"
```

### Extract Specific Nodes

Extract one or more specific elements from your Figma file:

```bash
# Extract a single node by ID
figma-extractor \
  --url "https://www.figma.com/file/YOUR_FILE_KEY/Design" \
  --token "YOUR_ACCESS_TOKEN" \
  --node-ids "123:456"

# Extract multiple nodes
figma-extractor \
  --url "https://www.figma.com/file/YOUR_FILE_KEY/Design" \
  --token "YOUR_ACCESS_TOKEN" \
  --node-ids "123:456,789:012,345:678"
```

### Extract from Figma Share Links

The tool automatically detects node IDs from Figma share URLs:

```bash
# URL with node-id query parameter (most common)
figma-extractor \
  --url "https://www.figma.com/file/ABC123/Design?node-id=123:456" \
  --token "YOUR_ACCESS_TOKEN"

# URL with multiple nodes
figma-extractor \
  --url "https://www.figma.com/file/ABC123/Design?node-id=123:456,789:012" \
  --token "YOUR_ACCESS_TOKEN"
```

### Getting a Figma Personal Access Token

1. Log in to your Figma account
2. Go to **Settings** → **Account**
3. Scroll down to **Personal access tokens**
4. Click **Generate new token**
5. Give it a name (e.g., "Design Extractor")
6. Copy the token (you won't be able to see it again)


### Options

- `--url, -u`: Figma file URL (required)
- `--token, -t`: Figma Personal Access Token (required)
- `--output, -o`: Output markdown file (default: `FIGMA_DESIGN_SPECIFICATIONS.md`)
- `--node-ids, -n`: Comma-separated node IDs to extract (optional)
- `--inherit-context, -i`: Inherit file-level context (colors, styles) when extracting specific nodes (default: false)
- `--export-images`: Export images/assets from Figma (default: false)
- `--image-format`: Image format: `png`, `svg`, `jpg`, `pdf` (default: `png`)
- `--image-scales`: Comma-separated scale factors, e.g. `"1,2,3"` (default: `1`; ignored for SVG/PDF)
- `--image-dir`: Output directory for exported images (default: `figma-assets`)

### Examples

**Extract entire file:**
```bash
figma-extractor \
  --url "https://www.figma.com/file/abc123xyz/My-Design-System" \
  --token "figd_xxxxxxxxxxxxxxxxxxxxxxxxxxxx" \
  --output "design-specs.md"
```

**Extract specific component:**
```bash
figma-extractor \
  --url "https://www.figma.com/file/abc123xyz/My-Design-System" \
  --token "figd_xxxxxxxxxxxxxxxxxxxxxxxxxxxx" \
  --node-ids "123:456" \
  --output "button-component.md"
```

**Extract from share link with node ID:**
```bash
figma-extractor \
  --url "https://www.figma.com/file/abc123xyz/My-Design-System?node-id=123:456" \
  --token "figd_xxxxxxxxxxxxxxxxxxxxxxxxxxxx" \
  --output "component-specs.md"
```

**Extract node with file-level context (colors, styles):**
```bash
figma-extractor \
  --url "https://www.figma.com/file/abc123xyz/My-Design-System?node-id=123:456" \
  --token "figd_xxxxxxxxxxxxxxxxxxxxxxxxxxxx" \
  --inherit-context \
  --output "component-with-styles.md"
```

**Extract multiple related components:**
```bash
figma-extractor \
  --url "https://www.figma.com/file/abc123xyz/My-Design-System" \
  --token "figd_xxxxxxxxxxxxxxxxxxxxxxxxxxxx" \
  --node-ids "123:456,789:012,345:678" \
  --output "button-variants.md"
```

**Export all assets/images from the entire file as PNG (1x and 2x):**
```bash
figma-extractor \
  --url "https://www.figma.com/file/abc123xyz/My-Design-System" \
  --token "figd_xxxxxxxxxxxxxxxxxxxxxxxxxxxx" \
  --export-images \
  --image-scales "1,2" \
  --image-dir "assets"
```

**Export specific nodes as SVG:**
```bash
figma-extractor \
  --url "https://www.figma.com/file/abc123xyz/My-Design-System" \
  --token "figd_xxxxxxxxxxxxxxxxxxxxxxxxxxxx" \
  --node-ids "123:456,789:012" \
  --export-images \
  --image-format svg \
  --image-dir "icons"
```

## Output Format

The tool generates a markdown file with the following sections:

### Design System
- **Color Palette**: All colors categorized by usage (primary, background, text, etc.)
- **Typography**: Font families, sizes, weights, and line heights
- **Spacing**: Standardized spacing scale
- **Border Radius**: Border radius values
- **Shadows**: Shadow definitions with offsets, blur, and colors

### Layout Specifications
- Header height
- Sidebar width
- Content padding
- Other layout measurements

### Implementation Notes
- Instructions for applying the design system
- Usage examples for Tailwind CSS configuration
- Guidelines for implementing the design

## How It Works

### Full File Extraction

1. **Authentication**: Connects to the Figma API using your personal access token
2. **File Retrieval**: Fetches the complete file data including all nodes and styles
3. **Recursive Extraction**: Traverses the entire document tree and extracts:
   - Fill colors from all elements
   - Stroke colors and border properties
   - Text styles (fonts, sizes, weights)
   - Layout properties (padding, spacing, dimensions)
   - Visual effects (shadows, blur)
4. **Categorization**: Automatically categorizes extracted values based on node names and properties
5. **Normalization**: Deduplicates and normalizes values to standard scales
6. **Markdown Generation**: Formats all specifications as CSS variables in a markdown document

### Node-Specific Extraction

1. **Node ID Detection**: Extracts node IDs from the URL or `--node-ids` flag
   - Supports query parameter format: `?node-id=123:456`
   - Supports URL-encoded format: `?node-id=123-456`
   - Supports multiple nodes: `?node-id=123:456,789:012`
2. **Targeted Fetch**: Fetches only the specified nodes using the Figma `/nodes` API endpoint
3. **Smart Extraction**: Extracts specifications from target nodes and their children
4. **Optional Context Inheritance**: Use `--inherit-context` flag to include:
   - Published styles and colors from document root
   - Global typography definitions
   - File-level design tokens
5. **Normalization & Deduplication**: Normalizes values to standard scales
6. **Markdown Output**: Generates the same comprehensive markdown format

**Benefits of Node Extraction:**
- ⚡ **Faster**: Only fetches and processes specific elements
- 🎯 **Focused**: Perfect for extracting individual components or screens
- 📦 **Efficient**: Works great with large Figma files
- 🔗 **Convenient**: Works directly with Figma share links
- 🎛️ **Flexible**: Choose whether to include file-level context or not

### Image/Asset Export

1. **Opt-in**: Enabled via the `--export-images` flag
2. **Node Discovery**:
   - **Full file mode**: Automatically discovers all nodes that have export settings defined by the designer in Figma
   - **Node-specific mode**: Exports the targeted nodes directly
3. **Batched API Requests**: Sends node IDs to the Figma Images API in batches of 100 for efficiency
4. **Concurrent Downloads**: Downloads images in parallel (up to 5 at a time) for speed
5. **Smart Naming**: Generates kebab-case filenames from node names, with `@2x`/`@3x` suffixes for raster scales > 1 and automatic deduplication of colliding names
6. **Multi-Format**: Supports PNG, SVG, JPG, and PDF output formats
7. **Multi-Scale**: Generate multiple scale variants (e.g., 1x, 2x, 3x) in a single run; scale is ignored for vector formats (SVG/PDF)
8. **Integrated Output**: Exported asset info is included in the generated markdown file

## Integration with Claude

The generated markdown file is specifically formatted to work with [Claude Sonnet 4.5](https://claude.ai), [ChatGPT](https://chatgpt.com/) and e.t.c. for implementing the design. Simply provide the generated markdown file to AI along with your implementation request, and it will use the exact specifications to build your UI.

## Example Output

The tool generates CSS variables like:

```css
/* Primary Colors */
--color-primary-main: #3B82F6;
--color-primary-hover: #2563EB;

/* Background Colors */
--color-bg-main: #FFFFFF;
--color-bg-surface: #F9FAFB;

/* Typography */
--font-primary: 'Inter', system-ui, -apple-system, sans-serif;
--text-base: 16px;
--text-lg: 18px;
--font-medium: 500;
--font-semibold: 600;

/* Spacing */
--space-4: 16px;
--space-6: 24px;
--space-8: 32px;

/* Border Radius */
--radius-md: 6px;
--radius-lg: 8px;
```

## Node ID Formats

The tool supports all common Figma URL formats for node identification:

### Query Parameter (Most Common)
```
https://www.figma.com/file/ABC123/Design?node-id=123:456
https://www.figma.com/design/ABC123/Design?node-id=123-456  (URL-encoded colon)
```

### Multiple Nodes
```
https://www.figma.com/file/ABC123/Design?node-id=123:456,789:012
```

### Hash Fragment
```
https://www.figma.com/file/ABC123/Design#123:456
```

### Finding Node IDs

To find a node ID in Figma:
1. Right-click on any element in Figma
2. Select "Copy link to selection"
3. The copied URL will contain the node ID in the format shown above
4. Use that URL directly with figma-extractor

## Limitations

- Requires a valid Figma Personal Access Token
- Can only access files you have permission to view
- Color categorization is based on node naming conventions
- Very large files may take longer to process (use node extraction for better performance)
- Node IDs must exist in the specified file

## Contributing

Contributions are welcome! Please feel free to submit pull requests or open issues for bugs and feature requests.

## License

MIT License - feel free to use this tool in your projects.

## Support

If you encounter any issues:
1. Verify your Figma token is valid
2. Ensure you have access to the Figma file
3. Check that the file URL is correct
4. Review the error messages for specific issues

For additional help, please open an issue on GitHub.

## License

This project is licensed under the [BSD 3-clause license](LICENSE), just like the Go project itself.
