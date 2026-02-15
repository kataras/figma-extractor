package imager

import (
	"testing"

	"github.com/kataras/figma-extractor/pkg/figma"
)

func TestCollectImageFillNodes(t *testing.T) {
	tests := []struct {
		name     string
		root     figma.Node
		wantLen  int
		wantRefs []string // expected ImageRef values
	}{
		{
			name: "no image fills",
			root: figma.Node{
				ID:   "0:1",
				Name: "Frame",
				Fills: []figma.Paint{
					{Type: "SOLID", Color: &figma.Color{R: 1, G: 0, B: 0, A: 1}},
				},
			},
			wantLen: 0,
		},
		{
			name: "single image fill at root",
			root: figma.Node{
				ID:   "1:1",
				Name: "Human Figure",
				Fills: []figma.Paint{
					{Type: "IMAGE", ImageRef: "abc123", ScaleMode: "FILL"},
				},
			},
			wantLen:  1,
			wantRefs: []string{"abc123"},
		},
		{
			name: "image fill in child node",
			root: figma.Node{
				ID:   "0:1",
				Name: "Frame",
				Children: []figma.Node{
					{
						ID:   "2:1",
						Name: "Background",
						Fills: []figma.Paint{
							{Type: "SOLID"},
						},
					},
					{
						ID:   "2:2",
						Name: "Photo",
						Fills: []figma.Paint{
							{Type: "IMAGE", ImageRef: "img456"},
						},
					},
				},
			},
			wantLen:  1,
			wantRefs: []string{"img456"},
		},
		{
			name: "multiple image fills in nested tree",
			root: figma.Node{
				ID:   "0:1",
				Name: "Page",
				Children: []figma.Node{
					{
						ID:   "1:1",
						Name: "Frame A",
						Children: []figma.Node{
							{
								ID:   "3:1",
								Name: "Avatar",
								Fills: []figma.Paint{
									{Type: "IMAGE", ImageRef: "ref1"},
								},
							},
						},
					},
					{
						ID:   "1:2",
						Name: "Frame B",
						Fills: []figma.Paint{
							{Type: "IMAGE", ImageRef: "ref2"},
						},
					},
				},
			},
			wantLen:  2,
			wantRefs: []string{"ref1", "ref2"},
		},
		{
			name: "image fill with empty ImageRef is skipped",
			root: figma.Node{
				ID:   "1:1",
				Name: "Broken",
				Fills: []figma.Paint{
					{Type: "IMAGE", ImageRef: ""},
				},
			},
			wantLen: 0,
		},
		{
			name: "mixed fills - only IMAGE type collected",
			root: figma.Node{
				ID:   "1:1",
				Name: "Mixed",
				Fills: []figma.Paint{
					{Type: "SOLID"},
					{Type: "GRADIENT_LINEAR"},
					{Type: "IMAGE", ImageRef: "mixedRef"},
				},
			},
			wantLen:  1,
			wantRefs: []string{"mixedRef"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CollectImageFillNodes(&tt.root)
			if len(got) != tt.wantLen {
				t.Errorf("CollectImageFillNodes() returned %d nodes, want %d", len(got), tt.wantLen)
				return
			}
			for i, ref := range tt.wantRefs {
				if got[i].ImageRef != ref {
					t.Errorf("CollectImageFillNodes()[%d].ImageRef = %q, want %q", i, got[i].ImageRef, ref)
				}
			}
		})
	}
}

func TestDetectExtensionFromURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "png URL",
			url:  "https://s3-alpha.figma.com/img/abc123/image.png",
			want: "png",
		},
		{
			name: "jpg URL",
			url:  "https://s3-alpha.figma.com/img/abc123/photo.jpg",
			want: "jpg",
		},
		{
			name: "svg URL",
			url:  "https://s3-alpha.figma.com/img/abc123/icon.svg",
			want: "svg",
		},
		{
			name: "URL with query params",
			url:  "https://s3-alpha.figma.com/img/abc123/image.png?X-Amz-Algorithm=AWS4-HMAC-SHA256",
			want: "png",
		},
		{
			name: "URL without extension defaults to png",
			url:  "https://s3-alpha.figma.com/img/abc123/image",
			want: "png",
		},
		{
			name: "empty URL defaults to png",
			url:  "",
			want: "png",
		},
		{
			name: "invalid URL defaults to png",
			url:  "://bad-url",
			want: "png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectExtensionFromURL(tt.url)
			if got != tt.want {
				t.Errorf("detectExtensionFromURL(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestCollectExportableNodes_WalksChildren(t *testing.T) {
	// Verifies that CollectExportableNodes walks into children
	// and finds nodes with ExportSettings, not just the root.
	root := figma.Node{
		ID:   "0:1",
		Name: "Frame",
		Type: "FRAME",
		Children: []figma.Node{
			{
				ID:   "1:1",
				Name: "Icon Button",
				Type: "COMPONENT",
				ExportSettings: []figma.ExportSetting{
					{Format: "PNG", Suffix: "", Constraint: struct {
						Type  string  `json:"type"`
						Value float64 `json:"value"`
					}{Type: "SCALE", Value: 1}},
				},
			},
			{
				ID:   "1:2",
				Name: "Label",
				Type: "TEXT",
				// No export settings
			},
			{
				ID:   "1:3",
				Name: "Nested Group",
				Type: "GROUP",
				Children: []figma.Node{
					{
						ID:   "2:1",
						Name: "Logo",
						Type: "RECTANGLE",
						ExportSettings: []figma.ExportSetting{
							{Format: "SVG"},
						},
					},
				},
			},
		},
	}

	got := CollectExportableNodes(&root)

	// Should find 2 nodes with ExportSettings: "Icon Button" and "Logo"
	if len(got) != 2 {
		t.Fatalf("CollectExportableNodes() returned %d nodes, want 2", len(got))
	}

	if name, ok := got["1:1"]; !ok || name != "Icon Button" {
		t.Errorf("expected node 1:1 (Icon Button), got %v", got)
	}
	if name, ok := got["2:1"]; !ok || name != "Logo" {
		t.Errorf("expected node 2:1 (Logo), got %v", got)
	}
}
