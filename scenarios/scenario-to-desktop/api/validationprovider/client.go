// Package validationprovider adapts the shared durable scenario-validation
// contract to a target-owned desktop execution. It deliberately knows only
// the provider contract and normalized workflow path; workflow formats and
// browser automation remain owned by the provider.
package validationprovider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
)

type URLResolver interface {
	ResolveScenarioURLDefault(context.Context, string) (string, error)
}

type discoveryResolver struct{}

func (discoveryResolver) ResolveScenarioURLDefault(ctx context.Context, scenario string) (string, error) {
	return discovery.ResolveScenarioURLDefault(ctx, scenario)
}

type Request struct {
	MatrixRunID    string
	CellID         string
	ScenarioName   string
	ScenarioPath   string
	WorkflowPath   string
	WorkflowID     string
	Target         *domainv1.ElectronTarget
	ProfileID      string
	ContextID      string
	ArtifactDigest string
	Timeout        time.Duration
}

type Result struct {
	ProviderRunID string
	Passed        bool
	Reason        string
	Evidence      []*domainv1.LayeredEvidence
}

type Client struct {
	resolver URLResolver
	http     *http.Client
}

func NewWorkflowHealthClient() *Client {
	return NewClient(discoveryResolver{}, &http.Client{})
}

func NewClient(resolver URLResolver, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	return &Client{resolver: resolver, http: httpClient}
}

func (c *Client) Execute(ctx context.Context, request Request) Result {
	if c == nil || c.resolver == nil || c.http == nil {
		return Result{Reason: "validation provider client is unavailable"}
	}
	if request.Target == nil {
		return Result{Reason: "Electron target identity is missing"}
	}
	if strings.TrimSpace(request.WorkflowPath) == "" {
		return Result{Reason: "provider workflow path is missing"}
	}
	baseURL, err := c.resolver.ResolveScenarioURLDefault(ctx, "workflow-health")
	if err != nil {
		return Result{Reason: fmt.Sprintf("resolve validation provider: %v", err)}
	}
	client := scenariovalidationconnect.NewDurableValidationRunServiceClient(c.http, strings.TrimRight(baseURL, "/"))
	key := fmt.Sprintf("scenario-to-desktop:%s:%s", request.MatrixRunID, request.CellID)
	start, err := client.StartValidationRun(ctx, connect.NewRequest(&scenariovalidationv1.StartValidationRunRequest{
		Scenario:       request.ScenarioName,
		Path:           request.ScenarioPath,
		IdempotencyKey: key,
		ParentRunId:    request.MatrixRunID,
		DesktopBinding: &scenariovalidationv1.DesktopValidationBinding{
			TargetId:       request.Target.GetTargetId(),
			CdpEndpoint:    request.Target.GetCdpEndpoint(),
			RendererId:     request.Target.GetRendererId(),
			RendererUrl:    request.Target.GetRendererUrl(),
			RendererTitle:  request.Target.GetRendererTitle(),
			ScenarioName:   request.ScenarioName,
			ArtifactDigest: request.ArtifactDigest,
			ContextId:      request.ContextID,
			ProfileId:      request.ProfileID,
			CdpTransport:   request.Target.GetCdpTransport(),
			WorkflowPath:   request.WorkflowPath,
			WorkflowId:     request.WorkflowID,
		},
	}))
	if err != nil {
		return Result{Reason: fmt.Sprintf("start provider validation: %v", err)}
	}
	if start == nil || start.Msg == nil || start.Msg.GetRun() == nil || strings.TrimSpace(start.Msg.GetRun().GetRunId()) == "" {
		return Result{Reason: "validation provider returned no durable run identity"}
	}
	providerRunID := start.Msg.GetRun().GetRunId()
	timeout := request.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}
	wait, err := client.WaitValidationRun(ctx, connect.NewRequest(&scenariovalidationv1.WaitValidationRunRequest{
		RunId:   providerRunID,
		Timeout: durationpb.New(timeout),
	}))
	if err != nil {
		return Result{ProviderRunID: providerRunID, Reason: fmt.Sprintf("wait provider validation: %v", err)}
	}
	if wait == nil || wait.Msg == nil || wait.Msg.GetRun() == nil {
		return Result{ProviderRunID: providerRunID, Reason: "validation provider returned no terminal run"}
	}
	run := wait.Msg.GetRun()
	evidence, evidenceErr := artifactEvidence(run, request.ScenarioPath, request)
	result := Result{ProviderRunID: providerRunID, Passed: run.GetState() == scenariovalidationv1.ValidationRunState_VALIDATION_RUN_STATE_SUCCEEDED && run.GetTerminalResult().GetStatus() == scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED, Evidence: evidence}
	if run.GetError() != nil && strings.TrimSpace(run.GetError().GetMessage()) != "" {
		result.Reason = run.GetError().GetMessage()
	} else if !result.Passed {
		result.Reason = providerVerdictReason(run)
	}
	if evidenceErr != nil {
		result.Passed = false
		result.Reason = evidenceErr.Error()
	}
	if isolationErr := validateIsolationEvidence(run.GetTerminalResult().GetNativeDetail()); isolationErr != nil {
		result.Passed = false
		result.Reason = isolationErr.Error()
	}
	return result
}

func validateIsolationEvidence(native *anypb.Any) error {
	if native == nil {
		return fmt.Errorf("provider validation omitted routed isolation evidence")
	}
	detail := &structpb.Struct{}
	if err := native.UnmarshalTo(detail); err != nil {
		return fmt.Errorf("provider returned invalid native validation evidence: %w", err)
	}
	execution := structValue(detail, "execution")
	isolation := structValue(execution, "isolation")
	if isolation == nil {
		return fmt.Errorf("provider validation omitted routed isolation evidence")
	}
	if !boolValue(isolation, "installed") {
		return fmt.Errorf("provider validation did not prove routed test isolation")
	}
	if strings.TrimSpace(stringValue(isolation, "lease_id")) == "" {
		return fmt.Errorf("provider validation omitted routed isolation lease identity")
	}
	for _, field := range []string{"install_error", "heartbeat_error", "clear_error"} {
		if strings.TrimSpace(stringValue(isolation, field)) != "" {
			return fmt.Errorf("provider validation reported routed isolation %s", field)
		}
	}
	if numberValue(isolation, "primary_during_test_mode_requests") != 0 || numberValue(isolation, "primary_root_writes_during_test_mode") != 0 {
		return fmt.Errorf("provider validation reported primary-storage activity during test mode")
	}
	return nil
}

func structValue(parent *structpb.Struct, field string) *structpb.Struct {
	if parent == nil {
		return nil
	}
	value := parent.GetFields()[field]
	if value == nil {
		return nil
	}
	return value.GetStructValue()
}

func stringValue(parent *structpb.Struct, field string) string {
	if parent == nil || parent.GetFields()[field] == nil {
		return ""
	}
	return parent.GetFields()[field].GetStringValue()
}

func boolValue(parent *structpb.Struct, field string) bool {
	if parent == nil || parent.GetFields()[field] == nil {
		return false
	}
	return parent.GetFields()[field].GetBoolValue()
}

func numberValue(parent *structpb.Struct, field string) float64 {
	if parent == nil || parent.GetFields()[field] == nil {
		return 0
	}
	return parent.GetFields()[field].GetNumberValue()
}

func providerVerdictReason(run *scenariovalidationv1.ValidationRun) string {
	if run == nil {
		return "validation provider returned no run"
	}
	if terminal := run.GetTerminalResult(); terminal != nil && terminal.GetStatus() != scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_UNSPECIFIED {
		return "validation provider verdict: " + terminal.GetStatus().String()
	}
	return "validation provider did not pass the selected workflow"
}

func artifactEvidence(run *scenariovalidationv1.ValidationRun, scenarioPath string, request Request) ([]*domainv1.LayeredEvidence, error) {
	if run == nil {
		return nil, fmt.Errorf("provider returned no run for workflow evidence")
	}
	var evidence []*domainv1.LayeredEvidence
	for _, reference := range run.GetArtifactReferences() {
		path, ok := unpackReference(reference)
		if !ok {
			continue
		}
		if !filepath.IsAbs(path) {
			path = filepath.Join(scenarioPath, filepath.FromSlash(path))
		}
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		digest := sha256.Sum256(data)
		checksum := "sha256:" + hex.EncodeToString(digest[:])
		id := filepath.Base(path)
		if id == "." || id == string(filepath.Separator) || id == "" {
			id = "workflow-artifact"
		}
		uri := fmt.Sprintf("workflow-health://%s/%s", run.GetRunId(), strings.TrimLeft(filepath.ToSlash(path), "/"))
		mediaType := "application/json"
		evidence = append(evidence, &domainv1.LayeredEvidence{Kind: domainv1.LayeredEvidence_KIND_BAS_WORKFLOW, EvidenceId: "provider-" + id, Uri: uri, Sha256: checksum, MediaType: &mediaType, Redacted: true})
	}
	if len(evidence) == 0 {
		return nil, fmt.Errorf("provider validation completed without readable checksummed workflow evidence")
	}
	return evidence, nil
}

func unpackReference(value *anypb.Any) (string, bool) {
	if value == nil {
		return "", false
	}
	var object structpb.Struct
	if err := value.UnmarshalTo(&object); err != nil {
		return "", false
	}
	field := object.GetFields()["reference"]
	if field == nil {
		return "", false
	}
	path := strings.TrimSpace(field.GetStringValue())
	return path, path != ""
}
