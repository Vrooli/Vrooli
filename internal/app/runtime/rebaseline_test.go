package runtimeapp

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/cliinstall"
	"github.com/vrooli/vrooli/internal/coreactivation"
	"github.com/vrooli/vrooli/internal/rebaselineprogress"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

type rebaselineFixture struct {
	t          *testing.T
	home       string
	layout     coreactivation.Layout
	candidate  string
	identities map[string]*cliv1.CliVersion
	enrolled   bool
	paused     bool
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

func TestRebaselineRefusesPendingOrdinaryBundle(t *testing.T) {
	f := newRebaselineFixture(t)
	path, _, err := buildinfo.PreparedBundlePaths(f.home)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	intent, err := json.Marshal(buildinfo.PreparedBundleIntent{Version: 1, Manifest: strings.Repeat("a", 64), Root: strings.Repeat("b", 64)})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, intent, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = f.effects().run(context.Background(), f.home, f.request(true))
	if err == nil || !strings.Contains(err.Error(), "ordinary bundle install is incomplete") {
		t.Fatalf("expected pending bundle refusal, got %v", err)
	}
	if len(f.calls) != 0 || len(f.confirmed) != 0 {
		t.Fatalf("rebaseline proceeded past fence: calls=%v confirmations=%v", f.calls, f.confirmed)
	}
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
		guardLocked:  func(context.Context, string) error { return nil },
		lockRoot:     func(context.Context, string) (func(), error) { return func() {}, nil },
		lockWatchdog: func(context.Context, string) (func(), error) { return func() {}, nil },
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
		selectWatchdog: func(_ string, _ string, _ *rebaselineprogress.Progress, _, snapshot string) (string, string, error) {
			return snapshot, "", nil
		},
		buildWatchdog: func(_, output string) error {
			if err := step("build-watchdog"); err != nil {
				return err
			}
			return os.WriteFile(output, []byte("new watchdog"), 0o700)
		},
		probeWorker: func(context.Context, string, *cliv1.CliVersion, string) error { return step("qualify-watchdog") },
		enrollment: func(string) (coreactivation.Enrollment, bool, error) {
			return coreactivation.Enrollment{Enabled: f.enrolled, RebaselinePaused: f.paused, UpdatedUnixNano: f.updated}, f.updated > 0, nil
		},
		compareEnrollment: func(_ context.Context, _ string, expected coreactivation.Enrollment, enabled, paused bool) error {
			if expected.Enabled != f.enrolled || expected.UpdatedUnixNano != f.updated || expected.RebaselinePaused != f.paused {
				return errEnrollmentSuperseded
			}
			name := "disable"
			if enabled {
				name = "enroll"
			}
			if err := step(name); err != nil {
				return err
			}
			f.operatorSets(enabled)
			f.paused = paused
			return nil
		},
		installRoot: func(source, destination, expected string) error {
			if err := step("install"); err != nil {
				return err
			}
			current, err := fileDigest(destination)
			if err != nil || current != expected {
				return errors.New("installed image differs from expected")
			}
			data, err := os.ReadFile(source)
			if err != nil {
				return err
			}
			return os.WriteFile(destination, data, 0o700)
		},
		retainBundle: func(_ string, source, _ string) (string, error) { return source, nil },
		removeBundle: func(string, string) error { return nil },
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
	f.paused = false
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
	stateDir, err := repocontract.RuntimeHomeEntryPath(f.home, repocontract.HomeKeyState)
	if err != nil {
		t.Fatal(err)
	}
	attemptFiles, err := os.ReadDir(filepath.Join(stateDir, "root-rebaseline", "attempts"))
	if err != nil {
		t.Fatal(err)
	}
	var startedStages []string
	for _, file := range attemptFiles {
		data, err := os.ReadFile(filepath.Join(stateDir, "root-rebaseline", "attempts", file.Name()))
		if err != nil {
			t.Fatal(err)
		}
		var record struct {
			Events []struct {
				Stage  string `json:"stage"`
				Reason string `json:"reason"`
			} `json:"events"`
		}
		if err := json.Unmarshal(data, &record); err != nil {
			t.Fatal(err)
		}
		for _, event := range record.Events {
			if event.Reason == "started" && event.Stage != "confirmation" {
				startedStages = append(startedStages, event.Stage)
			}
		}
	}
	wantStages := []string{"pause", "fence", "install", "watchdog", "supervisor", "verify", "companions", "restore"}
	if !slices.Equal(startedStages, wantStages) {
		t.Fatalf("stage labels = %v, want %v; effect calls = %v", startedStages, wantStages, f.calls)
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
			effects.probeWorker = func(ctx context.Context, path string, identity *cliv1.CliVersion, rootImage string) error {
				if when == "watchdog" {
					mutate()
				}
				return worker(ctx, path, identity, rootImage)
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
			effects.installRoot = func(source, destination, expected string) error {
				if err := install(source, destination, expected); err != nil {
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
		}, "make install", "guard"},
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
			if name == "equal contract" && !errors.Is(err, ErrRebaselineNotRequired) {
				t.Fatalf("same-contract result must route to normal activation: %v", err)
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
	if got := strings.Join(f.calls, ","); got != "guard,qualify-watchdog,publish-watchdog,converge,verify,enroll" {
		t.Fatalf("finish order = %s (install and pause must be skipped once the candidate serves)", got)
	}
	f.calls = nil
	if _, err := f.effects().run(context.Background(), f.home, f.request(true)); !errors.Is(err, ErrRebaselineNotRequired) {
		t.Fatalf("already-current enrolled image should use normal activation: %v", err)
	}
	if got := strings.Join(f.calls, ","); got != "guard" {
		t.Fatalf("an idempotent rerun performed recovery: %s", got)
	}
}

func TestAlreadyServingWithoutRecoveryEvidenceUsesNormalInstall(t *testing.T) {
	for _, tc := range []struct {
		name    string
		updated int64
	}{{"absent enrollment", 0}, {"disabled enrollment", 1}} {
		t.Run(tc.name, func(t *testing.T) {
			f := newRebaselineFixture(t)
			f.enrolled, f.updated = false, tc.updated
			if err := os.WriteFile(f.layout.Executable, []byte("candidate image"), 0700); err != nil {
				t.Fatal(err)
			}
			_, err := f.effects().run(context.Background(), f.home, f.request(true))
			if !errors.Is(err, ErrRebaselineNotRequired) {
				t.Fatalf("already current: %v", err)
			}
			if got := strings.Join(f.calls, ","); got != "guard" {
				t.Fatalf("unexpected recovery effects: %s", got)
			}
			if len(f.confirmed) != 0 {
				t.Fatal("requested unnecessary confirmation")
			}
		})
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

func TestRebuiltCandidateCannotReplaceUnfinishedRebaseline(t *testing.T) {
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
	_, err := f.effects().run(context.Background(), f.home, f.request(true))
	if err == nil || !strings.Contains(err.Error(), "resume the original") || f.enrolled {
		t.Fatalf("changed candidate crossed unfinished transition: %v", err)
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

func TestRebaselineRefusesServingImageChangedAfterConfirmation(t *testing.T) {
	f := newRebaselineFixture(t)
	effects := f.effects()
	configure := effects.compareEnrollment
	effects.compareEnrollment = func(ctx context.Context, home string, expected coreactivation.Enrollment, enabled, paused bool) error {
		if err := configure(ctx, home, expected, enabled, paused); err != nil {
			return err
		}
		if !enabled {
			return os.WriteFile(f.layout.Executable, []byte("concurrent root image"), 0700)
		}
		return nil
	}
	_, err := effects.run(context.Background(), f.home, f.request(true))
	if err == nil || !strings.Contains(err.Error(), "installed image differs from expected") {
		t.Fatalf("expected stale image refusal, got %v", err)
	}
	data, err := os.ReadFile(f.layout.Executable)
	if err != nil || string(data) != "concurrent root image" {
		t.Fatalf("concurrent image overwritten: %q, %v", data, err)
	}
}

func TestRebaselineSerializesConcurrentTransitions(t *testing.T) {
	f := newRebaselineFixture(t)
	entered := make(chan struct{})
	release := make(chan struct{})
	request := f.request(false)
	request.Confirm = func(RebaselinePlan) (bool, error) { close(entered); <-release; return false, nil }
	done := make(chan error, 1)
	go func() { _, err := f.effects().run(context.Background(), f.home, request); done <- err }()
	<-entered
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := f.effects().run(ctx, f.home, f.request(true))
	close(release)
	<-done
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("second transition was not serialized: %v", err)
	}
}

func TestRebaselineRejectsWatchdogReplacedDuringQualification(t *testing.T) {
	f := newRebaselineFixture(t)
	effects := f.effects()
	effects.probeWorker = func(_ context.Context, image string, _ *cliv1.CliVersion, _ string) error {
		return os.WriteFile(image, []byte("different watchdog"), 0700)
	}
	_, err := effects.run(context.Background(), f.home, f.request(true))
	if err == nil || !strings.Contains(err.Error(), "watchdog image changed during qualification") {
		t.Fatalf("expected watchdog mismatch, got %v", err)
	}
	if len(f.confirmed) != 0 || !f.enrolled {
		t.Fatal("changed watchdog was admitted")
	}
}

func TestRebaselineDoesNotRestoreEnrollmentFromMatchingLegacyMarker(t *testing.T) {
	f := newRebaselineFixture(t)
	f.operatorSets(false)
	data, err := json.Marshal(rebaselinePause{SchemaVersion: rebaselinePauseSchemaVersion, EnrollmentUpdatedUnixNano: f.updated})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(f.pausePath()), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f.pausePath(), data, 0600); err != nil {
		t.Fatal(err)
	}
	_, err = f.effects().run(context.Background(), f.home, f.request(true))
	if err == nil || !strings.Contains(err.Error(), "explicit enrollment decision") || f.enrolled {
		t.Fatalf("legacy marker authorized enrollment: %v", err)
	}
}

func TestRebaselinePublishesCompanionsUnderRootFence(t *testing.T) {
	f := newRebaselineFixture(t)
	e := f.effects()
	locked := false
	e.lockRoot = func(context.Context, string) (func(), error) {
		if locked {
			t.Fatal("root lock acquired twice")
		}
		locked = true
		return func() { locked = false }, nil
	}
	request := f.request(true)
	request.BundleSHA = strings.Repeat("a", 64)
	called := false
	var admitted RebaselineAdmission
	request.BeforeEffects = func(_ context.Context, admission RebaselineAdmission) error {
		admitted = admission
		return nil
	}
	request.AfterActivation = func(_ context.Context, recovery RebaselineRecovery) error {
		called = true
		if !locked {
			t.Fatal("companions ran outside root fence")
		}
		if recovery.Admission != admitted || recovery.BundleDir != filepath.Dir(request.Candidate) {
			t.Fatalf("recovery callbacks did not share the qualified value: admitted=%+v activated=%+v", admitted, recovery)
		}
		actual, err := fileDigest(f.layout.Executable)
		if err != nil || actual != recovery.Admission.CandidateSHA {
			t.Fatalf("wrong serving root %s: %v", actual, err)
		}
		return nil
	}
	if _, err := e.run(context.Background(), f.home, request); err != nil {
		t.Fatal(err)
	}
	if !called || locked {
		t.Fatalf("callback=%v lock=%v", called, locked)
	}
}

func TestCompanionFailureKeepsRootPausedAndNamesResume(t *testing.T) {
	f := newRebaselineFixture(t)
	e := f.effects()
	request := f.request(true)
	request.BeforeEffects = func(context.Context, RebaselineAdmission) error { return nil }
	request.AfterActivation = func(context.Context, RebaselineRecovery) error { return errors.New("companion unavailable") }
	if _, err := e.run(context.Background(), f.home, request); err == nil || !strings.Contains(err.Error(), "runtime update rebaseline") {
		t.Fatalf("failure lacked resume guidance: %v", err)
	}
	if f.enrolled || !f.paused {
		t.Fatalf("enrollment unexpectedly restored: enabled=%v paused=%v", f.enrolled, f.paused)
	}
	request.AfterActivation = func(context.Context, RebaselineRecovery) error { return nil }
	if _, err := e.run(context.Background(), f.home, request); err != nil {
		t.Fatalf("resume failed: %v", err)
	}
	if !f.enrolled || f.paused {
		t.Fatalf("resume did not restore enrollment: enabled=%v paused=%v", f.enrolled, f.paused)
	}
}

func TestPreparedInteractiveFailureGuidanceRunsRetainedCandidate(t *testing.T) {
	f := newRebaselineFixture(t)
	// Shell metacharacters must remain literal in both the executable and its argument.
	f.candidate = filepath.Join(filepath.Dir(f.candidate), "candidate ' $HOME `false`", "vrooli")
	if err := os.MkdirAll(filepath.Dir(f.candidate), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f.candidate, []byte("candidate image"), 0700); err != nil {
		t.Fatal(err)
	}
	request := f.request(true)
	request.AfterActivation = func(context.Context, RebaselineRecovery) error { return errors.New("companion unavailable") }
	_, err := f.effects().run(context.Background(), f.home, request)
	if err == nil {
		t.Fatal("expected interrupted transition")
	}
	message := err.Error()
	start := strings.Index(message, "rerun `")
	end := strings.LastIndex(message, "`; it re-qualifies")
	if start < 0 || end < start {
		t.Fatalf("missing recovery command: %s", message)
	}
	command := message[start+len("rerun `") : end]
	// Replace the fixture image only after the failure, so executing the printed
	// command can safely record exactly what a shell would invoke.
	output := filepath.Join(t.TempDir(), "arguments")
	if err := os.WriteFile(f.candidate, []byte("#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$RECOVERY_ARGS\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("sh", "-c", command)
	cmd.Env = append(os.Environ(), "RECOVERY_ARGS="+output)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("recovery command failed: %v: %s\n%s", err, out, command)
	}
	got, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	want := "runtime\nupdate\nrebaseline\n" + f.candidate + "\n"
	if string(got) != want {
		t.Fatalf("recovery arguments = %q, want %q", got, want)
	}
}

func TestRebaselineWaitsForOrdinaryBundleWorkflow(t *testing.T) {
	f := newRebaselineFixture(t)
	release, err := buildinfo.AcquireRootBundleWorkflow(context.Background(), f.home)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if _, err := f.effects().run(ctx, f.home, f.request(true)); err == nil {
		t.Fatal("transition passed active bundle workflow")
	}
	if len(f.calls) != 0 {
		t.Fatalf("transition observed or changed owners while ordinary bundle held lock: %v", f.calls)
	}
}

func TestRebaselineRejectsCandidateReplacedAfterBundleVerification(t *testing.T) {
	f := newRebaselineFixture(t)
	approved, err := fileDigest(f.candidate)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f.candidate, []byte("replacement"), 0700); err != nil {
		t.Fatal(err)
	}
	req := f.request(true)
	req.ExpectedCandidateSHA = approved
	if _, err := f.effects().run(context.Background(), f.home, req); err == nil || !strings.Contains(err.Error(), "differs from prepared bundle") {
		t.Fatalf("replacement accepted: %v", err)
	}
	if got := strings.Join(f.calls, ","); got != "guard" {
		t.Fatalf("replaced candidate reached effects: %s", got)
	}
}

func TestSameContractDoesNotRequireWatchdogBuild(t *testing.T) {
	f := newRebaselineFixture(t)
	f.identities["candidate image"].PersistentStateContractSha256 = strings.Repeat("c", 64)
	f.failAt = "build-watchdog"
	if _, err := f.effects().run(context.Background(), f.home, f.request(true)); !errors.Is(err, ErrRebaselineNotRequired) {
		t.Fatalf("matching contract did not take normal path: %v", err)
	}
	if got := strings.Join(f.calls, ","); got != "guard" {
		t.Fatalf("unexpected effects: %s", got)
	}
}

func TestTransitionCallerGateAppliesToTransitionAndRecoveryOnly(t *testing.T) {
	denied := errors.New("caller denied")
	for _, mode := range []string{"same contract", "transition", "recovery"} {
		t.Run(mode, func(t *testing.T) {
			f := newRebaselineFixture(t)
			if mode == "same contract" {
				f.identities["candidate image"].PersistentStateContractSha256 = strings.Repeat("c", 64)
			}
			if mode == "recovery" {
				f.failAt = "converge"
				if _, err := f.effects().run(context.Background(), f.home, f.request(true)); err == nil {
					t.Fatal("expected interrupted transition")
				}
				f.failAt, f.calls = "", nil
			}
			req := f.request(true)
			req.RequireTransitionCaller = func() error { return denied }
			_, err := f.effects().run(context.Background(), f.home, req)
			if mode == "same contract" {
				if !errors.Is(err, ErrRebaselineNotRequired) {
					t.Fatalf("ordinary install: %v", err)
				}
			} else if !errors.Is(err, denied) {
				t.Fatalf("gate bypassed: %v", err)
			}
			if got := strings.Join(f.calls, ","); got != "guard" {
				t.Fatalf("effects before gate: %s", got)
			}
		})
	}
}

func TestDisabledEnrollmentProgressRecoversAndCompletedRetrySkipsEffects(t *testing.T) {
	for _, failure := range []string{"install", "publish-watchdog", "converge", "verify"} {
		t.Run(failure, func(t *testing.T) {
			f := newRebaselineFixture(t)
			f.operatorSets(false)
			f.failAt = failure
			req := f.request(true)
			req.BundleSHA = strings.Repeat("b", 64)
			req.AfterActivation = func(context.Context, RebaselineRecovery) error { return nil }
			if _, err := f.effects().run(context.Background(), f.home, req); err == nil {
				t.Fatal("expected interrupted transition")
			}
			path := filepath.Join(f.home, ".vrooli", "state", "root-rebaseline", "progress.json")
			p, err := rebaselineprogress.Read(path)
			if err != nil || p == nil || p.Complete || p.Bundle != req.BundleSHA {
				t.Fatalf("interrupted progress: %+v %v", p, err)
			}
			f.failAt, f.calls = "", nil
			if _, err := f.effects().run(context.Background(), f.home, req); err != nil {
				t.Fatalf("resume: %v", err)
			}
			p, err = rebaselineprogress.Read(path)
			if err != nil || p == nil || !p.Complete || p.Watchdog == "" {
				t.Fatalf("completed progress: %+v %v", p, err)
			}
			f.calls = nil
			// A new companion bundle does not undo completion of the same root.
			req.BundleSHA = strings.Repeat("c", 64)
			if _, err := f.effects().run(context.Background(), f.home, req); !errors.Is(err, ErrRebaselineNotRequired) {
				t.Fatalf("completed retry: %v", err)
			}
			if got := strings.Join(f.calls, ","); got != "guard" {
				t.Fatalf("completed retry repeated effects: %s", got)
			}
		})
	}
}

func TestPreparedRecoveryCannotBeCompletedByAnotherPath(t *testing.T) {
	f := newRebaselineFixture(t)
	req := f.request(true)
	req.BundleSHA = strings.Repeat("b", 64)
	req.AfterActivation = func(context.Context, RebaselineRecovery) error { return errors.New("crash before companions") }
	if _, err := f.effects().run(context.Background(), f.home, req); err == nil {
		t.Fatal("expected interrupted publication")
	}
	path := filepath.Join(f.home, ".vrooli", "state", "root-rebaseline", "progress.json")
	for _, retry := range []RebaselineRequest{f.request(true), func() RebaselineRequest { r := f.request(true); r.BundleSHA = req.BundleSHA; return r }(), func() RebaselineRequest { r := req; r.BundleSHA = strings.Repeat("c", 64); return r }(), func() RebaselineRequest { r := req; r.Source = t.TempDir(); return r }()} {
		if _, err := f.effects().run(context.Background(), f.home, retry); err == nil {
			t.Fatal("different or unprepared recovery accepted")
		}
		p, err := rebaselineprogress.Read(path)
		if err != nil || p == nil || p.Complete || p.Bundle != req.BundleSHA {
			t.Fatalf("recovery evidence changed: %+v %v", p, err)
		}
	}
	pending := filepath.Join(filepath.Dir(path), "watchdog-pending")
	original, err := os.ReadFile(pending)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pending, []byte("different watchdog"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := f.effects().run(context.Background(), f.home, req); err == nil || !strings.Contains(err.Error(), "watchdog differs") {
		t.Fatalf("changed watchdog accepted: %v", err)
	}
	if err := os.WriteFile(pending, original, 0700); err != nil {
		t.Fatal(err)
	}
	req.AfterActivation = func(context.Context, RebaselineRecovery) error { return nil }
	if _, err := f.effects().run(context.Background(), f.home, req); err != nil {
		t.Fatalf("original recovery failed: %v", err)
	}
}

func TestRecoveryResumeGuidanceRequiresProgressAndAuthority(t *testing.T) {
	original := "/snapshot/candidate"
	bundle := "/snapshot/retained"
	prepared := RebaselineRequest{AfterActivation: func(context.Context, RebaselineRecovery) error { return nil }}
	tests := []struct {
		name, got, want string
	}{
		{"prepared progress write failed", preparedRebaselineResumeCommand(prepared, original, bundle, false), "'/snapshot/candidate' runtime update rebaseline '/snapshot/candidate'"},
		{"prepared recovery is ready", preparedRebaselineResumeCommand(prepared, original, bundle, true), "'/snapshot/retained/vrooli' runtime update rebaseline '/snapshot/retained/vrooli'"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Fatalf("resume command = %q, want %q", tc.got, tc.want)
			}
		})
	}

	// A later mutation failure is resumable from the retained, progress-bound
	// Hub bundle. A pre-progress failure above must continue from the snapshot.
	f := newRebaselineFixture(t)
	f.failAt = "converge"
	request := f.request(true)
	request.BundleSHA = strings.Repeat("a", 64)
	request.BeforeEffects = func(context.Context, RebaselineAdmission) error { return nil }
	request.AfterActivation = func(context.Context, RebaselineRecovery) error { return nil }
	retained := filepath.Join(f.home, "retained-bundle")
	effects := f.effects()
	effects.retainBundle = func(string, string, string) (string, error) { return retained, nil }
	if _, err := effects.run(context.Background(), f.home, request); err == nil {
		t.Fatal("expected mutation interruption")
	} else {
		want := "'" + filepath.Join(retained, "vrooli") + "' runtime update rebaseline '" + filepath.Join(retained, "vrooli") + "'"
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("post-progress mutation failure did not use retained recovery bundle: %v", err)
		}
	}
}

func TestCancellationBetweenStagesRecordsFailedStage(t *testing.T) {
	f := newRebaselineFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	effects := f.effects()
	publish := effects.publishWatchdog
	effects.publishWatchdog = func(ctx context.Context, home, image, digest string) error {
		if err := publish(ctx, home, image, digest); err != nil {
			return err
		}
		cancel()
		return nil
	}
	if _, err := effects.run(ctx, f.home, f.request(true)); err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled between stages: %v", err)
	}
	stateDir, err := repocontract.RuntimeHomeEntryPath(f.home, repocontract.HomeKeyState)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(filepath.Join(stateDir, "root-rebaseline", "attempts"))
	if err != nil {
		t.Fatal(err)
	}
	var supervisorEvents []string
	for _, entry := range entries {
		data, err := os.ReadFile(filepath.Join(stateDir, "root-rebaseline", "attempts", entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		var record struct {
			Events []struct {
				Stage  string `json:"stage"`
				Reason string `json:"reason"`
			} `json:"events"`
		}
		if err := json.Unmarshal(data, &record); err != nil {
			t.Fatal(err)
		}
		for _, event := range record.Events {
			if event.Stage == "supervisor" {
				supervisorEvents = append(supervisorEvents, event.Reason)
			}
		}
	}
	if !slices.Equal(supervisorEvents, []string{"started", "failed"}) {
		t.Fatalf("canceled stage events = %v, want [started failed]", supervisorEvents)
	}
}

func TestPreInstallFailureReconfirmsSameImages(t *testing.T) {
	f := newRebaselineFixture(t)
	f.failAt = "install"
	req := f.request(true)
	req.BundleSHA = strings.Repeat("b", 64)
	req.AfterActivation = func(context.Context, RebaselineRecovery) error { return nil }
	if _, err := f.effects().run(context.Background(), f.home, req); err == nil {
		t.Fatal("expected pre-install failure")
	}
	if len(f.confirmed) != 1 {
		t.Fatalf("initial confirmation missing: %+v", f.confirmed)
	}
	f.failAt = ""
	if _, err := f.effects().run(context.Background(), f.home, req); err != nil {
		t.Fatalf("retry: %v", err)
	}
	if len(f.confirmed) != 2 || f.confirmed[0].CandidateSHA != f.confirmed[1].CandidateSHA || f.confirmed[0].WatchdogSHA != f.confirmed[1].WatchdogSHA {
		t.Fatalf("retry did not reconfirm the same images: %+v", f.confirmed)
	}
}

// Kill the transition process after durable effects, then resume in a fresh
// process context using the same isolated home and bundle.
func TestInterruptedProcessResumesPreparedBundle(t *testing.T) {
	if point := os.Getenv("VROOLI_REBASELINE_CRASH_POINT"); point != "" {
		f := newRebaselineFixture(t)
		f.home = os.Getenv("VROOLI_REBASELINE_CRASH_HOME")
		f.candidate = os.Getenv("VROOLI_REBASELINE_CRASH_CANDIDATE")
		var err error
		f.layout, err = rootUpdateLayout(f.home)
		if err != nil {
			t.Fatal(err)
		}
		f.operatorSets(false)
		e := f.effects()
		e.retainBundle, e.removeBundle = cliinstall.RetainRebaselineBundle, cliinstall.RemoveRebaselineBundle
		install := e.installRoot
		e.installRoot = func(src, dst, expected string) error {
			if err := install(src, dst, expected); err != nil {
				return err
			}
			if point == "root" {
				os.Exit(42)
			}
			return nil
		}
		publish := e.publishWatchdog
		e.publishWatchdog = func(ctx context.Context, home, image, digest string) error {
			if err := publish(ctx, home, image, digest); err != nil {
				return err
			}
			if point == "watchdog" {
				os.Exit(42)
			}
			return nil
		}
		req := f.request(true)
		req.BundleSHA = os.Getenv("VROOLI_REBASELINE_CRASH_BUNDLE")
		req.AfterActivation = func(context.Context, RebaselineRecovery) error {
			if point == "companions" {
				os.Exit(42)
			}
			return nil
		}
		_, err = e.run(context.Background(), f.home, req)
		t.Fatalf("expected abrupt process exit, got %v", err)
	}
	for _, point := range []string{"root", "watchdog", "companions"} {
		t.Run(point, func(t *testing.T) {
			f := newRebaselineFixture(t)
			f.operatorSets(false)
			source := t.TempDir()
			for _, name := range []string{"vrooli", "vrooli-atomic-install", "vrooli-api", "vrooli-agent-launcher", "vrooli-policy-runner", "vrooli-gocacheprog", "vrooli-watchdog"} {
				data := []byte(name)
				if name == "vrooli" {
					data = []byte("candidate image")
				}
				if err := os.WriteFile(filepath.Join(source, name), data, 0700); err != nil {
					t.Fatal(err)
				}
			}
			bundle := filepath.Join(t.TempDir(), "bundle")
			if err := cliinstall.PrepareBundle(source, bundle); err != nil {
				t.Fatal(err)
			}
			manifest, err := fileDigest(filepath.Join(bundle, "SHA256SUMS"))
			if err != nil {
				t.Fatal(err)
			}
			f.candidate = filepath.Join(bundle, "vrooli")
			cmd := exec.Command(os.Args[0], "-test.run=^TestInterruptedProcessResumesPreparedBundle$")
			cmd.Env = append(os.Environ(), "VROOLI_REBASELINE_CRASH_POINT="+point, "VROOLI_REBASELINE_CRASH_HOME="+f.home, "VROOLI_REBASELINE_CRASH_CANDIDATE="+f.candidate, "VROOLI_REBASELINE_CRASH_BUNDLE="+manifest)
			if err := cmd.Run(); err == nil {
				t.Fatal("child did not crash")
			} else if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 42 {
				t.Fatalf("child failed before crash point: %v", err)
			}
			path := filepath.Join(f.home, ".vrooli", "state", "root-rebaseline", "progress.json")
			p, err := rebaselineprogress.Read(path)
			if err != nil || p == nil || p.Complete {
				t.Fatalf("missing incomplete intent: %+v %v", p, err)
			}
			if _, err := f.effects().run(context.Background(), f.home, f.request(true)); err == nil || !strings.Contains(err.Error(), "original prepared bundle") {
				t.Fatalf("unprepared retry crossed crashed prepared transition: %v", err)
			}
			retained, err := cliinstall.RebaselineBundlePath(f.home, manifest)
			if err != nil {
				t.Fatal(err)
			}
			if err := buildinfo.GuardRebaselineCompletion(f.home); err == nil || !strings.Contains(err.Error(), filepath.Join(retained, "vrooli")) {
				t.Fatalf("operator cannot discover retained recovery: %v", err)
			}
			if err := os.RemoveAll(bundle); err != nil {
				t.Fatal(err)
			}
			if err := os.RemoveAll(source); err != nil {
				t.Fatal(err)
			}
			f.candidate = filepath.Join(retained, "vrooli")
			req := f.request(true)
			req.BundleSHA = manifest
			published := false
			req.AfterActivation = func(_ context.Context, recovery RebaselineRecovery) error {
				actual, err := fileDigest(f.layout.Executable)
				if err != nil || actual != recovery.Admission.CandidateSHA {
					return errors.New("companion root mismatch")
				}
				published = true
				return nil
			}
			e := f.effects()
			e.retainBundle, e.removeBundle = cliinstall.RetainRebaselineBundle, cliinstall.RemoveRebaselineBundle
			if _, err := e.run(context.Background(), f.home, req); err != nil {
				t.Fatalf("resume: %v", err)
			}
			p, err = rebaselineprogress.Read(path)
			if err != nil || !p.Complete || !published {
				t.Fatalf("incomplete recovery: %+v published=%v err=%v", p, published, err)
			}
		})
	}
}

func TestRootUpdateAdmissionRejectsInterruptedRebaseline(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".vrooli", "state", "root-rebaseline", "progress.json")
	if err := rebaselineprogress.Write(path, rebaselineprogress.Progress{Version: 1, Candidate: strings.Repeat("a", 64), Watchdog: strings.Repeat("b", 64)}); err != nil {
		t.Fatal(err)
	}
	_, err := SubmitRootUpdate(context.Background(), home, RootUpdateRequest{Source: "/source", Candidate: "/candidate"})
	if err == nil || !strings.Contains(err.Error(), "rebaseline is incomplete") {
		t.Fatalf("admission ignored interrupted transition: %v", err)
	}
}

func TestIncompleteProgressStillRecoversWhenOperatorEnablesEnrollment(t *testing.T) {
	f := newRebaselineFixture(t)
	f.failAt = "converge"
	if _, err := f.effects().run(context.Background(), f.home, f.request(true)); err == nil {
		t.Fatal("expected interruption")
	}
	f.operatorSets(true)
	f.failAt, f.calls = "", nil
	if _, err := f.effects().run(context.Background(), f.home, f.request(true)); err != nil {
		t.Fatalf("incomplete transition must recover: %v", err)
	}
	if !strings.Contains(strings.Join(f.calls, ","), "publish-watchdog") {
		t.Fatalf("recovery was skipped: %v", f.calls)
	}
}

func TestIncompleteProgressContinuesAfterServingSameContractImageChanges(t *testing.T) {
	f := newRebaselineFixture(t)
	f.operatorSets(false)
	f.failAt = "converge"
	if _, err := f.effects().run(context.Background(), f.home, f.request(true)); err == nil {
		t.Fatal("expected interrupted transition")
	}
	if err := os.WriteFile(f.layout.Executable, []byte("other same-contract image"), 0700); err != nil {
		t.Fatal(err)
	}
	f.identities["other same-contract image"] = &cliv1.CliVersion{CliVersion: "1", PlatformVersion: "1", BuildIdentity: strings.Repeat("9", 64), RuntimeHomeContractSha256: strings.Repeat("e", 64), PersistentStateContractSha256: strings.Repeat("d", 64)}
	f.failAt, f.calls = "", nil
	if _, err := f.effects().run(context.Background(), f.home, f.request(true)); err != nil {
		t.Fatalf("continuation: %v", err)
	}
	if !strings.Contains(strings.Join(f.calls, ","), "publish-watchdog") {
		t.Fatalf("recovery skipped: %v", f.calls)
	}
}

func TestSuccessfulRebaselineAttributionUsesRetainedPriorAndInstalledCandidate(t *testing.T) {
	f := newRebaselineFixture(t)
	effects := f.effects()
	var attempt *rootAttempt
	effects.newAttempt = func(home, kind string) *rootAttempt { attempt = beginRootAttempt(home, kind); return attempt }
	result, err := effects.run(context.Background(), f.home, f.request(true))
	if err != nil {
		t.Fatal(err)
	}
	if attempt.serving != filepath.Join(result.RetainedDir, "vrooli") || attempt.candidate != f.layout.Executable {
		t.Fatalf("attribution images: serving=%q candidate=%q", attempt.serving, attempt.candidate)
	}
	prior, err := os.ReadFile(attempt.serving)
	if err != nil || string(prior) != "serving image" {
		t.Fatalf("retained prior %q: %v", prior, err)
	}
	candidate, err := os.ReadFile(attempt.candidate)
	if err != nil || string(candidate) != "candidate image" {
		t.Fatalf("installed candidate %q: %v", candidate, err)
	}
}

func TestPreparedTransitionUsesManifestWatchdogAfterSourceChanges(t *testing.T) {
	f := newRebaselineFixture(t)
	input := t.TempDir()
	for _, name := range []string{"vrooli", "vrooli-api", "vrooli-atomic-install", "vrooli-agent-launcher", "vrooli-policy-runner", "vrooli-gocacheprog", "vrooli-watchdog"} {
		data := name
		if name == "vrooli" {
			data = "candidate image"
		}
		if name == "vrooli-watchdog" {
			data = "original cohort watchdog"
		}
		if err := os.WriteFile(filepath.Join(input, name), []byte(data), 0700); err != nil {
			t.Fatal(err)
		}
	}
	bundle := filepath.Join(t.TempDir(), "bundle")
	if err := cliinstall.PrepareBundle(input, bundle); err != nil {
		t.Fatal(err)
	}
	manifest, err := fileDigest(filepath.Join(bundle, "SHA256SUMS"))
	if err != nil {
		t.Fatal(err)
	}
	source := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "behavior.go"), []byte("changed after preparation"), 0600); err != nil {
		t.Fatal(err)
	}
	f.candidate = filepath.Join(bundle, "vrooli")
	request := f.request(true)
	request.Source = source
	request.BundleSHA = manifest
	request.AfterActivation = func(context.Context, RebaselineRecovery) error { return nil }
	e := f.effects()
	e.selectWatchdog = selectPreparedRebaselineWatchdog
	e.buildWatchdog = func(string, string) error { t.Fatal("rebuilt watchdog from changed source"); return nil }
	e.probeWorker = func(_ context.Context, image string, _ *cliv1.CliVersion, _ string) error {
		b, err := os.ReadFile(image)
		if err != nil {
			return err
		}
		if string(b) != "original cohort watchdog" {
			t.Fatalf("watchdog=%q", b)
		}
		return nil
	}
	if _, err := e.run(context.Background(), f.home, request); err != nil {
		t.Fatal(err)
	}
}

func TestFreshTransitionRequiresManifestBoundWatchdog(t *testing.T) {
	if _, _, err := selectPreparedRebaselineWatchdog(t.TempDir(), "", nil, filepath.Join(t.TempDir(), "pending"), ""); err == nil || !strings.Contains(err.Error(), "make rebaseline") {
		t.Fatalf("unprepared transition: %v", err)
	}
}

func TestLegacyPreparedWatchdogRecovery(t *testing.T) {
	for _, tc := range []struct {
		name                               string
		progress, pending, tamper, success bool
	}{
		{"fresh legacy refused", false, false, false, false},
		{"missing pending refused", true, false, false, false},
		{"tampered pending refused", true, true, true, false},
		{"matching pending resumes", true, true, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newRebaselineFixture(t)
			f.operatorSets(false)
			input := t.TempDir()
			for _, name := range []string{"vrooli", "vrooli-api", "vrooli-atomic-install", "vrooli-agent-launcher", "vrooli-policy-runner", "vrooli-gocacheprog", "vrooli-watchdog"} {
				data := name
				if name == "vrooli" {
					data = "candidate image"
				}
				if err := os.WriteFile(filepath.Join(input, name), []byte(data), 0700); err != nil {
					t.Fatal(err)
				}
			}
			bundle := filepath.Join(t.TempDir(), "bundle")
			if err := cliinstall.PrepareBundle(input, bundle); err != nil {
				t.Fatal(err)
			}
			manifestPath := filepath.Join(bundle, "SHA256SUMS")
			data, err := os.ReadFile(manifestPath)
			if err != nil {
				t.Fatal(err)
			}
			var lines []string
			for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
				if !strings.HasSuffix(line, "  vrooli-watchdog") {
					lines = append(lines, line)
				}
			}
			if err := os.WriteFile(manifestPath, []byte(strings.Join(lines, "\n")+"\n"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(filepath.Join(bundle, "vrooli-watchdog")); err != nil {
				t.Fatal(err)
			}
			manifest, err := fileDigest(manifestPath)
			if err != nil {
				t.Fatal(err)
			}
			f.candidate = filepath.Join(bundle, "vrooli")
			request := f.request(true)
			request.BundleSHA = manifest
			request.AfterActivation = func(context.Context, RebaselineRecovery) error { return nil }
			transitionDir := filepath.Dir(f.pausePath())
			if err := os.MkdirAll(transitionDir, 0700); err != nil {
				t.Fatal(err)
			}
			digest := sha256.Sum256([]byte("qualified legacy watchdog"))
			watchdogSHA := hex.EncodeToString(digest[:])
			if tc.progress {
				candidateSHA, err := fileDigest(f.candidate)
				if err != nil {
					t.Fatal(err)
				}
				if err := rebaselineprogress.Write(filepath.Join(transitionDir, "progress.json"), rebaselineprogress.Progress{Version: 1, Candidate: candidateSHA, Watchdog: watchdogSHA, Bundle: manifest, SourceRoot: request.Source}); err != nil {
					t.Fatal(err)
				}
			}
			if tc.pending {
				data := []byte("qualified legacy watchdog")
				if tc.tamper {
					data = []byte("tampered")
				}
				if err := os.WriteFile(filepath.Join(transitionDir, "watchdog-pending"), data, 0700); err != nil {
					t.Fatal(err)
				}
			}
			e := f.effects()
			e.selectWatchdog = selectPreparedRebaselineWatchdog
			e.buildWatchdog = func(string, string) error { t.Fatal("rebuilt legacy watchdog"); return nil }
			_, err = e.run(context.Background(), f.home, request)
			if tc.success && err != nil {
				t.Fatalf("recovery: %v", err)
			}
			if !tc.success && err == nil {
				t.Fatal("unsafe legacy case accepted")
			}
		})
	}
}
