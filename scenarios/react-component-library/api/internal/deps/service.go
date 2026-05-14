package deps

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// PackageJSONReader is the target-scenario-tree seam ValidateAdoption
// uses to read a package.json. Production walks the configured
// scenarios root with a traversal guard; tests inject a fake.
type PackageJSONReader interface {
	Read(ctx context.Context, scenario string) ([]byte, error)
}

// Service is the application-layer surface handlers depend on.
type Service interface {
	// SyncForComponent re-records declarations for one component
	// (driven by the indexer's UpsertListener). Service strips
	// whitespace and rejects malformed dep names; the rest is delegated
	// to the repository.
	SyncForComponent(ctx context.Context, in SyncInput) error

	// ListForComponent returns the declarations the indexer recorded
	// for a component.
	ListForComponent(ctx context.Context, componentID string) ([]Declaration, error)

	// ValidateAdoption is the core req 10 verb: read the target
	// scenario's package.json, intersect each declared range against
	// the resolved version, fold issues into a Verdict.
	ValidateAdoption(ctx context.Context, componentID, scenario string) (Verdict, error)
}

type service struct {
	repo Repository
	pkgs PackageJSONReader
}

// NewService constructs the production Service. pkgs may be nil for
// callers that only use SyncForComponent / ListForComponent (e.g. the
// indexer); ValidateAdoption returns an error in that case so the
// missing-seam is loud rather than silent.
func NewService(repo Repository, pkgs PackageJSONReader) Service {
	return &service{repo: repo, pkgs: pkgs}
}

var _ Service = (*service)(nil)

func (s *service) SyncForComponent(ctx context.Context, in SyncInput) error {
	if strings.TrimSpace(in.ComponentID) == "" {
		return fmt.Errorf("sync deps: component_id required")
	}
	cleaned := make([]DeclarationFields, 0, len(in.Declarations))
	for _, d := range in.Declarations {
		name := strings.TrimSpace(d.DepName)
		if name == "" {
			continue
		}
		cleaned = append(cleaned, DeclarationFields{
			DepName:      name,
			VersionRange: strings.TrimSpace(d.VersionRange),
		})
	}
	in.Declarations = cleaned
	return s.repo.SyncForComponent(ctx, in)
}

func (s *service) ListForComponent(ctx context.Context, componentID string) ([]Declaration, error) {
	return s.repo.ListForComponent(ctx, componentID)
}

func (s *service) ValidateAdoption(ctx context.Context, componentID, scenario string) (Verdict, error) {
	cid := strings.TrimSpace(componentID)
	if cid == "" {
		return Verdict{}, fmt.Errorf("component_id required")
	}
	scn := strings.TrimSpace(scenario)
	if scn == "" {
		return Verdict{}, fmt.Errorf("scenario required")
	}
	if s.pkgs == nil {
		return Verdict{}, fmt.Errorf("package.json reader not configured")
	}

	declarations, err := s.repo.ListForComponent(ctx, cid)
	if err != nil {
		return Verdict{}, fmt.Errorf("list declarations: %w", err)
	}
	if len(declarations) == 0 {
		// No declarations recorded means: either the component has no
		// deps (perfectly valid — verdict OK) or it was never indexed.
		// We can't tell from the repo alone; the indexer is responsible
		// for writing an empty set on every header parse so absence
		// truly means "not indexed". For a deterministic verdict here
		// we return OK with no issues — UI/CLI can call ListForComponent
		// to disambiguate.
		return Verdict{Kind: VerdictOK}, nil
	}

	raw, err := s.pkgs.Read(ctx, scn)
	if err != nil {
		return Verdict{}, ErrScenarioPackageJSONMissing{Scenario: scn, Cause: err}
	}
	targetDeps, err := parsePackageJSONDeps(raw)
	if err != nil {
		return Verdict{}, fmt.Errorf("parse package.json: %w", err)
	}

	verdict := Verdict{Kind: VerdictOK}
	for _, d := range declarations {
		targetRange, present := targetDeps[d.DepName]
		if !present {
			verdict.Issues = append(verdict.Issues, Issue{
				DepName:       d.DepName,
				DeclaredRange: d.VersionRange,
				Kind:          IssueMissingDep,
				Detail:        fmt.Sprintf("scenario %q has no dependency %q", scn, d.DepName),
			})
			continue
		}
		kind, detail := classify(d.VersionRange, targetRange)
		if kind == "" {
			continue
		}
		verdict.Issues = append(verdict.Issues, Issue{
			DepName:         d.DepName,
			DeclaredRange:   d.VersionRange,
			ScenarioVersion: targetRange,
			Kind:            kind,
			Detail:          detail,
		})
	}

	sort.Slice(verdict.Issues, func(i, j int) bool { return verdict.Issues[i].DepName < verdict.Issues[j].DepName })

	// Fold severity: any block → block; any warn → warn; else ok.
	for _, iss := range verdict.Issues {
		switch iss.Severity() {
		case VerdictBlock:
			verdict.Kind = VerdictBlock
		case VerdictWarn:
			if verdict.Kind != VerdictBlock {
				verdict.Kind = VerdictWarn
			}
		}
	}
	return verdict, nil
}

// parsePackageJSONDeps merges the `dependencies` + `devDependencies` +
// `peerDependencies` maps from a package.json. Later entries do NOT
// overwrite earlier ones — first match wins (dependencies > peer > dev).
// This matches what bundlers actually resolve at runtime.
func parsePackageJSONDeps(raw []byte) (map[string]string, error) {
	var pkg struct {
		Dependencies     map[string]string `json:"dependencies"`
		DevDependencies  map[string]string `json:"devDependencies"`
		PeerDependencies map[string]string `json:"peerDependencies"`
	}
	if err := json.Unmarshal(raw, &pkg); err != nil {
		return nil, err
	}
	out := map[string]string{}
	merge := func(src map[string]string) {
		for k, v := range src {
			if _, ok := out[k]; ok {
				continue
			}
			out[k] = v
		}
	}
	merge(pkg.Dependencies)
	merge(pkg.PeerDependencies)
	merge(pkg.DevDependencies)
	return out, nil
}
