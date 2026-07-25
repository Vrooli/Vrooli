package interactive

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"agent-manager/internal/domain"
)

// LaunchCommandParams is the input to BuildLaunchCommand.
type LaunchCommandParams struct {
	RunnerType domain.RunnerType
	// BinaryPath is the resolved interactive CLI binary (from AgentLaunchInfo).
	BinaryPath string
	// TagEnvKey / Tag carry the per-run reconciler tag (from AgentLaunchInfo).
	TagEnvKey string
	Tag       string
	// WorkingDir is the agent's cwd; the command cd's into it first.
	WorkingDir string
	// RunDir is the agent-manager-owned per-run directory under which the
	// relocated CODEX_HOME/GROK_HOME lives (ignored for claude).
	RunDir string
	// Model and Effort are resolved run controls forwarded to the interactive
	// CLI. Empty values preserve the CLI default.
	Model  string
	Effort domain.Effort
	// InitialPrompt is supplied as one trailing positional argument so the
	// interactive process receives the same initial task as codec-pipe runs.
	InitialPrompt string
}

// BuildLaunchCommand builds the shell command web-console pastes+executes to
// start the real interactive agent CLI, per design §4. The shape is:
//
//	cd <workdir> && <TAG_KEY>=<tag> [<HOME_VAR>=<runDir>/<subdir>] <binary>
//
// The inline env prefix wins over web-console's own per-session env for that
// process (design §4), pinning the tag (so the reconciler attributes the
// process from /proc) and, for codex/grok, an agent-manager-owned session home
// (so the transcript path is deterministic). claude gets no home relocation.
//
// All interpolated values are single-quote shell-escaped so paths or tags with
// spaces/metacharacters cannot break or inject into the command.
func BuildLaunchCommand(p LaunchCommandParams) (string, error) {
	spec, ok := specFor(p.RunnerType)
	if !ok {
		return "", fmt.Errorf("interactive mode is not supported for runner %q", p.RunnerType)
	}
	if strings.TrimSpace(p.BinaryPath) == "" {
		return "", fmt.Errorf("interactive launch for %s: binary path is empty (runner unavailable?)", p.RunnerType)
	}
	if strings.TrimSpace(p.WorkingDir) == "" {
		return "", fmt.Errorf("interactive launch for %s: working directory is required", p.RunnerType)
	}
	if strings.TrimSpace(p.TagEnvKey) == "" {
		return "", fmt.Errorf("interactive launch for %s: tag env key is required", p.RunnerType)
	}

	var envPrefix []string
	envPrefix = append(envPrefix, fmt.Sprintf("%s=%s", p.TagEnvKey, shellQuote(p.Tag)))
	if spec.relocatesHome() {
		if strings.TrimSpace(p.RunDir) == "" {
			return "", fmt.Errorf("interactive launch for %s: run dir is required to relocate %s", p.RunnerType, spec.homeEnvVar)
		}
		home := filepath.Join(p.RunDir, spec.homeSubdir)
		envPrefix = append(envPrefix, fmt.Sprintf("%s=%s", spec.homeEnvVar, shellQuote(home)))
	}

	args := []string{shellQuote(p.BinaryPath)}
	if model := strings.TrimSpace(p.Model); model != "" {
		args = append(args, "--model", shellQuote(model))
	}
	if p.Effort != "" {
		switch p.RunnerType {
		case domain.RunnerTypeCodex:
			args = append(args, "-c", shellQuote("model_reasoning_effort="+string(p.Effort)))
		case domain.RunnerTypeClaudeCode, domain.RunnerTypeGrok:
			args = append(args, "--effort", shellQuote(string(p.Effort)))
		}
	}
	if p.InitialPrompt != "" {
		args = append(args, shellQuote(p.InitialPrompt))
	}

	cmd := fmt.Sprintf("cd %s && %s %s",
		shellQuote(p.WorkingDir),
		strings.Join(envPrefix, " "),
		strings.Join(args, " "),
	)
	return cmd, nil
}

// seedRelocatedHome prepares a relocated-home agent's fresh run-scoped home so
// the launched CLI boots straight to its input prompt instead of a first-run
// gate (design §4). It (1) copies the shared home's credential/config seedFiles
// (auth.json, config.toml) so the CLI stays authenticated, and (2) appends a
// directory-trust entry to the run's private config so the CLI does not block on
// its "trust this directory?" prompt. Copies (not symlinks) keep the run's
// writes off the shared home, which dozens of concurrent CLI processes contend
// on. Missing source files are skipped — the shared home may legitimately lack
// one, and the launch surfaces any real auth gap as its own clear failure.
//
// userHome overrides the resolved home directory (tests point it at a fixture);
// empty uses os.UserHomeDir.
func seedRelocatedHome(spec agentSpec, relocatedHome, workingDir, userHome string) error {
	if spec.sharedHomeBase == "" {
		return nil
	}
	if userHome == "" {
		h, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("resolve home dir to seed interactive %s home: %w", spec.runnerType, err)
		}
		userHome = h
	}
	sharedHome := filepath.Join(userHome, spec.sharedHomeBase)
	for _, name := range spec.seedFiles {
		src := filepath.Join(sharedHome, name)
		if _, err := os.Stat(src); err != nil {
			continue // shared home lacks this file; skip (best-effort seed)
		}
		if err := copyFile(src, filepath.Join(relocatedHome, name)); err != nil {
			return fmt.Errorf("seed %s into interactive %s home: %w", name, spec.runnerType, err)
		}
	}
	if spec.dirTrustTOML != nil && spec.trustConfigFile != "" {
		if err := appendToFile(filepath.Join(relocatedHome, spec.trustConfigFile), spec.dirTrustTOML(workingDir)); err != nil {
			return fmt.Errorf("pre-trust working dir for interactive %s: %w", spec.runnerType, err)
		}
	}
	return nil
}

// cleanupSeededHome removes the credential/config files that seedRelocatedHome
// copied into a run's relocated home. It is called at session teardown — once
// the CLI that used them is dead — so a run's private copy of the shared home's
// credentials (codex auth.json/config.toml) does not linger on disk after the
// run is done. Best-effort and idempotent: a shared-home agent (claude) or a
// missing file is a no-op. Only the seeded files are removed; the discovered
// transcript and the rest of the run dir are left intact for diagnostics.
func cleanupSeededHome(spec agentSpec, relocatedHome string) error {
	if spec.sharedHomeBase == "" || relocatedHome == "" {
		return nil
	}
	var errs []error
	for _, name := range spec.seedFiles {
		if err := os.Remove(filepath.Join(relocatedHome, name)); err != nil && !os.IsNotExist(err) {
			errs = append(errs, fmt.Errorf("remove seeded %s from interactive %s home: %w", name, spec.runnerType, err))
		}
	}
	return errors.Join(errs...)
}

// copyFile copies src to dst (truncating dst), preserving nothing but contents.
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o600)
}

// appendToFile appends s to the file at path, creating it if absent.
func appendToFile(path, s string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	if _, werr := f.WriteString(s); werr != nil {
		_ = f.Close()
		return werr
	}
	return f.Close()
}

// homeDirFor returns the relocated session-home path for codex/grok under the
// run dir, or "" when the agent uses a shared home (claude). Used both to build
// the launch command and to know where to create the directory / discover the
// transcript.
func homeDirFor(spec agentSpec, runDir string) string {
	if !spec.relocatesHome() {
		return ""
	}
	return filepath.Join(runDir, spec.homeSubdir)
}

// shellQuote wraps s in single quotes, escaping embedded single quotes, so the
// result is a single safe shell word. Empty strings become ” (a valid empty
// argument) rather than a bare word.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
