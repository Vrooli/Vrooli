package componenttests

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"react-component-library/internal/components"
)

// ContractReader isolates authored contract storage from execution. The
// production reader is filesystem-backed; tests can use a map without writing
// source files.
type ContractReader interface {
	Load(components.Component, components.ComponentVersion) (Contract, error)
}

type FSContractReader struct{ Root string }

func (r FSContractReader) Load(asset components.Component, version components.ComponentVersion) (Contract, error) {
	kind := "components"
	if asset.AssetKind == components.AssetKindHook {
		kind = "hooks"
	}
	path := filepath.Join(r.Root, kind, asset.Slug, "versions", version.Version, ContractFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return Contract{}, fmt.Errorf("read %s: %w", ContractFileName, err)
	}
	return ParseContract(data, string(asset.AssetKind))
}

type Stage string

const (
	StageClosure  Stage = "closure_integrity"
	StageStatic   Stage = "source_integrity"
	StageContract Stage = "contract_validation"
	StageDeclared Stage = "declared_behavior"
	StageEvidence Stage = "experience_evidence"
)

type Verdict string

const (
	VerdictPassed  Verdict = "passed"
	VerdictFailed  Verdict = "failed"
	VerdictBlocked Verdict = "blocked"
)

type Result struct {
	Stage          Stage   `json:"stage"`
	AssetLibraryID string  `json:"assetLibraryId"`
	Version        string  `json:"version"`
	Subject        string  `json:"subject,omitempty"`
	Verdict        Verdict `json:"verdict"`
	Message        string  `json:"message,omitempty"`
	Remediation    string  `json:"remediation,omitempty"`
}
type Artifact struct {
	Kind, Label, AssetLibraryID, Version, Reference string
}
type Report struct {
	ID              string     `json:"id"`
	RootComponentID string     `json:"rootComponentId"`
	RootLibraryID   string     `json:"rootLibraryId"`
	RootVersion     string     `json:"rootVersion"`
	IncludeClosure  bool       `json:"includeClosure"`
	CreatedAt       time.Time  `json:"createdAt"`
	Verdict         Verdict    `json:"verdict"`
	Results         []Result   `json:"results"`
	Artifacts       []Artifact `json:"artifacts"`
}
type Request struct {
	ComponentID, Version string
	IncludeClosure       bool
}

// ExampleReader supplies indexed example expectations. It never returns or
// executes setupJson: preview setup is intentionally outside the test DSL's
// security boundary.
type ExampleReader interface {
	ListExamples(context.Context, components.ExampleQuery) ([]components.ComponentExample, error)
}

// ClaimReader verifies that a contract claim names a claim in the asset's
// published experience document. It only reads declarative JSON; reconciliation
// captures remain owned by Experience Manager.
type ClaimReader interface {
	HasClaim(components.Component, string) (bool, error)
}
type FSClaimReader struct{ Root string }

func (r FSClaimReader) HasClaim(asset components.Component, claim string) (bool, error) {
	name := kebabName(asset.Slug)
	data, err := os.ReadFile(filepath.Join(r.Root, "experience", "components", name+".json"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	var document struct {
		Claims []struct {
			ID string `json:"id"`
		} `json:"claims"`
	}
	if err := json.Unmarshal(data, &document); err != nil {
		return false, fmt.Errorf("decode experience contract: %w", err)
	}
	for _, candidate := range document.Claims {
		if candidate.ID == claim {
			return true, nil
		}
	}
	return false, nil
}
func kebabName(value string) string {
	var out []rune
	for i, character := range strings.TrimSpace(value) {
		if character >= 'A' && character <= 'Z' && i > 0 {
			out = append(out, '-')
		}
		if (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') || (character >= '0' && character <= '9') {
			out = append(out, rune(strings.ToLower(string(character))[0]))
		}
	}
	return strings.Trim(string(out), "-")
}

type Runner struct {
	Assets    components.DependencyReader
	Examples  ExampleReader
	Contracts ContractReader
	Claims    ClaimReader
	Now       func() time.Time
}

// Run performs the deterministic, safe pre-harness stages. Browser and React
// harness adapters consume the normalized report later; this stage proves that
// no runtime can start until the versioned closure and restricted contract are
// valid. It is deliberately failure-isolating: one bad sibling contract does
// not hide the others.
func (r Runner) Run(ctx context.Context, request Request) (Report, error) {
	if r.Assets == nil || r.Contracts == nil {
		return Report{}, fmt.Errorf("component test runner is not configured")
	}
	if strings.TrimSpace(request.ComponentID) == "" || strings.TrimSpace(request.Version) == "" {
		return Report{}, ValidationError{Code: "version_required", Detail: "component id and explicit version are required"}
	}
	closure, err := components.ResolveDependencyClosure(ctx, r.Assets, request.ComponentID, request.Version)
	if err != nil {
		return Report{}, err
	}
	if !request.IncludeClosure {
		closure = closure[len(closure)-1:]
	}
	now := time.Now().UTC()
	if r.Now != nil {
		now = r.Now().UTC()
	}
	report := Report{RootComponentID: request.ComponentID, RootLibraryID: closure[len(closure)-1].Asset.LibraryID, RootVersion: request.Version, IncludeClosure: request.IncludeClosure, CreatedAt: now}
	report.ID = reportID(report)
	for _, resolved := range closure {
		asset, version := resolved.Asset, resolved.Version
		report.Results = append(report.Results, Result{Stage: StageClosure, AssetLibraryID: asset.LibraryID, Version: version.Version, Verdict: VerdictPassed, Message: "version-pinned asset resolved"})
		report.Results = append(report.Results, staticSourceResult(asset, version))
		contract, loadErr := r.Contracts.Load(asset, version)
		if loadErr != nil {
			if errors.Is(loadErr, os.ErrNotExist) {
				report.Results = append(report.Results, Result{Stage: StageContract, AssetLibraryID: asset.LibraryID, Version: version.Version, Verdict: VerdictBlocked, Message: "no versioned test contract is declared", Remediation: "add test-contract.json to opt this asset into component testing; examples.setup is never executed"})
			} else {
				report.Results = append(report.Results, Result{Stage: StageContract, AssetLibraryID: asset.LibraryID, Version: version.Version, Verdict: VerdictFailed, Message: loadErr.Error(), Remediation: "repair the versioned test-contract.json; examples.setup is never executed"})
			}
			continue
		}
		report.Results = append(report.Results, Result{Stage: StageContract, AssetLibraryID: asset.LibraryID, Version: version.Version, Verdict: VerdictPassed, Message: "declarative contract accepted"})
		report.Artifacts = append(report.Artifacts, Artifact{Kind: "test-contract", Label: ContractFileName, AssetLibraryID: asset.LibraryID, Version: version.Version, Reference: asset.LibraryID + "@" + version.Version + ":" + ContractFileName})
		if asset.AssetKind == components.AssetKindHook {
			for _, fixture := range contract.Fixtures {
				declared := Result{Stage: StageDeclared, AssetLibraryID: asset.LibraryID, Version: version.Version, Subject: fixture.Name, Verdict: VerdictPassed, Message: "approved hook fixture accepted for React harness"}
				report.Results = append(report.Results, declared)
				report.Results = append(report.Results, claimResults(asset, version, fixture.Name, fixture.Claims, declared.Verdict, r.Claims)...)
			}
			continue
		}
		if r.Examples == nil {
			return Report{}, fmt.Errorf("component test example reader is not configured")
		}
		examples, err := r.Examples.ListExamples(ctx, components.ExampleQuery{ComponentID: asset.ID, Version: version.Version, Limit: 500})
		if err != nil {
			return Report{}, fmt.Errorf("list examples for %s@%s: %w", asset.LibraryID, version.Version, err)
		}
		byName := make(map[string]components.ComponentExample, len(examples))
		for _, example := range examples {
			byName[example.Name] = example
		}
		for _, trace := range contract.Examples {
			declared := declaredTraceResult(asset, version, trace, byName[trace.Example])
			report.Results = append(report.Results, declared)
			report.Results = append(report.Results, claimResults(asset, version, trace.Example, trace.Claims, declared.Verdict, r.Claims)...)
		}
	}
	report.Verdict = VerdictPassed
	for _, result := range report.Results {
		if result.Verdict == VerdictFailed {
			report.Verdict = VerdictFailed
			break
		}
		if result.Verdict == VerdictBlocked {
			report.Verdict = VerdictBlocked
		}
	}
	return report, nil
}

func staticSourceResult(asset components.Component, version components.ComponentVersion) Result {
	result := Result{Stage: StageStatic, AssetLibraryID: asset.LibraryID, Version: version.Version, Verdict: VerdictPassed, Message: "versioned source integrity accepted"}
	if strings.TrimSpace(version.Content) == "" {
		result.Verdict = VerdictFailed
		result.Message = "versioned entry source is empty"
		result.Remediation = "restore the version entry source before running behavior checks"
		return result
	}
	if strings.TrimSpace(version.ContentSHA256) == "" {
		result.Verdict = VerdictBlocked
		result.Message = "versioned source has no indexed content digest"
		result.Remediation = "reindex the catalog so the immutable source digest is recorded"
	}
	return result
}

func claimResults(asset components.Component, version components.ComponentVersion, subject string, claims []string, traceVerdict Verdict, reader ClaimReader) []Result {
	results := make([]Result, 0, len(claims))
	for _, claim := range claims {
		result := Result{Stage: StageEvidence, AssetLibraryID: asset.LibraryID, Version: version.Version, Subject: claim, Verdict: VerdictPassed, Message: "declared experience claim linked to this run"}
		if traceVerdict != VerdictPassed {
			result.Verdict, result.Message, result.Remediation = VerdictBlocked, "claim evidence is blocked by its failed behavior trace", "repair the trace before treating this claim as run-backed evidence"
		} else if reader == nil {
			result.Verdict, result.Message, result.Remediation = VerdictBlocked, "experience contract was not available to verify this claim", "add the claim to the asset experience contract"
		} else if found, err := reader.HasClaim(asset, claim); err != nil {
			result.Verdict, result.Message, result.Remediation = VerdictFailed, err.Error(), "repair the asset experience contract"
		} else if !found {
			result.Verdict, result.Message, result.Remediation = VerdictBlocked, "claim is not declared by the asset experience contract", "add the named claim or remove it from test-contract.json"
		}
		results = append(results, result)
	}
	return results
}

func declaredTraceResult(asset components.Component, version components.ComponentVersion, trace ExampleTrace, example components.ComponentExample) Result {
	result := Result{Stage: StageDeclared, AssetLibraryID: asset.LibraryID, Version: version.Version, Subject: trace.Example, Verdict: VerdictPassed, Message: "declared example expectation matches safe trace"}
	if example.Name == "" {
		result.Verdict = VerdictFailed
		result.Message = "contract references an example that does not exist"
		result.Remediation = "add the named example or correct test-contract.json"
		return result
	}
	var expectations []map[string]any
	if err := json.Unmarshal([]byte(example.ExpectJSON), &expectations); err != nil {
		result.Verdict = VerdictFailed
		result.Message = "example expectation is invalid JSON"
		result.Remediation = "repair examples.json before running the trace"
		return result
	}
	for _, assertion := range trace.Assertions {
		if !declaresAssertion(expectations, assertion) {
			result.Verdict = VerdictFailed
			result.Message = fmt.Sprintf("example does not declare %s assertion", assertion.Kind)
			result.Remediation = "align examples.json and test-contract.json"
			return result
		}
	}
	return result
}
func declaresAssertion(expectations []map[string]any, assertion Assertion) bool {
	for _, candidate := range expectations {
		if fmt.Sprint(candidate["kind"]) != assertion.Kind {
			continue
		}
		switch assertion.Kind {
		case "role":
			if fmt.Sprint(candidate["role"]) == assertion.Role && fmt.Sprint(candidate["name"]) == assertion.Name {
				return true
			}
		case "text":
			if fmt.Sprint(candidate["value"]) == assertion.Value {
				return true
			}
		case "attribute":
			if fmt.Sprint(candidate["selector"]) == assertion.Target && fmt.Sprint(candidate["name"]) == assertion.Attribute {
				return true
			}
		default:
			return true
		}
	}
	return false
}

func reportID(report Report) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{report.RootLibraryID, report.RootVersion, fmt.Sprintf("%t", report.IncludeClosure), report.CreatedAt.Format(time.RFC3339Nano)}, "\x00")))
	return "ctr_" + hex.EncodeToString(sum[:8])
}
