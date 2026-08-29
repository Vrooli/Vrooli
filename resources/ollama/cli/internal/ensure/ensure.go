// Package ensure implements `resource-ollama ensure`: it reads a scenario's
// ollama dependency config (model roles and direct model exceptions) and pulls
// any resolved models that aren't already installed on the local Ollama instance.
package ensure

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/tuning"

	"github.com/vrooli/vrooli/resources/ollama/cli/internal/policy"

	"github.com/vrooli/cli-core/cliapp"
)

var (
	defaultPerModelTimeout = tuning.ResourceLongHTTPTimeout()
	logPrefix              = "ollama-ensure:"
)

// AdmissionValidator is the fail-closed model-admission gate run after pulls.
// It receives the role names resolved in this ensure; returning an error
// aborts ensure so a model that cannot satisfy a tool role is not seated. It
// is injected (rather than imported) so the low-level ensure package never
// depends on the higher-level probe/doctor logic. nil disables the gate.
type AdmissionValidator func(ctx context.Context, resolvedRoles []string) error

// CommandGroup returns the cliapp group registering the `ensure` verb on the
// resource's CLI. The optional validator wires the fail-closed admission gate
// (built from the models-doctor SSOT by main). Pass the result into
// app.SetCommands alongside app.StandardLifecycleCommands().
func CommandGroup(validator AdmissionValidator) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Dependency Ensurance",
		Commands: []cliapp.Command{
			{
				Name:        "ensure",
				Description: "Resolve and pull models declared in a scenario's ollama dependency config",
				Usage:       "resource-ollama ensure --config-base64 <base64-json>",
				Run:         func(args []string) error { return runCLI(args, validator) },
			},
		},
	}
}

func runCLI(args []string, validator AdmissionValidator) error {
	var (
		configB64      string
		timeoutSeconds int
	)
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--config-base64":
			if i+1 >= len(args) {
				return errors.New("--config-base64 requires a value")
			}
			configB64 = args[i+1]
			i++
		case "--timeout-seconds":
			if i+1 >= len(args) {
				return errors.New("--timeout-seconds requires a value")
			}
			v, err := strconv.Atoi(args[i+1])
			if err != nil || v <= 0 {
				return fmt.Errorf("--timeout-seconds must be a positive integer, got %q", args[i+1])
			}
			timeoutSeconds = v
			i++
		default:
			return fmt.Errorf("unknown flag %q", args[i])
		}
	}
	if configB64 == "" {
		return errors.New("--config-base64 is required")
	}

	raw, err := base64.StdEncoding.DecodeString(configB64)
	if err != nil {
		return fmt.Errorf("decode --config-base64: %w", err)
	}
	cfg, err := ParseConfig(raw)
	if err != nil {
		return err
	}

	ctx := context.Background()
	if timeoutSeconds > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
		defer cancel()
	}
	return Run(ctx, cfg, NewClient(), os.Stdout, validator)
}

// Run is the testable entry point: given a parsed config, a client, and a
// writer, it lists installed tags, diffs, pulls the missing ones, and — when a
// validator is supplied — runs the fail-closed admission gate over the
// resolved roles before returning success.
func Run(ctx context.Context, cfg Config, client *Client, stdout io.Writer, validator AdmissionValidator) error {
	resolution, err := resolveConfigModels(cfg)
	if err != nil {
		return err
	}
	for _, warning := range resolution.Warnings {
		fmt.Fprintf(stdout, "%s warning: %s\n", logPrefix, warning)
	}
	if len(resolution.Models) == 0 {
		fmt.Fprintf(stdout, "%s no models requested; nothing to do\n", logPrefix)
		return nil
	}

	refs := make([]string, 0, len(resolution.Models))
	resolvedRoles := make([]string, 0, len(resolution.Models))
	for _, m := range resolution.Models {
		refs = append(refs, m.Ref)
		if m.Source == "role" {
			resolvedRoles = append(resolvedRoles, m.Role)
			fmt.Fprintf(stdout, "%s resolved role %s -> %s\n", logPrefix, m.Role, m.Ref)
		}
	}
	if len(refs) == 0 {
		fmt.Fprintf(stdout, "%s config listed only empty model specs; nothing to do\n", logPrefix)
		return nil
	}

	// gate runs the fail-closed admission validator over the resolved roles
	// after the models are confirmed present. A nil validator is a no-op.
	gate := func() error {
		if validator == nil {
			return nil
		}
		if err := validator(ctx, resolvedRoles); err != nil {
			return fmt.Errorf("model admission gate: %w", err)
		}
		return nil
	}

	installed, err := client.ListTags(ctx)
	if err != nil {
		return fmt.Errorf("list installed models: %w", err)
	}

	missing := make([]string, 0, len(refs))
	for _, ref := range refs {
		if !installed[ref] && !installed[withLatestTag(ref)] {
			missing = append(missing, ref)
		}
	}
	if len(missing) == 0 {
		fmt.Fprintf(stdout, "%s all %d model(s) already installed\n", logPrefix, len(refs))
		return gate()
	}

	fmt.Fprintf(stdout, "%s pulling %d missing model(s): %s\n", logPrefix, len(missing), strings.Join(missing, ", "))

	var errs []error
	for _, ref := range missing {
		pullCtx, cancel := context.WithTimeout(ctx, defaultPerModelTimeout)
		start := time.Now()
		err := client.Pull(pullCtx, ref, func(p PullProgress) {
			reportProgress(stdout, ref, p)
		})
		cancel()
		elapsed := time.Since(start).Round(tuning.ProgressDisplayResolution())
		if err != nil {
			fmt.Fprintf(stdout, "%s pull %s FAILED after %s: %v\n", logPrefix, ref, elapsed, err)
			errs = append(errs, err)
			continue
		}
		fmt.Fprintf(stdout, "%s pull %s OK in %s\n", logPrefix, ref, elapsed)
	}
	if len(errs) > 0 {
		return fmt.Errorf("%d model pull(s) failed: %w", len(errs), errors.Join(errs...))
	}
	return gate()
}

func resolveConfigModels(cfg Config) (policy.Resolution, error) {
	if len(cfg.ModelRoles) == 0 && len(cfg.Models) == 0 && strings.TrimSpace(cfg.Model) == "" {
		return policy.Resolution{}, nil
	}
	p, _, err := policy.LoadDefaultFile(os.Getenv)
	if err != nil {
		return policy.Resolution{}, err
	}
	resolution, err := p.Resolve(cfg.ResolveRequest())
	if err != nil {
		return policy.Resolution{}, err
	}
	return resolution, nil
}

func withLatestTag(ref string) string {
	if strings.Contains(ref, ":") {
		return ref
	}
	return ref + ":latest"
}

func reportProgress(w io.Writer, ref string, p PullProgress) {
	status := strings.TrimSpace(p.Status)
	if status == "" {
		return
	}
	if p.Total > 0 {
		pct := float64(p.Completed) / float64(p.Total) * 100
		fmt.Fprintf(w, "%s %s: %s (%.0f%%)\n", logPrefix, ref, status, pct)
		return
	}
	fmt.Fprintf(w, "%s %s: %s\n", logPrefix, ref, status)
}
