package query

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestSharedAPIQueryFixturePreservesCLIHumanAndMachineContracts(t *testing.T) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source path")
	}
	fixturePath := filepath.Join(filepath.Dir(sourceFile), "../../../api/handlers/routing/testdata/query-task-response.json")
	fixture, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read shared API response fixture: %v", err)
	}
	var response routingv1.QueryResponse
	if err := protojson.Unmarshal(fixture, &response); err != nil {
		t.Fatalf("decode shared API ProtoJSON: %v", err)
	}

	h := &handlers{}
	report := h.queryReport(nil, &response)
	var human bytes.Buffer
	if err := cliapp.RenderListReport(&human, report); err != nil {
		t.Fatalf("render ordinary human output: %v", err)
	}
	humanText := human.String()
	for _, want := range []string{
		"Task evidence by request part",
		"Find the relevant interface",
		"HealthCard component",
		"unrecognized retrieval source (93)",
		"Find routing guidance",
		"supporting hit references unavailable",
		"Identify the owning capability",
		"not routed",
		"Preserve a future coverage state",
		"unrecognized status (93)",
		"Preserve a future request-part source",
		"unrecognized source (93)",
	} {
		if !strings.Contains(humanText, want) {
			t.Errorf("human report omitted %q: %s", want, humanText)
		}
	}
	if len(report.RetrievalHints) == 0 || !strings.Contains(report.RetrievalHints[0], `request part "Find routing guidance"`) {
		t.Fatalf("next action did not identify the first uncovered part: %#v", report.RetrievalHints)
	}
	if strings.Contains(report.RetrievalHints[0], "HealthCard component") || strings.Contains(report.RetrievalHints[0], "docs") {
		t.Fatalf("scoped continuation borrowed a different part's evidence: %q", report.RetrievalHints[0])
	}

	call := func(cliapp.OperationContext) (*routingv1.QueryResponse, error) { return &response, nil }
	render := func(ctx cliapp.OperationContext, msg *routingv1.QueryResponse) cliapp.ListReport {
		return h.queryReport(ctx, msg)
	}
	machineHandler := cliapp.ProtoListWithConnectJSONFlag(call, render, "json-connect")
	for _, connectNames := range []bool{false, true} {
		var machine bytes.Buffer
		ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
			Schema: cliapp.ArgSchema{
				Positionals: []cliapp.Positional{{Name: "text"}},
				Flags: []cliapp.Flag{
					{Name: "json-connect", Bool: true},
					{Name: "show-all-results", Bool: true},
					{Name: "explain", Bool: true},
				},
			},
			Positionals: map[string]string{"text": "Find HealthCard route and owner"},
			JSON:        true, BoolFlags: map[string]bool{"json-connect": connectNames}, Stdout: &machine,
		})
		if err := machineHandler.Run(ctx); err != nil {
			t.Fatalf("render machine JSON (connect names %t): %v", connectNames, err)
		}
		output := machine.String()
		partField, hitIDField, statusField := "query_parts", "hit_id", "status"
		retrievalSourcesField := "retrieval_sources"
		if connectNames {
			partField, hitIDField, statusField = "queryParts", "hitId", "status"
			retrievalSourcesField = "retrievalSources"
		}
		var envelope map[string]any
		if err := json.Unmarshal([]byte(output), &envelope); err != nil {
			t.Fatalf("decode machine JSON: %v", err)
		}
		parts, ok := envelope[partField].([]any)
		if !ok || len(parts) != 5 {
			t.Errorf("machine contract (connect names %t) lost typed query parts: %s", connectNames, output)
			continue
		}
		stalePart, ok := parts[1].(map[string]any)
		if !ok {
			t.Errorf("stale part was not an object: %s", output)
			continue
		}
		refs, ok := stalePart["evidence"].([]any)
		if !ok || len(refs) != 1 {
			t.Errorf("stale reference was lost: %s", output)
			continue
		}
		staleRef, ok := refs[0].(map[string]any)
		surfacePart, surfaceOK := parts[0].(map[string]any)
		surfaceEvidence, evidenceOK := surfacePart["evidence"].([]any)
		var retrievalSources []any
		if evidenceOK && len(surfaceEvidence) == 1 {
			if surfaceRef, ok := surfaceEvidence[0].(map[string]any); ok {
				retrievalSources, _ = surfaceRef[retrievalSourcesField].([]any)
			}
		}
		futurePart, futurePartOK := parts[3].(map[string]any)
		futureSourcePart, futureSourcePartOK := parts[4].(map[string]any)
		if !ok || staleRef[hitIDField] != "stale-hit" || !surfaceOK || len(retrievalSources) != 2 || retrievalSources[1] != float64(93) || parts[2].(map[string]any)[statusField] != "STATUS_NOT_ROUTED" || !futurePartOK || futurePart[statusField] != float64(93) || !futureSourcePartOK || futureSourcePart["source"] != float64(93) {
			t.Errorf("machine contract (connect names %t) lost typed evidence: %s", connectNames, output)
		}
		if strings.Contains(output, "Continue with request part") || strings.Contains(output, "retrieval_hints") {
			t.Errorf("human-only continuation leaked into machine output: %s", output)
		}
	}
}
