package profilevalidation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/browser-automation-studio/internal/cancellationqualification"
	"github.com/vrooli/browser-automation-studio/internal/evidencecompletenessqualification"
	"github.com/vrooli/browser-automation-studio/internal/motionqualification"
	"github.com/vrooli/browser-automation-studio/internal/passivefidelityqualification"
	"github.com/vrooli/browser-automation-studio/internal/resourcebudgetqualification"
	"github.com/vrooli/maturity-go/assessment"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
)

const (
	providerScenario   = "browser-automation-studio"
	providerPhase      = "rehabilitation-evidence"
	cohortEvidenceGlob = ".vrooli/runtime/rehabilitation-evidence/profile-durability-*.json"
)

type deps struct {
	ScenarioDir   string
	BuildIdentity func(context.Context) (string, error)
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

func Module(scenarioDir string) (scenariovalidationv1connect.ScenarioValidationServiceHandler, error) {
	spec, err := assessment.LoadSpecFromScenario(scenarioDir)
	if err != nil {
		return nil, err
	}
	describer, err := assessment.LoadDescriber(scenarioDir)
	if err != nil {
		return nil, err
	}
	p := &provider{deps: deps{ScenarioDir: scenarioDir, BuildIdentity: liveBuildIdentity}, spec: spec}
	return assessment.Serve(p, describer), nil
}

func (p *provider) ValidateScenario(ctx context.Context, req *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	scenario := strings.TrimSpace(req.Msg.GetScenario())
	if scenario == "" {
		scenario = providerScenario
	}
	if scenario != providerScenario {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("provider only validates %s", providerScenario))
	}
	if !req.Msg.GetIncludeExecution() {
		return p.response(scenario, nil)
	}
	return p.response(scenario, p.validate(ctx))
}

func (p *provider) response(scenario string, findings []assessment.Finding) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	a, err := assessment.BuildProtoAssessment(assessment.BuildInput{Scenario: scenario, Spec: *p.spec, Findings: findings})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	r, err := assessment.BuildValidationResponse(scenario, a, nil, nil)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(r), nil
}

func (p *provider) validate(ctx context.Context) []assessment.Finding {
	root, err := filepath.Abs(p.deps.ScenarioDir)
	if err != nil {
		return []assessment.Finding{profileFinding(err), cancellationFinding(err), passiveFidelityFinding(err), resourceBudgetFinding(err), motionFinding(err), evidenceCompletenessFinding(err)}
	}
	build, err := p.deps.BuildIdentity(ctx)
	if err != nil {
		return []assessment.Finding{profileFinding(err), cancellationFinding(err), passiveFidelityFinding(err), resourceBudgetFinding(err), motionFinding(err), evidenceCompletenessFinding(err)}
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
	if err := evidencecompletenessqualification.Validate(root, build); err != nil {
		findings = append(findings, evidenceCompletenessFinding(err))
	}
	return findings
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
	return assessment.Finding{Code: "EVIDENCE_COMPLETENESS_INVALID", Severity: "SEVERITY_ERROR", Title: "Evidence-completeness owner tests are stale or invalid", Message: err.Error(), Location: evidencecompletenessqualification.EvidenceGlob, Remediation: "Run the focused screenshot, inline telemetry, external video/trace, and active-retention owners for the current BAS source and build."}
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
