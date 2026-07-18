package componenttests

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"react-component-library/internal/components"
)

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

// ValidationError identifies an invalid runner request with a stable code for
// the CLI and workbench instead of returning an opaque transport error.
type ValidationError struct {
	Code, Field, Detail string
}

func (e ValidationError) Error() string {
	if e.Field == "" {
		return e.Code + ": " + e.Detail
	}
	return fmt.Sprintf("%s at %s: %s", e.Code, e.Field, e.Detail)
}

// StoryReader supplies validated story expectations. It never exposes setup
// or executable values: the story contract is the sole preview/test boundary.
type StoryReader interface {
	ListStories(context.Context, components.StoryQuery) ([]components.ComponentStory, error)
}

type Runner struct {
	Assets  components.DependencyReader
	Stories StoryReader
	Now     func() time.Time
}

// Run performs the deterministic, safe pre-harness stages. Browser and React
// harness adapters consume the normalized report later; this stage proves that
// no runtime can start until the versioned closure and restricted contract are
// valid. It is deliberately failure-isolating: one bad sibling contract does
// not hide the others.
func (r Runner) Run(ctx context.Context, request Request) (Report, error) {
	if r.Assets == nil || r.Stories == nil {
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
		results, artifacts, storyErr := r.directStoryResults(ctx, asset, version)
		if storyErr != nil {
			return Report{}, storyErr
		}
		report.Results = append(report.Results, results...)
		report.Artifacts = append(report.Artifacts, artifacts...)
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

// directStoryResults accepts the same validated story.json contract that
// drives the preview workbench; no second behavior declaration is required.
func (r Runner) directStoryResults(ctx context.Context, asset components.Component, version components.ComponentVersion) ([]Result, []Artifact, error) {
	projected, err := r.Stories.ListStories(ctx, components.StoryQuery{ComponentID: asset.ID, Version: version.Version, Limit: 20})
	if err != nil {
		return nil, nil, fmt.Errorf("list stories for %s@%s: %w", asset.LibraryID, version.Version, err)
	}
	if len(projected) != 1 {
		return []Result{{Stage: StageContract, AssetLibraryID: asset.LibraryID, Version: version.Version, Verdict: VerdictFailed, Message: "asset version must have exactly one indexed story contract", Remediation: "add one valid story.json and reindex"}}, nil, nil
	}
	var story components.StoryContract
	if err := json.Unmarshal([]byte(projected[0].ContractJSON), &story); err != nil {
		return nil, nil, fmt.Errorf("decode story contract for %s@%s: %w", asset.LibraryID, version.Version, err)
	}
	results := []Result{{Stage: StageContract, AssetLibraryID: asset.LibraryID, Version: version.Version, Verdict: VerdictPassed, Message: "validated story contract accepted"}}
	for _, definition := range story.Stories {
		message := "declared component story accepted for preview and runner"
		if asset.AssetKind == components.AssetKindHook {
			message = "declared hook fixture accepted for runner"
		}
		results = append(results, Result{Stage: StageDeclared, AssetLibraryID: asset.LibraryID, Version: version.Version, Subject: definition.ID, Verdict: VerdictPassed, Message: message})
	}
	return results, []Artifact{{Kind: "story-contract", Label: "story.json", AssetLibraryID: asset.LibraryID, Version: version.Version, Reference: asset.LibraryID + "@" + version.Version + ":story.json"}}, nil
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

func reportID(report Report) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{report.RootLibraryID, report.RootVersion, fmt.Sprintf("%t", report.IncludeClosure), report.CreatedAt.Format(time.RFC3339Nano)}, "\x00")))
	return "ctr_" + hex.EncodeToString(sum[:8])
}
