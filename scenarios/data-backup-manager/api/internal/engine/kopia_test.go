package engine

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

type captureRunner struct{ args []string }

func (r *captureRunner) Run(_ context.Context, args ...string) ([]byte, error) {
	r.args = append([]string(nil), args...)
	return nil, nil
}

func TestRepoCreatePassesSFTPMetadataWithoutCredentialValues(t *testing.T) {
	runner := &captureRunner{}
	cli := &KopiaCLI{Runner: runner}
	err := cli.RepoCreate(context.Background(), RepoSpec{
		Name: "remote", Backend: BackendSFTP,
		SFTPHost: "backup.example", SFTPPort: 2222, SFTPUsername: "vrooli",
		SFTPRemotePath: "/backups/vrooli", SFTPKnownHosts: "/home/operator/.ssh/known_hosts",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"repo", "create", "--name", "remote", "--backend", "sftp", "--host", "backup.example", "--port", "2222", "--username", "vrooli", "--remote-path", "/backups/vrooli", "--known-hosts", "/home/operator/.ssh/known_hosts"}
	if !reflect.DeepEqual(runner.args, want) {
		t.Fatalf("resource-kopia args = %#v, want %#v", runner.args, want)
	}
}

func TestRepoCreatePassesProviderS3MetadataWithoutCredentials(t *testing.T) {
	runner := &captureRunner{}
	cli := &KopiaCLI{Runner: runner}
	err := cli.RepoCreate(context.Background(), RepoSpec{
		Name: "remote", Backend: BackendS3, Bucket: "backups", Endpoint: "objects.example:9443",
		Prefix: "vrooli/host-a", Region: "eu-west-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"repo", "create", "--name", "remote", "--backend", "s3", "--bucket", "backups", "--endpoint", "objects.example:9443", "--prefix", "vrooli/host-a", "--region", "eu-west-1"}
	if !reflect.DeepEqual(runner.args, want) {
		t.Fatalf("resource-kopia args = %#v, want %#v", runner.args, want)
	}
}

func TestRepoConnectUsesExistingRepositoryVerbAndNonSecretBackendMetadata(t *testing.T) {
	for _, tc := range []struct {
		name string
		spec RepoSpec
		want []string
	}{
		{name: "S3", spec: RepoSpec{Name: "remote", Backend: BackendS3, Bucket: "backups", Endpoint: "objects.example:9443", Prefix: "vrooli/node-a", Region: "us-east-1"}, want: []string{"repo", "connect", "--name", "remote", "--backend", "s3", "--bucket", "backups", "--endpoint", "objects.example:9443", "--prefix", "vrooli/node-a", "--region", "us-east-1"}},
		{name: "SFTP", spec: RepoSpec{Name: "remote", Backend: BackendSFTP, SFTPHost: "backup.example", SFTPPort: 2222, SFTPUsername: "vrooli", SFTPRemotePath: "/backups/vrooli", SFTPKnownHosts: "/home/operator/.ssh/known_hosts"}, want: []string{"repo", "connect", "--name", "remote", "--backend", "sftp", "--host", "backup.example", "--port", "2222", "--username", "vrooli", "--remote-path", "/backups/vrooli", "--known-hosts", "/home/operator/.ssh/known_hosts"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runner := &captureRunner{}
			cli := &KopiaCLI{Runner: runner}
			if err := cli.RepoConnect(context.Background(), tc.spec); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(runner.args, tc.want) {
				t.Fatalf("resource-kopia args = %#v, want %#v", runner.args, tc.want)
			}
		})
	}
}

type damageRunner struct {
	args []string
	out  []byte
	err  error
}

func (r *damageRunner) Run(_ context.Context, args ...string) ([]byte, error) {
	r.args = append([]string(nil), args...)
	return r.out, r.err
}

func TestRepoDamageScanUsesResourceOwnerAndValidatesTypedRelativeEvidence(t *testing.T) {
	runner := &damageRunner{out: []byte(`{"root":"/repo","started_at":"2026-11-13T18:00:00Z","duration":12,"files":9,"bytes":42,"readable_temp_files":1,"unreadable":[{"relative_path":"p/abc.f.tmp.0123456789abcdef","class":"leftover_temp","blob_kind":"data_pack","problem":"io_error","orphan_temp":true,"detail":"input/output error"}]}`)}
	cli := &KopiaCLI{Runner: runner}
	got, err := cli.RepoDamageScan(context.Background(), "primary")
	if err != nil {
		t.Fatal(err)
	}
	wantArgs := []string{"repo", "damage-scan", "--name", "primary", "--json"}
	if !reflect.DeepEqual(runner.args, wantArgs) {
		t.Fatalf("damage scan args = %#v, want %#v", runner.args, wantArgs)
	}
	if got.Files != 9 || got.Bytes != 42 || len(got.Unreadable) != 1 || got.Unreadable[0].Class != "leftover_temp" || !got.Unreadable[0].Orphan {
		t.Fatalf("typed damage result = %+v", got)
	}

	runner.out = []byte(`{"root":"/repo","started_at":"2026-11-13T18:00:00Z","unreadable":[{"relative_path":"../outside","class":"other","problem":"other"}]}`)
	if _, err := cli.RepoDamageScan(context.Background(), "primary"); err == nil {
		t.Fatal("owner scan path traversal must be rejected")
	}
	for name, raw := range map[string]string{
		"negative aggregate":           `{"root":"/repo","started_at":"2026-11-13T18:00:00Z","files":-1}`,
		"non-canonical separator":      `{"root":"/repo","started_at":"2026-11-13T18:00:00Z","unreadable":[{"relative_path":"p\\a.f","class":"repository_blob","blob_kind":"data_pack","problem":"io_error"}]}`,
		"path classification mismatch": `{"root":"/repo","started_at":"2026-11-13T18:00:00Z","unreadable":[{"relative_path":"p/a.f.tmp","class":"leftover_temp","blob_kind":"metadata_pack","problem":"io_error","orphan_temp":true}]}`,
		"duplicate path":               `{"root":"/repo","started_at":"2026-11-13T18:00:00Z","unreadable":[{"relative_path":"z/a","class":"other","blob_kind":"other","problem":"other"},{"relative_path":"z/a","class":"other","blob_kind":"other","problem":"other"}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			runner.out = []byte(raw)
			if _, err := cli.RepoDamageScan(context.Background(), "primary"); err == nil {
				t.Fatal("malformed owner damage evidence was accepted")
			}
		})
	}
	runner.out = []byte(`{"root":"/repo","started_at":"2026-11-13T18:00:00Z","unreadable":[{"relative_path":"s","class":"other","blob_kind":"other","problem":"io_error"}]}`)
	if _, err := cli.RepoDamageScan(context.Background(), "primary"); err != nil {
		t.Fatalf("unreadable shard directory is valid generic evidence: %v", err)
	}
	runner.err = errors.New("device disappeared")
	if _, err := cli.RepoDamageScan(context.Background(), "primary"); err == nil {
		t.Fatal("owner scan errors must not become a clean scan")
	}
}

func TestBrowseSnapshotRejectsOversizedManifestAndPreservesEntryType(t *testing.T) {
	t.Run("oversized JSON", func(t *testing.T) {
		runner := &damageRunner{out: []byte(`[` + strings.Repeat(`{"path":"x","sizeBytes":0,"type":"f"},`, 70000) + `]`)}
		cli := &KopiaCLI{Runner: runner}
		if _, err := cli.BrowseSnapshot(context.Background(), "repo", "snap", ""); err == nil {
			t.Fatal("oversized browse output must refuse before decoding")
		}
	})
	t.Run("special type retained", func(t *testing.T) {
		runner := &damageRunner{out: []byte(`[{"path":"pipe","sizeBytes":0,"type":"p"}]`)}
		cli := &KopiaCLI{Runner: runner}
		entries, err := cli.BrowseSnapshot(context.Background(), "repo", "snap", "")
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 1 || entries[0].Type != "p" || entries[0].IsDir {
			t.Fatalf("entry type was lost: %+v", entries)
		}
	})
}

func TestSnapshotListByTagsBuildsCrossHostTagFilter(t *testing.T) {
	runner := &damageRunner{out: []byte(`[]`)}
	_, err := (&KopiaCLI{Runner: runner}).SnapshotListByTags(context.Background(), "repo", []string{"dbm.owner:agent-manager", "dbm.name:data"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"snapshot", "list", "--repo", "repo", "--all", "--tags", "dbm.owner:agent-manager", "--tags", "dbm.name:data", "--json"}
	if !reflect.DeepEqual(runner.args, want) {
		t.Fatalf("args = %#v, want %#v", runner.args, want)
	}
}

func TestSnapshotListParsesKopiaObjectTags(t *testing.T) {
	runner := &damageRunner{out: []byte(`[{"id":"s","source":{"host":"h","userName":"u","path":"/p"},"tags":{"tag:dbm.owner":"agent-manager","tag:dbm.name":"data"},"stats":{"totalSize":1,"fileCount":1,"dirCount":1},"rootEntry":{"summ":{"symlinks":0}}}]`)}
	rows, err := (&KopiaCLI{Runner: runner}).SnapshotList(context.Background(), "repo", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want one", len(rows))
	}
	want := []string{"dbm.name:data", "dbm.owner:agent-manager"}
	if len(rows[0].Tags) != len(want) || rows[0].Tags[0] != want[0] || rows[0].Tags[1] != want[1] {
		t.Fatalf("tags = %v, want %v", rows[0].Tags, want)
	}
}

func TestSnapshotListParsesAggregateEntryCounts(t *testing.T) {
	runner := &damageRunner{out: []byte(`[{"id":"snap","source":{"host":"h","userName":"u","path":"/p"},"stats":{"totalSize":8,"fileCount":2,"dirCount":3},"rootEntry":{"summ":{"symlinks":4}}}]`)}
	cli := &KopiaCLI{Runner: runner}
	snapshots, err := cli.SnapshotList(context.Background(), "repo", "/p")
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshots) != 1 || snapshots[0].FileCount != 2 || snapshots[0].DirCount != 3 || snapshots[0].SymlinkCount != 4 {
		t.Fatalf("Kopia aggregate counts were not preserved: %+v", snapshots)
	}
}

func TestSnapshotListBoundsInventoryAndRequiresStatsTotalSize(t *testing.T) {
	t.Run("oversized inventory bytes", func(t *testing.T) {
		runner := &damageRunner{out: []byte(strings.Repeat(" ", 32<<20+1))}
		if _, err := (&KopiaCLI{Runner: runner}).SnapshotList(context.Background(), "repo", "/target"); err == nil {
			t.Fatal("oversized snapshot inventory must refuse")
		}
	})
	t.Run("large valid inventories are enumerable", func(t *testing.T) {
		one := `{"id":"s","stats":{"totalSize":1,"fileCount":1,"dirCount":1},"rootEntry":{"summ":{"symlinks":0}}}`
		runner := &damageRunner{out: []byte("[" + strings.TrimSuffix(strings.Repeat(one+",", 300), ",") + "]")}
		rows, err := (&KopiaCLI{Runner: runner}).SnapshotList(context.Background(), "repo", "/target")
		if err != nil {
			t.Fatalf("large valid inventory was refused: %v", err)
		}
		if len(rows) != 300 {
			t.Fatalf("rows = %d, want 300", len(rows))
		}
	})
	t.Run("missing total size despite compatibility field", func(t *testing.T) {
		runner := &damageRunner{out: []byte(`[{"id":"s","totalSize":1,"stats":{"fileCount":1,"dirCount":1},"rootEntry":{"summ":{"symlinks":0}}}]`)}
		rows, err := (&KopiaCLI{Runner: runner}).SnapshotList(context.Background(), "repo", "/target")
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != 1 || rows[0].StatsTotalSize != -1 {
			t.Fatalf("stats total size was inferred from fallback: %+v", rows)
		}
	})
}

func TestRepoQuarantineTempRequiresExactSessionPathAndVerifiesOwnerResult(t *testing.T) {
	runner := &damageRunner{out: []byte(`{"repository":"elements","original_path":"/repo/s/abc/blob.f.tmp.deadbeef","quarantine_path":"/repo-recovery-quarantine/s/abc/blob.f.tmp.deadbeef","final_blob_path":"/repo/s/abc/blob.f","source_data_deleted":false,"final_blob_written":false,"quarantine_same_volume":true,"committed":true}`)}
	cli := &KopiaCLI{Runner: runner}
	got, err := cli.RepoQuarantineTempBlob(context.Background(), "elements", "s/abc/blob.f.tmp.deadbeef")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"repo", "quarantine-orphan-temp-blob", "--name", "elements", "--relative-temp-path", "s/abc/blob.f.tmp.deadbeef", "--confirm-quarantine", "quarantine-orphan-temp:elements:s/abc/blob.f.tmp.deadbeef", "--json"}
	if !reflect.DeepEqual(runner.args, want) || !got.Committed || got.SourceDataDeleted || got.FinalBlobWritten {
		t.Fatalf("quarantine call/result = %#v / %+v", runner.args, got)
	}
	runner.args = nil
	runner.out = []byte(`{"repository":"elements","original_path":"/repo/s/abc/blob.f.tmp","quarantine_path":"/repo-recovery-quarantine/s/abc/blob.f.tmp","final_blob_path":"/repo/s/abc/blob.f","source_data_deleted":false,"final_blob_written":false,"quarantine_same_volume":true,"committed":true}`)
	if _, err := cli.RepoQuarantineTempBlob(context.Background(), "elements", "s/abc/blob.f.tmp"); err != nil {
		t.Fatalf("suffixless Kopia temp should reach the owner: %v", err)
	}
	wantSuffixless := []string{"repo", "quarantine-orphan-temp-blob", "--name", "elements", "--relative-temp-path", "s/abc/blob.f.tmp", "--confirm-quarantine", "quarantine-orphan-temp:elements:s/abc/blob.f.tmp", "--json"}
	if !reflect.DeepEqual(runner.args, wantSuffixless) {
		t.Fatalf("suffixless quarantine args = %#v, want %#v", runner.args, wantSuffixless)
	}
	runner.args = nil
	for _, unsafe := range []string{"../outside/s/blob.f.tmp.deadbeef", "s/abc/blob.f.tmp.not-a-hash", "s/blob.f.tmp.deadbeef", "s/abc/blob.f.tmp.deadbeef/extra"} {
		if _, err := cli.RepoQuarantineTempBlob(context.Background(), "elements", unsafe); err == nil || len(runner.args) != 0 {
			t.Fatalf("unsafe relative path %q reached resource-kopia: args=%v err=%v", unsafe, runner.args, err)
		}
	}
	if len(runner.args) != 0 {
		t.Fatalf("unsafe relative path reached resource-kopia: args=%v", runner.args)
	}
	runner.out = []byte(`{"repository":"elements","original_path":"/repo/x","quarantine_path":"/repo/q","final_blob_path":"/repo/f","source_data_deleted":true,"quarantine_same_volume":true,"committed":true}`)
	if _, err := cli.RepoQuarantineTempBlob(context.Background(), "elements", "s/abc/blob.f.tmp.deadbeef"); err == nil {
		t.Fatal("owner result claiming source data deletion must be rejected")
	}
}

func TestRepoCacheProbeRequiresOverlayVerifiedExactOrphanQBlob(t *testing.T) {
	hash := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	runner := &damageRunner{out: []byte(`{"status":"snapshot_list_failed","orphan_temp_blob_candidate":{"blob_id":"q1234","candidate_path":"/private/repo/q.tmp.hash","candidate_bytes":99,"candidate_sha256":"` + hash + `","final_blob_absent":true,"source_repository_read_only":true,"overlay_snapshot_status":"snapshot_list_passed"}}`)}
	cli := &KopiaCLI{Runner: runner}
	got, err := cli.RepoCacheProbe(context.Background(), "elements", "q1234")
	if err != nil {
		t.Fatal(err)
	}
	wantArgs := []string{"repo", "cache-probe", "--name", "elements", "--blob-id", "q1234", "--json"}
	if !reflect.DeepEqual(runner.args, wantArgs) || got.CandidateSHA256 != hash || got.OverlaySnapshotStatus != "snapshot_list_passed" {
		t.Fatalf("cache probe args/result = %#v / %+v", runner.args, got)
	}
	runner.args = nil
	if _, err := cli.RepoCacheProbe(context.Background(), "elements", "../q1234"); err == nil || len(runner.args) != 0 {
		t.Fatalf("invalid blob id reached owner: args=%v err=%v", runner.args, err)
	}
	runner.out = []byte(`{"status":"snapshot_list_failed","orphan_temp_blob_candidate":{"blob_id":"q1234","candidate_path":"/repo/tmp","candidate_bytes":99,"candidate_sha256":"` + hash + `","final_blob_absent":true,"source_repository_read_only":true,"overlay_snapshot_status":"snapshot_list_failed"}}`)
	if _, err := cli.RepoCacheProbe(context.Background(), "elements", "q1234"); err == nil {
		t.Fatal("candidate without a successful disposable overlay must not qualify")
	}
}

func TestRepoCacheProbeReturnsTypedMissingBlobWithoutCandidate(t *testing.T) {
	runner := &damageRunner{out: []byte(`{"status":"snapshot_list_failed","blob":{"blob_id":"q1234","relative_path":"q/12/34.f","state":"missing","orphan_temp":false}}`)}
	cli := &KopiaCLI{Runner: runner}
	got, err := cli.RepoCacheProbe(context.Background(), "elements", "q1234")
	if err != nil {
		t.Fatalf("typed missing blob should be returned as evidence: %v", err)
	}
	if got.Blob == nil || got.Blob.BlobID != "q1234" || got.Blob.State != "missing" || got.Blob.RelativePath != "q/12/34.f" || got.CandidateBytes != 0 {
		t.Fatalf("typed missing blob result = %+v", got)
	}
	runner.out = []byte(`{"status":"snapshot_list_failed","blob":{"blob_id":"q1234","relative_path":"/outside/repo","state":"missing","orphan_temp":false}}`)
	if _, err := cli.RepoCacheProbe(context.Background(), "elements", "q1234"); err == nil {
		t.Fatal("absolute repository path in owner evidence must be rejected")
	}
}

func TestRepoOrphanBlobCommitRejectsOverwriteAndVerifiesExactHash(t *testing.T) {
	hash := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	runner := &damageRunner{out: []byte(`{"repository":"elements","blob_id":"q1234","candidate_sha256":"` + hash + `","published_bytes":99,"candidate_preserved":true,"target_replaced":false,"committed":true}`)}
	cli := &KopiaCLI{Runner: runner}
	got, err := cli.RepoOrphanBlobCommit(context.Background(), "elements", "q1234", hash)
	if err != nil {
		t.Fatal(err)
	}
	wantArgs := []string{"repo", "orphan-temp-blob-commit", "--name", "elements", "--blob-id", "q1234", "--candidate-sha256", hash, "--confirm-commit", "restore-orphan-blob:elements:q1234", "--json"}
	if !reflect.DeepEqual(runner.args, wantArgs) || !got.Committed || got.TargetReplaced || !got.CandidatePreserved {
		t.Fatalf("orphan recovery args/result = %#v / %+v", runner.args, got)
	}
	runner.out = []byte(`{"repository":"elements","blob_id":"q1234","candidate_sha256":"` + hash + `","published_bytes":99,"candidate_preserved":true,"target_replaced":true,"committed":true}`)
	if _, err := cli.RepoOrphanBlobCommit(context.Background(), "elements", "q1234", hash); err == nil {
		t.Fatal("owner result claiming overwrite must be rejected")
	}
}

func TestRepoRestoreTempBindsCanonicalRepositoryAndExactSourceAndTarget(t *testing.T) {
	hash := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	repoPath := "/mnt/Elements/repositories/primary.kopia"
	rel := "q/1234.f.tmp.abcd1234"
	runner := &damageRunner{out: []byte(`{"repository":"elements","relative_temp_path":"` + rel + `","source_path":"` + repoPath + `/` + rel + `","target_path":"` + repoPath + `/q/1234.f","sha256":"` + hash + `","bytes":99,"source_preserved":true,"target_replaced":false,"replay":false,"committed":true}`)}
	cli := &KopiaCLI{Runner: runner}
	got, err := cli.RepoRestoreTemp(context.Background(), "elements", repoPath, rel, hash)
	if err != nil {
		t.Fatal(err)
	}
	wantArgs := []string{
		"repo", "restore-from-temp", "--name", "elements", "--repository-path", repoPath,
		"--relative-temp-path", rel, "--temp-sha256", hash,
		"--confirm-restore", "restore-temp:elements:" + repoPath + ":" + rel + ":" + hash, "--json",
	}
	if !reflect.DeepEqual(runner.args, wantArgs) || got.RelativePath != rel || !got.SourcePreserved || got.TargetReplaced || !got.Committed {
		t.Fatalf("temp restore args/result = %#v / %+v", runner.args, got)
	}
	runner.out = []byte(`{"repository":"elements","relative_temp_path":"` + rel + `","source_path":"` + repoPath + `/` + rel + `","target_path":"` + repoPath + `/q/9999.f","sha256":"` + hash + `","bytes":99,"source_preserved":true,"target_replaced":false,"committed":true}`)
	if _, err := cli.RepoRestoreTemp(context.Background(), "elements", repoPath, rel, hash); err == nil {
		t.Fatal("owner result naming a different final target must be rejected")
	}
}

func TestRepoRestoreTempRejectsUnsafePathsBeforeOwnerCall(t *testing.T) {
	runner := &damageRunner{}
	cli := &KopiaCLI{Runner: runner}
	hash := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	if _, err := cli.RepoRestoreTemp(context.Background(), "elements", "/repo", "../q/1234.f.tmp", hash); err == nil {
		t.Fatal("path traversal must be rejected")
	}
	if len(runner.args) != 0 {
		t.Fatalf("unsafe path reached resource-kopia: %#v", runner.args)
	}
}
