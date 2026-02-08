package figma

import (
	"testing"
)

func TestExtractFileKey(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		want    string
		wantErr bool
	}{
		{
			name:    "valid /file/ URL",
			url:     "https://www.figma.com/file/ABC123XYZ/Design-Name",
			want:    "ABC123XYZ",
			wantErr: false,
		},
		{
			name:    "valid /design/ URL",
			url:     "https://www.figma.com/design/ABC123XYZ/Design-Name",
			want:    "ABC123XYZ",
			wantErr: false,
		},
		{
			name:    "URL with node-id parameter",
			url:     "https://www.figma.com/design/4gkABR5gEZnIvlCaXmA4KI/Makis-s-file?node-id=11933-305884",
			want:    "4gkABR5gEZnIvlCaXmA4KI",
			wantErr: false,
		},
		{
			name:    "URL with additional parameters",
			url:     "https://www.figma.com/design/4gkABR5gEZnIvlCaXmA4KI/Makis-s-file?node-id=11933-305884&t=ObvUckUHZc8tSjeT-1",
			want:    "4gkABR5gEZnIvlCaXmA4KI",
			wantErr: false,
		},
		{
			name:    "URL without www subdomain",
			url:     "https://figma.com/file/ABC123XYZ/Design-Name",
			want:    "ABC123XYZ",
			wantErr: false,
		},
		{
			name:    "URL with http protocol",
			url:     "http://www.figma.com/file/ABC123XYZ/Design-Name",
			want:    "ABC123XYZ",
			wantErr: false,
		},
		{
			name:    "URL with trailing slash",
			url:     "https://www.figma.com/file/ABC123XYZ/",
			want:    "ABC123XYZ",
			wantErr: false,
		},
		{
			name:    "invalid URL - missing file key",
			url:     "https://www.figma.com/file/",
			want:    "",
			wantErr: true,
		},
		{
			name:    "invalid URL - wrong domain",
			url:     "https://www.example.com/file/ABC123XYZ",
			want:    "",
			wantErr: true,
		},
		{
			name:    "invalid URL - wrong path",
			url:     "https://www.figma.com/dashboard/ABC123XYZ",
			want:    "",
			wantErr: true,
		},
		{
			name:    "empty URL",
			url:     "",
			want:    "",
			wantErr: true,
		},
		{
			name:    "file key with mixed alphanumeric",
			url:     "https://www.figma.com/file/aB1cD2eF3gH4iJ5kL6/MyDesign",
			want:    "aB1cD2eF3gH4iJ5kL6",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractFileKey(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractFileKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ExtractFileKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractNodeIDs(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		want    []string
		wantErr bool
	}{
		{
			name:    "single node-id with colon",
			url:     "https://www.figma.com/file/ABC123/Design?node-id=123:456",
			want:    []string{"123:456"},
			wantErr: false,
		},
		{
			name:    "single node-id with dash (URL-encoded)",
			url:     "https://www.figma.com/design/4gkABR5gEZnIvlCaXmA4KI/Makis-s-file?node-id=11933-305884",
			want:    []string{"11933:305884"},
			wantErr: false,
		},
		{
			name:    "node-id with additional parameters",
			url:     "https://www.figma.com/design/4gkABR5gEZnIvlCaXmA4KI/Makis-s-file?node-id=11933-305884&t=ObvUckUHZc8tSjeT-1",
			want:    []string{"11933:305884"},
			wantErr: false,
		},
		{
			name:    "multiple node-ids with colons",
			url:     "https://www.figma.com/file/ABC123/Design?node-id=123:456,789:012",
			want:    []string{"123:456", "789:012"},
			wantErr: false,
		},
		{
			name:    "multiple node-ids with dashes",
			url:     "https://www.figma.com/file/ABC123/Design?node-id=123-456,789-012",
			want:    []string{"123:456", "789:012"},
			wantErr: false,
		},
		{
			name:    "multiple node-ids with mixed format",
			url:     "https://www.figma.com/file/ABC123/Design?node-id=123:456,789-012",
			want:    []string{"123:456", "789:012"},
			wantErr: false,
		},
		{
			name:    "hash fragment format single node",
			url:     "https://www.figma.com/file/ABC123/Design#123:456",
			want:    []string{"123:456"},
			wantErr: false,
		},
		{
			name:    "hash fragment format multiple nodes",
			url:     "https://www.figma.com/file/ABC123/Design#123:456,789:012",
			want:    []string{"123:456", "789:012"},
			wantErr: false,
		},
		{
			name:    "path format single node",
			url:     "https://www.figma.com/file/ABC123/Design/nodes/123:456",
			want:    []string{"123:456"},
			wantErr: false,
		},
		{
			name:    "path format multiple nodes",
			url:     "https://www.figma.com/file/ABC123/Design/nodes/123:456,789:012",
			want:    []string{"123:456", "789:012"},
			wantErr: false,
		},
		{
			name:    "no node-ids in URL",
			url:     "https://www.figma.com/file/ABC123/Design",
			want:    []string{},
			wantErr: false,
		},
		{
			name:    "node-id with spaces (should be trimmed)",
			url:     "https://www.figma.com/file/ABC123/Design?node-id=123:456, 789:012",
			want:    []string{"123:456", "789:012"},
			wantErr: false,
		},
		{
			name:    "duplicate node-ids (should deduplicate)",
			url:     "https://www.figma.com/file/ABC123/Design?node-id=123:456,123:456,789:012",
			want:    []string{"123:456", "789:012"},
			wantErr: false,
		},
		{
			name:    "node-id as first parameter",
			url:     "https://www.figma.com/file/ABC123/Design?node-id=123:456&other=value",
			want:    []string{"123:456"},
			wantErr: false,
		},
		{
			name:    "node-id as middle parameter",
			url:     "https://www.figma.com/file/ABC123/Design?first=value&node-id=123:456&last=value",
			want:    []string{"123:456"},
			wantErr: false,
		},
		{
			name:    "empty node-id parameter",
			url:     "https://www.figma.com/file/ABC123/Design?node-id=",
			want:    []string{},
			wantErr: false,
		},
		{
			name:    "node-id with long numeric values",
			url:     "https://www.figma.com/file/ABC123/Design?node-id=999999:888888",
			want:    []string{"999999:888888"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractNodeIDs(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractNodeIDs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != len(tt.want) {
				t.Errorf("ExtractNodeIDs() returned %d nodes, want %d nodes", len(got), len(tt.want))
				t.Errorf("ExtractNodeIDs() = %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ExtractNodeIDs() at index %d = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestDeduplicateNodeIDs(t *testing.T) {
	tests := []struct {
		name string
		ids  []string
		want []string
	}{
		{
			name: "no duplicates",
			ids:  []string{"123:456", "789:012", "345:678"},
			want: []string{"123:456", "789:012", "345:678"},
		},
		{
			name: "with duplicates",
			ids:  []string{"123:456", "789:012", "123:456", "345:678"},
			want: []string{"123:456", "789:012", "345:678"},
		},
		{
			name: "all duplicates",
			ids:  []string{"123:456", "123:456", "123:456"},
			want: []string{"123:456"},
		},
		{
			name: "empty slice",
			ids:  []string{},
			want: []string{},
		},
		{
			name: "single element",
			ids:  []string{"123:456"},
			want: []string{"123:456"},
		},
		{
			name: "preserves order",
			ids:  []string{"789:012", "123:456", "789:012", "345:678", "123:456"},
			want: []string{"789:012", "123:456", "345:678"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deduplicateNodeIDs(tt.ids)
			if len(got) != len(tt.want) {
				t.Errorf("deduplicateNodeIDs() returned %d nodes, want %d nodes", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("deduplicateNodeIDs() at index %d = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}
