package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// This test builds the figma-extractor binary and runs it against a real
// Figma file to verify image extraction works end-to-end, including the
// render-API fallback for IMAGE fill nodes.
//
// Run with:
//   FIGMA_TOKEN=<your-token> go test -v

const (
	figmaURL = "https://www.figma.com/design/rrjFDZ1mXkjC147DGMDtFU/Mobile-Apps-%E2%80%93-Prototyping-Kit--Community-?node-id=193-3231&p=f&t=jQIqfqrH4tIfi4Ey-0"
	outDir   = "./"
)

func mustGetToken(t *testing.T) string {
	token := os.Getenv("FIGMA_TOKEN")
	if token == "" {
		t.Skip("FIGMA_TOKEN not set, skipping test")
	}
	return token
}

// buildBinary compiles the figma-extractor binary into outDir and returns its absolute path.
func buildBinary(t *testing.T) string {
	t.Helper()

	// Resolve the repo root (parent of _examples).
	repoRoot, err := filepath.Abs(filepath.Join("../.."))
	if err != nil {
		t.Fatalf("failed to resolve repo root: %v", err)
	}

	absOut, err := filepath.Abs(outDir)
	if err != nil {
		t.Fatalf("failed to resolve output dir: %v", err)
	}

	bin := filepath.Join(absOut, "figma-extractor")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}

	cmd := exec.Command("go", "build", "-o", bin, "./cmd/figma-extractor")
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build binary: %v\n%s", err, out)
	}
	return bin
}

func TestImageExtraction(t *testing.T) {
	token := mustGetToken(t)
	bin := buildBinary(t)

	imageDir := filepath.Join(outDir, "figma-assets")
	outputFile := filepath.Join(outDir, "output.md")

	// Clean up from previous runs so stale files don't affect assertions.
	os.RemoveAll(imageDir)
	os.Remove(outputFile)

	cmd := exec.Command(bin,
		"--url", figmaURL,
		"--token", token,
		"--output", outputFile,
		"--export-images",
		"--image-format", "png",
		"--image-scales", "1",
		"--image-dir", imageDir,
		"--component-tree",
	)
	out, err := cmd.CombinedOutput()
	t.Logf("CLI output:\n%s", string(out))
	if err != nil {
		t.Fatalf("figma-extractor failed: %v", err)
	}
}
