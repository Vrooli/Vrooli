// Package structureprovider is the client-side seam for project structure
// validation. The structure-health scenario is the only authority for the
// verdict; callers in the control plane only transport and render its result.
package structureprovider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	contractapp "github.com/vrooli/vrooli/internal/app/contract"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
)

const (
	ScenarioName   = "structure-health"
	TargetID       = "repo"
	DefaultTimeout = 10 * time.Second
)

// ErrUnavailable identifies a missing or unreachable structure-health
// authority. Callers must surface this error rather than manufacture a local
// structural verdict.
var ErrUnavailable = errors.New("structure-health unavailable")

// URLResolver resolves the live scenario API URL. It is injectable so callers
// can test delegation without relying on a running control plane.
type URLResolver func(context.Context, string) (string, error)

// Client is the narrow provider seam used by hygiene and contract validation.
type Client interface {
	Validate(context.Context, string) (contractapp.ValidationOutput, error)
}

// Provider calls structure-health's shared scenario-validation contract.
type Provider struct {
	ResolveURL URLResolver
	HTTPClient *http.Client
	Timeout    time.Duration
}

// NewDefault returns the production provider client.
func NewDefault() Provider {
	return Provider{
		ResolveURL: discovery.ResolveScenarioURLDefault,
		Timeout:    DefaultTimeout,
	}
}

// Validate delegates project:repo validation to structure-health.
func (p Provider) Validate(ctx context.Context, root string) (contractapp.ValidationOutput, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return contractapp.ValidationOutput{}, fmt.Errorf("%w: repository root is required", ErrUnavailable)
	}
	resolveURL := p.ResolveURL
	if resolveURL == nil {
		resolveURL = discovery.ResolveScenarioURLDefault
	}
	timeout := p.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	baseURL, err := resolveURL(callCtx, ScenarioName)
	if err != nil {
		return contractapp.ValidationOutput{}, fmt.Errorf("%w: resolve %s: %v", ErrUnavailable, ScenarioName, err)
	}
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return contractapp.ValidationOutput{}, fmt.Errorf("%w: %s returned an empty API URL", ErrUnavailable, ScenarioName)
	}
	httpClient := p.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: timeout}
	}
	client := scenariovalidationconnect.NewScenarioValidationServiceClient(httpClient, baseURL)
	response, err := client.ValidateTarget(callCtx, connect.NewRequest(&scenariovalidationv1.ValidateTargetRequest{
		Target: &commonv1.ValidationTarget{
			Kind: commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_PROJECT,
			Id:   TargetID,
			Root: root,
		},
		Path: root,
	}))
	if err != nil {
		return contractapp.ValidationOutput{}, fmt.Errorf("%w: validate %s:%s: %v", ErrUnavailable, ScenarioName, TargetID, err)
	}
	if response == nil || response.Msg == nil {
		return contractapp.ValidationOutput{}, fmt.Errorf("%w: %s returned an empty validation response", ErrUnavailable, ScenarioName)
	}
	return outputFromResponse(root, response.Msg), nil
}

func outputFromResponse(root string, response *scenariovalidationv1.ValidateTargetResponse) contractapp.ValidationOutput {
	checks := make([]contractapp.CheckResult, 0, len(projectChecks))
	failures := map[string][]string{}
	if assessment := response.GetAssessment(); assessment != nil {
		for _, finding := range assessment.GetFindings() {
			if finding == nil {
				continue
			}
			name := checkName(finding.GetCode())
			if name == "" {
				name = finding.GetCode()
			}
			message := strings.TrimSpace(finding.GetMessage())
			if message == "" {
				message = "structure-health reported a finding"
			}
			failures[name] = append(failures[name], message)
		}
	}
	for _, check := range projectChecks {
		messages := failures[check.name]
		message := "ok"
		passed := len(messages) == 0
		if !passed {
			message = strings.Join(messages, "; ")
		}
		checks = append(checks, contractapp.CheckResult{Name: check.name, Passed: passed, Message: message})
	}
	for name, messages := range failures {
		if checkKnown(name) {
			continue
		}
		checks = append(checks, contractapp.CheckResult{Name: name, Passed: false, Message: strings.Join(messages, "; ")})
	}

	schemaPassed := true
	schemaMessage := "delegated to structure-health"
	if messages := failures["project_contract_invalid"]; len(messages) > 0 {
		schemaPassed = false
		schemaMessage = strings.Join(messages, "; ")
	}
	success := response.GetStatus() == scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED
	return contractapp.ValidationOutput{
		Success: success,
		Root:    root,
		Schema:  contractapp.ValidationCheck{Passed: schemaPassed, Message: schemaMessage},
		Report: contractapp.Report{
			Root:         root,
			ContractPath: filepath.Join(root, ".vrooli", "repo-contract.json"),
			Success:      success,
			Checks:       checks,
		},
	}
}

type projectCheck struct {
	name string
	code string
}

var projectChecks = []projectCheck{
	{name: "phase1_semantics", code: "PROJECT_PHASE1_SEMANTICS"},
	{name: "canonical_markers_and_paths", code: "PROJECT_CANONICAL_LAYOUT"},
	{name: "runtime_home_section", code: "PROJECT_RUNTIME_HOME"},
	{name: "live_repo_structure", code: "PROJECT_LIVE_STRUCTURE"},
	{name: "project_config_surface", code: "PROJECT_CONFIG_SURFACE"},
	{name: "excluded_legacy_rules_and_paths", code: "PROJECT_EXCLUDED_LEGACY"},
	{name: "profile_roots_within_canonical_layout", code: "PROJECT_PROFILE_ROOTS"},
	{name: "bundle_profile_policy", code: "PROJECT_BUNDLE_PROFILE"},
	{name: "docs_alignment", code: "PROJECT_DOCS_ALIGNMENT"},
	{name: "resource_schema_artifacts", code: "PROJECT_RESOURCE_ARTIFACTS"},
}

func checkName(code string) string {
	for _, check := range projectChecks {
		if check.code == code {
			return check.name
		}
	}
	if code == "PROJECT_CONTRACT_INVALID" {
		return "project_contract_invalid"
	}
	return ""
}

func checkKnown(name string) bool {
	for _, check := range projectChecks {
		if check.name == name {
			return true
		}
	}
	return name == "project_contract_invalid"
}
