// Package gocacheapp is the control plane's owner surface for the shared Go
// build cache: activation of the build-cache protocol, status, and the
// owner-budgeted reclaim Storage Manager invokes. The protocol itself lives in
// internal/gocache.
package gocacheapp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/api-core/retention"
	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/gocache"
	"github.com/vrooli/vrooli/internal/shell"
	"github.com/vrooli/vrooli/internal/tools"
)

// OwnerCommand is the command Storage Manager declares for this owner.
const OwnerCommand = "vrooli host go-cache"

// Service resolves the host's helper, store, and owner budget.
type Service struct {
	// Home overrides the user home; empty means the current user's.
	Home string
	// Now overrides the clock in tests.
	Now func() time.Time
}

// ReclaimOptions selects one owner reclaim.
type ReclaimOptions struct {
	Cache string
	// Token binds the reclaim to a plan; empty plans afresh first.
	Token string
	// MaxBytes bounds the reclaim; zero frees everything the plan selected.
	MaxBytes int64
	// TimeBudget stops the sweep cleanly before a caller's deadline.
	TimeBudget time.Duration
}

func (s Service) home() (string, error) {
	if os.Geteuid() == 0 {
		return "", errors.New("the Go build-cache protocol is per user; run this command as the user whose builds it serves, not as root")
	}
	if s.Home != "" {
		return s.Home, nil
	}
	return os.UserHomeDir()
}

// Program returns the installed helper's absolute path, the exact value
// GOCACHEPROG carries when the protocol is active.
func (s Service) Program() (string, error) {
	home, err := s.home()
	if err != nil {
		return "", err
	}
	root, err := repocontract.VrooliUserRoot(home)
	if err != nil {
		return "", fmt.Errorf("resolve Vrooli runtime home: %w", err)
	}
	program := filepath.Join(root, "bin", gocache.ProgramName)
	if strings.ContainsAny(program, " \t\n'\"") {
		return "", fmt.Errorf("helper path %q contains whitespace or quotes, which GOCACHEPROG cannot carry unquoted", program)
	}
	return program, nil
}

// Policy returns the owner budget declared by the Go tool manifest and the
// protocol's recency floor.
func (Service) Policy() (gocache.Policy, error) {
	data, err := tools.Manifests.ReadFile("go/tool.json")
	if err != nil {
		return gocache.Policy{}, err
	}
	var manifest struct {
		Storage struct {
			Entries map[string]struct {
				Budget struct {
					MaxBytes string `json:"max_bytes"`
				} `json:"budget"`
				Reclaim struct {
					Pruner  string `json:"pruner"`
					Command string `json:"command"`
				} `json:"reclaim"`
			} `json:"entries"`
		} `json:"storage"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return gocache.Policy{}, fmt.Errorf("parse Go tool manifest: %w", err)
	}
	entry, ok := manifest.Storage.Entries["build_cache"]
	if !ok || entry.Reclaim.Pruner != "custom" || entry.Reclaim.Command != OwnerCommand {
		return gocache.Policy{}, fmt.Errorf("the Go tool manifest's build_cache entry must delegate reclaim to %q", OwnerCommand)
	}
	budget, err := retention.ParseBytes(entry.Budget.MaxBytes)
	if err != nil || budget <= 0 {
		return gocache.Policy{}, fmt.Errorf("the Go tool manifest's build_cache budget %q is not a positive size", entry.Budget.MaxBytes)
	}
	return gocache.Policy{BudgetBytes: budget, MinIdle: gocache.DefaultMinIdle}, nil
}

// Store opens the named cache, or this user's default GOCACHE.
func (Service) Store(cache string) (gocache.Store, error) {
	if strings.TrimSpace(cache) == "" {
		resolved, err := gocache.DefaultCacheDir()
		if err != nil {
			return gocache.Store{}, err
		}
		cache = resolved
	}
	return gocache.OpenStore(cache)
}

// Status reports activation, sessions, the go-command audit, and the last
// reclaim for a store without scanning it.
func (s Service) Status(cache string) (gocache.Status, error) {
	store, program, err := s.storeAndProgram(cache)
	if err != nil {
		return gocache.Status{}, err
	}
	return gocache.ReadStatus(store, program, s.Now)
}

// Plan proposes an owner-budgeted reclaim.
func (s Service) Plan(cache string, fresh bool) (gocache.Plan, error) {
	store, program, err := s.storeAndProgram(cache)
	if err != nil {
		return gocache.Plan{}, err
	}
	policy, err := s.Policy()
	if err != nil {
		return gocache.Plan{}, err
	}
	return gocache.MakePlan(gocache.PlanOptions{Store: store, Program: program, Policy: policy, Fresh: fresh, Now: s.Now})
}

// Reclaim applies a plan. Without a token it plans afresh and applies that
// plan, so an operator invocation is one command.
func (s Service) Reclaim(opts ReclaimOptions) (gocache.Receipt, error) {
	store, program, err := s.storeAndProgram(opts.Cache)
	if err != nil {
		return gocache.Receipt{}, err
	}
	policy, err := s.Policy()
	if err != nil {
		return gocache.Receipt{}, err
	}
	token := opts.Token
	if token == "" {
		plan, err := gocache.MakePlan(gocache.PlanOptions{Store: store, Program: program, Policy: policy, Fresh: true, Now: s.Now})
		if err != nil {
			return gocache.Receipt{}, err
		}
		if plan.BlockedReason != "" || plan.Token == "" {
			reason := plan.BlockedReason
			stop := gocache.StopBlocked
			if reason == "" {
				stop = gocache.StopComplete
			}
			at := s.now().UTC()
			return gocache.Receipt{Schema: gocache.ReceiptSchema, Cache: store.Root, StoreID: plan.StoreID, StartedAt: at, FinishedAt: at, StopReason: stop, BlockedReason: reason, PlanExhausted: reason == "", Audit: plan.Audit}, nil
		}
		token = plan.Token
	}
	var deadline time.Time
	if opts.TimeBudget > 0 {
		deadline = s.now().Add(opts.TimeBudget)
	}
	return gocache.Apply(gocache.ApplyOptions{Store: store, Program: program, Policy: policy, Token: token, MaxBytes: opts.MaxBytes, Deadline: deadline, Now: s.Now})
}

// Activate self-tests the installed helper and then points this user's go
// env file at it, so every go command started afterwards uses the protocol.
func (s Service) Activate(ctx context.Context) (gocache.Activation, error) {
	program, err := s.Program()
	if err != nil {
		return gocache.Activation{}, err
	}
	current := gocache.ReadActivation(program)
	if current.ConfiguredProgram != "" && current.ConfiguredProgram != program {
		return current, fmt.Errorf("GOCACHEPROG in %s is already %q; Vrooli will not replace another cache program", current.EnvFile, current.ConfiguredProgram)
	}
	if !current.ProgramInstalled {
		return current, fmt.Errorf("the helper %s is not installed; install the control plane (make install) first", program)
	}
	if err := gocache.SelfTest(ctx, program); err != nil {
		return current, fmt.Errorf("helper self-test failed; GOCACHEPROG left unchanged: %w", err)
	}
	if current.Active {
		return current, nil
	}
	if err := s.goEnv(ctx, "-w", "GOCACHEPROG="+program); err != nil {
		return current, err
	}
	activation := gocache.ReadActivation(program)
	if !activation.Active {
		return activation, fmt.Errorf("go env -w completed but the protocol is not active: %s", activation.Reason)
	}
	return activation, nil
}

// Deactivate removes the helper from this user's go env file. Running go
// commands keep their sessions until they exit; reclaim refuses from now on.
func (s Service) Deactivate(ctx context.Context) (gocache.Activation, error) {
	program, err := s.Program()
	if err != nil {
		return gocache.Activation{}, err
	}
	current := gocache.ReadActivation(program)
	switch current.ConfiguredProgram {
	case "":
		return current, nil
	case program:
	default:
		return current, fmt.Errorf("GOCACHEPROG in %s is %q, which Vrooli does not manage", current.EnvFile, current.ConfiguredProgram)
	}
	if err := s.goEnv(ctx, "-u", "GOCACHEPROG"); err != nil {
		return current, err
	}
	return gocache.ReadActivation(program), nil
}

// goEnv edits the go env file through cmd/go, its canonical writer. It runs
// outside any module with toolchain switching off, so the local toolchain
// writes the same file every go command reads.
func (s Service) goEnv(ctx context.Context, args ...string) error {
	home, err := s.home()
	if err != nil {
		return err
	}
	env := append(withoutKeys(os.Environ(), "GOTOOLCHAIN", "GOCACHEPROG"), "GOTOOLCHAIN=local")
	output, err := shell.CombinedOutput(shell.Spec{Context: ctx, Name: "go", Args: append([]string{"env"}, args...), Dir: home, Env: env})
	if err != nil {
		return fmt.Errorf("go env %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return nil
}

func (s Service) storeAndProgram(cache string) (gocache.Store, string, error) {
	program, err := s.Program()
	if err != nil {
		return gocache.Store{}, "", err
	}
	store, err := s.Store(cache)
	if err != nil {
		return gocache.Store{}, "", err
	}
	return store, program, nil
}

func (s Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func withoutKeys(environ []string, keys ...string) []string {
	out := make([]string, 0, len(environ))
next:
	for _, entry := range environ {
		for _, key := range keys {
			if strings.HasPrefix(entry, key+"=") {
				continue next
			}
		}
		out = append(out, entry)
	}
	return out
}
