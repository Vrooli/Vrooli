package engine

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"
)

func TestEncryptionAlgorithmReadsCurrentKopiaStatusShape(t *testing.T) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(`{
		"contentFormat": {
			"hash": "BLAKE2B-256-128",
			"encryption": "AES256-GCM-HMAC-SHA256"
		}
	}`), &raw); err != nil {
		t.Fatalf("unmarshal status: %v", err)
	}
	if got := encryptionAlgorithm(raw); got != "AES256-GCM-HMAC-SHA256" {
		t.Fatalf("encryptionAlgorithm = %q", got)
	}
}

type recordingRunner struct {
	args   [][]string
	output []byte
	err    error
}

func (r *recordingRunner) Run(_ context.Context, args ...string) ([]byte, error) {
	r.args = append(r.args, append([]string(nil), args...))
	return r.output, r.err
}

func TestRepoDeleteUsesResourceKopiaDeleteCommand(t *testing.T) {
	runner := &recordingRunner{}
	cli := &KopiaCLI{Runner: runner}
	if err := cli.RepoDelete(context.Background(), "nightly"); err != nil {
		t.Fatalf("RepoDelete: %v", err)
	}
	want := []string{"repo", "delete", "--name", "nightly"}
	if len(runner.args) != 1 {
		t.Fatalf("calls = %d, want 1", len(runner.args))
	}
	if !slices.Equal(runner.args[0], want) {
		t.Fatalf("args = %v, want %v", runner.args[0], want)
	}
}

// TestPassphraseRefMatchesResourceKopiaConvention pins the credential authority
// identity/field convention shared with resource-kopia.
// It is a pure derivation — no Runner is invoked, so no secret can be read.
func TestPassphraseRefMatchesResourceKopiaConvention(t *testing.T) {
	cli := &KopiaCLI{Runner: &recordingRunner{}}
	got := cli.PassphraseRef("elements-local")
	want := "vrooli/kopia/elements-local:repository-passphrase"
	if got != want {
		t.Fatalf("PassphraseRef = %q, want %q", got, want)
	}
	// A reference only: it must never look like or contain a secret value.
	if strings.Contains(got, "passphrase=") || len(got) == 0 {
		t.Fatalf("PassphraseRef must be a reference path, got %q", got)
	}
}

func TestRepoStatsParsesPhysicalBytesShape(t *testing.T) {
	runner := &recordingRunner{output: []byte(`{
		"physicalBytes": 11097041777,
		"blobCount": 1332
	}`)}
	cli := &KopiaCLI{Runner: runner}
	got, err := cli.RepoStats(context.Background(), "elements-local")
	if err != nil {
		t.Fatalf("RepoStats: %v", err)
	}
	if got.SizeBytes != 11097041777 || got.BlobCount != 1332 {
		t.Fatalf("stats = %+v, want {SizeBytes:11097041777 BlobCount:1332}", got)
	}
	want := []string{"repo", "stats", "--name", "elements-local", "--json"}
	if !slices.Equal(runner.args[0], want) {
		t.Fatalf("args = %v, want %v", runner.args[0], want)
	}
}

func TestSnapshotCreateParsesResourceKopiaJSONShape(t *testing.T) {
	runner := &recordingRunner{output: []byte(`{
		"id": "kabc123",
		"startTime": "2026-05-31T22:00:00Z",
		"stats": {"totalSize": 11}
	}`)}
	cli := &KopiaCLI{Runner: runner}
	got, err := cli.SnapshotCreate(context.Background(), "nightly", "/stage/fs", SnapshotMetadata{})
	if err != nil {
		t.Fatalf("SnapshotCreate: %v", err)
	}
	if got.ID != "kabc123" || got.Path != "/stage/fs" || got.SizeBytes != 11 {
		t.Fatalf("snapshot = %+v", got)
	}
	want := []string{"snapshot", "create", "--repo", "nightly", "--path", "/stage/fs", "--json"}
	if !slices.Equal(runner.args[0], want) {
		t.Fatalf("args = %v, want %v", runner.args[0], want)
	}
}

func TestSnapshotCreatePassesMetadataFlagsAndNeverASecret(t *testing.T) {
	runner := &recordingRunner{output: []byte(`{"id":"k1","stats":{"totalSize":1}}`)}
	cli := &KopiaCLI{Runner: runner}
	_, err := cli.SnapshotCreate(context.Background(), "nightly", "/stage/fs", SnapshotMetadata{
		Description:    "Data Backup Manager target acme/db run r1",
		OverrideSource: "dbm://acme/db",
		Tags:           []string{"dbm:true", "dbm.run_id:r1"},
	})
	if err != nil {
		t.Fatalf("SnapshotCreate: %v", err)
	}
	want := []string{
		"snapshot", "create", "--repo", "nightly", "--path", "/stage/fs",
		"--description", "Data Backup Manager target acme/db run r1",
		"--override-source", "dbm://acme/db",
		"--tags", "dbm:true", "--tags", "dbm.run_id:r1",
		"--json",
	}
	if !slices.Equal(runner.args[0], want) {
		t.Fatalf("args = %v, want %v", runner.args[0], want)
	}
	// Defense-in-depth: no token that looks like a passphrase/secret flag.
	for _, a := range runner.args[0] {
		if strings.Contains(strings.ToLower(a), "passphrase") || strings.Contains(strings.ToLower(a), "password") {
			t.Fatalf("secret-looking token in argv: %q", a)
		}
	}
}

func TestBrowseSnapshotUsesResourceKopiaBrowseAndParsesEntries(t *testing.T) {
	runner := &recordingRunner{output: []byte(`[
		{"path": "nested", "sizeBytes": 0, "type": "d", "isDir": true},
		{"path": "root.txt", "sizeBytes": 6, "type": "f", "isDir": false}
	]`)}
	cli := &KopiaCLI{Runner: runner}
	got, err := cli.BrowseSnapshot(context.Background(), "nightly", "snap-abc", "")
	if err != nil {
		t.Fatalf("BrowseSnapshot: %v", err)
	}
	wantArgs := []string{"snapshot", "browse", "--repo", "nightly", "--snapshot", "snap-abc", "--json"}
	if !slices.Equal(runner.args[0], wantArgs) {
		t.Fatalf("args = %v, want %v", runner.args[0], wantArgs)
	}
	if len(got) != 2 {
		t.Fatalf("entries = %+v, want 2", got)
	}
	if got[0].Path != "nested" || !got[0].IsDir {
		t.Fatalf("entry[0] = %+v", got[0])
	}
	if got[1].Path != "root.txt" || got[1].SizeBytes != 6 || got[1].IsDir {
		t.Fatalf("entry[1] = %+v", got[1])
	}
}

// scriptedRunner returns one canned output per call, in order.
type scriptedRunner struct {
	args    [][]string
	outputs []string
	errAt   int // 1-based call index that fails; 0 = never
}

func (r *scriptedRunner) Run(_ context.Context, args ...string) ([]byte, error) {
	r.args = append(r.args, append([]string(nil), args...))
	n := len(r.args)
	if n == r.errAt {
		return nil, errors.New("kopia exploded")
	}
	if n <= len(r.outputs) {
		return []byte(r.outputs[n-1]), nil
	}
	return nil, nil
}

// TestSnapshotCreateReportsTheRecordedSource pins that the engine reports the
// source kopia actually recorded (an override source is absolutized by kopia),
// because that is the only identity retention can act on.
func TestSnapshotCreateReportsTheRecordedSource(t *testing.T) {
	runner := &recordingRunner{output: []byte(`{
		"id": "k1",
		"source": {"host": "box", "userName": "me", "path": "/srv/api/dbm:/acme/db"}
	}`)}
	cli := &KopiaCLI{Runner: runner}
	got, err := cli.SnapshotCreate(context.Background(), "nightly", "/stage/fs", SnapshotMetadata{OverrideSource: "dbm://acme/db"})
	if err != nil {
		t.Fatalf("SnapshotCreate: %v", err)
	}
	if got.Source != "me@box:/srv/api/dbm:/acme/db" {
		t.Fatalf("source = %q", got.Source)
	}
}

// TestApplyRetentionBoundsToKeepLatestAndExpiresNow proves keep-latest N means
// at most N: every GFS bucket is zeroed (else the inherited global policy keeps
// more), the exact source is targeted, expiry runs now, and the remaining count
// is reported from that source.
func TestApplyRetentionBoundsToKeepLatestAndExpiresNow(t *testing.T) {
	const source = "me@box:/srv/api/dbm:/acme/db"
	runner := &scriptedRunner{outputs: []string{"", "", `[{"id":"a"},{"id":"b"},{"id":"c"}]`}}
	cli := &KopiaCLI{Runner: runner}
	remaining, err := cli.ApplyRetention(context.Background(), "nightly", source, 3)
	if err != nil {
		t.Fatalf("ApplyRetention: %v", err)
	}
	if remaining != 3 {
		t.Fatalf("remaining = %d, want 3", remaining)
	}
	want := [][]string{
		{
			"policy", "set", "--repo", "nightly", "--path", source, "--keep-latest", "3",
			"--keep-hourly", "0", "--keep-daily", "0", "--keep-weekly", "0", "--keep-monthly", "0", "--keep-annual", "0",
		},
		{"snapshot", "expire", "--repo", "nightly", "--path", source, "--delete"},
		{"snapshot", "list", "--repo", "nightly", "--json", "--path", source},
	}
	if len(runner.args) != len(want) {
		t.Fatalf("calls = %v, want %v", runner.args, want)
	}
	for i := range want {
		if !slices.Equal(runner.args[i], want[i]) {
			t.Fatalf("call %d = %v, want %v", i, runner.args[i], want[i])
		}
	}
}

func TestApplyRetentionRefusesUnknownSourceOrBound(t *testing.T) {
	runner := &scriptedRunner{}
	cli := &KopiaCLI{Runner: runner}
	if _, err := cli.ApplyRetention(context.Background(), "nightly", "", 3); err == nil {
		t.Fatal("empty source must fail, never fall back to a guessed path")
	}
	if _, err := cli.ApplyRetention(context.Background(), "nightly", "me@box:/x", 0); err == nil {
		t.Fatal("keep-latest 0 must fail, never zero every bucket")
	}
	if len(runner.args) != 0 {
		t.Fatalf("no engine call may run on invalid input: %v", runner.args)
	}
}

func TestApplyRetentionStopsWhenPolicySetFails(t *testing.T) {
	runner := &scriptedRunner{errAt: 1}
	cli := &KopiaCLI{Runner: runner}
	if _, err := cli.ApplyRetention(context.Background(), "nightly", "me@box:/x", 3); err == nil {
		t.Fatal("policy failure must surface")
	}
	if len(runner.args) != 1 {
		t.Fatalf("expiry must not run under an unapplied policy: %v", runner.args)
	}
}
