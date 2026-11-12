package resourcecli

import (
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
	n8nresource "github.com/vrooli/vrooli/internal/resources/n8n"
)

type N8NRuntimeRequest struct {
	SandboxID                  string
	RuntimeID                  string
	CompanionRuntimeID         string
	PostgresCompanionRuntimeID string
	CompanionName              string
	ServiceVersion             string
	Database                   string
	HTTPMockConfigJSON         []byte
	OperationID                uuid.UUID
	JSON                       bool
}

func ParseN8NRuntimeRequest(action N8NRuntimeCommandID, args []string) (N8NRuntimeRequest, error) {
	var spec *commandtree.Spec[N8NRuntimeCommandID]
	for _, item := range N8NRuntimeCommandSpecs() {
		if item.Handler == action {
			copy := item
			spec = &copy
			break
		}
	}
	if spec == nil {
		return N8NRuntimeRequest{}, fmt.Errorf("unknown n8n runtime operation %q", action)
	}
	parsed, err := commandtree.ParseArgs("resource n8n runtime "+string(action), "", spec.Args, args)
	if err != nil {
		return N8NRuntimeRequest{}, err
	}
	request := N8NRuntimeRequest{
		SandboxID:                  strings.TrimSpace(parsed.FlagValue("--sandbox-id")),
		RuntimeID:                  strings.TrimSpace(parsed.FlagValue("--runtime-id")),
		CompanionRuntimeID:         strings.TrimSpace(parsed.FlagValue("--companion-runtime-id")),
		PostgresCompanionRuntimeID: strings.TrimSpace(parsed.FlagValue("--postgres-companion-runtime-id")),
		CompanionName:              strings.TrimSpace(parsed.FlagValue("--companion-name")),
		ServiceVersion:             strings.TrimSpace(parsed.FlagValue("--service-version")),
		Database:                   strings.TrimSpace(parsed.FlagValue("--database")),
		JSON:                       parsed.HasFlag("--json"),
	}
	switch action {
	case N8NRuntimeAcquire:
	case N8NRuntimeStart, N8NRuntimeStatus:
		if request.SandboxID == "" {
			return N8NRuntimeRequest{}, fmt.Errorf("--sandbox-id is required")
		}
	case N8NRuntimeStop:
		if request.SandboxID == "" || request.RuntimeID == "" {
			return N8NRuntimeRequest{}, fmt.Errorf("--sandbox-id and --runtime-id are required")
		}
	case N8NRuntimeAttachCompanion, N8NRuntimeDetachCompanion:
		if request.SandboxID == "" || request.RuntimeID == "" || request.CompanionRuntimeID == "" {
			return N8NRuntimeRequest{}, fmt.Errorf("--sandbox-id, --runtime-id and --companion-runtime-id are required")
		}
	case N8NRuntimeCompanionStart:
		if request.SandboxID == "" || request.CompanionName == "" || request.ServiceVersion == "" || request.Database == "" {
			return N8NRuntimeRequest{}, fmt.Errorf("--sandbox-id, --companion-name, --service-version and --database are required")
		}
		if request.CompanionName != n8nresource.SyntheticPostgresName || request.ServiceVersion != n8nresource.SyntheticPostgresVersion || request.Database != n8nresource.SyntheticPostgresDB {
			return N8NRuntimeRequest{}, fmt.Errorf("unsupported synthetic Postgres profile")
		}
	case N8NRuntimeCompanionStop:
		if request.SandboxID == "" || request.CompanionRuntimeID == "" {
			return N8NRuntimeRequest{}, fmt.Errorf("--sandbox-id and --companion-runtime-id are required")
		}
	case N8NRuntimeCompanionStartHTTPMock:
		if request.SandboxID == "" || !parsed.HasFlag("--config-stdin") {
			return N8NRuntimeRequest{}, fmt.Errorf("--sandbox-id and --config-stdin are required")
		}
	case N8NRuntimeCompanionStopHTTPMock:
		if request.SandboxID == "" || request.CompanionRuntimeID == "" {
			return N8NRuntimeRequest{}, fmt.Errorf("--sandbox-id and --companion-runtime-id are required")
		}
	case N8NRuntimeImportWorkflow, N8NRuntimeExecuteWorkflow, N8NRuntimeExecuteManualWorkflow:
		if request.SandboxID == "" || request.RuntimeID == "" || parsed.FlagValue("--operation-id") == "" {
			return N8NRuntimeRequest{}, fmt.Errorf("--sandbox-id, --runtime-id and --operation-id are required")
		}
		operationID, err := uuid.Parse(strings.TrimSpace(parsed.FlagValue("--operation-id")))
		if err != nil || operationID == uuid.Nil {
			return N8NRuntimeRequest{}, fmt.Errorf("--operation-id must be a nonzero UUID")
		}
		request.OperationID = operationID
	case N8NRuntimeDispatchWebhookEvent:
		if request.SandboxID == "" || request.RuntimeID == "" || parsed.FlagValue("--operation-id") == "" || !parsed.HasFlag("--event-stdin") {
			return N8NRuntimeRequest{}, fmt.Errorf("--sandbox-id, --runtime-id, --operation-id and --event-stdin are required")
		}
		operationID, err := uuid.Parse(strings.TrimSpace(parsed.FlagValue("--operation-id")))
		if err != nil || operationID == uuid.Nil {
			return N8NRuntimeRequest{}, fmt.Errorf("--operation-id must be a nonzero UUID")
		}
		request.OperationID = operationID
	}
	return request, nil
}

// ReadN8NWebhookEventInput reads one bounded synthetic JSON event from stdin.
func ReadN8NWebhookEventInput(input io.Reader) ([]byte, error) {
	if input == nil {
		return nil, fmt.Errorf("webhook event requires JSON on stdin")
	}
	data, err := io.ReadAll(io.LimitReader(input, n8nresource.MaxSyntheticWebhookEventBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read webhook event JSON from stdin: %w", err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("webhook event requires JSON on stdin")
	}
	if len(data) > n8nresource.MaxSyntheticWebhookEventBytes {
		return nil, fmt.Errorf("webhook event JSON exceeds %d-byte limit", n8nresource.MaxSyntheticWebhookEventBytes)
	}
	return data, nil
}

// ReadN8NHTTPMockInput reads a bounded JSON config from stdin only.
func ReadN8NHTTPMockInput(input io.Reader) ([]byte, error) {
	if input == nil {
		return nil, fmt.Errorf("HTTP mock requires JSON config on stdin")
	}
	data, err := io.ReadAll(io.LimitReader(input, n8nresource.MaxSyntheticHTTPMockConfigBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read HTTP mock config from stdin: %w", err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("HTTP mock requires JSON config on stdin")
	}
	if len(data) > n8nresource.MaxSyntheticHTTPMockConfigBytes {
		return nil, fmt.Errorf("HTTP mock config exceeds %d-byte limit", n8nresource.MaxSyntheticHTTPMockConfigBytes)
	}
	return data, nil
}

// ReadN8NWorkflowInput reads one workflow document without accepting a path or
// allowing an unbounded stdin payload at the CLI boundary.
func ReadN8NWorkflowInput(input io.Reader) ([]byte, error) {
	if input == nil {
		return nil, fmt.Errorf("workflow import requires JSON on stdin")
	}
	data, err := io.ReadAll(io.LimitReader(input, n8nresource.MaxSyntheticWorkflowBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read workflow JSON from stdin: %w", err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("workflow import requires JSON on stdin")
	}
	if len(data) > n8nresource.MaxSyntheticWorkflowBytes {
		return nil, fmt.Errorf("workflow JSON exceeds %d-byte limit", n8nresource.MaxSyntheticWorkflowBytes)
	}
	return data, nil
}
