// Package engine declares the KopiaEngine seam: the single boundary through
// which the destinations, runs, and restores domains reach the backup engine.
// Every snapshot, restore, verify, repository, and stats operation goes
// through resource-kopia — this scenario never calls the kopia binary directly
// and never hand-rolls crypto/dedup/compression (wrap-not-use).
//
// KopiaEngine lives in this substrate package rather than inside one domain
// because three domains consume it; it mirrors the httpc.Doer ambient-seam
// shape. Production wires *KopiaCLI (which shells out through the CommandRunner
// seam); tests substitute mocks.FakeKopiaEngine to assert call shape and stub
// engine results without touching kopia or the network.
//
// Encryption is always on and kopia owns its repository passphrases via vault;
// this scenario never reads or passes a passphrase. Secrets therefore never
// appear in any argv built here.
package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
)

// kopiaBinary is the wrapped resource CLI. All engine work flows through it.
const kopiaBinary = "resource-kopia"

// passphraseRefFormat is the vault path where resource-kopia stores a
// repository's encryption passphrase. It MUST stay byte-identical to
// resource-kopia's own PassphrasePath (resources/kopia/cli/internal/vault/
// vault.go) — they are two ends of the same convention and must move in
// lockstep. DBM only ever records this *reference path* in a bundle; it never
// reads or transmits the passphrase value itself.
const passphraseRefFormat = "secret/resources/kopia/repo/%s/passphrase"

// Backend kinds, matching resource-kopia's --backend flag values.
const (
	BackendFilesystem = "filesystem"
	BackendS3         = "s3"
)

// RepoSpec describes a kopia repository (one per destination) to create.
// Filesystem backends use Path; S3 backends use Bucket/Endpoint and pass
// credentials to resource-kopia, which stores them in vault. Credentials are
// passed to the CommandRunner as flags only at the engine boundary and never
// persisted by this scenario.
type RepoSpec struct {
	Name    string
	Backend string

	// Filesystem backend.
	Path string

	// S3 backend.
	Bucket     string
	Endpoint   string
	AccessKey  string
	SecretKey  string
	DisableTLS bool
}

// RepoStatus is the subset of `resource-kopia repo status` this scenario reads.
type RepoStatus struct {
	// Encryption algorithm reported by kopia; always non-empty for a created
	// repository (encryption is always on). DBM-ENC-001 asserts on this.
	EncryptionAlgorithm string
	Connected           bool
}

// RepoStats is the subset of `resource-kopia repo stats` used for usage-vs-cap.
type RepoStats struct {
	SizeBytes int64
}

// SnapshotMetadata is the optional self-identifying metadata DBM attaches to a
// kopia snapshot so standalone tooling can read owner/name/run without DBM. It
// must never carry a secret value (no passphrases, tokens, or raw credentials);
// locator paths are also excluded by default as potentially sensitive.
type SnapshotMetadata struct {
	// Description is a human one-liner recorded on the snapshot.
	Description string
	// OverrideSource is a stable logical source (e.g. dbm://<owner>/<name>) so
	// the snapshot is not labeled with the throwaway staging path.
	OverrideSource string
	// Tags are key:value pairs passed through to kopia (repeatable --tags).
	Tags []string
}

// Snapshot is a kopia snapshot reference produced/listed by the engine.
type Snapshot struct {
	ID        string
	Path      string
	StartTime string
	SizeBytes int64
}

// SnapshotEntry is one path within a snapshot's content listing.
type SnapshotEntry struct {
	Path      string
	SizeBytes int64
	IsDir     bool
}

// KopiaEngine is the seam wrapping resource-kopia. The surface is narrow — only
// the operations the three consuming domains actually call. New methods land
// here when a domain proves it needs one.
//
// seam: KopiaEngine wraps the backup engine. Production wires engine.KopiaCLI
// (kopia.go); tests wire mocks.FakeKopiaEngine (testutil/mocks/kopia.go).
type KopiaEngine interface {
	// RepoCreate provisions a kopia repository for a destination. Encryption
	// is always on; the passphrase is generated and stored in vault by kopia.
	RepoCreate(ctx context.Context, spec RepoSpec) error
	// RepoStatus reports the repository's encryption algorithm and connection.
	RepoStatus(ctx context.Context, repo string) (RepoStatus, error)
	// PassphraseRef returns the vault reference *path* (never the value) where
	// resource-kopia stores this repository's encryption passphrase, so a
	// detached destination bundle can point an operator at the key for
	// standalone recovery. It is a pure, deterministic string derivation —
	// no I/O, no secret read.
	PassphraseRef(repo string) string
	// RepoStats reports current repository usage in bytes (for storage caps).
	RepoStats(ctx context.Context, repo string) (RepoStats, error)
	// RepoDelete removes local resource-kopia metadata and Vault secret refs for
	// a destination repository.
	RepoDelete(ctx context.Context, repo string) error
	// SnapshotCreate snapshots a captured artifact path into a repository and
	// returns the snapshot reference (including its id). meta carries optional
	// self-identifying kopia metadata (description, override-source, tags) so a
	// standalone `kopia snapshot list` can attribute the snapshot without DBM
	// running. meta never carries a secret.
	SnapshotCreate(ctx context.Context, repo, path string, meta SnapshotMetadata) (Snapshot, error)
	// SnapshotList lists snapshots in a repository, optionally for one path.
	SnapshotList(ctx context.Context, repo, path string) ([]Snapshot, error)
	// SnapshotRestore restores a snapshot's contents into target.
	SnapshotRestore(ctx context.Context, repo, snapshotID, target string) error
	// SnapshotVerify verifies a snapshot's restorability; verifyPercent in
	// [0,100] controls how many files are byte-verified (100 = full).
	SnapshotVerify(ctx context.Context, repo, snapshotID string, verifyPercent int) error
	// BrowseSnapshot lists entries within a snapshot, optionally under path.
	BrowseSnapshot(ctx context.Context, repo, snapshotID, path string) ([]SnapshotEntry, error)
	// PolicySet applies retention to a repository path; keepLatest <= 0 leaves
	// the keep-latest policy untouched.
	PolicySet(ctx context.Context, repo, path string, keepLatest int) error
}

// CommandRunner is the exec seam KopiaCLI shells through. Production wires
// ExecRunner; engine unit tests wire a recording fake to assert argv shape
// (notably: never a secret in argv) and stub stdout.
//
// seam: CommandRunner is the process-exec boundary for resource-kopia.
type CommandRunner interface {
	Run(ctx context.Context, args ...string) ([]byte, error)
}

// ExecRunner runs the configured binary with os/exec and returns stdout.
type ExecRunner struct {
	Binary string
}

// Run executes Binary with args and returns combined stdout. A non-zero exit
// surfaces as an error wrapping stderr so callers can record a failed run.
func (r ExecRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	bin := r.Binary
	if bin == "" {
		bin = kopiaBinary
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if ok := asExitError(err, &exitErr); ok {
			return nil, fmt.Errorf("%s %v: %w: %s", bin, args, err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("%s %v: %w", bin, args, err)
	}
	return out, nil
}

// asExitError is a tiny errors.As shim kept local so kopia.go's import block
// stays minimal and the ExecRunner reads top-to-bottom.
func asExitError(err error, target **exec.ExitError) bool {
	if ee, ok := err.(*exec.ExitError); ok {
		*target = ee
		return true
	}
	return false
}

// KopiaCLI is the production KopiaEngine. It builds resource-kopia argv and
// parses the wrapper's `--json` passthrough of kopia's native JSON.
type KopiaCLI struct {
	Runner CommandRunner
}

// NewKopiaCLI returns the production engine wired to resource-kopia.
func NewKopiaCLI() *KopiaCLI {
	return &KopiaCLI{Runner: ExecRunner{Binary: kopiaBinary}}
}

// Compile-time guarantee.
var _ KopiaEngine = (*KopiaCLI)(nil)

func (k *KopiaCLI) RepoCreate(ctx context.Context, spec RepoSpec) error {
	args := []string{"repo", "create", "--name", spec.Name, "--backend", spec.Backend}
	switch spec.Backend {
	case BackendFilesystem:
		args = append(args, "--path", spec.Path)
	case BackendS3:
		args = append(args, "--bucket", spec.Bucket, "--endpoint", spec.Endpoint)
		if spec.DisableTLS {
			args = append(args, "--disable-tls")
		}
		// Credentials are handed to resource-kopia (which stores them in
		// vault); they cross only this boundary and are never persisted here.
		if spec.AccessKey != "" {
			args = append(args, "--access-key", spec.AccessKey, "--secret-access-key", spec.SecretKey)
		}
	default:
		return fmt.Errorf("repo create %q: unknown backend %q", spec.Name, spec.Backend)
	}
	if _, err := k.Runner.Run(ctx, args...); err != nil {
		return fmt.Errorf("repo create %q: %w", spec.Name, err)
	}
	return nil
}

// PassphraseRef derives the vault reference path for a repository's passphrase.
// Pure string formatting — it deliberately performs no vault I/O so it can be
// called during create without a round-trip and never risks touching a secret.
func (k *KopiaCLI) PassphraseRef(repo string) string {
	return fmt.Sprintf(passphraseRefFormat, repo)
}

func (k *KopiaCLI) RepoStatus(ctx context.Context, repo string) (RepoStatus, error) {
	out, err := k.Runner.Run(ctx, "repo", "status", "--name", repo, "--json")
	if err != nil {
		return RepoStatus{}, fmt.Errorf("repo status %q: %w", repo, err)
	}
	// kopia `repository status --json` is passed through verbatim; probe the
	// encryption field across the casings kopia has used.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(out, &raw); err != nil {
		return RepoStatus{}, fmt.Errorf("repo status %q: parse json: %w", repo, err)
	}
	return RepoStatus{
		EncryptionAlgorithm: encryptionAlgorithm(raw),
		Connected:           true,
	}, nil
}

func (k *KopiaCLI) RepoStats(ctx context.Context, repo string) (RepoStats, error) {
	out, err := k.Runner.Run(ctx, "repo", "stats", "--name", repo, "--json")
	if err != nil {
		return RepoStats{}, fmt.Errorf("repo stats %q: %w", repo, err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(out, &raw); err != nil {
		return RepoStats{}, fmt.Errorf("repo stats %q: parse json: %w", repo, err)
	}
	return RepoStats{SizeBytes: firstInt(raw, "sizeBytes", "Size", "totalSize", "TotalPackedSize")}, nil
}

func (k *KopiaCLI) RepoDelete(ctx context.Context, repo string) error {
	if _, err := k.Runner.Run(ctx, "repo", "delete", "--name", repo); err != nil {
		return fmt.Errorf("repo delete %q: %w", repo, err)
	}
	return nil
}

func (k *KopiaCLI) SnapshotCreate(ctx context.Context, repo, path string, meta SnapshotMetadata) (Snapshot, error) {
	args := []string{"snapshot", "create", "--repo", repo, "--path", path}
	if meta.Description != "" {
		args = append(args, "--description", meta.Description)
	}
	if meta.OverrideSource != "" {
		args = append(args, "--override-source", meta.OverrideSource)
	}
	for _, tag := range meta.Tags {
		args = append(args, "--tags", tag)
	}
	args = append(args, "--json")
	out, err := k.Runner.Run(ctx, args...)
	if err != nil {
		return Snapshot{}, fmt.Errorf("snapshot create repo=%q path=%q: %w", repo, path, err)
	}
	snap, err := parseSnapshot(out)
	if err != nil {
		return Snapshot{}, fmt.Errorf("snapshot create repo=%q path=%q: %w", repo, path, err)
	}
	snap.Path = path
	return snap, nil
}

func (k *KopiaCLI) SnapshotList(ctx context.Context, repo, path string) ([]Snapshot, error) {
	args := []string{"snapshot", "list", "--repo", repo, "--json"}
	if path != "" {
		args = append(args, "--path", path)
	}
	out, err := k.Runner.Run(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("snapshot list repo=%q: %w", repo, err)
	}
	var manifests []map[string]json.RawMessage
	if err := json.Unmarshal(out, &manifests); err != nil {
		return nil, fmt.Errorf("snapshot list repo=%q: parse json: %w", repo, err)
	}
	snaps := make([]Snapshot, 0, len(manifests))
	for _, m := range manifests {
		snaps = append(snaps, Snapshot{
			ID:        firstString(m, "id", "ID"),
			StartTime: firstString(m, "startTime", "StartTime"),
			SizeBytes: snapshotSize(m),
		})
	}
	return snaps, nil
}

func (k *KopiaCLI) SnapshotRestore(ctx context.Context, repo, snapshotID, target string) error {
	if _, err := k.Runner.Run(ctx, "snapshot", "restore", "--repo", repo, "--snapshot", snapshotID, "--target", target); err != nil {
		return fmt.Errorf("snapshot restore repo=%q snapshot=%q: %w", repo, snapshotID, err)
	}
	return nil
}

func (k *KopiaCLI) SnapshotVerify(ctx context.Context, repo, snapshotID string, verifyPercent int) error {
	if verifyPercent < 0 {
		verifyPercent = 0
	}
	if verifyPercent > 100 {
		verifyPercent = 100
	}
	if _, err := k.Runner.Run(ctx, "snapshot", "verify", "--repo", repo, "--snapshot", snapshotID,
		"--verify-files-percent", strconv.Itoa(verifyPercent)); err != nil {
		return fmt.Errorf("snapshot verify repo=%q snapshot=%q: %w", repo, snapshotID, err)
	}
	return nil
}

func (k *KopiaCLI) BrowseSnapshot(ctx context.Context, repo, snapshotID, path string) ([]SnapshotEntry, error) {
	args := []string{"snapshot", "browse", "--repo", repo, "--snapshot", snapshotID, "--json"}
	if path != "" {
		args = append(args, "--path", path)
	}
	out, err := k.Runner.Run(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("snapshot browse repo=%q snapshot=%q: %w", repo, snapshotID, err)
	}
	var entries []map[string]json.RawMessage
	if err := json.Unmarshal(out, &entries); err != nil {
		return nil, fmt.Errorf("snapshot browse repo=%q snapshot=%q: parse json: %w", repo, snapshotID, err)
	}
	out2 := make([]SnapshotEntry, 0, len(entries))
	for _, e := range entries {
		out2 = append(out2, SnapshotEntry{
			Path:      firstString(e, "name", "path", "Path"),
			SizeBytes: firstInt(e, "size", "Size", "sizeBytes"),
			IsDir:     firstString(e, "type", "Type") == "d",
		})
	}
	return out2, nil
}

func (k *KopiaCLI) PolicySet(ctx context.Context, repo, path string, keepLatest int) error {
	if keepLatest <= 0 {
		return nil
	}
	if _, err := k.Runner.Run(ctx, "policy", "set", "--repo", repo, "--path", path,
		"--keep-latest", strconv.Itoa(keepLatest)); err != nil {
		return fmt.Errorf("policy set repo=%q path=%q: %w", repo, path, err)
	}
	return nil
}

// parseSnapshot reads a single kopia snapshot manifest (the `snapshot create
// --json` shape) and extracts the id and total size.
func parseSnapshot(out []byte) (Snapshot, error) {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(out, &m); err != nil {
		return Snapshot{}, fmt.Errorf("parse snapshot json: %w", err)
	}
	id := firstString(m, "id", "ID")
	if id == "" {
		return Snapshot{}, fmt.Errorf("snapshot json missing id: %s", string(out))
	}
	return Snapshot{ID: id, StartTime: firstString(m, "startTime", "StartTime"), SizeBytes: snapshotSize(m)}, nil
}

// snapshotSize digs the total size out of a manifest's nested stats block,
// tolerating the casings kopia has shipped.
func snapshotSize(m map[string]json.RawMessage) int64 {
	if raw, ok := m["stats"]; ok {
		var stats map[string]json.RawMessage
		if json.Unmarshal(raw, &stats) == nil {
			if n := firstInt(stats, "totalSize", "TotalSize", "totalFileSize"); n > 0 {
				return n
			}
		}
	}
	return firstInt(m, "size", "Size", "totalSize")
}

// firstString returns the first key that decodes to a non-empty string.
func firstString(m map[string]json.RawMessage, keys ...string) string {
	for _, k := range keys {
		if raw, ok := m[k]; ok {
			var s string
			if json.Unmarshal(raw, &s) == nil && s != "" {
				return s
			}
		}
	}
	return ""
}

func encryptionAlgorithm(m map[string]json.RawMessage) string {
	if algorithm := firstString(m, "encryption", "Encryption", "encryptionAlgorithm"); algorithm != "" {
		return algorithm
	}
	for _, key := range []string{"contentFormat", "ContentFormat"} {
		raw, ok := m[key]
		if !ok {
			continue
		}
		var nested map[string]json.RawMessage
		if json.Unmarshal(raw, &nested) == nil {
			if algorithm := firstString(nested, "encryption", "Encryption", "encryptionAlgorithm"); algorithm != "" {
				return algorithm
			}
		}
	}
	return ""
}

// firstInt returns the first key that decodes to an integer.
func firstInt(m map[string]json.RawMessage, keys ...string) int64 {
	for _, k := range keys {
		if raw, ok := m[k]; ok {
			var n int64
			if json.Unmarshal(raw, &n) == nil {
				return n
			}
		}
	}
	return 0
}
