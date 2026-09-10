package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"
)

// Registrar registers a data binding with the backup owner and returns the
// owner's reference for it. data-backup-manager is the production owner;
// scenarios self-register targets keyed by (owner, name), so registration is
// idempotent and safe to repeat on every capture.
type Registrar interface {
	Register(ctx context.Context, owner string, binding domain.DataBinding) (string, error)
}

// ArgvRunner runs one argv and returns its combined output.
type ArgvRunner func(ctx context.Context, tool string, args ...string) ([]byte, error)

// DBMRegistrar reaches data-backup-manager through its CLI argv
// (`data-backup-manager targets register --owner --name --kind --locator`).
// Unreachable is a typed backup_provider_unavailable, never a silent skip.
type DBMRegistrar struct {
	Runner  ArgvRunner
	Tool    string
	Timeout time.Duration
}

// DBMTool is the backup owner's CLI name.
const DBMTool = "data-backup-manager"

func (d DBMRegistrar) runner() ArgvRunner {
	if d.Runner != nil {
		return d.Runner
	}
	return func(ctx context.Context, tool string, args ...string) ([]byte, error) {
		timeout := d.Timeout
		if timeout <= 0 {
			timeout = 30 * time.Second
		}
		cctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		return exec.CommandContext(cctx, tool, args...).CombinedOutput() //nolint:gosec // argv built from typed fields, never a shell string
	}
}

func (d DBMRegistrar) tool() string {
	if d.Tool != "" {
		return d.Tool
	}
	return DBMTool
}

// dbmKind maps a binding kind/provider onto data-backup-manager's source kinds.
func dbmKind(b domain.DataBinding) (string, error) {
	switch {
	case b.Kind == domain.DataBindingKindSQL && b.Provider == domain.BackupProviderPostgres:
		return "postgres", nil
	case b.Kind == domain.DataBindingKindSQL && b.Provider == domain.BackupProviderSQLite:
		return "sqlite", nil
	case b.Kind == domain.DataBindingKindFiles:
		return "filesystem", nil
	case b.Kind == domain.DataBindingKindApplication:
		return "workspace-checkpoint", nil
	}
	return "", fmt.Errorf("binding %s: no data-backup-manager source kind for %s/%s", b.ID, b.Kind, b.Provider)
}

// Register implements Registrar.
func (d DBMRegistrar) Register(ctx context.Context, owner string, binding domain.DataBinding) (string, error) {
	kind, err := dbmKind(binding)
	if err != nil {
		return "", apierrors.New(apierrors.CodeInvalidRequest, err.Error())
	}
	args := []string{"targets", "register", "--owner", owner, "--name", binding.ID, "--kind", kind, "--locator", binding.Locator, "--json"}
	out, err := d.runner()(ctx, d.tool(), args...)
	if err != nil {
		return "", apierrors.New(apierrors.CodeBackupProviderUnavailable, "backup owner "+d.tool()+" is unreachable: "+err.Error()).
			WithDetail("binding", binding.ID).WithDetail("owner", d.tool()).WithDetail("output", strings.TrimSpace(string(out))).WithRetryable(true)
	}
	return parseTargetRef(out, binding.ID), nil
}

// parseTargetRef pulls the target id from the owner's JSON reply; when the
// reply carries none the reference is owner-scoped (owner:name).
func parseTargetRef(out []byte, name string) string {
	var reply struct {
		ID     string `json:"id"`
		Target struct {
			ID string `json:"id"`
		} `json:"target"`
	}
	if err := json.Unmarshal(out, &reply); err == nil {
		if reply.Target.ID != "" {
			return DBMTool + ":" + reply.Target.ID
		}
		if reply.ID != "" {
			return DBMTool + ":" + reply.ID
		}
	}
	return DBMTool + ":" + name
}

// StaticRegistrar is a test registrar returning fixed references.
type StaticRegistrar map[string]string

// Register implements Registrar.
func (s StaticRegistrar) Register(_ context.Context, _ string, binding domain.DataBinding) (string, error) {
	ref, ok := s[binding.ID]
	if !ok {
		return "", apierrors.New(apierrors.CodeBackupProviderUnavailable, "no backup owner for binding "+binding.ID).WithDetail("binding", binding.ID)
	}
	return ref, nil
}
