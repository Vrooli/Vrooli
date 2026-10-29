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
	if len(paths) == 0 {
		return pending{}, fmt.Errorf("--path is required (repeat it for several locations)")
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
	for _, path := range p.Paths {
		report.Changes = append(report.Changes, "requested "+path)
	}
	if p.Await.ParkCommand != "" {
		report.NextCommand = []string{p.Await.ParkCommand}
	}
	return report
}
