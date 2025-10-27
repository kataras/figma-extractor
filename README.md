# Figma Design Extractor

A Go tool to extract design specifications from Figma files using the Figma REST API. This tool automatically extracts colors, typography, spacing, shadows, border radii, and layout specifications from any Figma file and outputs them in a markdown format that can be used to implement the design with AI.

## Features

- 🎨 **Color Extraction**: Automatically categorizes colors into primary, secondary, background, text, status, and border colors
- 📝 **Typography**: Extracts font families, sizes, weights, and line heights
- 📏 **Spacing**: Identifies spacing patterns and normalizes them to a standard scale
- 🌈 **Visual Effects**: Extracts shadows and border radii
- 📐 **Layout Specs**: Captures layout dimensions like header height and sidebar width
- 📄 **Markdown Output**: Generates a comprehensive markdown file with all specifications

## Installation

The only requirement is the [Go Programming Language](https://go.dev/dl/).

```bash
go install github.com/kataras/figma-extractor/cmd/figma-extractor@latest
```

## Usage

```bash
figma-extractor --url "https://www.figma.com/file/YOUR_FILE_KEY/Design-Name" --token "YOUR_ACCESS_TOKEN"
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

### Example

```bash
figma-extractor \
  --url "https://www.figma.com/file/abc123xyz/My-Design-System" \
  --token "figd_xxxxxxxxxxxxxxxxxxxxxxxxxxxx" \
  --output "design-specs.md"
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

## Limitations

- Requires a valid Figma Personal Access Token
- Can only access files you have permission to view
- Color categorization is based on node naming conventions
- Very large files may take longer to process

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
