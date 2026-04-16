package sync

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"device-sync-hub/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `sync` subcommand group that wraps /api/v1/sync/*.
// File operations (upload, download) use direct HTTP requests because the
// unified HTTP helper only supports JSON bodies; all other commands go
// through core.Get / core.Request.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "sync",
		Description: "Share files, text, and clipboard content across devices",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List active sync items", Run: func(args []string) error { return runList(core, args) }},
			{Name: "get", Aliases: []string{"show"}, Description: "Show a single sync item", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "upload", Description: "Upload a file", Run: func(args []string) error { return runUploadFile(core, args) }},
			{Name: "upload-text", Description: "Upload text or clipboard content", Run: func(args []string) error { return runUploadText(core, args) }},
			{Name: "clipboard", Description: "Push clipboard content to target devices (--body-file PATH)", Run: func(args []string) error { return runClipboard(core, args) }},
			{Name: "notification", Description: "Push a notification to target devices (--body-file PATH)", Run: func(args []string) error { return runNotification(core, args) }},
			{Name: "download", Description: "Download a sync item", Run: func(args []string) error { return runDownload(core, args) }},
			{Name: "delete", Aliases: []string{"rm"}, Description: "Delete a sync item", Run: func(args []string) error { return runDelete(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("sync list")
	typeFilter := fs.String("type", "", "Filter by type (file|text|clipboard|notification)")
	status := fs.String("status", "", "Filter by status")
	search := fs.String("search", "", "Substring search across content")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	query := support.BuildQuery(map[string]string{
		"type":   *typeFilter,
		"status": *status,
		"search": *search,
	})
	body, err := core.Get("/sync/items", query)
	if err != nil {
		return err
	}
	var items []support.SyncItem
	if err := support.Decode(body, &items); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Active sync items: %d", len(items))},
		ResultsHeading: "Items",
		Results:        itemRows(items),
		RetrievalHints: []string{
			fmt.Sprintf("%s sync get <item-id>", support.CLIName),
			fmt.Sprintf("%s sync download <item-id>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("sync get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: sync get <item-id>")
	}
	id := fs.Arg(0)

	body, err := core.Get("/sync/items/"+id, nil)
	if err != nil {
		return err
	}
	var item support.SyncItem
	if err := support.Decode(body, &item); err != nil {
		return err
	}

	results := []string{
		fmt.Sprintf("ID: %s", item.ID),
		fmt.Sprintf("Type: %s", item.Type),
		fmt.Sprintf("Status: %s", item.Status),
		fmt.Sprintf("Created: %s", support.FormatTimeValue(item.CreatedAt)),
		fmt.Sprintf("Expires: %s", support.FormatTimeValue(item.ExpiresAt)),
	}
	if item.SourceDevice != "" {
		results = append(results, fmt.Sprintf("Source device: %s", item.SourceDevice))
	}
	if len(item.TargetDevices) > 0 {
		results = append(results, fmt.Sprintf("Target devices: %v", item.TargetDevices))
	}
	if len(item.Content) > 0 {
		results = append(results, "Content:")
		results = append(results, support.MapRows(item.Content)...)
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Sync item: %s (%s)", item.ID, item.Type)},
		ResultsHeading: "Details",
		Results:        results,
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runUploadFile(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("sync upload")
	expires := fs.Int("expires", 0, "Expiration time in hours (server default when 0)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: sync upload <file> [--expires HOURS]")
	}
	filePath := fs.Arg(0)

	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("file not found: %s", filePath)
	}
	if info.IsDir() {
		return fmt.Errorf("path is a directory, not a file: %s", filePath)
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	fileWriter, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return fmt.Errorf("prepare upload: %w", err)
	}
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	if _, err := io.Copy(fileWriter, file); err != nil {
		file.Close()
		return fmt.Errorf("read file: %w", err)
	}
	file.Close()

	_ = writer.WriteField("content_type", "file")
	if *expires > 0 {
		_ = writer.WriteField("expires_in", strconv.Itoa(*expires))
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("finalize upload: %w", err)
	}

	respBody, err := doFormRequest(core, "POST", "/sync/upload", writer.FormDataContentType(), body.Bytes())
	if err != nil {
		return err
	}

	var resp support.UploadResponse
	if err := support.Decode(respBody, &resp); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("File uploaded: %s", filepath.Base(filePath)),
			fmt.Sprintf("Item ID: %s", resp.ItemID),
			fmt.Sprintf("Expires: %s", resp.ExpiresAt),
		},
		Changes: []string{"Created sync item and broadcast to connected devices"},
		NextCommand: []string{
			fmt.Sprintf("%s sync get %s", support.CLIName, resp.ItemID),
			fmt.Sprintf("%s sync download %s", support.CLIName, resp.ItemID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runUploadText(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("sync upload-text")
	expires := fs.Int("expires", 0, "Expiration time in hours (server default when 0)")
	clipboard := fs.Bool("clipboard", false, "Mark content as clipboard text")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: sync upload-text <text> [--expires HOURS] [--clipboard]")
	}
	text := strings.Join(fs.Args(), " ")

	contentType := "text"
	if *clipboard {
		contentType = "clipboard"
	}

	payload := map[string]interface{}{
		"text":         text,
		"content_type": contentType,
	}
	if *expires > 0 {
		payload["expires_in"] = *expires
	}

	respBody, err := core.Request("POST", "/sync/upload", nil, payload)
	if err != nil {
		return err
	}
	var resp support.UploadResponse
	if err := support.Decode(respBody, &resp); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("Text uploaded (%s)", contentType),
			fmt.Sprintf("Item ID: %s", resp.ItemID),
			fmt.Sprintf("Expires: %s", resp.ExpiresAt),
		},
		Changes: []string{"Created sync item and broadcast to connected devices"},
		NextCommand: []string{
			fmt.Sprintf("%s sync get %s", support.CLIName, resp.ItemID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runClipboard(core *cliapp.ScenarioApp, args []string) error {
	return runTargetedPost(core, args, "sync clipboard", "/sync/clipboard", "Clipboard pushed")
}

func runNotification(core *cliapp.ScenarioApp, args []string) error {
	return runTargetedPost(core, args, "sync notification", "/sync/notification", "Notification dispatched")
}

// runTargetedPost sends a JSON payload from --body-file to an endpoint whose
// request shape is variable (e.g., notification title/body/icon or clipboard
// content plus target_devices). Building these payloads hand in Go would be
// brittle and duplicate server-side validation; the body-file approach keeps
// the wrapper thin.
func runTargetedPost(core *cliapp.ScenarioApp, args []string, cmdName, path, verb string) error {
	fs := support.NewFlagSet(cmdName)
	bodyFile := fs.String("body-file", "", "Path to a JSON file with the request payload")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	payload, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}

	respBody, err := core.Request("POST", path, nil, payload)
	if err != nil {
		return err
	}

	var item support.SyncItem
	_ = support.Decode(respBody, &item)

	message := verb
	if item.ID != "" {
		message = fmt.Sprintf("%s (item %s)", verb, item.ID)
	}
	report := cliapp.MutationReport{
		Result:  []string{message},
		Changes: []string{"Broadcast to target devices via WebSocket"},
	}
	if item.ID != "" {
		report.NextCommand = []string{fmt.Sprintf("%s sync get %s", support.CLIName, item.ID)}
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runDownload(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("sync download")
	output := fs.String("output", "", "Output path (defaults to <item-id>)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: sync download <item-id> [--output PATH]")
	}
	id := fs.Arg(0)

	outPath := strings.TrimSpace(*output)
	if outPath == "" {
		outPath = id
	}
	if fs.NArg() >= 2 && outPath == id {
		outPath = fs.Arg(1)
	}

	filename, n, err := downloadToFile(core, "/sync/items/"+id+"/download", outPath)
	if err != nil {
		_ = os.Remove(outPath)
		return err
	}

	report := cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("Downloaded %d bytes", n),
			fmt.Sprintf("Saved to: %s", outPath),
		},
		Changes: []string{fmt.Sprintf("Server filename: %s", filename)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runDelete(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("sync delete")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: sync delete <item-id>")
	}
	id := fs.Arg(0)

	body, err := core.Request("DELETE", "/sync/items/"+id, nil, nil)
	if err != nil {
		return err
	}
	var resp support.DeleteResponse
	_ = support.Decode(body, &resp)

	msg := fmt.Sprintf("Sync item %s deleted", id)
	if resp.DeletedAt != "" {
		msg = fmt.Sprintf("%s at %s", msg, resp.DeletedAt)
	}
	report := cliapp.MutationReport{
		Result:      []string{msg},
		Changes:     []string{"Deleted item and broadcast removal to connected devices"},
		NextCommand: []string{fmt.Sprintf("%s sync list", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func itemRows(items []support.SyncItem) []string {
	if len(items) == 0 {
		return []string{"No active items"}
	}
	rows := make([]string, 0, len(items))
	for _, item := range items {
		filename := "-"
		if name, ok := item.Content["filename"].(string); ok && name != "" {
			filename = name
		} else if text, ok := item.Content["text"].(string); ok && text != "" {
			filename = truncate(text, 40)
		}
		size := "-"
		if s, ok := item.Content["file_size"].(float64); ok && s > 0 {
			size = fmt.Sprintf("%dB", int64(s))
		}
		rows = append(rows, fmt.Sprintf("%s | %s | %s | size=%s | expires=%s",
			support.ShortID(item.ID), item.Type, filename, size, support.FormatTimeValue(item.ExpiresAt)))
	}
	return rows
}

func truncate(s string, max int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// doFormRequest sends an HTTP request whose body is non-JSON (e.g., multipart).
// It resolves the API base + auth token from the ScenarioApp, so calls stay
// aligned with the standard API configuration plumbing.
func doFormRequest(core *cliapp.ScenarioApp, method, path, contentType string, body []byte) ([]byte, error) {
	endpoint, err := buildEndpoint(core, path)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(method, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "application/json")
	applyAuth(core, req)

	client := &http.Client{Timeout: clientTimeout(core)}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, cliutil.ParseAPIError(resp.StatusCode, data)
	}
	return data, nil
}

// downloadToFile streams a GET response directly to disk so large files do not
// balloon memory. Returns the server-provided filename (parsed from
// Content-Disposition) and the byte count written.
func downloadToFile(core *cliapp.ScenarioApp, path, outPath string) (string, int64, error) {
	endpoint, err := buildEndpoint(core, path)
	if err != nil {
		return "", 0, err
	}
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return "", 0, fmt.Errorf("create request: %w", err)
	}
	applyAuth(core, req)

	client := &http.Client{Timeout: clientTimeout(core)}
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		data, _ := io.ReadAll(resp.Body)
		return "", 0, cliutil.ParseAPIError(resp.StatusCode, data)
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil && filepath.Dir(outPath) != "." {
		return "", 0, fmt.Errorf("create output directory: %w", err)
	}
	f, err := os.Create(outPath)
	if err != nil {
		return "", 0, fmt.Errorf("create %s: %w", outPath, err)
	}
	defer f.Close()

	n, err := io.Copy(f, resp.Body)
	if err != nil {
		return "", n, fmt.Errorf("write %s: %w", outPath, err)
	}

	filename := parseFilename(resp.Header.Get("Content-Disposition"))
	return filename, n, nil
}

func buildEndpoint(core *cliapp.ScenarioApp, path string) (string, error) {
	base := strings.TrimRight(core.APIRootBase(), "/")
	if base == "" {
		return "", fmt.Errorf("api base URL is empty; configure an API base or set an API port")
	}
	return base + core.APIPath(path), nil
}

func applyAuth(core *cliapp.ScenarioApp, req *http.Request) {
	if core.APIClient == nil {
		return
	}
	for key, value := range core.APIClient.AuthHeaders() {
		req.Header.Set(key, value)
	}
}

func clientTimeout(core *cliapp.ScenarioApp) time.Duration {
	if core.HTTPClient == nil {
		return 60 * time.Second
	}
	if t := core.HTTPClient.Timeout(); t > 0 {
		return t
	}
	return 60 * time.Second
}

func parseFilename(disposition string) string {
	if disposition == "" {
		return ""
	}
	parts := strings.Split(disposition, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(strings.ToLower(part), "filename=") {
			name := strings.TrimPrefix(part, "filename=")
			name = strings.TrimPrefix(name, "\"")
			name = strings.TrimSuffix(name, "\"")
			return name
		}
	}
	return ""
}
