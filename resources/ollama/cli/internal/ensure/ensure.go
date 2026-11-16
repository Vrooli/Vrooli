// Package ensure implements `resource-ollama ensure`: it reads a scenario's
// ollama dependency config (model roles and direct model exceptions), preserves
// already-installed models, and admits any missing-model pull before writing.
package ensure

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
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

// PullLease owns one atomically admitted model acquisition. Implementations
// must renew the shared lease independently of progress frequency and keep it
// active after caller cancellation or uncertain remote activity. It may finish
// only after VerifyInstalled and VerifyCleanup have succeeded and Complete is
// called. Progress errors stop the client stream, but do not prove remote work
// or cleanup stopped.
type PullLease interface {
	LeaseID() string
	WorkContext() context.Context
	PullReference() string
	Install(context.Context, string) error
	Abandon()
	VerifyRemote(context.Context) error
	ObserveProgress(context.Context, PullProgress) error
	VerifyInstalled(context.Context, ModelInfo) error
	VerifyCleanup(context.Context) error
	RecordInstallProvenance(context.Context, ModelInfo) error
	Complete(context.Context) error
}

// PullAdmission reserves the complete model-store peak before a pull starts.
// A nil PullAdmission refuses all missing-model pulls.
type PullAdmission func(ctx context.Context, modelRef string) (PullLease, error)

// PullEndpointPreparer acquires the managed lifecycle lock and binds
// missing-model work to the managed Ollama process. It is called only after an
// initial inventory proves at least one model is missing, preserving the
// existing already-installed inventory path. The returned release is held
// until ensure returns.
type PullEndpointPreparer func(ctx context.Context, client *Client) (release func(), err error)

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
	client := NewClient()
	var admission *DiskAdmission
	var admissionClose func() error
	pullAdmission := func(admitCtx context.Context, modelRef string) (PullLease, error) {
		if admission == nil {
			var setupErr error
			admission, admissionClose, setupErr = NewDiskAdmission(admitCtx, client)
			if setupErr != nil {
				return nil, setupErr
			}
		}
		return admission.Admit(admitCtx, modelRef)
	}
	prepareEndpoint := func(prepareCtx context.Context, target *Client) (func(), error) {
		release, lockErr := acquireManagedOllamaLifecycleLock(prepareCtx)
		if lockErr != nil {
			return nil, lockErr
		}
		root, rootErr := effectiveModelRoot()
		if rootErr != nil {
			release()
			return nil, rootErr
		}
		managedClient, _, identityErr := resolveManagedOllamaClient(prepareCtx, root)
		if identityErr != nil {
			release()
			return nil, identityErr
		}
		*target = *managedClient
		return release, nil
	}
	err = runWithPullAdmissionAndEndpoint(ctx, cfg, client, os.Stdout, validator, prepareEndpoint, pullAdmission)
	if admissionClose != nil {
		if closeErr := admissionClose(); err == nil && closeErr != nil {
			err = fmt.Errorf("close shared capacity broker: %w", closeErr)
		}
	}
	return err
}

// Run is the fail-closed entry point. It lists installed tags and preserves the
// no-pull path for already-installed models. It has no pull adapter, so missing
// models are refused. When a validator is supplied, it runs after ensure.
func Run(ctx context.Context, cfg Config, client *Client, stdout io.Writer, validator AdmissionValidator) error {
	return runWithPullAdmissionAndEndpoint(ctx, cfg, client, stdout, validator, nil, nil)
}

// runWithPullAdmissionAndEndpoint preserves the no-pull inventory route, then
// optionally prepares and rechecks the managed endpoint before any admitted
// missing-model pull can start. Every missing-model caller must supply the
// production lifecycle-lock and managed-endpoint preparation boundary.
func runWithPullAdmissionAndEndpoint(ctx context.Context, cfg Config, client *Client, stdout io.Writer, validator AdmissionValidator, prepareEndpoint PullEndpointPreparer, pullAdmission PullAdmission) error {
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
		if !modelReferenceInstalled(installed, ref) {
			missing = append(missing, ref)
		}
	}
	if len(missing) == 0 {
		fmt.Fprintf(stdout, "%s all %d model(s) already installed\n", logPrefix, len(refs))
		return gate()
	}
	if pullAdmission == nil {
		return fmt.Errorf("%s shared disk-job admission is unavailable; refusing missing-model pull(s): %s", logPrefix, strings.Join(missing, ", "))
	}
	if prepareEndpoint == nil {
		return fmt.Errorf("%s managed lifecycle lock and endpoint preparation are unavailable; refusing missing-model pull(s): %s", logPrefix, strings.Join(missing, ", "))
	}
	release, prepareErr := prepareEndpoint(ctx, client)
	if release != nil {
		defer release()
	}
	if prepareErr != nil {
		return fmt.Errorf("%s could not bind missing-model acquisition to the managed Ollama process: %w", logPrefix, prepareErr)
	}
	if release == nil {
		return fmt.Errorf("%s managed lifecycle preparation returned no lock release handle; refusing missing-model pull(s)", logPrefix)
	}
	localClient, err := newLoopbackClient(client.BaseURL, net.DefaultResolver.LookupIP)
	if err != nil {
		return fmt.Errorf("%s missing-model acquisition requires the managed local Ollama endpoint: %w", logPrefix, err)
	}
	*client = *localClient
	// The initial inventory may have come from the caller's configured endpoint.
	// Once bound to the verified managed process, re-read tags before admitting
	// so we neither pull a model already installed there nor trust foreign tags.
	installed, err = client.ListTags(ctx)
	if err != nil {
		return fmt.Errorf("verify managed Ollama model inventory: %w", err)
	}
	missing = missing[:0]
	for _, ref := range refs {
		if !modelReferenceInstalled(installed, ref) {
			missing = append(missing, ref)
		}
	}
	if len(missing) == 0 {
		fmt.Fprintf(stdout, "%s all %d model(s) already installed on managed Ollama\n", logPrefix, len(refs))
		return gate()
	}

	fmt.Fprintf(stdout, "%s pulling %d missing model(s): %s\n", logPrefix, len(missing), strings.Join(missing, ", "))

	var errs []error
	for _, ref := range missing {
		pullCtx, cancel := context.WithTimeout(ctx, defaultPerModelTimeout)
		start := time.Now()
		lease, err := pullAdmission(pullCtx, ref)
		workCtx := pullCtx
		if err == nil && lease == nil {
			err = errors.New("shared disk-job admission returned no lease")
		}
		if err == nil {
			if lease.WorkContext() != nil {
				workCtx = lease.WorkContext()
			}
			pullRef := strings.TrimSpace(lease.PullReference())
			if pullRef == "" {
				err = errors.New("shared disk-job admission returned an empty pinned pull reference")
			} else {
				err = lease.VerifyRemote(workCtx)
				if err == nil {
					err = client.pullPinned(workCtx, pullRef, func(progress PullProgress) error {
						if progressErr := lease.ObserveProgress(workCtx, progress); progressErr != nil {
							return fmt.Errorf("sample admitted pull progress: %w", progressErr)
						}
						reportProgress(stdout, ref, progress)
						return nil
					})
				}
				if err == nil {
					err = lease.Install(workCtx, ref)
				}
			}
		}
		if err == nil {
			var inventory []ModelInfo
			inventory, err = client.ListModelInventory(workCtx)
			if err != nil {
				err = fmt.Errorf("verify installed model inventory: %w", err)
			} else {
				var installed *ModelInfo
				canonicalRef := normalizedDefaultModelRef(ref)
				for i := range inventory {
					if inventory[i].Name == ref || inventory[i].Name == withLatestTag(ref) || inventory[i].Name == canonicalRef || inventory[i].Name == withLatestTag(canonicalRef) {
						installed = &inventory[i]
						break
					}
				}
				if installed == nil || installed.Size <= 0 || strings.TrimSpace(installed.Digest) == "" {
					err = fmt.Errorf("verify installed model state: %s is absent or has unknown size/digest", ref)
				} else if err = lease.VerifyInstalled(workCtx, *installed); err == nil {
					err = lease.VerifyCleanup(workCtx)
					if err == nil {
						err = lease.RecordInstallProvenance(workCtx, *installed)
						if err == nil {
							err = lease.Complete(workCtx)
						}
					}
				}
			}
		}
		cancel()
		if err != nil && lease != nil {
			if id := strings.TrimSpace(lease.LeaseID()); id != "" {
				err = fmt.Errorf("disk lease %s remains active for owner recovery: %w", id, err)
			}
			lease.Abandon()
		}
		elapsed := time.Since(start).Round(tuning.ProgressDisplayResolution())
		if err != nil {
			fmt.Fprintf(stdout, "%s pull %s FAILED after %s: %v\n", logPrefix, ref, elapsed, err)
			errs = append(errs, err)
			// A failed pull can leave remote work or partial files behind. Do not
			// begin a second acquisition until the owner has reconciled that state.
			break
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
