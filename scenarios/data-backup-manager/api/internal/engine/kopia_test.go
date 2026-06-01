package engine

import (
	"context"
	"encoding/json"
	"slices"
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

func TestSnapshotCreateParsesResourceKopiaJSONShape(t *testing.T) {
	runner := &recordingRunner{output: []byte(`{
		"id": "kabc123",
		"startTime": "2026-05-31T22:00:00Z",
		"stats": {"totalSize": 11}
	}`)}
	cli := &KopiaCLI{Runner: runner}
	got, err := cli.SnapshotCreate(context.Background(), "nightly", "/stage/fs")
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
