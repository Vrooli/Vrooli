package resourcecli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/google/uuid"
	n8nresource "github.com/vrooli/vrooli/internal/resources/n8n"
)

const (
	testN8NSandboxID = "5e33a6f0-b1bf-4f5d-9315-95ba60d9e025"
	testN8NRuntimeID = "n8n:5e33a6f0-b1bf-4f5d-9315-95ba60d9e025"
	testN8NOperation = "f0a7d3e6-2a6b-4f8a-9714-1bb8be3eb1e2"
)

func TestParseN8NWorkflowBrokerCommands(t *testing.T) {
	importArgs := []string{
		"--sandbox-id", testN8NSandboxID,
		"--runtime-id", testN8NRuntimeID,
		"--operation-id", testN8NOperation,
		"--postgres-companion-runtime-id", "sandbox:postgres",
		"--json",
	}
	request, err := ParseN8NRuntimeRequest(N8NRuntimeImportWorkflow, importArgs)
	if err != nil {
		t.Fatal(err)
	}
	if request.SandboxID != testN8NSandboxID || request.RuntimeID != testN8NRuntimeID || request.OperationID != uuid.MustParse(testN8NOperation) || request.PostgresCompanionRuntimeID != "sandbox:postgres" || !request.JSON {
		t.Fatalf("unexpected import request: %#v", request)
	}
	execute, err := ParseN8NRuntimeRequest(N8NRuntimeExecuteWorkflow, importArgs[:6])
	if err != nil {
		t.Fatal(err)
	}
	if execute.OperationID != uuid.MustParse(testN8NOperation) || execute.PostgresCompanionRuntimeID != "" {
		t.Fatalf("unexpected execution request: %#v", execute)
	}
	manual, err := ParseN8NRuntimeRequest(N8NRuntimeExecuteManualWorkflow, importArgs[:6])
	if err != nil || manual.OperationID != uuid.MustParse(testN8NOperation) || manual.RuntimeID != testN8NRuntimeID {
		t.Fatalf("unexpected manual execution request: %#v, %v", manual, err)
	}
}

func TestParseN8NWorkflowBrokerRejectsFreeFormBoundaryInputs(t *testing.T) {
	base := []string{"--sandbox-id", testN8NSandboxID, "--runtime-id", testN8NRuntimeID, "--operation-id", testN8NOperation}
	for name, args := range map[string][]string{
		"workflow path positional": append(append([]string{}, base...), "workflow.json"),
		"caller workflow id":       append(append([]string{}, base...), "--workflow-id", "arbitrary"),
		"arbitrary command":        append(append([]string{}, base...), "--command", "rm -rf /"),
		"invalid operation id":     {"--sandbox-id", testN8NSandboxID, "--runtime-id", testN8NRuntimeID, "--operation-id", "not-a-uuid"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseN8NRuntimeRequest(N8NRuntimeImportWorkflow, args); err == nil {
				t.Fatalf("accepted forbidden broker input %#v", args)
			}
		})
	}
	if _, err := ParseN8NRuntimeRequest(N8NRuntimeExecuteWorkflow, append(base, "--postgres-companion-runtime-id", "extra")); err == nil {
		t.Fatal("execute accepted an undeclared Postgres companion option")
	}
	forbiddenManual := append(append([]string(nil), base...), "--workflow-id", "arbitrary")
	if _, err := ParseN8NRuntimeRequest(N8NRuntimeExecuteManualWorkflow, forbiddenManual); err == nil {
		t.Fatal("manual execute accepted a caller-selected workflow ID")
	}
}

func TestReadN8NWorkflowInputIsBounded(t *testing.T) {
	if _, err := ReadN8NWorkflowInput(strings.NewReader("")); err == nil {
		t.Fatal("empty stdin was accepted")
	}
	max := bytes.Repeat([]byte("x"), n8nresource.MaxSyntheticWorkflowBytes)
	got, err := ReadN8NWorkflowInput(bytes.NewReader(max))
	if err != nil || len(got) != len(max) {
		t.Fatalf("maximum valid payload: bytes=%d err=%v", len(got), err)
	}
	tooLarge := append(max, 'x')
	if _, err := ReadN8NWorkflowInput(bytes.NewReader(tooLarge)); err == nil {
		t.Fatal("over-limit stdin was accepted")
	}
}

func TestWebhookEventCommandAcceptsOnlyOwnerIDsAndBoundedStdin(t *testing.T) {
	args := []string{"--sandbox-id", testN8NSandboxID, "--runtime-id", testN8NRuntimeID, "--operation-id", testN8NOperation, "--event-stdin", "--json"}
	request, err := ParseN8NRuntimeRequest(N8NRuntimeDispatchWebhookEvent, args)
	if err != nil || request.OperationID != uuid.MustParse(testN8NOperation) || !request.JSON {
		t.Fatalf("dispatch request = %+v, %v", request, err)
	}
	for _, forbidden := range [][]string{
		args[:6],
		append(append([]string{}, args...), "--url", "http://example.com"),
		append(append([]string{}, args...), "--path", "/webhook/other"),
		append(append([]string{}, args...), "--workflow-id", "caller-id"),
		append(append([]string{}, args...), "--command", "curl"),
	} {
		if _, err := ParseN8NRuntimeRequest(N8NRuntimeDispatchWebhookEvent, forbidden); err == nil {
			t.Fatalf("accepted free-form webhook dispatch arguments: %#v", forbidden)
		}
	}
}

func TestReadN8NWebhookEventInputIsBounded(t *testing.T) {
	if _, err := ReadN8NWebhookEventInput(strings.NewReader("")); err == nil {
		t.Fatal("empty event stdin was accepted")
	}
	max := bytes.Repeat([]byte("x"), n8nresource.MaxSyntheticWebhookEventBytes)
	got, err := ReadN8NWebhookEventInput(bytes.NewReader(max))
	if err != nil || len(got) != len(max) {
		t.Fatalf("maximum event: bytes=%d err=%v", len(got), err)
	}
	if _, err := ReadN8NWebhookEventInput(bytes.NewReader(append(max, 'x'))); err == nil {
		t.Fatal("oversized event was accepted")
	}
}

func TestParseSyntheticPostgresOwnerCommandsAreFixedProfileOnly(t *testing.T) {
	start, err := ParseN8NRuntimeRequest(N8NRuntimeCompanionStart, []string{
		"--sandbox-id", testN8NSandboxID, "--companion-name", n8nresource.SyntheticPostgresName,
		"--service-version", n8nresource.SyntheticPostgresVersion, "--database", n8nresource.SyntheticPostgresDB, "--json",
	})
	if err != nil {
		t.Fatal(err)
	}
	if start.CompanionName != "pg" || start.ServiceVersion != "16.4" || start.Database != "workflow" || !start.JSON {
		t.Fatalf("unexpected fixed profile request: %+v", start)
	}
	for _, args := range [][]string{
		{"--sandbox-id", testN8NSandboxID, "--companion-name", "pg", "--service-version", "latest", "--database", "workflow"},
		{"--sandbox-id", testN8NSandboxID, "--companion-name", "pg", "--service-version", "16.4", "--database", "client"},
		{"--sandbox-id", testN8NSandboxID, "--companion-name", "pg", "--service-version", "16.4", "--database", "workflow", "--image", "evil"},
	} {
		if _, err := ParseN8NRuntimeRequest(N8NRuntimeCompanionStart, args); err == nil {
			t.Fatalf("accepted caller-selected profile or image: %#v", args)
		}
	}
	stop, err := ParseN8NRuntimeRequest(N8NRuntimeCompanionStop, []string{"--sandbox-id", testN8NSandboxID, "--companion-runtime-id", "sandbox.postgres:" + testN8NSandboxID, "--json"})
	if err != nil || stop.CompanionRuntimeID != "sandbox.postgres:"+testN8NSandboxID {
		t.Fatalf("stop request = %+v, %v", stop, err)
	}
	if _, err := ParseN8NRuntimeRequest(N8NRuntimeCompanionStop, []string{"--sandbox-id", testN8NSandboxID, "--companion-runtime-id", "x", "--container", "foreign"}); err == nil {
		t.Fatal("stop accepted arbitrary Docker object selection")
	}
}

func TestHTTPMockCLICommandsAreStdinBoundedAndHaveNoFreeFormFields(t *testing.T) {
	request, err := ParseN8NRuntimeRequest(N8NRuntimeCompanionStartHTTPMock, []string{"--sandbox-id", testN8NSandboxID, "--config-stdin", "--json"})
	if err != nil || request.SandboxID != testN8NSandboxID || !request.JSON {
		t.Fatalf("HTTP mock start request = %+v, %v", request, err)
	}
	for _, args := range [][]string{
		{"--sandbox-id", testN8NSandboxID},
		{"--sandbox-id", testN8NSandboxID, "--config-stdin", "fixtures.json"},
		{"--sandbox-id", testN8NSandboxID, "--config-stdin", "--server-script", "arbitrary"},
		{"--sandbox-id", testN8NSandboxID, "--config-stdin", "--port", "8080"},
	} {
		if _, err := ParseN8NRuntimeRequest(N8NRuntimeCompanionStartHTTPMock, args); err == nil {
			t.Fatalf("accepted free-form HTTP mock args: %#v", args)
		}
	}
	stop, err := ParseN8NRuntimeRequest(N8NRuntimeCompanionStopHTTPMock, []string{"--sandbox-id", testN8NSandboxID, "--companion-runtime-id", "n8n-httpmock:" + testN8NSandboxID + ":" + strings.Repeat("a", 64), "--json"})
	if err != nil || stop.CompanionRuntimeID == "" {
		t.Fatalf("HTTP mock stop request = %+v, %v", stop, err)
	}
	if _, err := ParseN8NRuntimeRequest(N8NRuntimeCompanionStopHTTPMock, []string{"--sandbox-id", testN8NSandboxID, "--companion-runtime-id", "owner", "--container", "foreign"}); err == nil {
		t.Fatal("HTTP mock stop accepted arbitrary container selection")
	}
}

func TestReadN8NHTTPMockInputIsBounded(t *testing.T) {
	if _, err := ReadN8NHTTPMockInput(strings.NewReader("")); err == nil {
		t.Fatal("empty config was accepted")
	}
	max := bytes.Repeat([]byte("x"), n8nresource.MaxSyntheticHTTPMockConfigBytes)
	got, err := ReadN8NHTTPMockInput(bytes.NewReader(max))
	if err != nil || len(got) != len(max) {
		t.Fatalf("maximum config: bytes=%d err=%v", len(got), err)
	}
	if _, err := ReadN8NHTTPMockInput(bytes.NewReader(append(max, 'x'))); err == nil {
		t.Fatal("oversized config was accepted")
	}
}
