package conversations

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"
)

const maxStoredAttachmentBytes = 64 << 20

type storedAttachmentResult struct {
	AskID  string `json:"ask_id"`
	Name   string `json:"name"`
	Output string `json:"output"`
	Size   int    `json:"size_bytes"`
	SHA256 string `json:"sha256"`
}

func (s *service) downloadStoredAttachment(ctx cliapp.RunContext) error {
	askID := strings.TrimSpace(ctx.Positional("id"))
	name := strings.TrimSpace(ctx.Positional("name"))
	output := strings.TrimSpace(ctx.Flag("output"))
	if askID == "" || name == "" {
		return fmt.Errorf("give the ask id and attachment name")
	}
	if output == "" {
		return fmt.Errorf("give --output so attachment bytes are written to a file")
	}
	if output == "-" {
		return fmt.Errorf("--output must be a file path")
	}
	content, digest, err := getStoredAttachment(s.httpClient, s.apiBase, askID, name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o700); err != nil {
		return fmt.Errorf("create attachment output directory: %w", err)
	}
	if err := os.WriteFile(output, content, 0o600); err != nil {
		return fmt.Errorf("write attachment output: %w", err)
	}
	result := storedAttachmentResult{AskID: askID, Name: name, Output: output, Size: len(content), SHA256: digest}
	if ctx.JSON() {
		encoder := json.NewEncoder(ctx.Stdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(result)
	}
	return ctx.RenderMutation(cliapp.MutationReport{Result: []string{
		fmt.Sprintf("Saved stored attachment %s (%d bytes, sha256 %s) to %s.", name, len(content), digest, output),
	}})
}

func getStoredAttachment(client interface {
	Do(*http.Request) (*http.Response, error)
}, apiBase, askID, name string) ([]byte, string, error) {
	if client == nil || strings.TrimSpace(apiBase) == "" {
		return nil, "", fmt.Errorf("Notification Hub API client is unavailable")
	}
	if strings.ContainsAny(askID, "/\\") || strings.ContainsAny(name, "/\\") || name == "." || name == ".." {
		return nil, "", fmt.Errorf("ask id and attachment name must each be one path segment")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	endpoint := strings.TrimRight(apiBase, "/") + "/api/v1/asks/" + url.PathEscape(askID) + "/attachments/" + url.PathEscape(name)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, "", fmt.Errorf("build stored attachment request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("request stored attachment: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("Notification Hub refused stored attachment with status %d", resp.StatusCode)
	}
	content, err := io.ReadAll(io.LimitReader(resp.Body, maxStoredAttachmentBytes+1))
	if err != nil {
		return nil, "", fmt.Errorf("read stored attachment bytes: %w", err)
	}
	if len(content) > maxStoredAttachmentBytes {
		return nil, "", fmt.Errorf("stored attachment exceeds %d bytes", maxStoredAttachmentBytes)
	}
	digest := sha256.Sum256(content)
	return content, hex.EncodeToString(digest[:]), nil
}
