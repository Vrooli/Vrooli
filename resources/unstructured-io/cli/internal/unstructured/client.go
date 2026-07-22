// Package unstructured implements the supported document partitioning contract.
package unstructured

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}
type Element struct {
	Type     string          `json:"type"`
	Text     string          `json:"text"`
	Metadata json.RawMessage `json:"metadata"`
}

func (c Client) client() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: 60 * time.Second}
}

func (c Client) Health(ctx context.Context) error {
	req, e := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(c.BaseURL, "/")+"/healthcheck", nil)
	if e != nil {
		return e
	}
	resp, e := c.client().Do(req)
	if e != nil {
		return fmt.Errorf("call Unstructured health: %w", e)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Unstructured health returned HTTP %d", resp.StatusCode)
	}
	return nil
}
func SupportedFormats() []string { return []string{"txt", "html", "htm", "pdf"} }
func (c Client) Process(ctx context.Context, path string) ([]Element, error) {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
	supported := false
	for _, f := range SupportedFormats() {
		if ext == f {
			supported = true
		}
	}
	if !supported {
		return nil, fmt.Errorf("unsupported document format %q", ext)
	}
	file, e := os.Open(path)
	if e != nil {
		return nil, e
	}
	defer file.Close()
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	part, e := w.CreateFormFile("files", filepath.Base(path))
	if e != nil {
		return nil, e
	}
	if _, e = io.Copy(part, file); e != nil {
		return nil, e
	}
	if e = w.Close(); e != nil {
		return nil, e
	}
	req, e := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.BaseURL, "/")+"/general/v0/general", &b)
	if e != nil {
		return nil, e
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, e := c.client().Do(req)
	if e != nil {
		return nil, fmt.Errorf("call Unstructured process: %w", e)
	}
	defer resp.Body.Close()
	data, e := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if e != nil {
		return nil, e
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Unstructured returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	var out []Element
	if e = json.Unmarshal(data, &out); e != nil {
		return nil, fmt.Errorf("decode Unstructured response: %w", e)
	}
	return out, nil
}
