package runtimeapp

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/coreactivation"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

type rebaselineFixture struct {
	t          *testing.T
	home       string
	layout     coreactivation.Layout
	candidate  string
	identities map[string]*cliv1.CliVersion
	enrolled   bool
	updated    int64
	serving    bool
	calls      []string
	failAt     string
	confirmed  []RebaselinePlan
}

func newRebaselineFixture(t *testing.T) *rebaselineFixture {
	t.Helper()
	home := t.TempDir()
	layout, err := rootUpdateLayout(home)
	if err != nil {
		t.Fatal(err)
	}
	f := &rebaselineFixture{t: t, home: home, layout: layout, candidate: filepath.Join(t.TempDir(), "vrooli"), enrolled: true, updated: 1}
	worker, err := coreactivation.InstalledWatchdogPath(home)
	if err != nil {
		t.Fatal(err)
	}
	for path, content := range map[string]string{layout.Executable: "serving image", f.candidate: "candidate image", worker: "watchdog image", layout.UnitPath: "[Unit]\n"} {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	identity := func(build, state string) *cliv1.CliVersion {
		return &cliv1.CliVersion{CliVersion: "1", PlatformVersion: "1", BuildIdentity: build, RuntimeHomeContractSha256: strings.Repeat("e", 64), PersistentStateContractSha256: state}
	}
	f.identities = map[string]*cliv1.CliVersion{
		"serving image":   identity(strings.Repeat("a", 64), strings.Repeat("c", 64)),
		"candidate image": identity(strings.Repeat("b", 64), strings.Repeat("d", 64)),
	}
	return f
}

func fileDigest(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func (f *rebaselineFixture) effects() rebaselineEffects {
	step := func(name string) error {
		f.calls = append(f.calls, name)
		if f.failAt == name {
			return errors.New(name + " failed")
		}
		return nil
	}
	return rebaselineEffects{
		guard:        func(context.Context, string) error { return step("guard") },
		validateRole: func(string) error { return nil },
		imageDigest:  func(_ context.Context, path string) (string, error) { return fileDigest(path) },
		probeIdentity: func(_ context.Context, path string) (*cliv1.CliVersion, error) {
			data, err := os.ReadFile(path)
			if err != nil {
				return nil, err
			}
			if identity, ok := f.identities[string(data)]; ok {
				return identity, nil
			}
			return nil, errors.New("unknown image")
		},
		explain: func(context.Context, string, string) []string {
			return []string{"owner fixture: format fact SchemaVersion changed"}
		},
		buildWatchdog: func(_, output string) error {
			if err := step("build-watchdog"); err != nil {
				return err
			}
			return os.WriteFile(output, []byte("new watchdog"), 0o700)
		},
		probeWorker: func(context.Context, string, *cliv1.CliVersion) error { return step("qualify-watchdog") },
		enrollment: func(string) (coreactivation.Enrollment, bool, error) {
			return coreactivation.Enrollment{Enabled: f.enrolled, UpdatedUnixNano: f.updated}, f.updated > 0, nil
		},
		configure: func(_ context.Context, _ string, enabled bool) error {
			name := "disable"
			if enabled {
				name = "enroll"
			}
			if err := step(name); err != nil {
				return err
			}
			f.operatorSets(enabled)
			return nil
		},
		installRoot: func(source, destination string) error {
			if err := step("install"); err != nil {
				return err
			}
			data, err := os.ReadFile(source)
			if err != nil {
				return err
			}
			return os.WriteFile(destination, data, 0o700)
		},
		publishWatchdog: func(_ context.Context, _, image, digest string) error {
			if actual, err := fileDigest(image); err != nil || actual != digest {
				return errors.New("published watchdog is not the qualified image")
			}
			return step("publish-watchdog")
		},
		converge: func(context.Context, string, string, string) error {
			if err := step("converge"); err != nil {
				return err
			}
			f.serving = true
			return nil
		},
		waitServing: func(context.Context, string, string, string) error {
			if !f.serving {
				return errors.New("not serving the candidate")
			}
			return step("verify")
		},
		now: time.Now,
	}
}

// operatorSets writes a new enrollment record, as ConfigureRootUpdates does.
func (f *rebaselineFixture) operatorSets(enabled bool) {
	f.enrolled = enabled
	f.updated++
}

func (f *rebaselineFixture) request(answer bool) RebaselineRequest {
	return RebaselineRequest{Candidate: f.candidate, Source: "/source", Progress: &bytes.Buffer{}, Confirm: func(plan RebaselinePlan) (bool, error) {
		f.confirmed = append(f.confirmed, plan)
		return answer, nil
	}}
}

func TestRebaselineQualifiesBeforeActingAndComposesTheOwnersInOrder(t *testing.T) {
	f := newRebaselineFixture(t)
	result, err := f.effects().run(context.Background(), f.home, f.request(true))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(f.calls, ","); got != "guard,build-watchdog,qualify-watchdog,disable,install,publish-watchdog,converge,verify,enroll" {
		t.Fatalf("transition order = %s", got)
	}
	if len(f.confirmed) != 1 || f.confirmed[0].Changes[0] != "owner fixture: format fact SchemaVersion changed" || f.confirmed[0].CandidateContract != strings.Repeat("d", 64) {
		t.Fatalf("operator was not shown the owner-level change: %+v", f.confirmed)
	}
	if !result.Enrolled || result.Finished || result.Contract != strings.Repeat("d", 64) || result.RetainedDir != f.confirmed[0].RetainedDir {
		t.Fatalf("unexpected result: %+v", result)
	}
	for name, want := range map[string]string{"vrooli": "serving image", "vrooli-watchdog": "watchdog image", f.layout.UnitName: "[Unit]\n"} {
		data, err := os.ReadFile(filepath.Join(result.RetainedDir, name))
		if err != nil || string(data) != want {
			t.Fatalf("retained %s = %q, %v", name, data, err)
		}
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(result.RetainedDir), "pause.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("a completed transition left its pause marker")
	}
	if installed, _ := os.ReadFile(f.layout.Executable); string(installed) != "candidate image" {
		t.Fatal("candidate was not installed")
	}
}

func TestRebaselineInstallsConfirmedBytesWhenTheBuildPathIsReplaced(t *testing.T) {
	f := newRebaselineFixture(t)
	effects := f.effects()
	probe := effects.probeIdentity
	qualifiedPath := ""
	effects.probeIdentity = func(ctx context.Context, path string) (*cliv1.CliVersion, error) {
		if path != f.layout.Executable {
			qualifiedPath = path
		}
		return probe(ctx, path)
	}
	request := f.request(true)
	confirm := request.Confirm
	request.Confirm = func(plan RebaselinePlan) (bool, error) {
		// Another make build atomically publishes a new candidate while the
		// operator is reading the confirmation. It can even change contract.
		replacement := f.candidate + ".new"
		if err := os.WriteFile(replacement, []byte("a different contract image"), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(replacement, f.candidate); err != nil {
			t.Fatal(err)
		}
		return confirm(plan)
	}
	result, err := effects.run(context.Background(), f.home, request)
	if err != nil {
		t.Fatalf("a concurrent candidate build changed the confirmed transition: %v", err)
	}
	if qualifiedPath == f.candidate || qualifiedPath == "" {
		t.Fatalf("qualification used the mutable build path: %q", qualifiedPath)
	}
	installed, _ := os.ReadFile(f.layout.Executable)
	if string(installed) != "candidate image" || result.Contract != strings.Repeat("d", 64) || !result.Enrolled {
		t.Fatalf("installed bytes or contract differ from the confirmed image: bytes=%q result=%+v", installed, result)
	}
	if current, _ := os.ReadFile(f.candidate); string(current) != "a different contract image" {
		t.Fatal("transition overwrote the concurrent build")
	}
	if _, err := os.Stat(qualifiedPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("private candidate was not cleaned up: %v", err)
	}
}

func TestRebaselineRefusesPrivateCandidateChangesBeforePausingEnrollment(t *testing.T) {
	for _, when := range []string{"role", "identity", "watchdog", "explanation", "confirmation"} {
		t.Run(when, func(t *testing.T) {
			f := newRebaselineFixture(t)
			effects := f.effects()
			privatePath := ""
			mutate := func() {
				t.Helper()
				if err := os.WriteFile(privatePath, []byte("changed private candidate"), 0o700); err != nil {
					t.Fatal(err)
				}
			}
			role := effects.validateRole
			effects.validateRole = func(path string) error {
				privatePath = path
				if when == "role" {
					mutate()
				}
				return role(path)
			}
			probe := effects.probeIdentity
			effects.probeIdentity = func(ctx context.Context, path string) (*cliv1.CliVersion, error) {
				identity, err := probe(ctx, path)
				if when == "identity" && path == privatePath {
					mutate()
				}
				return identity, err
			}
			worker := effects.probeWorker
			effects.probeWorker = func(ctx context.Context, path string, identity *cliv1.CliVersion) error {
				if when == "watchdog" {
					mutate()
				}
				return worker(ctx, path, identity)
			}
			explain := effects.explain
			effects.explain = func(ctx context.Context, serving, candidate string) []string {
				if candidate != privatePath {
					t.Fatalf("explanation used unqualified bytes: %q", candidate)
				}
				if when == "explanation" {
					mutate()
				}
				return explain(ctx, serving, candidate)
			}
			request := f.request(true)
			confirm := request.Confirm
			request.Confirm = func(plan RebaselinePlan) (bool, error) {
				if when == "confirmation" {
					mutate()
				}
				return confirm(plan)
			}
			_, err := effects.run(context.Background(), f.home, request)
			if err == nil || !strings.Contains(err.Error(), "private candidate image") {
				t.Fatalf("changed private image was not refused: %v", err)
			}
			if !f.enrolled || f.updated != 1 {
				t.Fatal("private image changed before confirmation but enrollment was paused")
			}
			if installed, _ := os.ReadFile(f.layout.Executable); string(installed) != "serving image" {
				t.Fatalf("private image mismatch changed installed bytes: %q", installed)
			}
			for _, call := range f.calls {
				if call == "install" || call == "disable" || call == "publish-watchdog" || call == "converge" {
					t.Fatalf("private image mismatch reached activation: %v", f.calls)
				}
			}
			if when != "confirmation" && len(f.confirmed) != 0 {
				t.Fatal("operator was asked to confirm a changed private image")
			}
		})
	}
}

func TestRebaselineRefusesCandidateReplacementWhilePreparingItsSnapshot(t *testing.T) {
	f := newRebaselineFixture(t)
	effects := f.effects()
	digest := effects.imageDigest
	effects.imageDigest = func(ctx context.Context, path string) (string, error) {
		observed, err := digest(ctx, path)
		if path == f.candidate {
			if err := os.WriteFile(path, []byte("concurrent replacement"), 0o700); err != nil {
				t.Fatal(err)
			}
		}
		return observed, err
	}
	_, err := effects.run(context.Background(), f.home, f.request(true))
	if err == nil || !strings.Contains(err.Error(), "changed while preparing its private image") {
		t.Fatalf("mixed candidate observation was accepted: %v", err)
	}
	if len(f.confirmed) != 0 || strings.Join(f.calls, ",") != "guard" || !f.enrolled {
		t.Fatalf("raced snapshot was qualified or activated: confirmed=%d calls=%v enrolled=%t", len(f.confirmed), f.calls, f.enrolled)
	}
	if installed, _ := os.ReadFile(f.layout.Executable); string(installed) != "serving image" {
		t.Fatal("raced snapshot changed the installed root")
	}
}

func TestRebaselinePreservesInstalledDigestFailureDetails(t *testing.T) {
	for _, failure := range []string{"mismatch", "read-error"} {
		t.Run(failure, func(t *testing.T) {
			f := newRebaselineFixture(t)
			effects := f.effects()
			installed := false
			install := effects.installRoot
			effects.installRoot = func(source, destination string) error {
				if err := install(source, destination); err != nil {
					return err
				}
				installed = true
				if failure == "mismatch" {
					return os.WriteFile(destination, []byte("unexpected installed image"), 0o700)
				}
				return nil
			}
			readErr := errors.New("injected installed digest read failure")
			digest := effects.imageDigest
			effects.imageDigest = func(ctx context.Context, path string) (string, error) {
				if installed && path == f.layout.Executable && failure == "read-error" {
					return "", readErr
				}
				return digest(ctx, path)
			}
			_, err := effects.run(context.Background(), f.home, f.request(true))
			if err == nil || !strings.Contains(err.Error(), `stopped at "install the candidate root CLI"`) || !strings.Contains(err.Error(), "paused by an unfinished rebaseline") {
				t.Fatalf("post-install failure lost its live transition state: %v", err)
			}
			if failure == "read-error" && !errors.Is(err, readErr) {
				t.Fatalf("digest read cause was discarded: %v", err)
			}
			if failure == "mismatch" {
				expected, _ := fileDigest(f.candidate)
				actual, _ := fileDigest(f.layout.Executable)
				if !strings.Contains(err.Error(), expected) || !strings.Contains(err.Error(), actual) {
					t.Fatalf("digest mismatch omitted expected or observed digest: %v", err)
				}
			}
			if f.enrolled || strings.Join(f.calls, ",") != "guard,build-watchdog,qualify-watchdog,disable,install" {
				t.Fatalf("failed post-install verification continued activation or re-enrollment: %v", f.calls)
			}
		})
	}
}

func TestRebaselineRefusesWhatAutomaticActivationOrNoOneShouldCross(t *testing.T) {
	cases := map[string]struct {
		mutate func(*rebaselineFixture)
		want   string
		calls  string
	}{
		"equal contract": {func(f *rebaselineFixture) {
			f.identities["candidate image"].PersistentStateContractSha256 = strings.Repeat("c", 64)
		}, "make install", "guard,build-watchdog,qualify-watchdog"},
		"layout change": {func(f *rebaselineFixture) {
			f.identities["candidate image"].RuntimeHomeContractSha256 = strings.Repeat("f", 64)
		}, "runtime-home layout", "guard,build-watchdog,qualify-watchdog"},
		"unstamped candidate": {func(f *rebaselineFixture) { f.identities["candidate image"].PersistentStateContractSha256 = "" }, "no qualified persistent-state contract", "guard,build-watchdog,qualify-watchdog"},
		"active operation":    {func(f *rebaselineFixture) { f.failAt = "guard" }, "guard failed", "guard"},
		"watchdog mismatch":   {func(f *rebaselineFixture) { f.failAt = "qualify-watchdog" }, "does not match the candidate", "guard,build-watchdog,qualify-watchdog"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			f := newRebaselineFixture(t)
			tc.mutate(f)
			_, err := f.effects().run(context.Background(), f.home, f.request(true))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("refusal = %v, want %q", err, tc.want)
			}
			if len(f.confirmed) != 0 || strings.Join(f.calls, ",") != tc.calls {
				t.Fatalf("a refused transition asked or acted: confirmed=%d calls=%v", len(f.confirmed), f.calls)
			}
			if installed, _ := os.ReadFile(f.layout.Executable); string(installed) != "serving image" || !f.enrolled {
				t.Fatal("a refused transition changed the serving image or enrollment")
			}
		})
	}
}

func TestDeclinedRebaselineChangesNothing(t *testing.T) {
	f := newRebaselineFixture(t)
	_, err := f.effects().run(context.Background(), f.home, f.request(false))
	if err == nil || !strings.Contains(err.Error(), "nothing was changed") {
		t.Fatalf("decline = %v", err)
	}
	if strings.Join(f.calls, ",") != "guard,build-watchdog,qualify-watchdog" || !f.enrolled {
		t.Fatalf("declined transition acted: %v", f.calls)
	}
}

func TestStoppedRebaselineFinishesFromLiveStateWithoutAskingAgain(t *testing.T) {
	f := newRebaselineFixture(t)
	f.failAt = "converge"
	_, err := f.effects().run(context.Background(), f.home, f.request(true))
	if err == nil || !strings.Contains(err.Error(), `stopped at "restart the runtime supervisor on the new image"`) || !strings.Contains(err.Error(), "paused by an unfinished rebaseline") || !strings.Contains(err.Error(), "re-qualifies") {
		t.Fatalf("stopped transition did not explain its state: %v", err)
	}
	f.failAt, f.calls = "", nil
	result, err := f.effects().run(context.Background(), f.home, f.request(true))
	if err != nil {
		t.Fatal(err)
	}
	if len(f.confirmed) != 1 || !result.Finished || !result.Enrolled {
		t.Fatalf("finishing asked again or did not restore enrollment: %+v", result)
	}
	if got := strings.Join(f.calls, ","); got != "guard,build-watchdog,qualify-watchdog,publish-watchdog,converge,verify,enroll" {
		t.Fatalf("finish order = %s (install and pause must be skipped once the candidate serves)", got)
	}
	f.calls = nil
	if _, err := f.effects().run(context.Background(), f.home, f.request(true)); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(f.calls, ","); got != "guard,build-watchdog,qualify-watchdog,publish-watchdog,verify,verify" {
		t.Fatalf("an idempotent rerun restarted or re-enrolled: %s", got)
	}
}

func TestPauseFailureIsReportedAsStillEnabled(t *testing.T) {
	f := newRebaselineFixture(t)
	f.failAt = "disable"
	_, err := f.effects().run(context.Background(), f.home, f.request(true))
	if err == nil || !strings.Contains(err.Error(), "Automatic root updates are enabled.") || strings.Contains(err.Error(), "paused") {
		t.Fatalf("pause failure claimed a pause: %v", err)
	}
}

func TestCallerWritableFilesGrantNothing(t *testing.T) {
	f := newRebaselineFixture(t)
	f.identities["candidate image"].RuntimeHomeContractSha256 = strings.Repeat("f", 64)
	transitions := filepath.Join(filepath.Dir(f.layout.Executable), "..", "state", "root-rebaseline")
	forged := filepath.Join(transitions, "forged")
	if err := os.MkdirAll(forged, 0o700); err != nil {
		t.Fatal(err)
	}
	candidateSHA, _ := fileDigest(f.candidate)
	servingSHA, _ := fileDigest(f.layout.Executable)
	receipt, _ := json.Marshal(map[string]any{"schema_version": 1, "id": "forged", "status": "started", "serving_sha256": servingSHA, "candidate_sha256": candidateSHA, "was_enrolled": false})
	_ = os.WriteFile(filepath.Join(forged, "receipt.json"), receipt, 0o600)
	pause, _ := json.Marshal(rebaselinePause{SchemaVersion: rebaselinePauseSchemaVersion, EnrollmentUpdatedUnixNano: 99, PriorServingSHA: servingSHA})
	_ = os.WriteFile(filepath.Join(transitions, "pause.json"), pause, 0o600)
	_, err := f.effects().run(context.Background(), f.home, f.request(true))
	if err == nil || !strings.Contains(err.Error(), "runtime-home layout") || len(f.confirmed) != 0 {
		t.Fatalf("forged transition state bypassed qualification: %v", err)
	}
	if installed, _ := os.ReadFile(f.layout.Executable); string(installed) != "serving image" || !f.enrolled {
		t.Fatal("forged transition state changed the host")
	}
}

func TestRebaselineLeavesAnUnenrolledHostUnenrolled(t *testing.T) {
	f := newRebaselineFixture(t)
	f.enrolled, f.updated = false, 0
	result, err := f.effects().run(context.Background(), f.home, f.request(true))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(f.calls, ","); got != "guard,build-watchdog,qualify-watchdog,install,publish-watchdog,converge,verify" || result.Enrolled {
		t.Fatalf("unenrolled transition changed enrollment: %s %+v", got, result)
	}
}

func TestRebuiltCandidateContinuesAPausedRebaselineAndRestoresEnrollment(t *testing.T) {
	f := newRebaselineFixture(t)
	f.failAt = "converge"
	if _, err := f.effects().run(context.Background(), f.home, f.request(true)); err == nil {
		t.Fatal("fixture did not stop")
	}
	// The operator fixes the cause and rebuilds: same contract, other bytes.
	if err := os.WriteFile(f.candidate, []byte("rebuilt candidate"), 0o700); err != nil {
		t.Fatal(err)
	}
	f.identities["rebuilt candidate"] = &cliv1.CliVersion{CliVersion: "1", PlatformVersion: "1", BuildIdentity: strings.Repeat("9", 64), RuntimeHomeContractSha256: strings.Repeat("e", 64), PersistentStateContractSha256: strings.Repeat("d", 64)}
	f.failAt, f.calls = "", nil
	result, err := f.effects().run(context.Background(), f.home, f.request(true))
	if err != nil {
		t.Fatal(err)
	}
	if len(f.confirmed) != 1 || !result.Enrolled || !f.enrolled {
		t.Fatalf("continuing a paused rebaseline asked again or left enrollment off: confirmed=%d %+v", len(f.confirmed), result)
	}
	if got := strings.Join(f.calls, ","); got != "guard,build-watchdog,qualify-watchdog,install,publish-watchdog,converge,verify,enroll" {
		t.Fatalf("continuation order = %s", got)
	}
	if _, err := os.Stat(f.pausePath()); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("a completed continuation left its pause marker")
	}
}

func TestOperatorEnrollmentDecisionAfterAPauseIsNeverOverridden(t *testing.T) {
	f := newRebaselineFixture(t)
	f.failAt = "converge"
	if _, err := f.effects().run(context.Background(), f.home, f.request(true)); err == nil {
		t.Fatal("fixture did not stop")
	}
	// The operator decides to keep automatic updates off: a new record.
	f.operatorSets(false)
	f.failAt, f.calls = "", nil
	result, err := f.effects().run(context.Background(), f.home, f.request(true))
	if err != nil {
		t.Fatal(err)
	}
	if result.Enrolled || f.enrolled || strings.Contains(strings.Join(f.calls, ","), "enroll") {
		t.Fatalf("a superseded pause marker re-enabled enrollment the operator disabled: %v", f.calls)
	}
	if _, err := os.Stat(f.pausePath()); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("a superseded pause marker was kept")
	}
}

func TestSameContractCandidateRefusalStatesEnrollmentHonestly(t *testing.T) {
	f := newRebaselineFixture(t)
	f.identities["candidate image"].PersistentStateContractSha256 = strings.Repeat("c", 64)
	_, err := f.effects().run(context.Background(), f.home, f.request(true))
	if err == nil || !strings.Contains(err.Error(), "which keeps automatic rollback") || !strings.Contains(err.Error(), "are enabled") {
		t.Fatalf("enrolled refusal = %v", err)
	}
	f.operatorSets(false)
	_, err = f.effects().run(context.Background(), f.home, f.request(true))
	if err == nil || strings.Contains(err.Error(), "keeps automatic rollback") || !strings.Contains(err.Error(), "installs it directly") {
		t.Fatalf("unenrolled refusal promised automatic rollback: %v", err)
	}
}

func (f *rebaselineFixture) pausePath() string {
	return filepath.Join(filepath.Dir(f.layout.Executable), "..", "state", "root-rebaseline", "pause.json")
}
