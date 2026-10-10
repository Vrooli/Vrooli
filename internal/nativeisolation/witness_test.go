package nativeisolation

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/vrooli/api-core/effortauthority"
)

type fixtureWitnessReader struct {
	clock                 time.Time
	snapshot              witnessSnapshot
	readErr               error
	qualifyErr            error
	reads, qualifications int
	afterQualification    func()
}

func (r *fixtureWitnessReader) now() time.Time { return r.clock }
func (r *fixtureWitnessReader) read(context.Context, ProbeSpec) (witnessSnapshot, error) {
	r.reads++
	return r.snapshot, r.readErr
}
func (r *fixtureWitnessReader) qualify(context.Context, *effortauthority.NativeUIDIsolation) error {
	r.qualifications++
	if r.afterQualification != nil {
		r.afterQualification()
	}
	return r.qualifyErr
}
func witnessFixture(t *testing.T) (*Witness, *fixtureWitnessReader) {
	t.Helper()
	s, now := probeFixture()
	r := &fixtureWitnessReader{clock: now, snapshot: witnessSnapshot{PID: 42, StartTicks: "100", InvocationID: "initial", Cgroup: "/system.slice/" + s.unit(), ExecutableHash: s.HelperSHA256, ImageHash: s.Manifest.RootFSDigest, RootIdentity: "1:2", WorkspaceIdentity: "3:4", MountNamespace: "mnt:[123]", MountInfoHash: s.HelperSHA256, UnitPostureHash: s.HelperSHA256}}
	w, e := captureWitness(context.Background(), s, r)
	if e != nil {
		t.Fatal(e)
	}
	return w, r
}

func TestContinuingWitnessCopiesQualifiedIsolationAndPinsManifest(t *testing.T) {
	w, r := witnessFixture(t)
	q, e := w.Isolation(context.Background())
	if e != nil || q.NativeProbePID != 42 || r.qualifications < 2 {
		t.Fatal("continuing evidence not qualified", e)
	}
	q.ProtectedPaths[0] = "/mutated"
	q.Services["issuer"] = effortauthority.ProtectedProcess{}
	q2, e := w.Isolation(context.Background())
	if e != nil || q2.ProtectedPaths[0] == "/mutated" || q2.Services["issuer"].PID == 0 {
		t.Fatal("returned isolation aliases witness", e)
	}
	m, _ := cloneSpec(w.spec)
	m.Manifest.Isolation.NativeProbePID = 999
	if w.ValidateManifest(m.Manifest) != nil {
		t.Fatal("obsolete PID prevented live replacement")
	}
	for _, change := range []func(*Manifest){func(m *Manifest) { m.RootFSDigest = digest("foreign") }, func(m *Manifest) { m.NativeUID++ }, func(m *Manifest) { m.Isolation.ProtectedPaths[0] = "/foreign" }, func(m *Manifest) { m.Isolation.Services["issuer"] = effortauthority.ProtectedProcess{} }, func(m *Manifest) {
		for id, b := range m.Bindings {
			b.PolicyDigest = digest("foreign")
			m.Bindings[id] = b
		}
	}} {
		copy, _ := cloneSpec(w.spec)
		change(&copy.Manifest)
		if w.ValidateManifest(copy.Manifest) == nil {
			t.Fatal("foreign manifest attached")
		}
	}
}

func TestContinuingWitnessRefusesDeadRestartRacesAndDeadline(t *testing.T) {
	for _, name := range []string{"dead", "restarted", "PID-reuse", "image", "exe", "root", "cgroup", "workspace", "mount-namespace", "mount-view", "unit-posture", "rollback-race", "permission", "qualifier", "deadline", "clock-rollback", "race", "deadline-race"} {
		t.Run(name, func(t *testing.T) {
			w, r := witnessFixture(t)
			switch name {
			case "dead":
				r.snapshot.PID = 0
			case "restarted":
				r.snapshot.InvocationID = "restart"
			case "PID-reuse":
				r.snapshot.StartTicks = "101"
			case "image":
				r.snapshot.ImageHash = digest("changed")
			case "exe":
				r.snapshot.ExecutableHash = digest("changed")
			case "root":
				r.snapshot.RootIdentity = "2:3"
			case "cgroup":
				r.snapshot.Cgroup = "/foreign"
			case "workspace":
				r.snapshot.WorkspaceIdentity = "foreign"
			case "mount-namespace":
				r.snapshot.MountNamespace = "mnt:[124]"
			case "mount-view":
				r.snapshot.MountInfoHash = digest("foreign")
			case "unit-posture":
				r.snapshot.UnitPostureHash = "changed"
			case "rollback-race":
				r.afterQualification = func() { r.clock = w.capturedAt.Add(-time.Second) }
			case "permission":
				r.readErr = errors.New("permission denied")
			case "qualifier":
				r.qualifyErr = errors.New("native qualification refused")
			case "deadline":
				r.clock = w.spec.Deadline
			case "clock-rollback":
				r.clock = w.capturedAt.Add(-time.Second)
			case "race":
				r.afterQualification = func() { r.snapshot.StartTicks = "raced" }
			case "deadline-race":
				r.afterQualification = func() { r.clock = w.spec.Deadline }
			}
			if _, e := w.Isolation(context.Background()); e == nil {
				t.Fatal("stale witness accepted")
			}
		})
	}
	s, now := probeFixture()
	s.Purpose = FirstQualification
	s.Deadline = now.Add(time.Minute)
	if _, e := captureWitness(context.Background(), s, &fixtureWitnessReader{clock: now}); e == nil {
		t.Fatal("H1 receipt became continuing witness")
	}
	if (*Witness)(nil).Qualify(context.Background()) == nil {
		t.Fatal("missing witness accepted")
	}
}
