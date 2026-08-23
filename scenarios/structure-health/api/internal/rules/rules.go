// Package rules turns a reconcile.Model into structure-health findings. The
// rules are profile-aware but the universal checks here apply to every scenario;
// profile-keyed conformance packs layer on in a later phase.
//
// Each finding's Code matches a code in .vrooli/maturity.json so the assessment
// builder can map it to a maturity level and severity.
package rules

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"structure-health/internal/intent"
	"structure-health/internal/reconcile"
)

// Finding is a structure-health rule finding (mapped to a native/proto finding
// and a maturity assessment finding downstream).
type Finding struct {
	Code        string
	Severity    string
	Title       string
	Message     string
	Location    string
	Remediation string
	Surface     string
}

const (
	sevError   = "error"
	sevWarning = "warning"
)

// Input carries everything the rules need.
type Input struct {
	Model reconcile.Model
	// ServiceJSONReadable is false when service.json could not be parsed.
	ServiceJSONReadable bool
}

// Evaluate runs every universal structure + lifecycle-wiring rule and returns
// the findings in a stable order.
func Evaluate(in Input) []Finding {
	var out []Finding
	out = append(out, serviceJSONRules(in)...)
	out = append(out, requiredFileRules(in)...)
	out = append(out, surfaceDirRules(in)...)
	out = append(out, apiEntrypointRules(in)...)
	out = append(out, dependencyRules(in)...)
	out = append(out, portBandRules(in)...)
	out = append(out, reconcileRules(in)...)
	out = append(out, deployabilityInstanceRules(in)...)
	return out
}

// serviceJSONRules checks the service.json contract: present, valid, and
// service.name matching the scenario directory.
func serviceJSONRules(in Input) []Finding {
	root := in.Model.RootPath
	if !exists(filepath.Join(root, ".vrooli", "service.json")) {
		return []Finding{{
			Code:        "SERVICE_JSON_MISSING",
			Severity:    sevError,
			Title:       "service.json is missing",
			Message:     "The scenario has no .vrooli/service.json; the lifecycle engine cannot start it.",
			Location:    ".vrooli/service.json",
			Remediation: "Add a .vrooli/service.json declaring the service, ports, and lifecycle.",
		}}
	}
	if !in.ServiceJSONReadable {
		return []Finding{{
			Code:        "SERVICE_JSON_INVALID",
			Severity:    sevError,
			Title:       "service.json is invalid",
			Message:     "The scenario's .vrooli/service.json could not be parsed.",
			Location:    ".vrooli/service.json",
			Remediation: "Fix the JSON syntax / schema so service.json parses.",
		}}
	}
	var out []Finding
	declaredName := strings.TrimSpace(in.Model.Intent.Name)
	dirName := filepath.Base(strings.TrimRight(root, string(filepath.Separator)))
	if declaredName != "" && dirName != "" && declaredName != dirName {
		out = append(out, Finding{
			Code:        "SERVICE_NAME_MISMATCH",
			Severity:    sevError,
			Title:       "service.name does not match the directory",
			Message:     "service.name is " + declaredName + " but the scenario directory is " + dirName + ".",
			Location:    ".vrooli/service.json",
			Remediation: "Set service.name to " + dirName + " (it must equal the scenario directory name).",
		})
	}
	return out
}

// requiredFileRules asserts the universal top-level files every scenario needs.
func requiredFileRules(in Input) []Finding {
	root := in.Model.RootPath
	var out []Finding
	for _, f := range []struct{ rel, why string }{
		{"Makefile", "the canonical lifecycle entrypoint (make start/test/logs/stop)"},
		{"README.md", "the scenario overview and documentation map"},
	} {
		if !exists(filepath.Join(root, filepath.FromSlash(f.rel))) {
			out = append(out, Finding{
				Code:        "REQUIRED_FILE_MISSING",
				Severity:    sevError,
				Title:       "required file missing: " + f.rel,
				Message:     f.rel + " is required — " + f.why + ".",
				Location:    f.rel,
				Remediation: "Add a " + f.rel + " to the scenario root.",
			})
		}
	}
	return out
}

// surfaceDirRules asserts each declared surface has a directory on disk.
func surfaceDirRules(in Input) []Finding {
	root := in.Model.RootPath
	var out []Finding
	for _, s := range in.Model.Surfaces {
		if !s.Declared {
			continue
		}
		dir := surfaceDir(s.Surface)
		if dir == "" {
			continue
		}
		if !dirExists(filepath.Join(root, dir)) {
			out = append(out, Finding{
				Code:        "SURFACE_DIR_MISSING",
				Severity:    sevError,
				Title:       "declared surface has no directory: " + s.Surface,
				Message:     "Surface " + s.Surface + " is declared but its " + dir + "/ directory is missing.",
				Location:    dir + "/",
				Remediation: "Create the " + dir + "/ directory and its entrypoint, or remove the declaration.",
				Surface:     s.Surface,
			})
		}
	}
	return out
}

// dependencyRules asserts dependency declarations use valid enums and no edge
// names the scenario itself (a trivial cycle).
func dependencyRules(in Input) []Finding {
	var out []Finding
	check := func(kind string, deps map[string]intent.Dependency) {
		for name, dep := range deps {
			if name == in.Model.Scenario {
				out = append(out, Finding{
					Code:        "DEPENDENCY_DECLARATION_INVALID",
					Severity:    sevError,
					Title:       "self-dependency declared",
					Message:     "The scenario declares a " + kind + " dependency on itself (" + name + ").",
					Location:    ".vrooli/service.json",
					Remediation: "Remove the self-referential dependency.",
				})
			}
			if p := dep.StartupPolicy; p != "" && !validStartupPolicy(p) {
				out = append(out, Finding{
					Code:        "DEPENDENCY_DECLARATION_INVALID",
					Severity:    sevError,
					Title:       "invalid startup_policy for " + name,
					Message:     "Dependency " + name + " has startup_policy " + p + " (must be must_start|try_start|ignore).",
					Location:    ".vrooli/service.json",
					Remediation: "Set a valid startup_policy.",
				})
			}
			if p := dep.FreshnessPolicy; p != "" && !validFreshnessPolicy(p) {
				out = append(out, Finding{
					Code:        "DEPENDENCY_DECLARATION_INVALID",
					Severity:    sevError,
					Title:       "invalid freshness_policy for " + name,
					Message:     "Dependency " + name + " has freshness_policy " + p + " (must be restart_when_stale|reuse_running|rebuild_only).",
					Location:    ".vrooli/service.json",
					Remediation: "Set a valid freshness_policy.",
				})
			}
		}
	}
	check("scenario", in.Model.Intent.Deps.Scenarios)
	check("resource", in.Model.Intent.Deps.Resources)
	return out
}

// portBandRules asserts that each canonical listener port (api/ui/websocket)
// declares its canonical env_var and, when it uses a range, the canonical band.
// It fires only on the safe-to-rewrite cases (a present-but-wrong env_var or
// range on a canonically-identified port) so the matching auto-fixer can correct
// it deterministically; missing allocations and fixed-port out-of-band cases are
// left to the broader (detection-only) PROFILE_PORTS pack. Advisory: the coarse
// pack already carries the blocking severity for the same misconfiguration.
func portBandRules(in Input) []Finding {
	var out []Finding
	for _, name := range sortedKeys(in.Model.Intent.Ports) {
		p := in.Model.Intent.Ports[name]
		band, env, ok := CanonicalPortBand(name, p.EnvVar)
		if !ok {
			continue
		}
		var problems []string
		if v := strings.TrimSpace(p.EnvVar); v != "" && v != env {
			problems = append(problems, "env_var "+v+" should be "+env)
		}
		if r := strings.TrimSpace(p.Range); r != "" && r != band {
			problems = append(problems, "range "+r+" should be "+band)
		}
		if len(problems) == 0 {
			continue
		}
		out = append(out, Finding{
			Code:        "PORT_BAND_NONCONFORMANT",
			Severity:    sevWarning,
			Title:       "canonical port " + name + " is outside its allocated band",
			Message:     "ports." + name + ": " + strings.Join(problems, "; ") + " (canonical " + name + " band).",
			Location:    ".vrooli/service.json",
			Remediation: "Set ports." + name + ".env_var to " + env + " and its range to " + band + ".",
		})
	}
	return out
}

// reconcileRules flags declared-but-not-detected and detected-but-not-declared
// surfaces.
func reconcileRules(in Input) []Finding {
	var out []Finding
	for _, s := range in.Model.Surfaces {
		switch {
		case s.Declared && !s.Actual:
			out = append(out, Finding{
				Code:        "SURFACE_RECONCILE_MISMATCH",
				Severity:    sevWarning,
				Title:       "declared surface not detected: " + s.Surface,
				Message:     "Surface " + s.Surface + " is declared in service.json but code facts detected no implementation.",
				Location:    surfaceDir(s.Surface) + "/",
				Remediation: "Implement the " + s.Surface + " surface or remove its declaration.",
				Surface:     s.Surface,
			})
		case !s.Declared && s.Actual:
			out = append(out, Finding{
				Code:        "SURFACE_RECONCILE_MISMATCH",
				Severity:    sevWarning,
				Title:       "detected surface not declared: " + s.Surface,
				Message:     "Code facts detected a " + s.Surface + " surface that service.json does not declare (no port/health/cli wiring).",
				Location:    surfaceDir(s.Surface) + "/",
				Remediation: "Declare the " + s.Surface + " surface in service.json (ports/health/cli) or remove the code.",
				Surface:     s.Surface,
			})
		}
	}
	return out
}

// --- helpers ---

func surfaceDir(surface string) string {
	switch strings.ToLower(surface) {
	case "api", "ui", "cli":
		return strings.ToLower(surface)
	default:
		return ""
	}
}

func apiDeclared(m reconcile.Model) bool { return surfaceDeclared(m, "api") }
func uiDeclared(m reconcile.Model) bool  { return surfaceDeclared(m, "ui") }

func surfaceDeclared(m reconcile.Model, kind string) bool {
	for _, s := range m.Surfaces {
		if strings.EqualFold(s.Kind, kind) && s.Declared {
			return true
		}
	}
	return false
}

// CanonicalPortBand returns the canonical range + env_var for a canonically
// identified listener port (api/ui/websocket, by name or env_var) and ok=false
// for scenario-defined ports that receive no band enforcement. It is the single
// source of truth shared by portBandRules and the port-band auto-fixer.
func CanonicalPortBand(name, envVar string) (band, env string, ok bool) {
	switch {
	case name == "api" || envVar == "API_PORT":
		return "15000-19999", "API_PORT", true
	case name == "ui" || envVar == "UI_PORT":
		return "20000-24999", "UI_PORT", true
	case name == "websocket" || name == "ws" || envVar == "WS_PORT":
		return "25000-29999", "WS_PORT", true
	default:
		return "", "", false
	}
}

// firstNonEmpty returns the first non-blank string.
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// sortedKeys returns the map keys in deterministic order.
func sortedKeys(m map[string]intent.Port) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func validStartupPolicy(p string) bool {
	switch p {
	case "must_start", "try_start", "ignore":
		return true
	}
	return false
}

func validFreshnessPolicy(p string) bool {
	switch p {
	case "restart_when_stale", "reuse_running", "rebuild_only":
		return true
	}
	return false
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func dirExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}
