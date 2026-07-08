package manifestvalidation

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"cli-health/internal/cliruntime"

	"github.com/vrooli/cli-core/cliapp"
)

// RuntimeProbe observes a scenario CLI binary's runtime behavior by resolving
// and exec'ing it (binary resolution + `--help`-tree walk both live in
// internal/cliruntime, shared with the AI-search index builder). It is the seam
// that turns cli-health's static manifest↔proto reconciliation into a static +
// runtime CLI authority.
//
// seam: RuntimeProbe — production wires cliRuntimeProbe (internal/cliruntime);
// tests inject a fake returning canned observations. Optional on Deps: a nil
// probe disables runtime probing entirely (the default static-only path). Even
// when wired, the probe runs only when the caller requests execution
// (include_execution) and the scenario declares a CLI surface.
type RuntimeProbe interface {
	Probe(ctx context.Context, scenario string) (RuntimeObservation, error)
}

// RuntimeObservation is what the probe saw when it ran the scenario CLI.
type RuntimeObservation struct {
	// Resolved reports whether the binary was found in this run context. When
	// false the scenario's CLI is simply not installed here (degrade, never a
	// hard error).
	Resolved bool
	// Binary is the resolved executable path (informational).
	Binary string
	// HelpFailed reports that the binary resolved but its root `--help` errored
	// or produced no output (a real defect — the CLI runs but its help is broken).
	HelpFailed bool
	// HelpError carries the help-failure detail when HelpFailed.
	HelpError string
	// Commands are the leaf commands the binary exposed at runtime, relative to
	// the scenario (origin stripped).
	Commands []RuntimeCommand
}

// RuntimeCommand is one leaf the binary exposed at runtime. Group is the first
// path segment ("" for a top-level command); Name is the leaf command.
type RuntimeCommand struct {
	Group string
	Name  string
}

// builtinCommands are commands cli-core injects into every CLI that are not
// part of the manifest's proto-bound surface. They must never be flagged as
// "undeclared" when observed at runtime.
var builtinCommands = map[string]bool{
	"help":       true,
	"version":    true,
	"configure":  true,
	"completion": true,
}

// cliRuntimeProbe is the production RuntimeProbe: it resolves the scenario's
// binary by name (matching the aisearch resolution rule) and walks its
// `--help` tree, both via internal/cliruntime.
type cliRuntimeProbe struct {
	timeout time.Duration
}

// NewCLIRuntimeProbe returns the production runtime probe. A non-positive
// timeout defaults inside cliruntime.ExecRunner.
func NewCLIRuntimeProbe(timeout time.Duration) RuntimeProbe {
	return cliRuntimeProbe{timeout: timeout}
}

func (p cliRuntimeProbe) Probe(ctx context.Context, scenario string) (RuntimeObservation, error) {
	bin := cliruntime.ResolveBinary(scenario, "")
	if bin == "" {
		return RuntimeObservation{Resolved: false}, nil
	}
	obs := RuntimeObservation{Resolved: true, Binary: bin}

	cmds := cliruntime.ParseHelpTree(ctx, cliruntime.ExecRunner(p.timeout), bin,
		cliruntime.HelpTreeOptions{Origin: scenario, MaxDepth: cliruntime.DefaultHelpMaxDepth})

	// A single help-failed stub means the root `--help` did not run cleanly.
	if len(cmds) == 1 && cmds[0].Source == cliruntime.SourceHelpFailed {
		obs.HelpFailed = true
		obs.HelpError = cmds[0].Description
		return obs, nil
	}
	for _, c := range cmds {
		if c.Source == cliruntime.SourceHelpFailed {
			continue
		}
		obs.Commands = append(obs.Commands, RuntimeCommand{Group: c.Group, Name: c.Name})
	}
	return obs, nil
}

// hasCLISurface reports whether the manifest declares any command (i.e. there is
// a CLI to probe at all). A manifest with no groups/commands has no runtime
// surface to validate.
func hasCLISurface(m *cliapp.Manifest) bool {
	if m == nil {
		return false
	}
	for _, g := range m.Groups {
		if len(g.Commands) > 0 {
			return true
		}
	}
	return false
}

// runtimeFindings turns a RuntimeObservation into findings, applying the
// degrade-don't-hard-fail policy:
//   - binary unresolved  -> warning (cli.binary_unrunnable): not installed here.
//   - root --help broken  -> error   (cli.help_failed): present but broken.
//   - command divergence  -> error   (cli.command_undeclared): runtime surface
//     diverges from the manifest SSOT (either direction).
func runtimeFindings(obs RuntimeObservation, m *cliapp.Manifest, manifestPath string) []Finding {
	if !obs.Resolved {
		return []Finding{{
			Severity:   SeverityWarning,
			Code:       CodeCLIBinaryUnrunnable,
			Location:   manifestPath,
			Message:    "scenario declares a CLI surface but its binary could not be resolved in this run context (PATH lookup failed)",
			Suggestion: "install/expose the scenario CLI so runtime validation can exercise it, or confirm cli.invoke.command matches the installed binary name",
		}}
	}
	if obs.HelpFailed {
		msg := "scenario CLI binary resolves but `--help` failed to run cleanly"
		if strings.TrimSpace(obs.HelpError) != "" {
			msg = fmt.Sprintf("%s: %s", msg, strings.TrimSpace(obs.HelpError))
		}
		return []Finding{{
			Severity:   SeverityError,
			Code:       CodeCLIHelpFailed,
			Location:   obs.Binary,
			Message:    msg,
			Suggestion: "run `<cli> --help` manually and fix the error so the CLI is introspectable",
		}}
	}
	return commandSurfaceFindings(obs, m, manifestPath)
}

// commandSurfaceFindings reconciles the binary's runtime command surface against
// the manifest's declared commands, in both directions, scoped to groups the
// manifest declares AND the binary actually exposes at runtime. Scoping this way
// keeps the check precise: framework-injected top-level commands (which live
// outside any manifest group) never trip it, and a help-parse gap that drops a
// whole group does not falsely report that group's commands as missing.
func commandSurfaceFindings(obs RuntimeObservation, m *cliapp.Manifest, manifestPath string) []Finding {
	// Commands declared as legitimate special cases in the manifest's top-level
	// exceptions[] are not "undeclared": they live outside the binding path on
	// purpose. Treat them like framework built-ins so declaring an exception
	// silences the undeclared-command finding for that command.
	declaredExceptions := exceptionCommandPaths(m)
	manifestByGroup := map[string]map[string]bool{}
	for _, g := range m.Groups {
		set := manifestByGroup[g.Name]
		if set == nil {
			set = map[string]bool{}
			manifestByGroup[g.Name] = set
		}
		for _, c := range g.Commands {
			set[c.Name] = true
		}
	}
	runtimeByGroup := map[string]map[string]bool{}
	for _, rc := range obs.Commands {
		set := runtimeByGroup[rc.Group]
		if set == nil {
			set = map[string]bool{}
			runtimeByGroup[rc.Group] = set
		}
		set[rc.Name] = true
	}

	var findings []Finding
	for _, group := range sortedKeys(manifestByGroup) {
		runtime, present := runtimeByGroup[group]
		if !present {
			// Group not observed at runtime — likely a help-parse gap or a
			// capability-gated group; do not flag its whole command set missing.
			continue
		}
		declared := manifestByGroup[group]

		// Declared-but-missing: the manifest claims a command the binary's help
		// tree does not expose under a group it otherwise does expose.
		for _, name := range sortedKeys(declared) {
			if runtime[name] {
				continue
			}
			findings = append(findings, Finding{
				Severity:   SeverityError,
				Code:       CodeCLICommandUndeclared,
				Location:   commandLocation(manifestPath, group, name),
				Message:    fmt.Sprintf("manifest declares command %q but the binary does not expose it at runtime under group %q", groupCmd(group, name), group),
				Suggestion: "rebuild/reinstall the CLI, or remove the stale command from the manifest",
			})
		}

		// Undeclared-accepted: the binary exposes a command under a manifest group
		// that the manifest neither binds nor (for built-ins) needs to.
		for _, name := range sortedKeys(runtime) {
			if declared[name] || builtinCommands[name] || declaredExceptions[normalizeCommandPath(groupCmd(group, name))] {
				continue
			}
			findings = append(findings, Finding{
				Severity:   SeverityError,
				Code:       CodeCLICommandUndeclared,
				Location:   commandLocation(manifestPath, group, name),
				Message:    fmt.Sprintf("binary exposes command %q at runtime but the manifest does not declare it (manifest is the CLI SSOT)", groupCmd(group, name)),
				Suggestion: "add the command to the manifest under its group/binding, declare it in exceptions[] if it is a legitimate special case (streaming/upload/passthrough/durable run), or remove it from the CLI",
			})
		}
	}
	return findings
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func groupCmd(group, name string) string {
	if group == "" {
		return name
	}
	return group + " " + name
}

func commandLocation(manifestPath, group, name string) string {
	return fmt.Sprintf("%s#/groups/%s/commands/%s", manifestPath, group, name)
}
