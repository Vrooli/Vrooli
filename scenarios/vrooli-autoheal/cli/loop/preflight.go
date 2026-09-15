package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/vrooli/repo-contract-go/cliinvoke"
	langrecover "vrooli-autoheal-langrecover"
)

// CheckStatus is a preflight check's verdict.
type CheckStatus string

const (
	CheckOK      CheckStatus = "ok"
	CheckFailed  CheckStatus = "failed"
	CheckSkipped CheckStatus = "skipped"
)

// PreflightCheck is one named probe and why it concluded what it did.
type PreflightCheck struct {
	Name   string      `json:"name"`
	Status CheckStatus `json:"status"`
	Reason string      `json:"reason,omitempty"`
	// Class is the invoker's classification when the check ran the CLI and
	// it failed (usage, binary-missing, timeout, ...). The status file
	// reports it as the failure class, because "preflight" is a stage.
	Class string `json:"class,omitempty"`
}

// PreflightResult is what --self-test prints and what the status file keeps.
type PreflightResult struct {
	At     time.Time        `json:"at"`
	OK     bool             `json:"ok"`
	Checks []PreflightCheck `json:"checks"`
}

// Failed lists the failed checks with their reasons, for a degraded reason.
func (r PreflightResult) Failed() []string {
	var names []string
	for _, check := range r.Checks {
		if check.Status == CheckFailed {
			names = append(names, check.Name+": "+check.Reason)
		}
	}
	return names
}

// FailureClass is the class of the first failed check that carries one, or
// "preflight" when the failure is not a CLI invocation's.
func (r PreflightResult) FailureClass() string {
	for _, check := range r.Checks {
		if check.Status == CheckFailed && check.Class != "" {
			return check.Class
		}
	}
	return "preflight"
}

// toolchainPathEntries is the table the toolchain check searches after PATH.
// A package variable so a test can take the host's real /usr/local/go/bin
// out of play.
var toolchainPathEntries = langrecover.DefaultPathEntries

// Preflight answers, before any heal is attempted, whether this loop is able
// to heal at all: it can find the CLI, the CLI answers, the CLI still speaks
// the contract the loop parses, the state directory takes writes, the
// recovery floor's toolchain is reachable, and the repository resolves. A
// failure here is non-healable by definition: retrying the lifecycle cannot
// fix a loop that cannot invoke it.
func Preflight(ctx context.Context, config *Config) PreflightResult {
	result := PreflightResult{At: time.Now().UTC(), OK: true}
	add := func(check PreflightCheck) {
		result.Checks = append(result.Checks, check)
		if check.Status == CheckFailed {
			result.OK = false
		}
	}
	plain := func(name string, status CheckStatus, reason string) PreflightCheck {
		return PreflightCheck{Name: name, Status: status, Reason: reason}
	}

	if config.VrooliCmdPath == "" {
		reason := "vrooli binary not found"
		if config.VrooliResolveErr != nil {
			reason = config.VrooliResolveErr.Error()
		}
		add(PreflightCheck{Name: "cli-resolves", Status: CheckFailed, Reason: reason, Class: cliinvoke.BinaryMissing.String()})
		add(plain("cli-answers", CheckSkipped, "no binary to invoke"))
		add(plain("cli-contract", CheckSkipped, "no binary to invoke"))
	} else {
		add(plain("cli-resolves", CheckOK, config.VrooliCmdPath))
		add(checkCLIAnswers(ctx, config))
		add(checkCLIContract(ctx, config))
	}

	add(checkStateWritable())

	if config.ManageAPILifecycle {
		add(checkToolchain())
	} else {
		add(plain("toolchain", CheckSkipped, "recovery floor disabled with --no-manage-api"))
	}

	if config.VrooliRoot == "" {
		add(plain("root-resolves", CheckFailed, "repository root could not be resolved from VROOLI_SOURCE_ROOT, VROOLI_ROOT, the working directory, or the executable path"))
	} else {
		add(plain("root-resolves", CheckOK, config.VrooliRoot))
	}
	return result
}

// checkCLIAnswers proves the binary runs and speaks JSON.
func checkCLIAnswers(ctx context.Context, config *Config) PreflightCheck {
	res := invokeVrooli(ctx, config, nil, nil, cliinvoke.VersionJSON()...)
	if res.Class != cliinvoke.OK {
		return failedInvocation("cli-answers", res)
	}
	var version struct {
		CLIVersion string `json:"cli_version"`
	}
	if err := json.Unmarshal(res.Stdout, &version); err != nil {
		return PreflightCheck{Name: "cli-answers", Status: CheckFailed, Reason: fmt.Sprintf("version --json did not parse: %v", err)}
	}
	return PreflightCheck{Name: "cli-answers", Status: CheckOK, Reason: "cli_version " + version.CLIVersion}
}

// checkCLIContract proves the status command still produces the shape the
// loop's port detection parses. A usage error here means the loop's argv is
// out of contract with the installed CLI.
func checkCLIContract(ctx context.Context, config *Config) PreflightCheck {
	res := invokeVrooli(ctx, config, nil, nil, cliinvoke.ScenarioStatusJSON(config.ScenarioName)...)
	if res.Class == cliinvoke.Usage || res.Class == cliinvoke.BinaryMissing || res.Class == cliinvoke.Timeout {
		return failedInvocation("cli-contract", res)
	}
	var status scenarioStatusResponse
	if err := json.Unmarshal(res.Stdout, &status); err != nil {
		return PreflightCheck{Name: "cli-contract", Status: CheckFailed, Reason: fmt.Sprintf("scenario status --json did not parse: %v", err)}
	}
	if status.Scenario.Name == "" {
		return PreflightCheck{Name: "cli-contract", Status: CheckFailed, Reason: "scenario status --json carries no scenario.name"}
	}
	return PreflightCheck{Name: "cli-contract", Status: CheckOK, Reason: fmt.Sprintf("scenario status --json parsed (%s)", res.Class)}
}

func checkStateWritable() PreflightCheck {
	dir, err := statusFileDir()
	if err != nil {
		return PreflightCheck{Name: "state-writable", Status: CheckFailed, Reason: err.Error()}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return PreflightCheck{Name: "state-writable", Status: CheckFailed, Reason: err.Error()}
	}
	probe, err := os.CreateTemp(dir, ".preflight-*")
	if err != nil {
		return PreflightCheck{Name: "state-writable", Status: CheckFailed, Reason: err.Error()}
	}
	name := probe.Name()
	_ = probe.Close()
	_ = os.Remove(name)
	return PreflightCheck{Name: "state-writable", Status: CheckOK, Reason: dir}
}

func checkToolchain() PreflightCheck {
	home, _ := os.UserHomeDir()
	path, err := langrecover.FindTool("go", exec.LookPath, toolchainPathEntries(runtime.GOOS, home))
	if err != nil {
		return PreflightCheck{Name: "toolchain", Status: CheckFailed, Reason: err.Error()}
	}
	return PreflightCheck{Name: "toolchain", Status: CheckOK, Reason: path}
}

// failedInvocation renders a failed CLI probe: the class, and the first
// line the CLI said about it.
func failedInvocation(name string, res cliinvoke.Result) PreflightCheck {
	detail := strings.TrimSpace(string(res.Stderr))
	if detail == "" && res.Err != nil {
		detail = res.Err.Error()
	}
	if line, _, found := strings.Cut(detail, "\n"); found {
		detail = line
	}
	return PreflightCheck{Name: name, Status: CheckFailed, Reason: fmt.Sprintf("%s: %s", res.Class, detail), Class: res.Class.String()}
}
