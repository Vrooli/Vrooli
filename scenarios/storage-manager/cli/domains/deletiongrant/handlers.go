package deletiongrant

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
)

const (
	requestPath = "/api/v1/deletion-grants/request"
	// webConsoleSessionEnv is the stable handle a web-console-hosted agent is
	// woken through when its identity token has aged out.
	webConsoleSessionEnv = "WC_WEB_CONSOLE_SESSION_ID"
)

type handlers struct {
	httpClient connect.HTTPClient
	baseURL    string
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{httpClient: httpClient, baseURL: baseURL}
}

// pending mirrors the request endpoint's 202 answer.
type pending struct {
	Status     string   `json:"status"`
	DecisionID string   `json:"decision_id"`
	Paths      []string `json:"paths"`
	Message    string   `json:"message"`
	Await      struct {
		ParkCommand string `json:"park_command"`
	} `json:"await"`
}

func (h *handlers) requestCall(ctx cliapp.OperationContext) (pending, error) {
	paths := ctx.FlagValues("path")
	if file := strings.TrimSpace(ctx.Flag("paths-file")); file != "" {
		listed, err := readPathsFile(file)
		if err != nil {
			return pending{}, err
		}
		paths = append(paths, listed...)
	}
	if len(paths) == 0 {
		return pending{}, fmt.Errorf("--path or --paths-file is required")
	}
	reason := strings.TrimSpace(ctx.Flag("reason"))
	if reason == "" {
		return pending{}, fmt.Errorf("--reason is required: the operator decides from it")
	}
	payload, err := json.Marshal(map[string]any{
		"paths":                  paths,
		"reason":                 reason,
		"expires_in":             strings.TrimSpace(ctx.Flag("expires-in")),
		"web_console_session_id": strings.TrimSpace(os.Getenv(webConsoleSessionEnv)),
	})
	if err != nil {
		return pending{}, err
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, h.baseURL+requestPath, bytes.NewReader(payload))
	if err != nil {
		return pending{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return pending{}, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return pending{}, err
	}
	if resp.StatusCode != http.StatusAccepted {
		return pending{}, fmt.Errorf("deletion grant request refused (%d): %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	var out pending
	if err := json.Unmarshal(data, &out); err != nil {
		return pending{}, fmt.Errorf("decode deletion grant answer: %w", err)
	}
	return out, nil
}

func requestReport(_ cliapp.OperationContext, p pending) cliapp.MutationReport {
	report := cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Operator approval requested (decision %s). The deletion is permitted once the operator approves it in the decision inbox; retry it then.", p.DecisionID)},
		Changes: make([]string, 0, len(p.Paths)),
	}
	// A large request names a sample rather than flooding the terminal; the
	// decision itself lists the paths for the operator.
	const shown = 10
	for i, path := range p.Paths {
		if i == shown && len(p.Paths) > shown+1 {
			report.Changes = append(report.Changes, fmt.Sprintf("… and %d more", len(p.Paths)-shown))
			break
		}
		report.Changes = append(report.Changes, "requested "+path)
	}
	if p.Await.ParkCommand != "" {
		report.NextCommand = []string{p.Await.ParkCommand}
	}
	return report
}

// readPathsFile reads one absolute path per line. Blank lines and lines starting
// with # are skipped, and anything after a tab is ignored, so an inventory with
// a size column can be passed as is.
func readPathsFile(name string) ([]string, error) {
	data, err := os.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("read --paths-file: %w", err)
	}
	var paths []string
	for _, line := range strings.Split(string(data), "\n") {
		line, _, _ = strings.Cut(line, "\t")
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		paths = append(paths, line)
	}
	return paths, nil
}
