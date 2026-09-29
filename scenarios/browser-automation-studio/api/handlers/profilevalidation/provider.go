package profilevalidation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/browser-automation-studio/internal/cancellationqualification"
	"github.com/vrooli/browser-automation-studio/internal/evidencecompletenessqualification"
	"github.com/vrooli/browser-automation-studio/internal/interactivefeedbackqualification"
	"github.com/vrooli/browser-automation-studio/internal/motionqualification"
	"github.com/vrooli/browser-automation-studio/internal/passivefidelityqualification"
	"github.com/vrooli/browser-automation-studio/internal/resourcebudgetqualification"
	"github.com/vrooli/maturity-go/assessment"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	"google.golang.org/protobuf/proto"
)

const (
	providerScenario   = "browser-automation-studio"
	providerPhase      = "rehabilitation-evidence"
	cohortEvidenceGlob = ".vrooli/runtime/rehabilitation-evidence/profile-durability-*.json"
)

type deps struct {
	ScenarioDir          string
	BuildIdentity        func(context.Context) (string, error)
	ReadRetainedArtifact func(context.Context, *scenariovalidationv1.RetainedEvidenceSet, *commonv1.EvidenceRef) ([]byte, error)
}

type cohort struct {
	ContractRow string            `json:"contractRow"`
	ContractSHA string            `json:"contractSha256"`
	SourceFiles map[string]string `json:"sourceFiles"`
	Runtime     struct {
		Build string `json:"managedBuildIdentityBeforeSeedAndAfterRestart"`
	} `json:"runtime"`
	Cohort struct {
		AllChecksPassed bool    `json:"allChecksPassed"`
		SeedChecks      int     `json:"seedChecks"`
		RestartChecks   int     `json:"restartChecks"`
		CheckpointMS    float64 `json:"checkpointVisibleAfterMs"`
		Isolation       string  `json:"alphaBetaIsolation"`
		Restart         string  `json:"alphaAndBetaAfterManagedApiDriverRestart"`
		Cleanup         int     `json:"syntheticProfilesDeleted"`
		Seed            struct {
			Path string `json:"path"`
			SHA  string `json:"sha256"`
		} `json:"seedOwnerReceipt"`
		RestartReceipt struct {
			Path string `json:"path"`
			SHA  string `json:"sha256"`
		} `json:"restartOwnerReceipt"`
	} `json:"cohort"`
}

type ownerReceipt struct {
	Stage        string `json:"stage"`
	ContractSHA  string `json:"contractSha256"`
	HarnessSHA   string `json:"harnessSha256"`
	BeforeHealth struct {
		Build string `json:"build_identity"`
	} `json:"beforeHealth"`
	Results []struct {
		Passed bool `json:"passed"`
	} `json:"results"`
}

type provider struct {
	scenariovalidationv1connect.UnimplementedScenarioValidationServiceHandler
	deps deps
	spec *assessment.Spec
}

type retainedEvidenceHandler struct {
	scenariovalidationv1connect.ScenarioValidationServiceHandler
}

func (h retainedEvidenceHandler) DescribeProvider(ctx context.Context, req *connect.Request[scenariovalidationv1.DescribeProviderRequest]) (*connect.Response[scenariovalidationv1.DescribeProviderResponse], error) {
	response, err := h.ScenarioValidationServiceHandler.DescribeProvider(ctx, req)
	if err != nil {
		return nil, err
	}
	copy := proto.Clone(response.Msg).(*scenariovalidationv1.DescribeProviderResponse)
	if copy.Capabilities == nil {
		copy.Capabilities = &scenariovalidationv1.ProviderCapabilities{}
	}
	copy.Capabilities.SupportsRetainedEvidence = true
	return connect.NewResponse(copy), nil
}

func Module(scenarioDir string) (scenariovalidationv1connect.ScenarioValidationServiceHandler, error) {
	spec, err := assessment.LoadSpecFromScenario(scenarioDir)
	if err != nil {
		return nil, err
	}
	describer, err := assessment.LoadDescriber(scenarioDir)
	if err != nil {
		return nil, err
	}
	p := &provider{deps: deps{ScenarioDir: scenarioDir, BuildIdentity: liveBuildIdentity, ReadRetainedArtifact: readRetainedArtifact}, spec: spec}
	return retainedEvidenceHandler{ScenarioValidationServiceHandler: assessment.Serve(p, describer)}, nil
}

func (p *provider) ValidateScenario(ctx context.Context, req *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	started := time.Now()
	scenario := strings.TrimSpace(req.Msg.GetScenario())
	if scenario == "" {
		scenario = providerScenario
	}
	if scenario != providerScenario {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("provider only validates %s", providerScenario))
	}
	if !req.Msg.GetIncludeExecution() {
		return p.response(scenario, nil, started)
	}
	return p.response(scenario, p.validateBoundEvidence(ctx, req.Msg.GetRetainedEvidenceSets()), started)
}

func (p *provider) ValidateTarget(ctx context.Context, req *connect.Request[scenariovalidationv1.ValidateTargetRequest]) (*connect.Response[scenariovalidationv1.ValidateTargetResponse], error) {
	target := req.Msg.GetTarget()
	if target == nil || target.GetKind() != commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SCENARIO {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("BAS provider requires a scenario target"))
	}
	result, err := p.ValidateScenario(ctx, connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: target.GetId(), Path: req.Msg.GetPath(), IncludeExecution: req.Msg.GetIncludeExecution(), CapabilitySubset: append([]string(nil), req.Msg.GetCapabilitySubset()...), RetainedEvidenceSets: req.Msg.GetRetainedEvidenceSets()}))
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&scenariovalidationv1.ValidateTargetResponse{Target: target, Status: result.Msg.GetStatus(), Assessment: result.Msg.GetAssessment(), NativeDetail: result.Msg.GetNativeDetail(), Metrics: result.Msg.GetMetrics(), FailureClassification: result.Msg.GetFailureClassification()}), nil
}

func (p *provider) validateBoundEvidence(ctx context.Context, sets []*scenariovalidationv1.RetainedEvidenceSet) []assessment.Finding {
	filtered := p.validateExecution(ctx)
	var err error
	if len(sets) != 1 {
		err = fmt.Errorf("exactly one retained evidence-completeness set is required")
	} else {
		set := sets[0]
		if set == nil || set.GetTarget() != providerScenario || set.GetProducer() != "evidence-completeness" {
			err = fmt.Errorf("retained evidence set does not identify the BAS evidence-completeness producer")
		} else if p.deps.ReadRetainedArtifact == nil {
			err = fmt.Errorf("Test Genie retained-artifact reader is unavailable")
		} else {
			root, rootErr := filepath.Abs(p.deps.ScenarioDir)
			if rootErr != nil {
				err = rootErr
			} else {
				build, buildErr := p.deps.BuildIdentity(ctx)
				if buildErr != nil {
					err = buildErr
				} else {
					err = evidencecompletenessqualification.ValidateRetained(root, build, set, func(ref *commonv1.EvidenceRef) ([]byte, error) { return p.deps.ReadRetainedArtifact(ctx, set, ref) })
				}
			}
		}
	}
	if err != nil {
		filtered = append(filtered, evidenceCompletenessFinding(err))
	}
	return filtered
}

func readRetainedArtifact(ctx context.Context, set *scenariovalidationv1.RetainedEvidenceSet, ref *commonv1.EvidenceRef) ([]byte, error) {
	baseURL, err := discovery.ResolveScenarioURLDefault(ctx, "test-genie")
	if err != nil {
		return nil, fmt.Errorf("resolve Test Genie artifact service: %w", err)
	}
	artifactURL := strings.TrimRight(baseURL, "/") + "/api/v1/scenarios/" + url.PathEscape(set.GetTarget()) + "/runs/" + url.PathEscape(set.GetRunId()) + "/artifacts/" + url.PathEscape(ref.GetArtifactId())
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, artifactURL, nil)
	if err != nil {
		return nil, err
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Test Genie artifact route returned %s", response.Status)
	}
	const maxArtifactBytes = 16 << 20
	data, err := io.ReadAll(io.LimitReader(response.Body, maxArtifactBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxArtifactBytes || int64(len(data)) != ref.GetSizeBytes() {
		return nil, fmt.Errorf("retained artifact exceeds the byte bound or declared size")
	}
	return data, nil
}

func (p *provider) response(scenario string, findings []assessment.Finding, started time.Time) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	a, err := assessment.BuildProtoAssessment(assessment.BuildInput{Scenario: scenario, Spec: *p.spec, Findings: findings})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	elapsed := time.Since(started).Milliseconds()
	if elapsed < 1 {
		elapsed = 1
	}
	r, err := assessment.BuildValidationResponse(scenario, a, nil, &commonv1.ExecutionMetrics{WallClockMs: elapsed})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(r), nil
}

func (p *provider) validate(ctx context.Context) []assessment.Finding {
	return p.validateExecution(ctx)
}

func (p *provider) validateExecution(ctx context.Context) []assessment.Finding {
	root, err := filepath.Abs(p.deps.ScenarioDir)
	if err != nil {
		return validationSetupFindings(err)
	}
	build, err := p.deps.BuildIdentity(ctx)
	if err != nil {
		return validationSetupFindings(err)
	}
	findings := make([]assessment.Finding, 0, 6)
	if err := p.validateProfile(root, build); err != nil {
		findings = append(findings, profileFinding(err))
	}
	if _, _, err := cancellationqualification.LatestCurrentReceipt(root, build); err != nil {
		findings = append(findings, cancellationFinding(err))
	}
	if err := passivefidelityqualification.Validate(root, build); err != nil {
		findings = append(findings, passiveFidelityFinding(err))
	}
	if err := resourcebudgetqualification.Validate(root, build); err != nil {
		findings = append(findings, resourceBudgetFinding(err))
	}
	if err := motionqualification.Validate(root, build); err != nil {
		findings = append(findings, motionFinding(err))
	}
	if err := interactivefeedbackqualification.Validate(root, build); err != nil {
		findings = append(findings, interactiveFeedbackFinding(err))
	}
	return findings
}

func validationSetupFindings(err error) []assessment.Finding {
	return []assessment.Finding{profileFinding(err), cancellationFinding(err), passiveFidelityFinding(err), resourceBudgetFinding(err), motionFinding(err), interactiveFeedbackFinding(err)}
}

func profileFinding(err error) assessment.Finding {
	return assessment.Finding{Code: "PROFILE_EVIDENCE_INVALID", Severity: "SEVERITY_ERROR", Title: "Profile durability evidence is stale or invalid", Message: err.Error(), Location: cohortEvidenceGlob, Remediation: "Rerun the managed profile durability cohort and retain owner receipts for the current build."}
}

func cancellationFinding(err error) assessment.Finding {
	return assessment.Finding{Code: "CANCELLATION_EVIDENCE_INVALID", Severity: "SEVERITY_ERROR", Title: "Cancellation/recovery evidence is stale or invalid", Message: err.Error(), Location: cancellationqualification.EvidenceGlob, Remediation: "Run the focused BAS cancellation qualification and retain all five independent fixture cases for the current build."}
}

func passiveFidelityFinding(err error) assessment.Finding {
	return assessment.Finding{Code: "PASSIVE_FIDELITY_EVIDENCE_INVALID", Severity: "SEVERITY_ERROR", Title: "Passive-fidelity evidence is stale or invalid", Message: err.Error(), Location: passivefidelityqualification.EvidenceDir, Remediation: "Rerun the managed 10,000-action, crash/reconnect, and browser-semantics owners for the current sources and build."}
}

func resourceBudgetFinding(err error) assessment.Finding {
	return assessment.Finding{Code: "RESOURCE_BUDGET_EVIDENCE_INVALID", Severity: "SEVERITY_ERROR", Title: "Resource-budget evidence is stale or invalid", Message: err.Error(), Location: resourcebudgetqualification.EvidenceDir, Remediation: "Run the 60-second managed idle and fixture-browser resource owner on the current BAS build, and report Windows private-memory status separately."}
}

func motionFinding(err error) assessment.Finding {
	return assessment.Finding{Code: "MOTION_EVIDENCE_INVALID", Severity: "SEVERITY_ERROR", Title: "Motion evidence is stale or invalid", Message: err.Error(), Location: motionqualification.EvidenceGlob, Remediation: "Run the five-minute managed motion owner and its slow-reader cohort against the current BAS build."}
}

func evidenceCompletenessFinding(err error) assessment.Finding {
	return assessment.Finding{Code: "EVIDENCE_COMPLETENESS_INVALID", Severity: "SEVERITY_ERROR", Title: "Evidence-completeness owner tests are stale or invalid", Message: err.Error(), Location: "retained-evidence-set", Remediation: "Admit the BAS evidence-completeness producer run and pass its exact retained artifact set to validation."}
}

func interactiveFeedbackFinding(err error) assessment.Finding {
	return assessment.Finding{Code: "INTERACTIVE_FEEDBACK_EVIDENCE_INVALID", Severity: "SEVERITY_ERROR", Title: "Interactive-feedback evidence is stale or incomplete", Message: err.Error(), Location: interactivefeedbackqualification.EvidenceGlob, Remediation: "Run both the local and 50 ms/10 Mbps remote interactive-feedback cohorts with 1,000 correlated inputs against the current BAS build."}
}

func (p *provider) validateProfile(root, build string) error {
	paths, err := filepath.Glob(filepath.Join(root, filepath.FromSlash(cohortEvidenceGlob)))
	if err != nil {
		return err
	}
	sort.Sort(sort.Reverse(sort.StringSlice(paths)))
	var receipt cohort
	selected := ""
	for _, path := range paths {
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			continue
		}
		var candidate cohort
		if json.Unmarshal(raw, &candidate) == nil && candidate.Runtime.Build == build && candidate.ContractRow == "profile-durability" {
			receipt, selected = candidate, path
			break
		}
	}
	if selected == "" {
		return fmt.Errorf("no retained profile cohort matches live build %q", build)
	}
	contractPath := filepath.Join(root, "docs/internal/REFRACTOR_CONTRACT.json")
	contract, err := os.ReadFile(contractPath)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(contract)
	if receipt.ContractRow != "profile-durability" || receipt.ContractSHA != hex.EncodeToString(sum[:]) {
		return fmt.Errorf("contract identity mismatch")
	}
	for rel, want := range receipt.SourceFiles {
		got, e := fileSHA(filepath.Join(root, filepath.FromSlash(rel)))
		if e != nil {
			return e
		}
		if got != want {
			return fmt.Errorf("source digest mismatch: %s", rel)
		}
	}
	for _, ref := range []struct{ path, sha, stage string }{{receipt.Cohort.Seed.Path, receipt.Cohort.Seed.SHA, "seed"}, {receipt.Cohort.RestartReceipt.Path, receipt.Cohort.RestartReceipt.SHA, "verify"}} {
		ownerPath := filepath.Join(root, filepath.FromSlash(ref.path))
		got, e := fileSHA(ownerPath)
		if e != nil {
			return e
		}
		if got != ref.sha {
			return fmt.Errorf("owner receipt digest mismatch: %s", ref.path)
		}
		var owner ownerReceipt
		raw, e := os.ReadFile(ownerPath)
		if e != nil {
			return e
		}
		if e = json.Unmarshal(raw, &owner); e != nil {
			return e
		}
		if len(owner.Results) == 0 {
			return fmt.Errorf("owner receipt has no checks: %s", ref.path)
		}
		if owner.Stage != ref.stage || owner.ContractSHA != receipt.ContractSHA || owner.HarnessSHA != receipt.SourceFiles["api/cmd/profile-durability-cohort/qualification.mjs"] || owner.BeforeHealth.Build != receipt.Runtime.Build {
			return fmt.Errorf("owner receipt identity mismatch: %s", ref.path)
		}
		for _, result := range owner.Results {
			if !result.Passed {
				return fmt.Errorf("owner receipt contains a failed check: %s", ref.path)
			}
		}
	}
	c := receipt.Cohort
	if !c.AllChecksPassed || c.SeedChecks < 5 || c.RestartChecks < 2 || c.CheckpointMS <= 0 || c.CheckpointMS > 5000 || c.Isolation != "passed" || c.Restart != "passed" || c.Cleanup != 2 {
		return fmt.Errorf("cohort assertions are incomplete")
	}
	if build == "" || build != receipt.Runtime.Build {
		return fmt.Errorf("live build identity does not match cohort receipt")
	}
	return nil
}

func liveBuildIdentity(ctx context.Context) (string, error) {
	port := strings.TrimSpace(os.Getenv("API_PORT"))
	if port == "" {
		return "", fmt.Errorf("API_PORT is not set")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1:"+port+"/health", nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("health returned %s", resp.Status)
	}
	var body struct {
		Build string `json:"build_identity"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}
	return body.Build, nil
}

func fileSHA(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:]), nil
}
