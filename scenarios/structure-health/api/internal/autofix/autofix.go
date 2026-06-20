// Package autofix registers structure-health's deterministic remediations
// against the shared maturity-go/autofix orchestrator. Every concrete edit is a
// Fixer here; the orchestrator owns preview/apply/idempotency. service.json edits
// are structured and format-preserving (see internal/svcedit) — they never blind
// round-trip the document, so key order and unknown fields survive.
package autofix

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"structure-health/internal/intent"
	"structure-health/internal/svcedit"

	autofixcore "github.com/vrooli/maturity-go/autofix"
)

// Candidate is the shared auto-fix candidate (re-exported for callers).
type Candidate = autofixcore.Candidate

// Finding/rule codes structure-health can auto-remediate. These mirror the codes
// emitted by internal/rules and listed in .vrooli/maturity.json.
const (
	RuleServiceNameMismatch = "SERVICE_NAME_MISMATCH"
	RuleHealthCheckMissing  = "HEALTH_CHECK_MISSING"
	RuleFreshnessMissing    = "FRESHNESS_CHECK_MISSING"
	RuleSurfaceDirMissing   = "SURFACE_DIR_MISSING"
	RuleRequiredFileMissing = "REQUIRED_FILE_MISSING"
)

var registry = autofixcore.NewRegistry(
	autofixcore.Fixer{RuleID: RuleServiceNameMismatch, Preview: previewServiceName, CanFix: canFix(previewServiceName)},
	autofixcore.Fixer{RuleID: RuleHealthCheckMissing, Preview: previewHealthCheck, CanFix: canFix(previewHealthCheck)},
	autofixcore.Fixer{RuleID: RuleFreshnessMissing, Preview: previewFreshnessChecks, CanFix: canFix(previewFreshnessChecks)},
	autofixcore.Fixer{RuleID: RuleSurfaceDirMissing, Preview: previewSurfaceDirs, CanFix: canFix(previewSurfaceDirs)},
	autofixcore.Fixer{RuleID: RuleRequiredFileMissing, Preview: previewRequiredFiles, CanFix: canFix(previewRequiredFiles)},
)

// Preview returns the proposed edits for the requested rules (all when empty).
func Preview(root string, ruleIDs []string) ([]Candidate, error) {
	return registry.Preview(root, ruleIDs)
}

// Apply writes the requested edits one at a time, re-previewing from fresh disk
// state after every write. Several structure-health rules (SERVICE_NAME_MISMATCH,
// HEALTH_CHECK_MISSING, FRESHNESS_CHECK_MISSING) edit the *same* service.json, so
// re-previewing is what makes them compose: each later candidate is recomputed
// against the already-applied earlier one instead of clobbering it with a stale
// full-file snapshot. Idempotency holds because Preview re-checks state and emits
// nothing once the tree is fixed; a (ruleID,filePath) pair is applied at most
// once per call as a termination backstop.
func Apply(root string, ruleIDs []string) ([]Candidate, error) {
	var applied []Candidate
	seen := map[string]bool{}
	for {
		candidates, err := registry.Preview(root, ruleIDs)
		if err != nil {
			return applied, err
		}
		next := -1
		for i := range candidates {
			if !seen[candidateKey(candidates[i])] {
				next = i
				break
			}
		}
		if next < 0 {
			return applied, nil
		}
		c := candidates[next]
		seen[candidateKey(c)] = true
		if err := os.MkdirAll(filepath.Dir(c.FilePath), 0o755); err != nil {
			return applied, err
		}
		if err := os.WriteFile(c.FilePath, []byte(c.After), 0o644); err != nil {
			return applied, err
		}
		c.Applied = true
		applied = append(applied, c)
	}
}

func candidateKey(c Candidate) string { return c.RuleID + "\x00" + c.FilePath }

// CanFix reports whether the named rule can currently remediate the finding.
func CanFix(root, ruleID, findingPath string) bool {
	return registry.CanFix(root, ruleID, findingPath)
}

// FixClassFor returns the fix classification for a finding code: autofix when a
// fixer is registered for it, detection_only otherwise.
func FixClassFor(code string) autofixcore.FixClass {
	switch code {
	case RuleServiceNameMismatch, RuleHealthCheckMissing, RuleFreshnessMissing, RuleSurfaceDirMissing, RuleRequiredFileMissing:
		return autofixcore.FixClassAutofix
	default:
		return autofixcore.FixClassDetectionOnly
	}
}

// canFix adapts a preview func into a CanFix predicate (the finding is fixable
// when the preview would produce at least one candidate).
func canFix(preview func(root string) ([]Candidate, error)) func(root, findingPath string) bool {
	return func(root, _ string) bool {
		c, err := preview(root)
		return err == nil && len(c) > 0
	}
}

// --- service.json fixers --------------------------------------------------

const serviceJSONRel = ".vrooli/service.json"

// loadServiceJSON reads + parses the target service.json into an editable doc,
// returning the original bytes for the candidate's Before snapshot.
func loadServiceJSON(root string) (*svcedit.Doc, []byte, error) {
	path := filepath.Join(root, filepath.FromSlash(serviceJSONRel))
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	doc, err := svcedit.Parse(raw)
	if err != nil {
		return nil, nil, err
	}
	return doc, raw, nil
}

// previewServiceName rewrites service.name to the scenario directory name.
func previewServiceName(root string) ([]Candidate, error) {
	in, err := intent.Load(root)
	if err != nil {
		return nil, nil
	}
	dir := scenarioDir(root)
	current := strings.TrimSpace(in.Name)
	if dir == "" || current == "" || current == dir {
		return nil, nil
	}
	doc, before, err := loadServiceJSON(root)
	if err != nil {
		return nil, nil
	}
	svc := svcedit.EnsureMap(doc.Root(), "service")
	svc.Set("name", dir)
	after, err := doc.Bytes()
	if err != nil {
		return nil, err
	}
	return []Candidate{{
		RuleID:      RuleServiceNameMismatch,
		FilePath:    filepath.Join(root, filepath.FromSlash(serviceJSONRel)),
		Description: fmt.Sprintf("Set service.name to %q (must equal the scenario directory).", dir),
		Before:      string(before),
		After:       string(after),
	}}, nil
}

// previewHealthCheck adds an http health check when an API surface is declared
// but no http check exists.
func previewHealthCheck(root string) ([]Candidate, error) {
	in, err := intent.Load(root)
	if err != nil {
		return nil, nil
	}
	api, _, _ := declaredSurfaces(in)
	if !api {
		return nil, nil
	}
	for _, c := range in.Lifecycle.Health.Checks {
		if strings.EqualFold(c.Type, "http") {
			return nil, nil
		}
	}
	doc, before, err := loadServiceJSON(root)
	if err != nil {
		return nil, nil
	}
	health := svcedit.EnsureMap(svcedit.EnsureMap(doc.Root(), "lifecycle"), "health")
	svcedit.AppendToSlice(health, "checks", svcedit.NewObject(
		"name", "api_endpoint",
		"type", "http",
		"target", "http://localhost:${API_PORT}/health",
		"critical", true,
		"timeout", 10000,
		"interval", 30000,
	))
	after, err := doc.Bytes()
	if err != nil {
		return nil, err
	}
	return []Candidate{{
		RuleID:      RuleHealthCheckMissing,
		FilePath:    filepath.Join(root, filepath.FromSlash(serviceJSONRel)),
		Description: "Add an http lifecycle.health check targeting the API /health endpoint.",
		Before:      string(before),
		After:       string(after),
	}}, nil
}

// previewFreshnessChecks adds the missing per-surface freshness checks in a
// single edit (binaries for api, cli for cli, ui-bundle for ui).
func previewFreshnessChecks(root string) ([]Candidate, error) {
	in, err := intent.Load(root)
	if err != nil {
		return nil, nil
	}
	api, ui, cli := declaredSurfaces(in)
	scenario := scenarioName(root, in)

	type want struct {
		declared bool
		typ      string
		build    func() interface{}
	}
	wants := []want{
		{api, "binaries", func() interface{} {
			return svcedit.NewObject("type", "binaries", "targets", []interface{}{"api/" + scenario + "-api"})
		}},
		{cli, "cli", func() interface{} {
			return svcedit.NewObject("type", "cli", "command", scenario)
		}},
		{ui, "ui-bundle", func() interface{} {
			return svcedit.NewObject("type", "ui-bundle", "bundle_path", "ui/dist/index.html", "source_dir", "ui/src")
		}},
	}

	var missing []want
	for _, w := range wants {
		if w.declared && len(in.FreshCheckByType(w.typ)) == 0 {
			missing = append(missing, w)
		}
	}
	if len(missing) == 0 {
		return nil, nil
	}
	doc, before, err := loadServiceJSON(root)
	if err != nil {
		return nil, nil
	}
	condition := svcedit.EnsureMap(svcedit.EnsureMap(svcedit.EnsureMap(doc.Root(), "lifecycle"), "setup"), "condition")
	types := make([]string, 0, len(missing))
	for _, w := range missing {
		svcedit.AppendToSlice(condition, "checks", w.build())
		types = append(types, w.typ)
	}
	after, err := doc.Bytes()
	if err != nil {
		return nil, err
	}
	return []Candidate{{
		RuleID:      RuleFreshnessMissing,
		FilePath:    filepath.Join(root, filepath.FromSlash(serviceJSONRel)),
		Description: "Add missing lifecycle.setup freshness checks: " + strings.Join(types, ", ") + ".",
		Before:      string(before),
		After:       string(after),
	}}, nil
}

// --- filesystem fixers ----------------------------------------------------

// previewSurfaceDirs creates a .gitkeep for each declared surface whose
// directory is absent (one candidate per missing surface directory).
func previewSurfaceDirs(root string) ([]Candidate, error) {
	in, err := intent.Load(root)
	if err != nil {
		return nil, nil
	}
	api, ui, cli := declaredSurfaces(in)
	var out []Candidate
	for _, s := range []struct {
		want bool
		dir  string
	}{{api, "api"}, {ui, "ui"}, {cli, "cli"}} {
		if !s.want || dirExists(filepath.Join(root, s.dir)) {
			continue
		}
		keep := filepath.Join(root, s.dir, ".gitkeep")
		out = append(out, Candidate{
			RuleID:      RuleSurfaceDirMissing,
			FilePath:    keep,
			Description: fmt.Sprintf("Create the %s/ surface directory.", s.dir),
			Before:      "",
			After:       "",
		})
	}
	return out, nil
}

// previewRequiredFiles creates a README.md stub when it is missing. The Makefile
// is intentionally NOT auto-generated: it must route real targets through vrooli
// and cannot be safely synthesized.
func previewRequiredFiles(root string) ([]Candidate, error) {
	readme := filepath.Join(root, "README.md")
	if fileExists(readme) {
		return nil, nil
	}
	scenario := scenarioDir(root)
	content := fmt.Sprintf("# %s\n\nProfile-aware structure/lifecycle validation scenario.\n\n"+
		"See `.vrooli/service.json` for the lifecycle contract and `docs/` for details.\n", scenario)
	return []Candidate{{
		RuleID:      RuleRequiredFileMissing,
		FilePath:    readme,
		Description: "Create a README.md stub at the scenario root.",
		Before:      "",
		After:       content,
	}}, nil
}

// --- helpers --------------------------------------------------------------

// declaredSurfaces mirrors reconcile's declared-surface derivation from intent
// alone (ports, health endpoints, cli.enabled) without needing code facts.
func declaredSurfaces(in intent.Intent) (api, ui, cli bool) {
	mark := func(name string) {
		switch strings.ToLower(name) {
		case "api":
			api = true
		case "ui":
			ui = true
		case "cli":
			cli = true
		}
	}
	for name := range in.Ports {
		mark(name)
	}
	for name := range in.Lifecycle.Health.Endpoints {
		mark(name)
	}
	if in.CLIEnabled {
		cli = true
	}
	return api, ui, cli
}

func scenarioDir(root string) string {
	return filepath.Base(strings.TrimRight(root, string(filepath.Separator)))
}

func scenarioName(root string, in intent.Intent) string {
	if n := strings.TrimSpace(in.Name); n != "" {
		return n
	}
	return scenarioDir(root)
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

func dirExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}
