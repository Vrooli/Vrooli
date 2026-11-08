package emergencywatchdog

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
)

const priorWatchdogStamp = "managed:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func setupWatchdogUpdateTest(t *testing.T) (paths, string) {
	t.Helper()
	// Publication fixtures must not fingerprint other agents' live checkout
	// edits between two serialized update requests.
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "cmd", "vrooli-watchdog"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "cmd", "vrooli-watchdog", "main.go"), []byte("package main\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	want, err := expectedWatchdogVersion(root)
	if err != nil {
		t.Fatal(err)
	}
	p := resolvePaths(t.TempDir())
	if err := os.MkdirAll(filepath.Dir(p.Binary), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p.Binary, []byte("prior"), 0o700); err != nil {
		t.Fatal(err)
	}
	oldRoot, oldBuild, oldVersion, oldPublish := resolveWatchdogRootFn, buildWatchdogFn, installedWatchdogVersion, publishWatchdogImage
	t.Cleanup(func() {
		resolveWatchdogRootFn, buildWatchdogFn, installedWatchdogVersion, publishWatchdogImage = oldRoot, oldBuild, oldVersion, oldPublish
	})
	resolveWatchdogRootFn = func() (string, error) { return root, nil }
	buildWatchdogFn = func(_, path string) error { return os.WriteFile(path, []byte("candidate"), 0o700) }
	installedWatchdogVersion = func(path string) string {
		data, _ := os.ReadFile(path)
		switch string(data) {
		case "prior":
			return priorWatchdogStamp
		case "candidate":
			return want
		default:
			return ""
		}
	}
	return p, want
}

func requireWatchdogBytes(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil || string(got) != want {
		t.Fatalf("image=%q want=%q error=%v", got, want, err)
	}
}

func requireWatchdogPhase(t *testing.T, p paths, want string) {
	t.Helper()
	r, err := readWatchdogUpdateReceipt(p)
	if err != nil || r == nil || r.Phase != want {
		t.Fatalf("receipt=%+v error=%v want=%s", r, err, want)
	}
}

func TestWatchdogPublicationRetainsPriorAndReconcilesLostWriteReply(t *testing.T) {
	for _, lostReply := range []bool{false, true} {
		t.Run(fmt.Sprint(lostReply), func(t *testing.T) {
			p, _ := setupWatchdogUpdateTest(t)
			calls := 0
			publishWatchdogImage = func(src, dst, sha string) error {
				calls++
				r, err := readWatchdogUpdateReceipt(p)
				if err != nil || r == nil || r.Phase != "publishing" {
					t.Fatalf("publication preceded durable intent: %+v %v", r, err)
				}
				requireWatchdogBytes(t, p.Binary+".previous", "prior")
				if err := copyWatchdogImage(src, dst, sha); err != nil {
					return err
				}
				if lostReply {
					return errors.New("injected lost write acknowledgement")
				}
				return nil
			}
			if err := buildAndInstallWatchdog(p); err != nil {
				t.Fatal(err)
			}
			requireWatchdogBytes(t, p.Binary, "candidate")
			requireWatchdogBytes(t, p.Binary+".previous", "prior")
			requireWatchdogPhase(t, p, "verified")
			buildWatchdogFn = func(string, string) error { t.Fatal("already qualified image rebuilt"); return nil }
			if err := buildAndInstallWatchdog(p); err != nil {
				t.Fatal(err)
			}
			if calls != 1 {
				t.Fatalf("publication executed %d times", calls)
			}
		})
	}
}

func TestWatchdogFailedPublicationRestoresExactPriorAndSuppressesSameImage(t *testing.T) {
	p, want := setupWatchdogUpdateTest(t)
	installedWatchdogVersion = func(path string) string {
		data, _ := os.ReadFile(path)
		if string(data) == "prior" {
			return priorWatchdogStamp
		}
		if string(data) == "candidate" && path != p.Binary {
			return want
		}
		return ""
	}
	if err := buildAndInstallWatchdog(p); err == nil {
		t.Fatal("failed installed canary reported success")
	}
	requireWatchdogBytes(t, p.Binary, "prior")
	requireWatchdogPhase(t, p, "failed_restored")
	publishWatchdogImage = func(string, string, string) error { t.Fatal("same failed image republished"); return nil }
	if err := buildAndInstallWatchdog(p); err == nil || !strings.Contains(err.Error(), "previously failed") {
		t.Fatalf("failed release was not suppressed: %v", err)
	}
	requireWatchdogBytes(t, p.Binary, "prior")
}

func TestWatchdogInterruptedPublicationUsesReceiptBeforeBuilding(t *testing.T) {
	for _, mode := range []string{"healthy candidate", "bad candidate", "restoration already copied", "foreign image", "corrupt recovery image"} {
		t.Run(mode, func(t *testing.T) {
			p, want := setupWatchdogUpdateTest(t)
			oldSHA, err := watchdogImageDigest(p.Binary)
			if err != nil {
				t.Fatal(err)
			}
			if err := copyWatchdogImage(p.Binary, p.Binary+".previous", oldSHA); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p.Binary, []byte("candidate"), 0o700); err != nil {
				t.Fatal(err)
			}
			candidateSHA, err := watchdogImageDigest(p.Binary)
			if err != nil {
				t.Fatal(err)
			}
			r := watchdogUpdateReceipt{SchemaVersion: 1, Phase: "publishing", CandidateSHA: candidateSHA, CandidateStamp: want, PreviousSHA: oldSHA, PreviousStamp: priorWatchdogStamp}
			if mode == "restoration already copied" {
				r.Phase = "restoring"
				if err := copyWatchdogImage(p.Binary+".previous", p.Binary, oldSHA); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "bad candidate" || mode == "corrupt recovery image" {
				installedWatchdogVersion = func(path string) string {
					data, _ := os.ReadFile(path)
					if string(data) == "prior" {
						return priorWatchdogStamp
					}
					return ""
				}
			}
			if mode == "foreign image" {
				if err := os.WriteFile(p.Binary, []byte("foreign"), 0o700); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "corrupt recovery image" {
				if err := os.WriteFile(p.Binary+".previous", []byte("corrupt"), 0o700); err != nil {
					t.Fatal(err)
				}
			}
			if err := writeWatchdogUpdateReceipt(p, r); err != nil {
				t.Fatal(err)
			}
			buildWatchdogFn = func(string, string) error { t.Fatal("pending publication skipped reconciliation"); return nil }
			err = buildAndInstallWatchdog(p)
			if mode == "healthy candidate" {
				if err != nil {
					t.Fatal(err)
				}
				requireWatchdogPhase(t, p, "verified")
				return
			}
			if err == nil {
				t.Fatal("interrupted failed/uncertain update reported success")
			}
			switch mode {
			case "bad candidate", "restoration already copied":
				requireWatchdogBytes(t, p.Binary, "prior")
				requireWatchdogPhase(t, p, "failed_restored")
			case "foreign image":
				requireWatchdogBytes(t, p.Binary, "foreign")
				requireWatchdogPhase(t, p, "publishing")
			case "corrupt recovery image":
				requireWatchdogBytes(t, p.Binary, "candidate")
				requireWatchdogPhase(t, p, "publishing")
			}
		})
	}
}

func TestWatchdogInvalidReceiptCannotAuthorizeWrites(t *testing.T) {
	for _, mode := range []string{"duplicate", "case alias", "unknown", "null field", "future schema", "trailing", "symlink", "oversized"} {
		t.Run(mode, func(t *testing.T) {
			p, want := setupWatchdogUpdateTest(t)
			r := watchdogUpdateReceipt{SchemaVersion: 1, Phase: "publishing", CandidateSHA: strings.Repeat("a", 64), CandidateStamp: want}
			data, _ := json.Marshal(r)
			switch mode {
			case "duplicate":
				data = append([]byte(`{"phase":"verified",`), data[1:]...)
			case "case alias":
				data = []byte(strings.Replace(string(data), `"phase"`, `"Phase"`, 1))
			case "unknown":
				data = append([]byte(`{"future":true,`), data[1:]...)
			case "null field":
				data = []byte(strings.Replace(string(data), `"previous_stamp":""`, `"previous_stamp":null`, 1))
			case "future schema":
				data = []byte(strings.Replace(string(data), `"schema_version":1`, `"schema_version":2`, 1))
			case "trailing":
				data = append(data, []byte(` {}`)...)
			case "oversized":
				data = []byte(strings.Repeat("x", 8193))
			}
			path := p.Binary + ".update.json"
			if mode == "symlink" {
				target := filepath.Join(p.Home, "foreign")
				if err := os.WriteFile(target, data, 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, path); err != nil {
					t.Fatal(err)
				}
			} else if err := os.WriteFile(path, data, 0o600); err != nil {
				t.Fatal(err)
			}
			buildWatchdogFn = func(string, string) error { t.Fatal("corrupt receipt triggered compiler"); return nil }
			if err := buildAndInstallWatchdog(p); err == nil {
				t.Fatal("invalid receipt accepted")
			}
			requireWatchdogBytes(t, p.Binary, "prior")
		})
	}
}

func TestWatchdogConcurrentUpdatesDoNotRotateRecoveryImage(t *testing.T) {
	p, _ := setupWatchdogUpdateTest(t)
	started, releaseBuild := make(chan struct{}), make(chan struct{})
	buildWatchdogFn = func(_, path string) error {
		close(started)
		<-releaseBuild
		return os.WriteFile(path, []byte("candidate"), 0o700)
	}
	result := make(chan error, 2)
	go func() { result <- buildAndInstallWatchdog(p) }()
	<-started
	go func() { result <- buildAndInstallWatchdog(p) }()
	close(releaseBuild)
	for range 2 {
		if err := <-result; err != nil {
			t.Fatal(err)
		}
	}
	requireWatchdogBytes(t, p.Binary, "candidate")
	requireWatchdogBytes(t, p.Binary+".previous", "prior")
	requireWatchdogPhase(t, p, "verified")
}

func TestMain(m *testing.M) {
	if fixture := os.Getenv("VROOLI_WATCHDOG_VERSION_FIXTURE"); fixture != "" {
		if fixture == "publication" {
			if len(os.Args) != 2 || os.Args[1] != "--version" {
				os.Exit(m.Run())
			}
			path, err := os.Executable()
			if err != nil {
				os.Exit(1)
			}
			file, err := os.Open(path)
			if err != nil {
				os.Exit(1)
			}
			info, err := file.Stat()
			if err != nil {
				os.Exit(1)
			}
			var marker [1]byte
			_, err = file.ReadAt(marker[:], info.Size()-1)
			_ = file.Close()
			if err != nil {
				os.Exit(1)
			}
			if marker[0] == 0xa1 {
				fmt.Println(priorWatchdogStamp)
			} else if marker[0] == 0xa2 && path != os.Getenv("VROOLI_WATCHDOG_CANONICAL_FIXTURE_PATH") {
				fmt.Println(os.Getenv("VROOLI_WATCHDOG_WANT_STAMP"))
			} else {
				fmt.Println("invalid")
			}
			os.Exit(0)
		}
		switch fixture {
		case "valid":
			fmt.Println(priorWatchdogStamp)
		case "overflow":
			fmt.Print(strings.Repeat("x", 16384) + priorWatchdogStamp)
		case "exit":
			os.Exit(1)
		case "timeout":
			time.Sleep(time.Minute)
		default:
			fmt.Print("invalid")
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestWatchdogIndependentPublicationReconcileDoesNotBuildOrEnroll(t *testing.T) {
	p, _ := setupWatchdogUpdateTest(t)
	buildWatchdogFn = func(string, string) error { t.Fatal("reconciliation attempted a new update"); return nil }
	if err := ReconcileBinaryPublication(context.Background(), p.Home); err != nil {
		t.Fatal(err)
	}
	requireWatchdogBytes(t, p.Binary, "prior")
	if _, err := os.Stat(p.Binary + ".update.json"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("empty reconciliation created an update: %v", err)
	}
	if err := ReconcileBinaryPublication(nil, p.Home); err == nil {
		t.Fatal("nil reconciliation context accepted")
	}
	for _, home := range []string{"", "relative-home"} {
		if err := ReconcileBinaryPublication(context.Background(), home); err == nil {
			t.Fatal("relative reconciliation home accepted")
		}
	}
}

func TestWatchdogBuildAliasCannotChangeForeignPermissions(t *testing.T) {
	p, _ := setupWatchdogUpdateTest(t)
	foreign := filepath.Join(p.Home, "foreign-executable")
	if err := os.WriteFile(foreign, []byte("foreign"), 0o644); err != nil {
		t.Fatal(err)
	}
	buildWatchdogFn = func(_, stage string) error {
		if err := os.Remove(stage); err != nil {
			return err
		}
		return os.Symlink(foreign, stage)
	}
	if err := buildAndInstallWatchdog(p); err == nil {
		t.Fatal("aliased build output accepted")
	}
	info, err := os.Stat(foreign)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Fatal("aliased build output changed foreign permissions")
	}
	requireWatchdogBytes(t, foreign, "foreign")
	requireWatchdogBytes(t, p.Binary, "prior")
}

func TestWatchdogInspectionExposesUnsettledReceiptAndForeignSameStampImage(t *testing.T) {
	p, want := setupWatchdogUpdateTest(t)
	sha, err := watchdogImageDigest(p.Binary)
	if err != nil {
		t.Fatal(err)
	}
	r := watchdogUpdateReceipt{SchemaVersion: 1, Phase: "publishing", CandidateSHA: sha, CandidateStamp: want}
	if err := writeWatchdogUpdateReceipt(p, r); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(watchdogBinaryPending(p), " "), "publishing") {
		t.Fatal("unsettled receipt hidden by binary observation")
	}
	r.Phase = "verified"
	if err := writeWatchdogUpdateReceipt(p, r); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p.Binary, []byte("foreign"), 0o700); err != nil {
		t.Fatal(err)
	}
	installedWatchdogVersion = func(string) string { return want }
	if !strings.Contains(strings.Join(watchdogBinaryPending(p), " "), "differs") {
		t.Fatal("matching version stamp masked changed image bytes")
	}
}

func TestWatchdogStageCleanupPreservesForeignFilesAndRecoverySlots(t *testing.T) {
	p, _ := setupWatchdogUpdateTest(t)
	dir := filepath.Dir(p.Binary)
	stale := filepath.Join(dir, ".vrooli-watchdog-stage-abandoned")
	if err := os.WriteFile(stale, []byte("abandoned stage"), 0o700); err != nil {
		t.Fatal(err)
	}
	foreign := filepath.Join(dir, "foreign")
	if err := os.WriteFile(foreign, []byte("foreign"), 0o600); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(dir, ".vrooli-watchdog-stage-foreign-alias")
	if err := os.Symlink(foreign, alias); err != nil {
		t.Fatal(err)
	}
	if err := buildAndInstallWatchdog(p); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("abandoned owned stage survived")
	}
	if _, err := os.Lstat(alias); err != nil {
		t.Fatal("unknown staging alias removed")
	}
	requireWatchdogBytes(t, foreign, "foreign")
	requireWatchdogBytes(t, p.Binary+".previous", "prior")
}

func TestWatchdogUnexpectedReplacementCannotOverwriteQualifiedRecovery(t *testing.T) {
	p, want := setupWatchdogUpdateTest(t)
	if err := buildAndInstallWatchdog(p); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p.Binary, []byte("foreign"), 0o700); err != nil {
		t.Fatal(err)
	}
	installedWatchdogVersion = func(path string) string {
		data, _ := os.ReadFile(path)
		switch string(data) {
		case "prior":
			return priorWatchdogStamp
		case "candidate", "foreign":
			return want
		}
		return ""
	}
	if err := buildAndInstallWatchdog(p); err == nil {
		t.Fatal("unexpected replacement admitted as the new recovery image")
	}
	requireWatchdogBytes(t, p.Binary, "foreign")
	requireWatchdogBytes(t, p.Binary+".previous", "prior")
	requireWatchdogPhase(t, p, "verified")
}

func TestWatchdogMissingCurrentRetainsExistingRecoveryIdentity(t *testing.T) {
	p, _ := setupWatchdogUpdateTest(t)
	if err := buildAndInstallWatchdog(p); err != nil {
		t.Fatal(err)
	}
	before, err := readWatchdogUpdateReceipt(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(p.Binary); err != nil {
		t.Fatal(err)
	}
	if err := buildAndInstallWatchdog(p); err != nil {
		t.Fatal(err)
	}
	after, err := readWatchdogUpdateReceipt(p)
	if err != nil {
		t.Fatal(err)
	}
	if after.PreviousSHA != before.PreviousSHA || after.PreviousStamp != before.PreviousStamp {
		t.Fatalf("recovery identity erased across missing current image: before=%+v after=%+v", before, after)
	}
	requireWatchdogBytes(t, p.Binary+".previous", "prior")
	requireWatchdogPhase(t, p, "verified")
}

func TestWatchdogVersionProbeBoundsExecutionAndOutput(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"valid", "overflow", "exit", "invalid", "timeout"} {
		t.Run(mode, func(t *testing.T) {
			t.Setenv("VROOLI_WATCHDOG_VERSION_FIXTURE", mode)
			started := time.Now()
			got := probeWatchdogVersion(executable)
			if mode == "valid" {
				if got != priorWatchdogStamp {
					t.Fatalf("valid stamp lost: %q", got)
				}
			} else if got != "" {
				t.Fatalf("invalid canary accepted: %q", got)
			}
			if time.Since(started) > 8*time.Second {
				t.Fatal("version probe exceeded its execution/pipe deadline")
			}
		})
	}
}

func TestWatchdogRestorationReceiptRequiresRetainedIdentityOnReadAndWrite(t *testing.T) {
	for _, phase := range []string{"restoring", "failed_restored"} {
		t.Run(phase, func(t *testing.T) {
			p, want := setupWatchdogUpdateTest(t)
			valid := watchdogUpdateReceipt{SchemaVersion: 1, Phase: "publishing", CandidateSHA: strings.Repeat("a", 64), CandidateStamp: want}
			if err := writeWatchdogUpdateReceipt(p, valid); err != nil {
				t.Fatal(err)
			}
			bad := valid
			bad.Phase = phase
			if err := writeWatchdogUpdateReceipt(p, bad); err == nil {
				t.Fatal("restoration without retained identity was published")
			}
			requireWatchdogPhase(t, p, "publishing")
			data, err := json.Marshal(bad)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p.Binary+".update.json", data, 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := readWatchdogUpdateReceipt(p); err == nil {
				t.Fatal("corrupt restoration receipt was accepted")
			}
			if phase == "failed_restored" {
				bad.PreviousSHA = strings.Repeat("b", 64)
				data, _ = json.Marshal(bad)
				if err := os.WriteFile(p.Binary+".update.json", data, 0o600); err != nil {
					t.Fatal(err)
				}
				if _, err := readWatchdogUpdateReceipt(p); err == nil {
					t.Fatal("restored outcome without prior canary stamp was accepted")
				}
			}
		})
	}
}

func TestWatchdogStageRejectsUnqualifiedImageBeforeReplacement(t *testing.T) {
	for _, candidateVersion := range []string{"", "managed:wrong"} {
		t.Run(candidateVersion, func(t *testing.T) {
			root, err := repocontract.ResolveRepoRoot()
			if err != nil {
				t.Fatal(err)
			}
			p := resolvePaths(t.TempDir())
			if err := os.MkdirAll(filepath.Dir(p.Binary), 0o700); err != nil {
				t.Fatal(err)
			}
			const prior = "prior watchdog bytes"
			if err := os.WriteFile(p.Binary, []byte(prior), 0o700); err != nil {
				t.Fatal(err)
			}
			oldRoot, oldBuild, oldVersion := resolveWatchdogRootFn, buildWatchdogFn, installedWatchdogVersion
			t.Cleanup(func() {
				resolveWatchdogRootFn, buildWatchdogFn, installedWatchdogVersion = oldRoot, oldBuild, oldVersion
			})
			resolveWatchdogRootFn = func() (string, error) { return root, nil }
			buildWatchdogFn = func(_, out string) error { return os.WriteFile(out, []byte("bad staged watchdog"), 0o700) }
			installedWatchdogVersion = func(path string) string {
				if path == p.Binary {
					return "managed:" + strings.Repeat("a", 64)
				}
				return candidateVersion
			}
			if err := buildAndInstallWatchdog(p); err == nil {
				t.Fatal("unqualified executable replaced the installed watchdog")
			}
			got, err := os.ReadFile(p.Binary)
			if err != nil || string(got) != prior {
				t.Fatalf("failed staging changed prior image: %q %v", got, err)
			}
		})
	}
}

func TestWatchdogPublicationRejectsInputChangesDuringBuild(t *testing.T) {
	for _, change := range []string{"Go source", "embedded policy", "module manifest", "missing embedded policy"} {
		t.Run(change, func(t *testing.T) {
			p, _ := setupWatchdogUpdateTest(t)
			root, err := resolveWatchdogRootFn()
			if err != nil {
				t.Fatal(err)
			}
			write := func(name, data string) {
				t.Helper()
				path := filepath.Join(root, name)
				if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			write("go.mod", "module example.test/watchdog\ngo 1.26\n")
			write("internal/policy/embed.go", "package policy\nimport _ \"embed\"\n//go:embed policy.json\nvar Policy string\n")
			write("internal/policy/policy.json", "initial policy")
			want, err := expectedWatchdogVersion(root)
			if err != nil {
				t.Fatal(err)
			}
			priorSHA, err := watchdogImageDigest(p.Binary)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p.Binary+".previous", []byte("retained rollback"), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := writeWatchdogUpdateReceipt(p, watchdogUpdateReceipt{SchemaVersion: 1, Phase: "verified", CandidateSHA: priorSHA, CandidateStamp: priorWatchdogStamp}); err != nil {
				t.Fatal(err)
			}
			receiptBefore, err := os.ReadFile(p.Binary + ".update.json")
			if err != nil {
				t.Fatal(err)
			}
			installedWatchdogVersion = func(path string) string {
				data, _ := os.ReadFile(path)
				if string(data) == "candidate" {
					return want // The candidate retains the stamp taken before compilation.
				}
				return priorWatchdogStamp
			}
			buildWatchdogFn = func(_, stage string) error {
				switch change {
				case "Go source":
					write("cmd/vrooli-watchdog/main.go", "package main\n// changed during compilation\n")
				case "embedded policy":
					write("internal/policy/policy.json", "changed during compilation")
				case "module manifest":
					write("go.mod", "module example.test/changed\ngo 1.26\n")
				case "missing embedded policy":
					if err := os.Remove(filepath.Join(root, "internal/policy/policy.json")); err != nil {
						return err
					}
				}
				return os.WriteFile(stage, []byte("candidate"), 0o700)
			}
			publishWatchdogImage = func(string, string, string) error {
				t.Fatal("publication attempted after build inputs changed")
				return nil
			}
			if err := buildAndInstallWatchdog(p); err == nil || !strings.Contains(err.Error(), "watchdog source") {
				t.Fatalf("changed or unreadable build inputs were not refused: %v", err)
			}
			requireWatchdogBytes(t, p.Binary, "prior")
			requireWatchdogBytes(t, p.Binary+".previous", "retained rollback")
			requireWatchdogBytes(t, p.Binary+".update.json", string(receiptBefore))
			stages, err := filepath.Glob(filepath.Join(filepath.Dir(p.Binary), ".vrooli-watchdog-stage-*"))
			if err != nil || len(stages) != 0 {
				t.Fatalf("refused staged image leaked: %v %v", stages, err)
			}
		})
	}
}

func setupRetainedBaselineTest(t *testing.T) (paths, RetainedWatchdogBaseline) {
	t.Helper()
	p, want := setupWatchdogUpdateTest(t)
	if err := os.WriteFile(p.Binary, []byte("candidate"), 0o700); err != nil {
		t.Fatal(err)
	}
	retained := filepath.Join(t.TempDir(), "retained-watchdog")
	if err := os.WriteFile(retained, []byte("prior"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p.Binary+".previous", []byte("prior"), 0o700); err != nil {
		t.Fatal(err)
	}
	currentSHA, _ := watchdogImageDigest(p.Binary)
	priorSHA, _ := watchdogImageDigest(retained)
	original := watchdogUpdateReceipt{SchemaVersion: 1, Phase: "verified", CandidateSHA: currentSHA, CandidateStamp: want, PreviousSHA: priorSHA, PreviousStamp: priorWatchdogStamp}
	if err := writeWatchdogUpdateReceipt(p, original); err != nil {
		t.Fatal(err)
	}
	receiptData, _ := os.ReadFile(p.Binary + ".update.json")
	spec := RetainedWatchdogBaseline{ExpectedCurrentReceiptSHA: fmt.Sprintf("%x", sha256.Sum256(receiptData)), Home: p.Home, ExpectedCurrentSHA: currentSHA, ExpectedCurrentStamp: want, RetainedImage: retained, RetainedSHA: priorSHA, RetainedStamp: priorWatchdogStamp}
	for _, name := range []string{serviceName, timerName, "vrooli-root-update-recovery.service", "vrooli-root-update-recovery.timer"} {
		path := filepath.Join(filepath.Dir(p.ServiceUnit), name)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		current := []byte("current " + name)
		old := []byte("[Unit]\nDescription=retained fixture\n[Service]\nType=oneshot\nExecStart=/bin/true\n")
		if strings.HasSuffix(name, ".timer") {
			old = []byte("[Unit]\nDescription=retained fixture timer\n[Timer]\nOnActiveSec=1h\n[Install]\nWantedBy=timers.target\n")
		}
		if err := os.WriteFile(path, current, 0o644); err != nil {
			t.Fatal(err)
		}
		backup := filepath.Join(filepath.Dir(retained), name)
		if err := os.WriteFile(backup, old, 0o644); err != nil {
			t.Fatal(err)
		}
		spec.Definitions = append(spec.Definitions, RetainedWatchdogDefinition{Name: name, ExpectedCurrentSHA: fmt.Sprintf("%x", sha256.Sum256(current)), RetainedPath: backup, RetainedSHA: fmt.Sprintf("%x", sha256.Sum256(old))})
	}
	resolveWatchdogRootFn = func() (string, error) { return "", errors.New("source checkout unavailable") }
	buildWatchdogFn = func(string, string) error { t.Fatal("source-free restoration called compiler"); return nil }
	oldRetain := baselineRetainCurrent
	oldProperties, oldCommand, oldDefinition, oldLink := baselineNativeProperties, baselineNativeCommand, baselineDefinitionPublish, baselineEvidenceLink
	t.Cleanup(func() {
		baselineRetainCurrent = oldRetain
		baselineNativeProperties, baselineNativeCommand, baselineDefinitionPublish, baselineEvidenceLink = oldProperties, oldCommand, oldDefinition, oldLink
	})
	baselineNativeProperties = func(_ context.Context, name string) (map[string]string, error) {
		return map[string]string{"LoadState": "loaded", "FragmentPath": filepath.Join(filepath.Dir(p.ServiceUnit), name), "DropInPaths": "", "NeedDaemonReload": "no", "ActiveState": "active", "UnitFileState": "enabled"}, nil
	}
	baselineNativeCommand = func(context.Context, []string) error { return nil }
	return p, spec
}

func TestRetainedBaselineRestoresVerifiedPublicationWithoutSource(t *testing.T) {
	p, spec := setupRetainedBaselineTest(t)
	if err := RestoreRetainedBaseline(context.Background(), spec); err != nil {
		t.Fatalf("qualified retained baseline must restore without source: %v", err)
	}
	requireWatchdogBytes(t, p.Binary, "prior")
	receipt, err := readWatchdogUpdateReceipt(p)
	if err != nil || receipt == nil || receipt.Phase != "verified" || receipt.CandidateSHA != spec.RetainedSHA || receipt.PreviousSHA != spec.ExpectedCurrentSHA {
		t.Fatalf("not a new honest restoration publication: %+v %v", receipt, err)
	}
}

func TestRetainedBaselineReplayRetiresCompletedActiveJournal(t *testing.T) {
	p, spec := setupRetainedBaselineTest(t)
	calls := 0
	baselineNativeCommand = func(_ context.Context, args []string) error {
		calls++
		if len(args) > 2 && args[1] == "enable" && !strings.HasSuffix(args[len(args)-1], ".timer") {
			t.Fatal("oneshot service triggered")
		}
		return nil
	}
	if err := RestoreRetainedBaseline(context.Background(), spec); err != nil {
		t.Fatal(err)
	}
	terminal, err := filepath.Glob(p.Binary + ".baseline-restore-*.verified.json")
	if err != nil || len(terminal) != 1 {
		t.Fatal(terminal, err)
	}
	data, _ := os.ReadFile(terminal[0])
	if err := os.WriteFile(p.Binary+".baseline-restore.json", data, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := RestoreRetainedBaseline(context.Background(), spec); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p.Binary + ".baseline-restore.json"); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("completed active fence remains")
	}
	if calls != 3 {
		t.Fatalf("idempotent replay reran native mutation: %d", calls)
	}
}

func TestRetainedBaselinePartialFenceAndExactResume(t *testing.T) {
	for _, edge := range []string{"resume", "foreign-receipt", "foreign-definition"} {
		t.Run(edge, func(t *testing.T) {
			p, spec := setupRetainedBaselineTest(t)
			real := baselineDefinitionPublish
			writes := 0
			baselineDefinitionPublish = func(path string, data []byte) error {
				writes++
				if writes == 2 {
					return errors.New("caller lost before second definition")
				}
				return real(path, data)
			}
			if err := RestoreRetainedBaseline(context.Background(), spec); err == nil {
				t.Fatal("partial restoration reported complete")
			}
			if err := updateWatchdogBinaryLocked(context.Background(), p, watchdogSource{}); err == nil || !strings.Contains(err.Error(), "pending retained-baseline") {
				t.Fatalf("normal update crossed owner fence: %v", err)
			}
			if err := ReconcileBinaryPublication(context.Background(), p.Home); err == nil || !strings.Contains(err.Error(), "pending retained-baseline") {
				t.Fatalf("generic reconciliation crossed owner fence: %v", err)
			}
			baselineDefinitionPublish = real
			if edge == "foreign-receipt" {
				r, _ := readWatchdogUpdateReceipt(p)
				r.PreviousStamp = priorWatchdogStamp
				if err := writeWatchdogUpdateReceipt(p, *r); err != nil {
					t.Fatal(err)
				}
			}
			if edge == "foreign-definition" {
				if err := os.WriteFile(filepath.Join(filepath.Dir(p.ServiceUnit), spec.Definitions[0].Name), []byte("foreign"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			err := RestoreRetainedBaseline(context.Background(), spec)
			if edge == "resume" {
				if err != nil {
					t.Fatal(err)
				}
				requireWatchdogPhase(t, p, "verified")
			} else if err == nil {
				t.Fatal("foreign state accepted on resume")
			}
		})
	}
}

func TestRetainedBaselineKnownInactiveTimerCanRecover(t *testing.T) {
	p, spec := setupRetainedBaselineTest(t)
	active := false
	old := baselineNativeProperties
	baselineNativeProperties = func(ctx context.Context, name string) (map[string]string, error) {
		v, e := old(ctx, name)
		if !active && strings.HasSuffix(name, ".timer") {
			v["ActiveState"] = "inactive"
			v["UnitFileState"] = "disabled"
		}
		return v, e
	}
	baselineNativeCommand = func(_ context.Context, args []string) error {
		if len(args) > 2 && args[1] == "enable" {
			active = true
		}
		return nil
	}
	if err := RestoreRetainedBaseline(context.Background(), spec); err != nil {
		t.Fatal(err)
	}
	requireWatchdogBytes(t, p.Binary, "prior")
}

func TestRetainedBaselineRejectsUnknownBeforePublication(t *testing.T) {
	for _, edge := range []string{"wrong-image", "wrong-receipt", "native-override", "masked-timer", "wrong-scope"} {
		t.Run(edge, func(t *testing.T) {
			p, spec := setupRetainedBaselineTest(t)
			switch edge {
			case "wrong-image":
				spec.RetainedSHA = strings.Repeat("f", 64)
			case "wrong-receipt":
				spec.ExpectedCurrentReceiptSHA = strings.Repeat("f", 64)
			case "wrong-scope":
				spec.Definitions[0].Name = "vrooli-runtime-supervisor.service"
			case "native-override", "masked-timer":
				old := baselineNativeProperties
				baselineNativeProperties = func(ctx context.Context, n string) (map[string]string, error) {
					v, e := old(ctx, n)
					if edge == "native-override" {
						v["DropInPaths"] = "/foreign.conf"
					} else if strings.HasSuffix(n, ".timer") {
						v["UnitFileState"] = "masked"
					}
					return v, e
				}
			}
			if err := RestoreRetainedBaseline(context.Background(), spec); err == nil {
				t.Fatal("unqualified input accepted")
			}
			requireWatchdogBytes(t, p.Binary, "candidate")
			for _, d := range spec.Definitions {
				if edge != "wrong-scope" {
					if err := baselineDefinitionMatches(filepath.Join(filepath.Dir(p.ServiceUnit), d.Name), d.ExpectedCurrentSHA); err != nil {
						t.Fatal(err)
					}
				}
			}
		})
	}
}

func TestRetainedBaselineEvidencePublicationIsCompleteAndNoReplace(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")
	data := []byte("complete reviewed evidence")
	// A crashed caller may leave a truncated private stage. It never becomes
	// immutable final evidence, and cannot prevent a fresh qualified stage.
	if err := os.WriteFile(filepath.Join(filepath.Dir(path), ".history.json.stage-crashed"), []byte("partial"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeBaselineEvidenceOnce(path, data); err != nil {
		t.Fatal(err)
	}
	if err := writeBaselineEvidenceOnce(path, data); err != nil {
		t.Fatal(err)
	}
	if err := writeBaselineEvidenceOnce(path, []byte("foreign")); err == nil {
		t.Fatal("immutable evidence overwritten")
	}
	requireWatchdogBytes(t, path, string(data))
}

func TestRetainedBaselineLostTerminalAckAndInactiveReplay(t *testing.T) {
	p, spec := setupRetainedBaselineTest(t)
	real := baselineEvidenceLink
	lost := false
	baselineEvidenceLink = func(stage, path string) error {
		err := real(stage, path)
		if err == nil && strings.HasSuffix(path, ".verified.json") && !lost {
			lost = true
			return errors.New("lost terminal link acknowledgement")
		}
		return err
	}
	if err := RestoreRetainedBaseline(context.Background(), spec); err == nil {
		t.Fatal("expected uncertain terminal acknowledgement")
	}
	baselineEvidenceLink = real
	old := baselineNativeProperties
	baselineNativeProperties = func(ctx context.Context, n string) (map[string]string, error) {
		v, e := old(ctx, n)
		if strings.HasSuffix(n, ".timer") {
			v["ActiveState"] = "inactive"
		}
		return v, e
	}
	if err := RestoreRetainedBaseline(context.Background(), spec); err == nil {
		t.Fatal("inactive terminal replay accepted")
	}
	baselineNativeProperties = old
	if err := RestoreRetainedBaseline(context.Background(), spec); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p.Binary + ".baseline-restore.json"); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("matching terminal fence not retired")
	}
}

func TestRetainedBaselineJournalRejectsAmbiguousOwnership(t *testing.T) {
	good := fmt.Sprintf(`{"version":1,"request_sha256":"%s","original_receipt_sha256":"%s","phase":"publishing"}`, strings.Repeat("a", 64), strings.Repeat("b", 64))
	for _, data := range []string{strings.Replace(good, `"phase":"publishing"`, `"phase":"publishing","phase":"verified"`, 1), strings.Replace(good, `"phase"`, `"Phase"`, 1), good + `{}`, strings.Replace(good, `"phase":"publishing"`, `"phase":null`, 1)} {
		path := filepath.Join(t.TempDir(), "active.json")
		if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := readBaselineJournal(path); err == nil {
			t.Fatal("ambiguous owner journal accepted")
		}
	}
}

func TestRetainedBaselineTerminalProvenanceCannotRetireForeignEvidence(t *testing.T) {
	for _, edge := range []string{"terminal-original", "active-original", "history"} {
		t.Run(edge, func(t *testing.T) {
			p, spec := setupRetainedBaselineTest(t)
			if err := RestoreRetainedBaseline(context.Background(), spec); err != nil {
				t.Fatal(err)
			}
			ts, _ := filepath.Glob(p.Binary + ".baseline-restore-*.verified.json")
			var r baselineRestoreJournal
			data, _ := os.ReadFile(ts[0])
			if err := json.Unmarshal(data, &r); err != nil {
				t.Fatal(err)
			}
			if edge == "terminal-original" {
				r.OriginalReceiptSHA = strings.Repeat("f", 64)
				if err := writeBaselineJournal(ts[0], r); err != nil {
					t.Fatal(err)
				}
			}
			if edge == "active-original" {
				r.OriginalReceiptSHA = strings.Repeat("f", 64)
				if err := writeBaselineJournal(p.Binary+".baseline-restore.json", r); err != nil {
					t.Fatal(err)
				}
			}
			if edge == "history" {
				hs, _ := filepath.Glob(p.Binary + ".baseline-restore-*.previous.json")
				if err := os.WriteFile(hs[0], []byte("foreign history"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if err := RestoreRetainedBaseline(context.Background(), spec); err == nil {
				t.Fatal("inconsistent provenance accepted")
			}
			if edge == "active-original" {
				if _, err := os.Stat(p.Binary + ".baseline-restore.json"); err != nil {
					t.Fatal("foreign active evidence retired")
				}
			}
		})
	}
}

func TestRetainedBaselineFinalReceiptWithoutHealthyTimersDoesNotComplete(t *testing.T) {
	p, spec := setupRetainedBaselineTest(t)
	if err := RestoreRetainedBaseline(context.Background(), spec); err != nil {
		t.Fatal(err)
	}
	ts, _ := filepath.Glob(p.Binary + ".baseline-restore-*.verified.json")
	data, _ := os.ReadFile(ts[0])
	var r baselineRestoreJournal
	if err := json.Unmarshal(data, &r); err != nil {
		t.Fatal(err)
	}
	r.Phase = "publishing"
	if err := writeBaselineJournal(p.Binary+".baseline-restore.json", r); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(ts[0]); err != nil {
		t.Fatal(err)
	}
	old := baselineNativeProperties
	baselineNativeProperties = func(ctx context.Context, n string) (map[string]string, error) {
		v, e := old(ctx, n)
		if strings.HasSuffix(n, ".timer") {
			v["ActiveState"] = "inactive"
		}
		return v, e
	}
	if err := RestoreRetainedBaseline(context.Background(), spec); err == nil {
		t.Fatal("final image/receipt bypassed healthy native completion")
	}
	if _, err := os.Stat(p.Binary + ".baseline-restore.json"); err != nil {
		t.Fatal("incomplete restoration unfenced")
	}
	baselineNativeProperties = old
	if err := RestoreRetainedBaseline(context.Background(), spec); err != nil {
		t.Fatal(err)
	}
}

func TestRetainedBaselineHistoryRefusesSymlinkAndOversize(t *testing.T) {
	for _, edge := range []string{"symlink", "oversize"} {
		t.Run(edge, func(t *testing.T) {
			p, spec := setupRetainedBaselineTest(t)
			real := baselineDefinitionPublish
			baselineDefinitionPublish = func(string, []byte) error { return errors.New("interrupted") }
			if err := RestoreRetainedBaseline(context.Background(), spec); err == nil {
				t.Fatal("expected partial definition failure")
			}
			baselineDefinitionPublish = real
			hs, _ := filepath.Glob(p.Binary + ".baseline-restore-*.previous.json")
			if err := os.Remove(hs[0]); err != nil {
				t.Fatal(err)
			}
			if edge == "symlink" {
				target := filepath.Join(t.TempDir(), "other")
				if err := os.WriteFile(target, []byte("other"), 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, hs[0]); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := os.WriteFile(hs[0], []byte(strings.Repeat("x", 8193)), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if err := RestoreRetainedBaseline(context.Background(), spec); err == nil {
				t.Fatal("unbounded/linked history accepted")
			}
		})
	}
}

func TestRetainedBaselineKeepsReceiptAndActualPreviousSlotConsistent(t *testing.T) {
	for _, lost := range []bool{false, true} {
		t.Run(fmt.Sprint(lost), func(t *testing.T) {
			p, spec := setupRetainedBaselineTest(t)
			real := baselineRetainCurrent
			calls := 0
			baselineRetainCurrent = func(a, b, c string) error {
				calls++
				if !lost && calls == 1 {
					return errors.New("retention interrupted before publication")
				}
				err := real(a, b, c)
				if lost && calls == 1 && err == nil {
					return errors.New("lost retention acknowledgement")
				}
				return err
			}
			err := RestoreRetainedBaseline(context.Background(), spec)
			if !lost {
				if err == nil {
					t.Fatal("interrupted retention did not surface")
				}
				requireWatchdogBytes(t, p.Binary, "candidate")
				if err := RestoreRetainedBaseline(context.Background(), spec); err != nil {
					t.Fatal(err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			r, err := readWatchdogUpdateReceipt(p)
			if err != nil {
				t.Fatal(err)
			}
			sum, err := watchdogImageDigest(p.Binary + ".previous")
			if err != nil || sum != r.PreviousSHA || sum != spec.ExpectedCurrentSHA {
				t.Fatal("receipt names unavailable rollback bytes", sum, err)
			}
			requireWatchdogBytes(t, p.Binary+".previous", "candidate")
		})
	}
}

func TestRetainedBaselineRefusesRotatingInputAndForeignBackup(t *testing.T) {
	for _, mode := range []string{"rotating-input", "foreign-backup"} {
		t.Run(mode, func(t *testing.T) {
			p, spec := setupRetainedBaselineTest(t)
			if mode == "rotating-input" {
				spec.RetainedImage = p.Binary + ".previous"
			} else {
				if err := os.WriteFile(p.Binary+".previous", []byte("foreign previous"), 0o700); err != nil {
					t.Fatal(err)
				}
			}
			if err := RestoreRetainedBaseline(context.Background(), spec); err == nil {
				t.Fatal("unsafe previous slot accepted")
			}
			requireWatchdogBytes(t, p.Binary, "candidate")
		})
	}
}
