package figma

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const (
	figmaAPIBase = "https://api.figma.com/v1"
)

// Client represents a Figma API client with configured HTTP settings for reliable communication
// with the Figma API. It includes retry logic and optimized transport settings for handling large files.
type Client struct {
	accessToken string
	httpClient  *http.Client
}

// NewClient creates a new Figma API client with the provided personal access token.
// The client is configured with optimized HTTP transport settings including connection pooling,
// disabled HTTP/2 (for large file stability), and a 10-minute timeout for very large files.
func NewClient(accessToken string) *Client {
	// Configure transport for better handling of large files
	transport := &http.Transport{
		MaxIdleConns:        10,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  false,
		DisableKeepAlives:   false,
		MaxIdleConnsPerHost: 10,
		// Disable HTTP/2 to avoid stream errors with large files
		ForceAttemptHTTP2: false,
	}

	return &Client{
		accessToken: accessToken,
		httpClient: &http.Client{
			Timeout:   10 * time.Minute, // Increased timeout for very large files
			Transport: transport,
		},
	}
}

// ExtractFileKey extracts the unique file identifier from a Figma URL.
// Supports both /file/ and /design/ URL patterns (e.g., figma.com/file/ABC123/Design-Name).
// Returns an error if the URL format is invalid or if the URL doesn't match the expected Figma domain pattern.
func ExtractFileKey(figmaURL string) (string, error) {
	// Match patterns like:
	// https://www.figma.com/file/ABC123/Design-Name
	// https://www.figma.com/design/ABC123/Design-Name
	// Anchored to ensure the entire URL matches the expected pattern and prevent bypass attacks.
	re := regexp.MustCompile(`^https?://(?:www\.)?figma\.com/(?:file|design)/([A-Za-z0-9]+)(?:/|$)`)
	matches := re.FindStringSubmatch(figmaURL)

	if len(matches) < 2 {
		return "", fmt.Errorf("invalid Figma URL format: must be a valid figma.com URL with /file/ or /design/ path")
	}

	return matches[1], nil
}

// ExtractNodeIDs extracts node identifiers from a Figma URL.
// Supports multiple formats:
//   - Query parameter: ?node-id=123:456 or ?node-id=123-456 or ?node-id=123:456,789:012
//   - Hash fragment: #123:456 or #123:456,789:012
//   - Path format: /nodes/123:456 or /nodes/123:456,789:012
//
// Returns an empty slice if no node IDs are found (not an error).
// Normalizes URL-encoded colons (123-456 → 123:456).
func ExtractNodeIDs(figmaURL string) ([]string, error) {
	nodeIDs := make([]string, 0)

	// Try query parameter format: ?node-id=123:456 or ?node-id=123-456
	queryRe := regexp.MustCompile(`[?&]node-id=([^&]+)`)
	if matches := queryRe.FindStringSubmatch(figmaURL); len(matches) >= 2 {
		// Split by comma for multiple nodes
		ids := strings.Split(matches[1], ",")
		for _, id := range ids {
			// Normalize: replace URL-encoded dash with colon
			id = strings.ReplaceAll(strings.TrimSpace(id), "-", ":")
			if id != "" {
				nodeIDs = append(nodeIDs, id)
			}
		}
		return deduplicateNodeIDs(nodeIDs), nil
	}

	// Try hash fragment format: #123:456 or #123:456,789:012
	hashRe := regexp.MustCompile(`#([0-9:-]+(?:,[0-9:-]+)*)`)
	if matches := hashRe.FindStringSubmatch(figmaURL); len(matches) >= 2 {
		ids := strings.Split(matches[1], ",")
		for _, id := range ids {
			id = strings.TrimSpace(id)
			if id != "" {
				nodeIDs = append(nodeIDs, id)
			}
		}
		return deduplicateNodeIDs(nodeIDs), nil
	}

	// Try path format: /nodes/123:456 or /nodes/123:456,789:012
	pathRe := regexp.MustCompile(`/nodes/([0-9:-]+(?:,[0-9:-]+)*)`)
	if matches := pathRe.FindStringSubmatch(figmaURL); len(matches) >= 2 {
		ids := strings.Split(matches[1], ",")
		for _, id := range ids {
			id = strings.TrimSpace(id)
			if id != "" {
				nodeIDs = append(nodeIDs, id)
			}
		}
		return deduplicateNodeIDs(nodeIDs), nil
	}

	// No node IDs found - return empty slice (not an error)
	return nodeIDs, nil
}

// deduplicateNodeIDs removes duplicate node IDs while preserving order.
func deduplicateNodeIDs(nodeIDs []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(nodeIDs))

	for _, id := range nodeIDs {
		if !seen[id] {
			seen[id] = true
			result = append(result, id)
		}
	}

	return result
}

// GetFile retrieves complete file data from the Figma API including document structure, styles, and metadata.
// Implements automatic retry logic (up to 3 attempts) with exponential backoff for handling rate limits
// and temporary failures. The request automatically retries on 429 (rate limit) and 5xx (server error) responses.
func (c *Client) GetFile(fileKey string) (*FileResponse, error) {
	url := fmt.Sprintf("%s/files/%s", figmaAPIBase, fileKey)

	var lastErr error
	maxRetries := 3

	for attempt := 1; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("X-Figma-Token", c.accessToken)
		// Disable HTTP/2 to avoid stream errors with large files
		req.Header.Set("Connection", "close")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("attempt %d failed to execute request: %w", attempt, err)
			if attempt < maxRetries {
				time.Sleep(time.Duration(attempt) * 2 * time.Second)
				continue
			}
			return nil, lastErr
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			lastErr = fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
			if attempt < maxRetries && (resp.StatusCode == 429 || resp.StatusCode >= 500) {
				time.Sleep(time.Duration(attempt) * 2 * time.Second)
				continue
			}
			return nil, lastErr
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("attempt %d failed to read response body: %w", attempt, err)
			if attempt < maxRetries {
				time.Sleep(time.Duration(attempt) * 2 * time.Second)
				continue
			}
			return nil, lastErr
		}

		var fileResp FileResponse
		if err := json.Unmarshal(body, &fileResp); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}

		return &fileResp, nil
	}

	return nil, lastErr
}

// GetFileNodes retrieves specific nodes from a Figma file by their node IDs.
// This is more efficient than fetching the entire file when you only need specific elements.
// Implements automatic retry logic (up to 3 attempts) with exponential backoff for handling rate limits.
// Parameters:
//   - fileKey: The Figma file identifier
//   - nodeIDs: Slice of node IDs to fetch (e.g., ["123:456", "789:012"])
//
// Returns a NodesResponse containing the requested nodes with their complete structure.
func (c *Client) GetFileNodes(fileKey string, nodeIDs []string) (*NodesResponse, error) {
	if len(nodeIDs) == 0 {
		return nil, fmt.Errorf("no node IDs provided")
	}

	// Join node IDs with comma for the API request
	idsParam := strings.Join(nodeIDs, ",")
	url := fmt.Sprintf("%s/files/%s/nodes?ids=%s", figmaAPIBase, fileKey, idsParam)

	var lastErr error
	maxRetries := 3

	for attempt := 1; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("X-Figma-Token", c.accessToken)
		req.Header.Set("Connection", "close")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("attempt %d failed to execute request: %w", attempt, err)
			if attempt < maxRetries {
				time.Sleep(time.Duration(attempt) * 2 * time.Second)
				continue
			}
			return nil, lastErr
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			lastErr = fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
			if attempt < maxRetries && (resp.StatusCode == 429 || resp.StatusCode >= 500) {
				time.Sleep(time.Duration(attempt) * 2 * time.Second)
				continue
			}
			return nil, lastErr
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("attempt %d failed to read response body: %w", attempt, err)
			if attempt < maxRetries {
				time.Sleep(time.Duration(attempt) * 2 * time.Second)
				continue
			}
			return nil, lastErr
		}

		var nodesResp NodesResponse
		if err := json.Unmarshal(body, &nodesResp); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}

		// Verify that all requested nodes were returned
		if len(nodesResp.Nodes) == 0 {
			return nil, fmt.Errorf("no nodes found for the provided IDs: %s", idsParam)
		}

		// Check for nodes that weren't found
		missingNodes := make([]string, 0)
		for _, id := range nodeIDs {
			if _, exists := nodesResp.Nodes[id]; !exists {
				missingNodes = append(missingNodes, id)
			}
		}

		if len(missingNodes) > 0 {
			return nil, fmt.Errorf("nodes not found: %s", strings.Join(missingNodes, ", "))
		}

		return &nodesResp, nil
	}

	return nil, lastErr
}

// GetImages retrieves rendered images for the specified nodes from the Figma Images API.
// Supports format (png, svg, jpg, pdf) and scale factor for raster formats.
// Implements automatic retry logic (up to 3 attempts) with exponential backoff.
func (c *Client) GetImages(fileKey string, nodeIDs []string, format string, scale float64) (*ImageResponse, error) {
	if len(nodeIDs) == 0 {
		return nil, fmt.Errorf("no node IDs provided")
	}

	if format == "" {
		format = "png"
	}
	if scale <= 0 {
		scale = 1
	}

	idsParam := strings.Join(nodeIDs, ",")
	url := fmt.Sprintf("%s/images/%s?ids=%s&format=%s&scale=%g", figmaAPIBase, fileKey, idsParam, format, scale)

	var lastErr error
	maxRetries := 3

	for attempt := 1; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("X-Figma-Token", c.accessToken)
		req.Header.Set("Connection", "close")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("attempt %d failed to execute request: %w", attempt, err)
			if attempt < maxRetries {
				time.Sleep(time.Duration(attempt) * 2 * time.Second)
				continue
			}
			return nil, lastErr
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			lastErr = fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
			if attempt < maxRetries && (resp.StatusCode == 429 || resp.StatusCode >= 500) {
				time.Sleep(time.Duration(attempt) * 2 * time.Second)
				continue
			}
			return nil, lastErr
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("attempt %d failed to read response body: %w", attempt, err)
			if attempt < maxRetries {
				time.Sleep(time.Duration(attempt) * 2 * time.Second)
				continue
			}
			return nil, lastErr
		}

		var imgResp ImageResponse
		if err := json.Unmarshal(body, &imgResp); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}

		if imgResp.Err != nil {
			return nil, fmt.Errorf("Figma images API error: %s", *imgResp.Err)
		}

		return &imgResp, nil
	}

	return nil, lastErr
}

// GetFileImages retrieves download URLs for all embedded images in a Figma file.
// Calls GET /v1/files/:key/images and returns a map of imageRef -> download URL.
// Implements automatic retry logic (up to 3 attempts) with exponential backoff.
func (c *Client) GetFileImages(fileKey string) (*FileImagesResponse, error) {
	url := fmt.Sprintf("%s/files/%s/images", figmaAPIBase, fileKey)

	var lastErr error
	maxRetries := 3

	for attempt := 1; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("X-Figma-Token", c.accessToken)
		req.Header.Set("Connection", "close")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("attempt %d failed to execute request: %w", attempt, err)
			if attempt < maxRetries {
				time.Sleep(time.Duration(attempt) * 2 * time.Second)
				continue
			}
			return nil, lastErr
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			lastErr = fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
			if attempt < maxRetries && (resp.StatusCode == 429 || resp.StatusCode >= 500) {
				time.Sleep(time.Duration(attempt) * 2 * time.Second)
				continue
			}
			return nil, lastErr
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("attempt %d failed to read response body: %w", attempt, err)
			if attempt < maxRetries {
				time.Sleep(time.Duration(attempt) * 2 * time.Second)
				continue
			}
			return nil, lastErr
		}

		var imgResp FileImagesResponse
		if err := json.Unmarshal(body, &imgResp); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}

		if imgResp.Err != nil {
			return nil, fmt.Errorf("Figma file images API error: %s", *imgResp.Err)
		}

		return &imgResp, nil
	}

	return nil, lastErr
}

// GetFileStyles retrieves all published styles (colors, text, effects, grids) from a Figma file.
// This includes style metadata such as names, descriptions, and type information.
func (c *Client) GetFileStyles(fileKey string) (*StylesResponse, error) {
	url := fmt.Sprintf("%s/files/%s/styles", figmaAPIBase, fileKey)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Figma-Token", c.accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var stylesResp StylesResponse
	if err := json.Unmarshal(body, &stylesResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &stylesResp, nil
}
