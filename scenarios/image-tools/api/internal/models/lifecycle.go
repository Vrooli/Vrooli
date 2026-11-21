package models

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"image-tools/internal/smoke"
)

// LifecycleState describes model preparation, independently of enablement and
// hardware fit. Ready always means the current artifact and runtime are proven.
type LifecycleState string

const (
	ModelAbsent     LifecycleState = "absent"
	ModelInstalling LifecycleState = "installing"
	ModelInstalled  LifecycleState = "installed"
	ModelBuilding   LifecycleState = "building"
	ModelSmoking    LifecycleState = "smoking"
	ModelReady      LifecycleState = "ready"
	ModelStale      LifecycleState = "stale"
	ModelFailed     LifecycleState = "failed"
)

// ModelLifecycle is shared by installation, admission, doctor and the picker.
// Reason is the actionable readiness code; state records the lifecycle phase.
type ModelLifecycle struct {
	ModelID      string         `json:"model_id"`
	State        LifecycleState `json:"state"`
	Reason       string         `json:"reason"`
	Detail       string         `json:"detail,omitempty"`
	ArtifactHash string         `json:"artifact_hash,omitempty"`
	RuntimeHash  string         `json:"runtime_hash,omitempty"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

func (s ModelLifecycle) Ready() bool { return s.State == ModelReady }

// Transition rejects out-of-order completion. Reinstallation is the explicit
// recovery path from every settled state; only a smoking/building/installed
// model may become ready. Evidence reconciliation uses Lifecycle below.
func (s ModelLifecycle) Transition(next LifecycleState) (ModelLifecycle, error) {
	valid := false
	switch next {
	case ModelInstalling:
		valid = s.State == "" || s.State == ModelAbsent || s.State == ModelInstalled || s.State == ModelReady || s.State == ModelStale || s.State == ModelFailed
	case ModelInstalled:
		valid = s.State == ModelInstalling || s.State == ModelBuilding || s.State == ModelSmoking
	case ModelBuilding:
		valid = s.State == ModelInstalled
	case ModelSmoking:
		valid = s.State == ModelInstalled || s.State == ModelBuilding
	case ModelReady:
		valid = s.State == ModelInstalled || s.State == ModelBuilding || s.State == ModelSmoking
	case ModelStale:
		valid = s.State == ModelReady || s.State == ModelInstalled || s.State == ModelSmoking
	case ModelFailed:
		valid = s.State == ModelInstalling || s.State == ModelInstalled || s.State == ModelBuilding || s.State == ModelSmoking
	case ModelAbsent:
		valid = s.State == "" || s.State == ModelAbsent || s.State == ModelInstalled || s.State == ModelReady || s.State == ModelStale || s.State == ModelFailed
	}
	if !valid {
		return s, fmt.Errorf("models: invalid lifecycle transition %q -> %q for %q", s.State, next, s.ModelID)
	}
	s.State = next
	return s, nil
}

func (s *InstallStore) lifecycle(ctx context.Context, id string) (ModelLifecycle, error) {
	r := ModelLifecycle{ModelID: id, State: ModelAbsent, Reason: "needs_model_install"}
	var at string
	err := s.db.QueryRowContext(ctx, `SELECT state,reason,detail,artifact_hash,runtime_hash,updated_at FROM model_lifecycle WHERE id=?`, id).
		Scan(&r.State, &r.Reason, &r.Detail, &r.ArtifactHash, &r.RuntimeHash, &at)
	if errors.Is(err, sql.ErrNoRows) {
		return r, nil
	}
	if err != nil {
		return r, fmt.Errorf("models: read lifecycle %q: %w", id, err)
	}
	r.UpdatedAt, err = time.Parse(time.RFC3339Nano, at)
	return r, err
}

func (s *InstallStore) saveLifecycle(ctx context.Context, r ModelLifecycle) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO model_lifecycle(id,state,reason,detail,artifact_hash,runtime_hash,updated_at)
VALUES(?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET state=excluded.state,reason=excluded.reason,detail=excluded.detail,
artifact_hash=excluded.artifact_hash,runtime_hash=excluded.runtime_hash,updated_at=excluded.updated_at`,
		r.ModelID, r.State, r.Reason, r.Detail, r.ArtifactHash, r.RuntimeHash, r.UpdatedAt.UTC().Format(time.RFC3339Nano))
	return err
}

func (in *Installer) advance(ctx context.Context, current *ModelLifecycle, next LifecycleState, reason string) error {
	r := *current
	if r.State != next {
		var err error
		r, err = current.Transition(next)
		if err != nil {
			return err
		}
	}
	r.Reason, r.UpdatedAt = reason, in.now()
	if in.State != nil {
		if err := in.State.saveLifecycle(ctx, r); err != nil {
			return err
		}
	}
	*current = r
	in.activeLifecycle.Store(r.ModelID, r)
	return nil
}

// Lifecycle reconciles durable preparation with the evidence that can become
// stale independently: artifact presence, derived output, runtime and smoke.
// A process restart cannot leave a model permanently "installing": transient
// stages are live only while this installer owns their execution.
func (in *Installer) Lifecycle(ctx context.Context, id string) (ModelLifecycle, error) {
	if active, ok := in.activeLifecycle.Load(id); ok {
		return active.(ModelLifecycle), nil
	}
	m, err := in.Resolve(ctx, id)
	if err != nil {
		return ModelLifecycle{}, err
	}
	return in.lifecycleForModel(ctx, m)
}

// lifecycleForModel reconciles lifecycle evidence for a model already resolved
// by the caller. Install already owns this catalog value; re-resolving it would
// repeat the custom-model store query without changing the evidence.
func (in *Installer) lifecycleForModel(ctx context.Context, m Model) (ModelLifecycle, error) {
	return in.lifecycleFromInstallRecord(ctx, m, nil)
}

// LifecycleFromInstallList reconciles a model against the install record a
// caller already bulk-loaded. A missing row is evidence too; this avoids one
// model_install query per picker candidate while keeping lifecycle state and
// artifact validation authoritative.
func (in *Installer) LifecycleFromInstallList(ctx context.Context, m Model, record InstallRecord, found bool) (ModelLifecycle, error) {
	if !found && !m.RequiresWeights() {
		record, _ = in.installedRecord(ctx, m.ID)
		found = true
	}
	if !found {
		record = InstallRecord{ID: m.ID}
	}
	return in.lifecycleFromInstallRecord(ctx, m, &record)
}

func (in *Installer) lifecycleFromInstallRecord(ctx context.Context, m Model, knownInstall *InstallRecord) (ModelLifecycle, error) {
	if active, ok := in.activeLifecycle.Load(m.ID); ok {
		return active.(ModelLifecycle), nil
	}
	r := ModelLifecycle{ModelID: m.ID, State: ModelAbsent, Reason: "needs_model_install"}
	if in.State != nil {
		var err error
		if r, err = in.State.lifecycle(ctx, m.ID); err != nil {
			return r, err
		}
	}
	observed := in.observeLifecycle(ctx, m, r, knownInstall)
	if r.State == ModelReady && observed.State == ModelStale {
		stale, err := r.Transition(ModelStale)
		if err != nil {
			return ModelLifecycle{}, err
		}
		stale.Reason = observed.Reason
		stale.Detail = observed.Detail
		stale.ArtifactHash = observed.ArtifactHash
		stale.RuntimeHash = observed.RuntimeHash
		stale.UpdatedAt = in.now()
		if in.State != nil {
			if err := in.State.saveLifecycle(ctx, stale); err != nil {
				return ModelLifecycle{}, err
			}
		}
		observed = stale
	}
	return observed, nil
}

func (in *Installer) observeLifecycle(ctx context.Context, m Model, r ModelLifecycle, knownInstall *InstallRecord) ModelLifecycle {
	set := func(state LifecycleState, reason string) ModelLifecycle {
		r.State, r.Reason = state, reason
		return r
	}
	var installed InstallRecord
	var ok bool
	if knownInstall != nil {
		installed = *knownInstall
		ok = installed.Installed && pathExists(installed.Path)
	} else {
		installed, ok = in.installedRecord(ctx, m.ID)
	}
	if !ok {
		if r.State == ModelFailed {
			return r
		}
		return set(ModelAbsent, "needs_model_install")
	}
	artifactHash := installed.Checksum
	if m.BuildProfile != nil {
		var ready bool
		artifactHash, ready = buildOutputReady(m.BuildProfile, installed.Path)
		if !ready {
			return set(ModelInstalled, "build_not_ready")
		}
	}
	// Backend preparation writes this state. Read paths never launch import or
	// device probes: health and doctor consume evidence, and explicit provision
	// revalidates it after a backend repair.
	if r.Reason == "needs_backend" {
		return set(ModelInstalled, "needs_backend")
	}
	dir := installed.Path
	if dir == "" {
		dir = in.ModelDir(m.ID)
	}
	r.ArtifactHash = artifactHash
	if in.Smoke != nil {
		r.RuntimeHash = in.Smoke.LockHash
	}
	st := in.smokeStatusFor(m, dir, artifactHash)
	if !st.Applicable {
		if m.Backend == "python-sidecar" || m.Backend == BackendDiffusers || m.Backend == "onnxruntime" || m.Backend == "sam2-transformers" || m.Backend == "triposr" {
			return set(ModelInstalled, "smoke_unsupported")
		}
		return set(ModelReady, "ready")
	}
	if !st.EnvProvisioned {
		return set(ModelInstalled, "env_not_provisioned")
	}
	if !st.HasVerdict {
		if _, exists := smoke.ReadVerdict(in.smokeEvidenceDir(m, dir)); exists {
			return set(ModelStale, "smoke_pending")
		}
		return set(ModelInstalled, "smoke_pending")
	}
	if !st.Verdict.Pass {
		r.Detail = st.Verdict.Reason
		return set(ModelFailed, "smoke_failed")
	}
	r.Detail = ""
	return set(ModelReady, "ready")
}
