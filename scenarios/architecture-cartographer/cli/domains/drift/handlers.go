package drift

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"google.golang.org/protobuf/types/known/structpb"
)

type handlers struct {
	client connect.HTTPClient
	base   string
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	client, base := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: client, base: strings.TrimRight(base, "/")}
}

func (h *handlers) searchCall(ctx cliapp.OperationContext) (*structpb.Struct, error) {
	scenario, query := strings.TrimSpace(ctx.Positional("scenario")), strings.TrimSpace(ctx.Positional("query"))
	if scenario == "" || query == "" {
		return nil, fmt.Errorf("scenario and query are required")
	}
	limit := 10
	body, err := json.Marshal(map[string]any{"scope": "scenario:" + scenario, "query": query, "limit": limit})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, h.base+"/api/v1/doc-code-drift/search", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, cliapp.WrapAPIError("search docs/code drift", err, nil)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("docs/code drift search returned HTTP %d", resp.StatusCode)
	}
	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode docs/code drift response: %w", err)
	}
	return structpb.NewStruct(result)
}

func (h *handlers) searchReport(_ cliapp.OperationContext, response *structpb.Struct) cliapp.MutationReport {
	body := response.AsMap()
	raw, _ := body["results"].([]any)
	lines := make([]string, 0, len(raw)+1)
	for _, value := range raw {
		row, _ := value.(map[string]any)
		doc := source(row["documentation"])
		var code []string
		if rows, ok := row["code"].([]any); ok {
			for _, v := range rows {
				code = append(code, source(v))
			}
		}
		var api []string
		if rows, ok := row["api"].([]any); ok {
			for _, v := range rows {
				api = append(api, source(v)+" ("+sourceBasis(v)+")")
			}
		}
		lines = append(lines, fmt.Sprintf("%v · %v [%v/%v] docs %s; code %s; API %v: %s", row["domain"], row["claim"], row["classification"], row["status"], doc, strings.Join(code, "; "), row["api_status"], strings.Join(api, "; ")))
	}
	for _, value := range body["degradations"].([]any) {
		lines = append(lines, "degraded: "+fmt.Sprint(value))
	}
	return cliapp.MutationReport{Result: append([]string{fmt.Sprintf("%d source-path drift candidate(s). Structural matches do not prove semantic conformance. %v", len(raw), body["scope"])}, lines...)}
}

func sourceBasis(value any) string {
	m, _ := value.(map[string]any)
	if m == nil {
		return "UNKNOWN"
	}
	return fmt.Sprint(m["basis"])
}

func source(value any) string {
	m, _ := value.(map[string]any)
	if m == nil {
		return "UNKNOWN"
	}
	return fmt.Sprintf("%v:%v sha256:%v", m["path"], m["line"], m["sha256"])
}
