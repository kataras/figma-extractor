package figma

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
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
