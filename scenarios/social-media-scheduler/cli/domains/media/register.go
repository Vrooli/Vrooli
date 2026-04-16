package media

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"social-media-scheduler/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register wraps /api/v1/media/*. Upload is a multipart request so it goes
// through a direct http.Client; everything else uses core.Request.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "media",
		Description: "Upload and manage media files",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "upload", Description: "Upload a media file (--file PATH)", Run: func(args []string) error { return runUpload(core, args) }},
			{Name: "list", Aliases: []string{"ls"}, Description: "List uploaded media", Run: func(args []string) error { return runList(core, args) }},
			{Name: "delete", Aliases: []string{"rm"}, Description: "Delete a media file", Run: func(args []string) error { return runDelete(core, args) }},
			{Name: "optimize", Description: "Optimize a media file for target platforms (--body-file PATH)", Run: func(args []string) error { return runOptimize(core, args) }},
		},
	}
}

func runUpload(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("media upload")
	filePath := fs.String("file", "", "Path to the media file to upload (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*filePath) == "" && fs.NArg() >= 1 {
		*filePath = fs.Arg(0)
	}
	if strings.TrimSpace(*filePath) == "" {
		return fmt.Errorf("usage: media upload --file PATH")
	}

	info, err := os.Stat(*filePath)
	if err != nil {
		return fmt.Errorf("file not found: %s", *filePath)
	}
	if info.IsDir() {
		return fmt.Errorf("path is a directory, not a file: %s", *filePath)
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	fileWriter, err := writer.CreateFormFile("file", filepath.Base(*filePath))
	if err != nil {
		return fmt.Errorf("prepare upload: %w", err)
	}
	file, err := os.Open(*filePath)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	if _, err := io.Copy(fileWriter, file); err != nil {
		file.Close()
		return fmt.Errorf("read file: %w", err)
	}
	file.Close()
	if err := writer.Close(); err != nil {
		return fmt.Errorf("finalize upload: %w", err)
	}

	respBody, err := doFormRequest(core, "POST", "/media/upload", writer.FormDataContentType(), body.Bytes())
	if err != nil {
		return err
	}
	var generic map[string]interface{}
	_ = support.Decode(respBody, &generic)

	report := cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Uploaded %s (%d bytes)", filepath.Base(*filePath), info.Size())},
		Changes: support.MapRows(generic),
		NextCommand: []string{
			fmt.Sprintf("%s media list", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("media list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/media", nil)
	if err != nil {
		return err
	}
	var items []map[string]interface{}
	if err := support.Decode(body, &items); err != nil {
		return err
	}

	rows := make([]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, renderMediaRow(item))
	}
	if len(rows) == 0 {
		rows = []string{"No media files found"}
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Media files: %d", len(items))},
		ResultsHeading: "Media",
		Results:        rows,
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runDelete(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("media delete")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: media delete <media-id>")
	}
	id := fs.Arg(0)

	body, err := core.Request("DELETE", "/media/"+id, nil, nil)
	if err != nil {
		return err
	}
	msg := support.EnvelopeMessage(body)
	if msg == "" {
		msg = fmt.Sprintf("Deleted media %s", id)
	}
	report := cliapp.MutationReport{
		Result:      []string{msg},
		Changes:     []string{fmt.Sprintf("Media %s deleted", id)},
		NextCommand: []string{fmt.Sprintf("%s media list", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runOptimize(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("media optimize")
	bodyFile := fs.String("body-file", "", "Path to a JSON file with the optimization payload")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: media optimize <media-id> --body-file PATH")
	}
	id := fs.Arg(0)
	payload, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.Request("POST", "/media/"+id+"/optimize", nil, payload)
	if err != nil {
		return err
	}
	var generic map[string]interface{}
	_ = support.Decode(body, &generic)

	msg := support.EnvelopeMessage(body)
	if msg == "" {
		msg = fmt.Sprintf("Optimization requested for media %s", id)
	}
	report := cliapp.MutationReport{
		Result:  []string{msg},
		Changes: support.MapRows(generic),
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func renderMediaRow(item map[string]interface{}) string {
	id, _ := item["id"].(string)
	name := support.RenderValue(item["filename"])
	contentType := support.RenderValue(item["content_type"])
	return fmt.Sprintf("%s | %s | %s", support.ShortID(id), name, contentType)
}

// doFormRequest sends a multipart request using core's resolved API base and
// auth headers, while staying on a dedicated http.Client for the non-JSON body.
func doFormRequest(core *cliapp.ScenarioApp, method, path, contentType string, body []byte) ([]byte, error) {
	base := strings.TrimRight(core.APIRootBase(), "/")
	if base == "" {
		return nil, fmt.Errorf("api base URL is empty; configure an API base or set an API port")
	}
	endpoint := base + core.APIPath(path)

	req, err := http.NewRequest(method, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "application/json")
	if core.APIClient != nil {
		for key, value := range core.APIClient.AuthHeaders() {
			req.Header.Set(key, value)
		}
	}

	timeout := 60 * time.Second
	if core.HTTPClient != nil {
		if t := core.HTTPClient.Timeout(); t > 0 {
			timeout = t
		}
	}
	client := &http.Client{Timeout: timeout}
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
