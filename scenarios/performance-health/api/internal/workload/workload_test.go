package workload

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	platform "github.com/vrooli/platform-go"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	sweepv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/sweep"
	_ "modernc.org/sqlite"
)

func nativeReceipt(d declaration, c candidate, started time.Time) producerReceipt {
	r := producerReceipt{SchemaVersion: 1, OperationID: "native-owner", StartedAt: started, FinishedAt: started.Add(time.Second),
		ProducerDigest: d.producerDigest, ContractDigest: d.contractDigest, FixtureRevision: strings.Repeat("f", 64),
		SampleCount: d.Samples, DeclaredWarmups: d.Warmups, Before: c, After: c}
	for i := -d.Warmups; i < d.Samples; i++ {
		ms := float64(i + 1)
		if i < 0 {
			ms = 10000
		}
		r.Attempts = append(r.Attempts, sample{Index: i, Warmup: i < 0, Status: "verified", OperationID: fmt.Sprintf("operation-%d", i),
			OwnerMS: ms, WallMS: ms + 100, ResponseFile: fmt.Sprintf("response-%d.json", i), ResponseSHA: digest([]byte("{}")),
			Artifacts: []artifact{{Type: "evidence", Path: "original.txt", SHA256: digest([]byte("evidence")), Bytes: 8}}})
	}
	return r
}

func TestWorkloadPercentileAndCompleteness(t *testing.T) {
	d := declaration{Definition: Definition{Samples: 100, Warmups: 1, P95MaxMS: 200}, producerDigest: "producer", contractDigest: "contract"}
	c := candidate{BuildIdentity: "build", DriverVersion: "driver", BrowserVersion: "browser"}
	start := time.Now()
	for _, fault := range []string{"none", "breach", "last-six-slow", "missing", "warmup-count", "denominator", "retry", "failed", "not-attempted", "status-missing", "zero", "negative", "wall-below-owner", "raw-missing", "artifacts-missing", "producer", "contract", "build-before", "build-after", "browser", "historical", "future", "malformed", "producer-error"} {
		t.Run(fault, func(t *testing.T) {
			r := nativeReceipt(d, c, start)
			switch fault {
			case "breach":
				for i := range r.Attempts {
					r.Attempts[i].WallMS += 1000
				}
			case "last-six-slow":
				for i := 95; i < len(r.Attempts); i++ {
					r.Attempts[i].OwnerMS = 1000
					r.Attempts[i].WallMS = 1100
				}
			case "missing":
				r.Attempts = r.Attempts[:100]
			case "warmup-count":
				r.DeclaredWarmups = 0
			case "denominator":
				r.SampleCount = 99
			case "retry":
				r.Attempts[2].OperationID = r.Attempts[1].OperationID
			case "failed":
				r.Attempts[2].Status = "failed"
			case "not-attempted":
				r.Attempts[2].Status = "not_attempted"
			case "status-missing":
				r.Attempts[2].Status = ""
			case "zero":
				r.Attempts[2].OwnerMS = 0
			case "negative":
				r.Attempts[2].OwnerMS = -1
			case "wall-below-owner":
				r.Attempts[2].WallMS = 1
			case "raw-missing":
				r.Attempts[2].ResponseFile = ""
			case "artifacts-missing":
				r.Attempts[2].Artifacts = nil
			case "producer":
				r.ProducerDigest = "other"
			case "contract":
				r.ContractDigest = "other"
			case "build-before":
				r.Before.BuildIdentity = "other"
			case "build-after":
				r.After.BuildIdentity = "other"
			case "browser":
				r.After.BrowserVersion = "other"
			case "historical":
				r.StartedAt = start.Add(-time.Hour)
			case "future":
				r.FinishedAt = start.Add(time.Hour)
			case "producer-error":
				r.Errors = []string{"failed"}
			}
			raw, _ := json.Marshal(r)
			if fault == "malformed" {
				raw = []byte(`{"schema_version":`)
			}
			reading := &sweepv1.WorkloadReading{}
			err := evaluate(raw, d, c, start, start.Add(2*time.Second), reading)
			valid := fault == "none" || fault == "breach" || fault == "last-six-slow"
			if (err == nil) != valid {
				t.Fatalf("fault=%s err=%v", fault, err)
			}
			if fault == "none" && (reading.P95Ms != 95 || reading.WallP95Ms != 195 || !reading.WithinBudget || reading.SampleCount != 100) {
				t.Fatalf("wrong full-cohort percentile %+v", reading)
			}
			if (fault == "breach" || fault == "last-six-slow") && reading.WithinBudget {
				t.Fatal("slow first attempts disappeared from budget verdict")
			}
		})
	}
}

func testService(t *testing.T) (*Service, string, *httptest.Server) {
	t.Helper()
	root := t.TempDir()
	contract, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "..", ".vrooli", "repo-contract.json"))
	if err != nil {
		t.Fatal(err)
	}
	put := func(path string, b []byte) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	put(filepath.Join(root, ".vrooli/repo-contract.json"), contract)
	scenarioRoot := filepath.Join(root, "scenarios/demo")
	put(filepath.Join(scenarioRoot, "producer.go"), []byte("source"))
	put(filepath.Join(scenarioRoot, "contract.json"), []byte("contract"))
	d := Definition{Command: []string{"producer", "{output}"}, Directory: ".", Sources: []string{"producer.go"}, Contract: "contract.json", Endpoints: map[string]string{"api_url": "API_PORT"}, Samples: 100, Warmups: 1, P95MaxMS: 200, TimeoutSeconds: 10, MaxAgeSeconds: 60}
	b, _ := json.Marshal(map[string]any{"performance": map[string]any{"workloads": map[string]Definition{"capture": d}}})
	put(filepath.Join(scenarioRoot, ".vrooli/testing.json"), b)
	db, err := sql.Open("sqlite", "file:"+filepath.Join(root, "evidence.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	s := NewService(root, filepath.Join(root, "receipts"), NewStore(db))
	s.environment = func(context.Context) (*commonv1.CaptureEnvironment, error) {
		return &commonv1.CaptureEnvironment{Os: "linux", Arch: "amd64", NumCpu: 8, TotalMemBytes: 32 << 30}, nil
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"build_identity": "sha256:" + strings.Repeat("a", 64)})
	}))
	t.Cleanup(server.Close)
	s.resolve = func(context.Context, string, string) (string, error) { return server.URL, nil }
	s.runCommand = func(ctx context.Context, dir string, args []string, ownerOutput string) error {
		output := args[1]
		if err := os.Mkdir(output, 0700); err != nil {
			return err
		}
		d, err := s.declaration("demo", "capture")
		if err != nil {
			return err
		}
		_, c, err := s.currentCandidate(ctx, "demo", d)
		if err != nil {
			return err
		}
		r := nativeReceipt(d, c, time.Now().UTC())
		r.FinishedAt = r.StartedAt
		original := filepath.Join(root, "original.txt")
		put(original, []byte("evidence"))
		for i := range r.Attempts {
			r.Attempts[i].Artifacts[0].Path = original
			put(filepath.Join(output, r.Attempts[i].ResponseFile), []byte("{}"))
		}
		raw, _ := json.Marshal(r)
		put(filepath.Join(output, "receipt.json"), raw)
		return nil
	}
	return s, scenarioRoot, server
}

func TestOwnerRetainsAndInvalidatesEvidence(t *testing.T) {
	for _, fault := range []string{"none", "original-removed", "retained-removed", "raw-changed", "response-changed", "producer-changed", "contract-changed", "configuration-changed", "build-changed", "expired", "newer-failure"} {
		t.Run(fault, func(t *testing.T) {
			s, root, _ := testService(t)
			r, err := s.Run(t.Context(), "demo", "capture")
			if err != nil {
				t.Fatal(err)
			}
			if r.Outcome != sweepv1.WorkloadOutcome_WORKLOAD_OUTCOME_MEASURED || !r.WithinBudget {
				t.Fatalf("owner did not measure fixture: %+v", r)
			}
			switch fault {
			case "original-removed":
				_ = os.Remove(filepath.Join(s.repoRoot, "original.txt"))
			case "retained-removed":
				_ = os.RemoveAll(filepath.Join(filepath.Dir(r.ReceiptPath), "artifacts"))
			case "raw-changed":
				_ = os.WriteFile(r.ReceiptPath, []byte("changed"), 0600)
			case "response-changed":
				_ = os.WriteFile(filepath.Join(filepath.Dir(r.ReceiptPath), "response-1.json"), []byte("changed"), 0600)
			case "producer-changed":
				_ = os.WriteFile(filepath.Join(root, "producer.go"), []byte("changed"), 0600)
			case "contract-changed":
				_ = os.WriteFile(filepath.Join(root, "contract.json"), []byte("changed"), 0600)
			case "configuration-changed":
				b, _ := os.ReadFile(filepath.Join(root, ".vrooli/testing.json"))
				_ = os.WriteFile(filepath.Join(root, ".vrooli/testing.json"), []byte(strings.Replace(string(b), `"p95_max_ms":200`, `"p95_max_ms":300`, 1)), 0600)
			case "build-changed":
				s.resolve = func(context.Context, string, string) (string, error) {
					return "", errors.New("deployed target unavailable")
				}
			case "expired":
				r.OperationId = "expired"
				r.CapturedAt = time.Now().Add(-time.Hour).Format(time.RFC3339Nano)
				other := NewStore(s.store.(*Store).db)
				_, _ = other.db.ExecContext(t.Context(), "DELETE FROM performance_workload_receipts")
				if err := other.Insert(t.Context(), r); err != nil {
					t.Fatal(err)
				}
			case "newer-failure":
				s.runCommand = func(context.Context, string, []string, string) error { return errors.New("producer died") }
				r, err = s.Run(t.Context(), "demo", "capture")
				if err != nil {
					t.Fatal(err)
				}
				if r.Outcome != sweepv1.WorkloadOutcome_WORKLOAD_OUTCOME_FAILED {
					t.Fatal("command failure disappeared")
				}
			}
			reading, err := s.Get(t.Context(), "demo", "capture")
			if err != nil {
				t.Fatal(err)
			}
			valid := fault == "none" || fault == "original-removed"
			if (reading.Outcome == sweepv1.WorkloadOutcome_WORKLOAD_OUTCOME_MEASURED) != valid {
				t.Fatalf("fault=%s reading=%+v", fault, reading)
			}
			if !valid && reading.WithinBudget {
				t.Fatal("invalid receipt retained pass flag")
			}
			if fault == "newer-failure" && reading.OperationId != r.OperationId {
				t.Fatal("fell back to older passing receipt")
			}
		})
	}
}

func TestUndeclaredAndConcurrentWorkloadsDoNotExecute(t *testing.T) {
	s, _, _ := testService(t)
	calls := 0
	s.runCommand = func(context.Context, string, []string, string) error { calls++; return nil }
	for _, name := range []string{"missing", "../capture"} {
		if _, err := s.Run(t.Context(), "demo", name); err == nil {
			t.Fatal("undeclared workload admitted")
		}
	}
	s.running.Lock()
	_, err := s.Run(t.Context(), "demo", "capture")
	s.running.Unlock()
	if err == nil || calls != 0 {
		t.Fatalf("busy admission executed: calls=%d error=%v", calls, err)
	}
}

type terminalWriteFailure struct{ repository }

func (s terminalWriteFailure) Complete(context.Context, *sweepv1.WorkloadReading) error {
	return errors.New("terminal storage unavailable")
}

func TestTerminalWriteFailureCannotRevealOlderPass(t *testing.T) {
	s, _, _ := testService(t)
	previous, err := s.Run(t.Context(), "demo", "capture")
	if err != nil {
		t.Fatal(err)
	}
	s.store = terminalWriteFailure{s.store}
	if _, err = s.Run(t.Context(), "demo", "capture"); err == nil {
		t.Fatal("terminal persistence fault acknowledged success")
	}
	latest, err := s.Get(t.Context(), "demo", "capture")
	if err != nil {
		t.Fatal(err)
	}
	if latest.OperationId == previous.OperationId || latest.Outcome != sweepv1.WorkloadOutcome_WORKLOAD_OUTCOME_UNAVAILABLE || latest.WithinBudget {
		t.Fatalf("older pass resurfaced: %+v", latest)
	}
}

func TestReferenceHardwareCannotBeAssumed(t *testing.T) {
	s, root, _ := testService(t)
	path := filepath.Join(root, ".vrooli/testing.json")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var config map[string]any
	if err := json.Unmarshal(b, &config); err != nil {
		t.Fatal(err)
	}
	d := config["performance"].(map[string]any)["workloads"].(map[string]any)["capture"].(map[string]any)
	d["min_cpu_cores"] = 16
	b, _ = json.Marshal(config)
	if err := os.WriteFile(path, b, 0600); err != nil {
		t.Fatal(err)
	}
	calls := 0
	s.runCommand = func(context.Context, string, []string, string) error { calls++; return nil }
	r, err := s.Run(t.Context(), "demo", "capture")
	if err != nil {
		t.Fatal(err)
	}
	if calls != 0 || r.Outcome == sweepv1.WorkloadOutcome_WORKLOAD_OUTCOME_MEASURED || r.WithinBudget {
		t.Fatalf("undersized host certified: calls=%d %+v", calls, r)
	}
}

func TestLatestAttemptUsesTimestampValueAndCannotBeRewritten(t *testing.T) {
	s, _, _ := testService(t)
	for _, r := range []*sweepv1.WorkloadReading{
		{Scenario: "demo", Workload: "capture", OperationId: "newer", CapturedAt: "2026-09-23T12:00:00.1001Z", Outcome: sweepv1.WorkloadOutcome_WORKLOAD_OUTCOME_FAILED},
		{Scenario: "demo", Workload: "capture", OperationId: "older", CapturedAt: "2026-09-23T12:00:00.1Z", Outcome: sweepv1.WorkloadOutcome_WORKLOAD_OUTCOME_MEASURED, WithinBudget: true},
	} {
		if err := s.store.Insert(t.Context(), r); err != nil {
			t.Fatal(err)
		}
		if err := s.store.Complete(t.Context(), r); err != nil {
			t.Fatal(err)
		}
	}
	latest, err := s.store.Latest(t.Context(), "demo", "capture")
	if err != nil {
		t.Fatal(err)
	}
	if latest.OperationId != "newer" {
		t.Fatalf("timestamp text ordering hid newest failure: %+v", latest)
	}
	latest.WithinBudget = true
	latest.Outcome = sweepv1.WorkloadOutcome_WORKLOAD_OUTCOME_MEASURED
	if err := s.store.Complete(t.Context(), latest); err == nil {
		t.Fatal("finalized receipt was overwritten")
	}
}

func TestCommandCancellationClosesOwnedTree(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("native Linux process tree assertion; no other-platform claim")
	}
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Fatal(err)
	}
	output := t.TempDir()
	pidPath := filepath.Join(output, "child.pid")
	ctx, cancel := context.WithTimeout(t.Context(), 300*time.Millisecond)
	defer cancel()
	// Both parent and child ignore TERM. Cancellation must reach force cleanup,
	// including the child that inherits the command's output pipes.
	err = runCommand(ctx, output, []string{sh, "-c", `trap '' TERM; sleep 60 & echo $! > "$1"; wait`, "fixture", pidPath}, output)
	if err == nil {
		t.Fatal("cancelled command succeeded")
	}
	b, err := os.ReadFile(pidPath)
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil {
		t.Fatal(err)
	}
	// SIGKILL delivery is asynchronous. Bound the observed exit instead of
	// requiring the kernel's process state to change in the signalling syscall.
	require.Eventually(t, func() bool { return !platform.IsPIDRunning(pid) }, time.Second, 10*time.Millisecond,
		"owned child %d survived cancellation", pid)
}

func TestCommandOutputIsBounded(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX fixture only")
	}
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Fatal(err)
	}
	output := t.TempDir()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	err = runCommand(ctx, output, []string{sh, "-c", `yes too-much-output`}, output)
	if err == nil {
		t.Fatal("unbounded command succeeded")
	}
	info, err := os.Stat(filepath.Join(output, "stdout.txt"))
	if err != nil || info.Size() > 1<<20 {
		t.Fatalf("unbounded output %v %v", info, err)
	}
}
