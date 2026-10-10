package nativeisolation

import (
	"context"
	"encoding/json"
	"time"

	"github.com/vrooli/api-core/effortauthority"
)

// Witness is captured explicitly by runtime ownership from live readonly
// evidence. It is not a JSON receipt or an HTTP-supplied PID. Exit, restart or
// PID reuse requires explicit owner recapture and a new immutable runtime.
type Witness struct {
	spec       ProbeSpec
	first      witnessSnapshot
	reader     witnessReader
	capturedAt time.Time
}

type witnessSnapshot struct {
	PID                                                                       int
	StartTicks, InvocationID, Cgroup, ExecutableHash, ImageHash, RootIdentity string
	MountNamespace, MountInfoHash, WorkspaceIdentity, UnitPostureHash         string
}

// The reader and effect seam are private. Production construction always uses
// exact pinned systemctl/proc/image reads and the unchanged native qualifier.
type witnessReader interface {
	read(context.Context, ProbeSpec) (witnessSnapshot, error)
	qualify(context.Context, *effortauthority.NativeUIDIsolation) error
	now() time.Time
}

func cloneSpec(s ProbeSpec) (ProbeSpec, error) {
	raw, e := json.Marshal(s)
	if e != nil {
		return ProbeSpec{}, ErrRefused
	}
	var copy ProbeSpec
	if json.Unmarshal(raw, &copy) != nil {
		return ProbeSpec{}, ErrRefused
	}
	return copy, nil
}

func CaptureWitness(ctx context.Context, spec ProbeSpec) (*Witness, error) {
	return captureWitness(ctx, spec, newWitnessReader())
}

func captureWitness(ctx context.Context, spec ProbeSpec, reader witnessReader) (*Witness, error) {
	s, e := cloneSpec(spec)
	if e != nil || reader == nil || s.Purpose != ContinuingWitness || s.validate(ctx, reader.now()) != nil {
		return nil, ErrRefused
	}
	a, e := reader.read(ctx, s)
	if e != nil || a.PID <= 0 || a.StartTicks == "" || a.InvocationID == "" || a.Cgroup == "" || a.RootIdentity == "" || a.MountNamespace == "" || a.MountInfoHash == "" || a.WorkspaceIdentity == "" || a.UnitPostureHash == "" || a.ExecutableHash != s.HelperSHA256 || a.ImageHash != s.Manifest.RootFSDigest {
		return nil, ErrRefused
	}
	w := &Witness{spec: s, first: a, reader: reader, capturedAt: reader.now()}
	if _, e = w.Isolation(ctx); e != nil {
		return nil, ErrRefused
	}
	return w, nil
}

// ValidateManifest compares all installed policy/image/UID/service/path data,
// ignoring only the obsolete probe PID that this witness replaces dynamically.
func (w *Witness) ValidateManifest(m Manifest) error {
	if w == nil || w.reader == nil || m.Isolation == nil {
		return ErrRefused
	}
	copy, e := cloneSpec(ProbeSpec{Manifest: m})
	if e != nil {
		return ErrRefused
	}
	expected, e := cloneSpec(w.spec)
	if e != nil {
		return ErrRefused
	}
	copy.Manifest.Isolation.NativeProbePID = 0
	expected.Manifest.Isolation.NativeProbePID = 0
	if digest(copy.Manifest) != digest(expected.Manifest) {
		return ErrRefused
	}
	return nil
}

// Isolation revalidates the exact continuing unit/process around the existing
// production qualifier. It returns an independent copy with its live PID.
func (w *Witness) Isolation(ctx context.Context) (*effortauthority.NativeUIDIsolation, error) {
	if w == nil || w.reader == nil || w.reader.now().Before(w.capturedAt) || w.spec.validate(ctx, w.reader.now()) != nil {
		return nil, ErrRefused
	}
	a, e := w.reader.read(ctx, w.spec)
	if e != nil || a != w.first {
		return nil, ErrRefused
	}
	s, e := cloneSpec(w.spec)
	if e != nil {
		return nil, ErrRefused
	}
	q := s.Manifest.Isolation
	q.NativeProbePID = a.PID
	if w.reader.qualify(ctx, q) != nil {
		return nil, ErrRefused
	}
	b, e := w.reader.read(ctx, w.spec)
	if e != nil || b != a || w.reader.now().Before(w.capturedAt) || w.spec.validate(ctx, w.reader.now()) != nil {
		return nil, ErrRefused
	}
	return q, nil
}

func (w *Witness) Qualify(ctx context.Context) error {
	_, e := w.Isolation(ctx)
	return e
}

// CheckPolicy binds the continuing witness to the exact immutable startup
// policy descriptor without host reads, callbacks, credentials or other effects.
func (w *Witness) CheckPolicy(p effortauthority.Policy) error {
	if w == nil || w.reader == nil || w.spec.Purpose != ContinuingWitness {
		return ErrRefused
	}
	return w.spec.Manifest.CheckPolicy(p)
}
