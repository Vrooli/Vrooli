package seeds

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestNativeSeedWorkerKeepsSyntheticChildSecretsOutOfArgvAndDiagnostics(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "bas", "seeds")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	secret := "synthetic-seed-credential-for-native-argv-proof"
	dsn := "postgres://synthetic-user:synthetic-password@127.0.0.1/private-fixture"
	script := `set -eu
printf '%s' "$TEST_GENIE_SYNTHETIC_SECRET" > observed.env
printf '%s' "$DATABASE_URL" > observed.dsn
printf '%s' "$BASHPID" > child.pid
printf '%s' "$TEST_GENIE_SYNTHETIC_SECRET"
while [ ! -f release ]; do sleep 0.05; done
`
	if err := os.WriteFile(filepath.Join(dir, "seed.sh"), []byte(script), 0600); err != nil {
		t.Fatal(err)
	}
	refs := make(chan OperationRecord, 4)
	recorder := func(record *OperationRecord) error {
		if err := writeWorkerJSON(filepath.Join(root, "operation.json"), record); err != nil {
			return err
		}
		if record.Ref != nil {
			copy := *record
			ref := *record.Ref
			copy.Ref = &ref
			select {
			case refs <- copy:
			default:
			}
		}
		return nil
	}
	manager := NewManager(root, root, root, nil).WithOperationRoot(filepath.Join(root, "native")).WithIsolationEnv(map[string]string{"TEST_GENIE_SYNTHETIC_SECRET": secret, "DATABASE_URL": dsn}).WithOperationRecorder(recorder)
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	defer func() { _ = os.WriteFile(filepath.Join(root, "release"), nil, 0600) }()
	done := make(chan error, 1)
	go func() { _, err := manager.Apply(ctx); done <- err }()
	var record OperationRecord
	select {
	case record = <-refs:
	case err := <-done:
		t.Fatalf("worker did not start: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("worker native ref was not witnessed")
	}
	deadline := time.Now().Add(5 * time.Second)
	var pid []byte
	for time.Now().Before(deadline) {
		raw, err := os.ReadFile(filepath.Join(root, "child.pid"))
		if err == nil {
			pid = raw
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if len(pid) == 0 {
		t.Fatal("seed child did not start")
	}
	observed, err := os.ReadFile(filepath.Join(root, "observed.env"))
	if err != nil || string(observed) != secret {
		t.Fatal("seed child did not receive its exact private environment")
	}
	observed, err = os.ReadFile(filepath.Join(root, "observed.dsn"))
	if err != nil || string(observed) != dsn {
		t.Fatal("seed child did not receive its exact isolated DSN")
	}
	native, err := exec.Command("systemctl", "--user", "show", "--property=ExecStart", "--", record.Ref.Unit).Output()
	if err != nil {
		t.Fatal("native invocation could not be inspected")
	}
	childArgs, err := os.ReadFile("/proc/" + string(pid) + "/cmdline")
	if err != nil {
		t.Fatal("seed child argv could not be inspected")
	}
	snapshot, err := os.ReadFile(filepath.Join(root, "operation.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, data := range [][]byte{native, childArgs, snapshot} {
		if bytes.Contains(data, []byte(secret)) || bytes.Contains(data, []byte(dsn)) {
			t.Fatal("synthetic seed credential leaked outside private input")
		}
	}
	input, err := os.ReadFile(record.InputPath)
	if err != nil || !bytes.Contains(input, []byte(secret)) {
		t.Fatal("private input does not contain the child's actual credential")
	}
	if info, err := os.Stat(record.InputPath); err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("secret input permissions are not private")
	}
	// Replaying the hidden worker outside its original native root is refused;
	// it cannot start an API or execute the seed a second time.
	replayCtx, replayCancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer replayCancel()
	replay := exec.CommandContext(replayCtx, record.WorkerPath, workerFlag, record.InputPath, workerInputHash(t, input))
	replay.Env = []string{"PATH=/usr/bin:/bin"}
	diagnostic, err := replay.CombinedOutput()
	if replayCtx.Err() != nil {
		t.Fatal("replayed worker claim did not refuse promptly")
	}
	if err == nil {
		t.Fatal("private worker claim was replayable")
	}
	if bytes.Contains(diagnostic, []byte(secret)) || bytes.Contains(diagnostic, []byte(dsn)) {
		t.Fatal("refused worker diagnostic exposed its private input")
	}
	if err := os.WriteFile(filepath.Join(root, "release"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("seed worker failed: %v", err)
		}
	case <-time.After(8 * time.Second):
		t.Fatal("seed worker did not finish")
	}
	for _, path := range []string{record.WorkerPath, record.InputPath, record.GrantPath} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatal("retired native operation retained image or credential input")
		}
	}
	journal, err := exec.Command("journalctl", "--user", "--unit="+record.Ref.Unit, "--no-pager", "-n", "50").Output()
	if err != nil {
		t.Fatal("native diagnostics could not be inspected")
	}
	if bytes.Contains(journal, []byte(secret)) || bytes.Contains(journal, []byte(dsn)) {
		t.Fatal("native journal exposed seed credentials")
	}
	log, err := os.ReadFile(record.LogPath)
	if err != nil || !bytes.Contains(log, []byte(secret)) {
		t.Fatal("seed output was not retained exclusively in its private owner log")
	}
}
func workerInputHash(t *testing.T, data []byte) string {
	t.Helper()
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
func TestSeedWorkerRejectsInvalidPrivateInputBeforeAnyExecution(t *testing.T) {
	root := t.TempDir()
	marker := filepath.Join(root, "must-not-execute")
	path := filepath.Join(root, "invalid.input.json")
	raw, _ := json.Marshal(workerInput{Version: 1, Path: "/usr/bin/touch", Args: []string{marker}, Dir: root, Env: []string{}, Deadline: time.Now().Add(time.Minute)})
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	var diagnostic bytes.Buffer
	handled, code := RunWorker([]string{workerFlag, path, "invalid-input-digest"}, &diagnostic)
	if !handled || code == 0 {
		t.Fatal("invalid worker input was accepted")
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("invalid private input executed a child")
	}
}
