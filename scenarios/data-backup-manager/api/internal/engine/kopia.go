// Package engine declares the KopiaEngine seam: the single boundary through
// which the destinations, runs, and restores domains reach the backup engine.
// Every snapshot, restore, verify, repository, and stats operation goes
// through resource-kopia — this scenario never calls the kopia binary directly
// and never hand-rolls crypto/dedup/compression (wrap-not-use).
//
// KopiaEngine lives in this substrate package rather than inside one domain
// because three domains consume it; it mirrors the outboundhttp.Doer ambient-seam
// shape. Production wires *KopiaCLI (which shells out through the CommandRunner
// seam); tests substitute mocks.FakeKopiaEngine to assert call shape and stub
// engine results without touching kopia or the network.
//
// Encryption is always on and kopia owns its repository passphrases via the
// credential authority;
// this scenario never reads or passes a passphrase. Secrets therefore never
// appear in any argv built here.
package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"data-backup-manager/internal/effectguard"

	kopiaregistry "github.com/vrooli/vrooli/packages/kopiaregistry-go"
)

// kopiaBinary is the wrapped resource CLI. All engine work flows through it.
const kopiaBinary = "resource-kopia"

// Backend kinds, matching resource-kopia's --backend flag values.
const (
	BackendFilesystem = "filesystem"
	BackendSFTP       = "sftp"
	BackendS3         = "s3"
)

// RepoSpec describes a kopia repository (one per destination) to create.
// Filesystem backends use Path; S3 backends use Bucket/Endpoint and resolve
// credentials inside resource-kopia from the credential authority. Secret
// values never cross this engine boundary.
type RepoSpec struct {
	Name    string
	Backend string

	// Filesystem backend.
	Path string

	// S3 backend.
	Bucket     string
	Endpoint   string
	Prefix     string
	Region     string
	DisableTLS bool

	// SFTP backend; PrivateKey is never part of this request and remains in
	// the credential authority used by resource-kopia.
	SFTPHost       string
	SFTPPort       int
	SFTPUsername   string
	SFTPRemotePath string
	SFTPKnownHosts string
}

// RepoStatus is the subset of `resource-kopia repo status` this scenario reads.
type RepoStatus struct {
	// Encryption algorithm reported by kopia; always non-empty for a created
	// repository (encryption is always on). DBM-ENC-001 asserts on this.
	EncryptionAlgorithm string
	Connected           bool
}

// RepoStats is the subset of `resource-kopia repo stats` used for usage-vs-cap.
// SizeBytes is the repository's physical on-disk footprint (post dedup +
// compression) — the real storage cost and the denominator for a dedup ratio.
type RepoStats struct {
	SizeBytes int64
	BlobCount int64
}

// RepoDamageScan is a typed, read-only inventory returned by resource-kopia.
// Paths are relative to the registered repository root; the owner command
// classifies Kopia blob names and filesystem failures.
type RepoDamageScan struct {
	Root       string            `json:"root"`
	StartedAt  string            `json:"started_at"`
	DurationNS int64             `json:"duration"`
	Files      int64             `json:"files"`
	Bytes      int64             `json:"bytes"`
	TempFiles  int64             `json:"readable_temp_files"`
	Unreadable []RepoDamageEntry `json:"unreadable"`
}

type RepoDamageEntry struct {
	RelativePath string `json:"relative_path"`
	Class        string `json:"class"`
	BlobKind     string `json:"blob_kind"`
	Problem      string `json:"problem"`
	Orphan       bool   `json:"orphan_temp"`
	Detail       string `json:"detail"`
}

// RepoDamageScanner is an optional owner capability implemented by KopiaCLI.
// Keeping it separate from KopiaEngine preserves narrow fakes that do not
// exercise repository diagnosis.
type RepoDamageScanner interface {
	RepoDamageScan(ctx context.Context, repo string) (RepoDamageScan, error)
}

type RepoQuarantineResult struct {
	Repository           string `json:"repository"`
	OriginalPath         string `json:"original_path"`
	QuarantinePath       string `json:"quarantine_path"`
	FinalBlobPath        string `json:"final_blob_path"`
	SourceDataDeleted    bool   `json:"source_data_deleted"`
	FinalBlobWritten     bool   `json:"final_blob_written"`
	QuarantineSameVolume bool   `json:"quarantine_same_volume"`
	Committed            bool   `json:"committed"`
}

type RepoDamageRepair interface {
	RepoQuarantineTempBlob(ctx context.Context, repo, relativePath string) (RepoQuarantineResult, error)
}

// OrphanBlobCandidate is the resource-kopia read-only overlay proof for one
// missing q-blob. CandidatePath is transient owner output and must not be
// persisted into notifications or DBM records.
type OrphanBlobCandidate struct {
	BlobID                string
	CandidatePath         string
	CandidateBytes        int64
	CandidateSHA256       string
	FinalBlobAbsent       bool
	SourceRepositoryRead  bool
	OverlaySnapshotStatus string
	Blob                  *RepoBlobInspection
}

// RepoBlobInspection is the resource-kopia owner's repository-relative
// classification for one exact q-blob implicated by Kopia.
type RepoBlobInspection struct {
	BlobID        string   `json:"blob_id"`
	RelativePath  string   `json:"relative_path"`
	State         string   `json:"state"`
	Problem       string   `json:"problem,omitempty"`
	OrphanTemp    bool     `json:"orphan_temp"`
	AdjacentTemps []string `json:"adjacent_temps,omitempty"`
}

type OrphanBlobRecoveryResult struct {
	Repository         string `json:"repository"`
	BlobID             string `json:"blob_id"`
	CandidateSHA256    string `json:"candidate_sha256"`
	PublishedBytes     int64  `json:"published_bytes"`
	CandidatePreserved bool   `json:"candidate_preserved"`
	TargetReplaced     bool   `json:"target_replaced"`
	Committed          bool   `json:"committed"`
}

// RepoTempBlobRestore records resource-kopia's exact-path .f.tmp recovery.
// It preserves the candidate and never replaces an existing final object.
type RepoTempBlobRestore struct {
	Repository      string
	RelativePath    string
	Source          string
	Target          string
	SHA256          string
	Bytes           int64
	SourcePreserved bool
	TargetReplaced  bool
	Replay          bool
	Committed       bool
}

type RepoTempBlobRestorer interface {
	RepoRestoreTemp(ctx context.Context, repo, repositoryPath, relativePath, sha256 string) (RepoTempBlobRestore, error)
}

type RepoOrphanBlobProbe interface {
	RepoCacheProbe(ctx context.Context, repo, blobID string) (OrphanBlobCandidate, error)
}

type RepoOrphanBlobRecovery interface {
	RepoOrphanBlobCommit(ctx context.Context, repo, blobID, candidateSHA256 string) (OrphanBlobRecoveryResult, error)
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
	ID   string
	Path string
	// Tags preserves Kopia's self-identifying metadata for safe idempotent
	// owner operations that must reconcile an effect after a process restart.
	Tags []string
	// Source is the kopia source identity (user@host:/path) the snapshot was
	// recorded under. With an override source it differs from Path, and it is
	// the only identity that retention policy and expiry act on.
	Source    string
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
	// is always on; the passphrase is generated and stored in the credential
	// authority by kopia.
	RepoCreate(ctx context.Context, spec RepoSpec) error
	// RepoConnect attaches to an already-existing repository and never creates
	// backend data. The caller must verify repository identity after connect.
	RepoConnect(ctx context.Context, spec RepoSpec) error
	// RepoRebind reconnects an existing filesystem repository name at a new
	// validated absolute path. It must not create a repository or change its
	// credential identity.
	RepoRebind(ctx context.Context, repo, absolutePath string) error
	// RepoStatus reports the repository's encryption algorithm and connection.
	RepoStatus(ctx context.Context, repo string) (RepoStatus, error)
	// PassphraseRef returns the credential identity/field reference (never the
	// value) where resource-kopia stores this repository's encryption
	// passphrase, so a detached destination bundle can point an operator at the
	// key for standalone recovery. It is a pure, deterministic string
	// derivation — no I/O, no secret read.
	PassphraseRef(repo string) string
	// RepoStats reports current repository usage in bytes (for storage caps).
	RepoStats(ctx context.Context, repo string) (RepoStats, error)
	// RepoDelete removes local resource-kopia metadata and credential refs for
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
	// ApplyRetention applies the count and time-bucket policy and enforces it now. source must be the exact identity
	// the snapshots were recorded under (Snapshot.Source). It returns how many
	// snapshots the source holds afterwards so callers can prove the bound
	// held. Blob space is reclaimed later by kopia maintenance.
	ApplyRetention(ctx context.Context, repo, source string, policy RetentionPolicy) (remaining int, err error)
}

// RetentionPolicy mirrors Kopia's count and GFS buckets without importing the plans domain.
type RetentionPolicy struct{ KeepLatest, KeepHourly, KeepDaily, KeepWeekly, KeepMonthly, KeepAnnual int }

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
	if err := effectguard.Check(ctx); err != nil {
		return nil, err
	}
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
	case BackendSFTP:
		args = append(args, "--host", spec.SFTPHost, "--port", strconv.Itoa(spec.SFTPPort), "--username", spec.SFTPUsername, "--remote-path", spec.SFTPRemotePath, "--known-hosts", spec.SFTPKnownHosts)
	case BackendS3:
		args = append(args, "--bucket", spec.Bucket)
		if spec.Endpoint != "" {
			args = append(args, "--endpoint", spec.Endpoint)
		}
		if spec.Prefix != "" {
			args = append(args, "--prefix", spec.Prefix)
		}
		if spec.Region != "" {
			args = append(args, "--region", spec.Region)
		}
		if spec.DisableTLS {
			args = append(args, "--disable-tls")
		}
	default:
		return fmt.Errorf("repo create %q: unknown backend %q", spec.Name, spec.Backend)
	}
	if _, err := k.Runner.Run(ctx, args...); err != nil {
		return fmt.Errorf("repo create %q: %w", spec.Name, err)
	}
	return nil
}

// RepoConnect connects to an existing repository using the typed, non-secret
// backend descriptor. Unlike RepoCreate, this operation must fail if no
// repository already exists at the selected backend location.
func (k *KopiaCLI) RepoConnect(ctx context.Context, spec RepoSpec) error {
	args := []string{"repo", "connect", "--name", spec.Name, "--backend", spec.Backend}
	switch spec.Backend {
	case BackendFilesystem:
		args = append(args, "--path", spec.Path)
	case BackendSFTP:
		args = append(args, "--host", spec.SFTPHost, "--port", strconv.Itoa(spec.SFTPPort), "--username", spec.SFTPUsername, "--remote-path", spec.SFTPRemotePath, "--known-hosts", spec.SFTPKnownHosts)
	case BackendS3:
		args = append(args, "--bucket", spec.Bucket)
		if spec.Endpoint != "" {
			args = append(args, "--endpoint", spec.Endpoint)
		}
		if spec.Prefix != "" {
			args = append(args, "--prefix", spec.Prefix)
		}
		if spec.Region != "" {
			args = append(args, "--region", spec.Region)
		}
		if spec.DisableTLS {
			args = append(args, "--disable-tls")
		}
	default:
		return fmt.Errorf("repo connect %q: unknown backend %q", spec.Name, spec.Backend)
	}
	if _, err := k.Runner.Run(ctx, args...); err != nil {
		return fmt.Errorf("repo connect %q: %w", spec.Name, err)
	}
	return nil
}

// RepoRebind updates resource-kopia's registration for an existing
// filesystem repository and connects to that path. The caller owns volume,
// bundle and repository validation; this boundary enforces an absolute path
// and deliberately uses repo connect (never repo create).
func (k *KopiaCLI) RepoRebind(ctx context.Context, repo, absolutePath string) error {
	if strings.TrimSpace(repo) == "" {
		return fmt.Errorf("repo rebind: repository name is required")
	}
	if strings.TrimSpace(absolutePath) == "" || !filepath.IsAbs(absolutePath) {
		return fmt.Errorf("repo rebind %q: filesystem path must be absolute", repo)
	}
	args := []string{"repo", "connect", "--name", repo, "--backend", BackendFilesystem, "--path", absolutePath}
	if _, err := k.Runner.Run(ctx, args...); err != nil {
		return fmt.Errorf("repo rebind %q: %w", repo, err)
	}
	return nil
}

// PassphraseRef derives the credential-authority reference for a repository's
// passphrase. It deliberately performs no credential I/O and never risks
// touching a secret.
func (k *KopiaCLI) PassphraseRef(repo string) string {
	identity, err := kopiaregistry.PassphraseIdentity(repo)
	if err != nil {
		return ""
	}
	return string(identity) + ":" + kopiaregistry.PassphraseField
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
	return RepoStats{
		SizeBytes: firstInt(raw, "physicalBytes", "sizeBytes", "Size", "totalSize", "TotalPackedSize"),
		BlobCount: firstInt(raw, "blobCount"),
	}, nil
}

var kopiaDamageTempName = regexp.MustCompile(`^[0-9a-z]+(?:-[0-9a-z]+)*\.f\.tmp(?:\.[0-9a-f]{8,64})?$`)
var kopiaDamageBlobName = regexp.MustCompile(`^[0-9a-z]+(?:-[0-9a-z]+)*\.f$`)

func expectedDamageClass(relativePath string) string {
	name := path.Base(relativePath)
	switch {
	case kopiaDamageTempName.MatchString(name):
		return "leftover_temp"
	case kopiaDamageBlobName.MatchString(name):
		return "repository_blob"
	default:
		return "other"
	}
}

func expectedDamageBlobKind(relativePath string) string {
	first, _, hasParent := strings.Cut(relativePath, "/")
	if !hasParent {
		return "other"
	}
	switch first {
	case "p":
		return "data_pack"
	case "q":
		return "metadata_pack"
	case "s":
		return "session_marker"
	case "x", "n":
		return "index"
	default:
		return "other"
	}
}

func (k *KopiaCLI) RepoDamageScan(ctx context.Context, repo string) (RepoDamageScan, error) {
	out, err := k.Runner.Run(ctx, "repo", "damage-scan", "--name", repo, "--json")
	if err != nil {
		return RepoDamageScan{}, fmt.Errorf("repo damage scan %q: %w", repo, err)
	}
	var result RepoDamageScan
	if err := json.Unmarshal(out, &result); err != nil {
		return RepoDamageScan{}, fmt.Errorf("repo damage scan %q: parse json: %w", repo, err)
	}
	if result.Root == "" || result.StartedAt == "" {
		return RepoDamageScan{}, fmt.Errorf("repo damage scan %q: missing root or start time", repo)
	}
	if result.DurationNS < 0 || result.Files < 0 || result.Bytes < 0 || result.TempFiles < 0 {
		return RepoDamageScan{}, fmt.Errorf("repo damage scan %q: negative aggregate count", repo)
	}
	seen := make(map[string]struct{}, len(result.Unreadable))
	for i, entry := range result.Unreadable {
		rel := strings.ReplaceAll(entry.RelativePath, `\`, "/")
		clean := path.Clean(rel)
		if entry.RelativePath != rel || rel == "" || filepath.IsAbs(rel) || path.IsAbs(rel) || clean != rel || clean == ".." || strings.HasPrefix(clean, "../") {
			return RepoDamageScan{}, fmt.Errorf("repo damage scan %q: invalid relative path at entry %d", repo, i)
		}
		if _, duplicate := seen[rel]; duplicate {
			return RepoDamageScan{}, fmt.Errorf("repo damage scan %q: duplicate unreadable path %q", repo, rel)
		}
		seen[rel] = struct{}{}
		switch entry.Class {
		case "leftover_temp", "repository_blob", "other":
		default:
			return RepoDamageScan{}, fmt.Errorf("repo damage scan %q: unknown damage class %q", repo, entry.Class)
		}
		if entry.Orphan && entry.Class != "leftover_temp" {
			return RepoDamageScan{}, fmt.Errorf("repo damage scan %q: only leftover temps can be orphaned", repo)
		}
		if entry.Class != expectedDamageClass(rel) || entry.BlobKind != expectedDamageBlobKind(rel) {
			return RepoDamageScan{}, fmt.Errorf("repo damage scan %q: classification does not match path %q", repo, rel)
		}
		switch entry.Problem {
		case "io_error", "permission_denied", "other":
		default:
			return RepoDamageScan{}, fmt.Errorf("repo damage scan %q: unknown problem %q", repo, entry.Problem)
		}
		switch entry.BlobKind {
		case "session_marker", "data_pack", "metadata_pack", "index", "other":
		default:
			return RepoDamageScan{}, fmt.Errorf("repo damage scan %q: unknown blob kind %q", repo, entry.BlobKind)
		}
	}
	return result, nil
}

var sessionTempBlobName = regexp.MustCompile(`^[0-9a-z]+(?:-[0-9a-z]+)*\.f\.tmp(?:\.[a-f0-9]{8,64})?$`)

func validSessionTempPath(relativePath string) bool {
	if relativePath == "" || strings.Contains(relativePath, `\`) || path.Clean(relativePath) != relativePath || !strings.HasPrefix(relativePath, "s/") {
		return false
	}
	parent := path.Dir(relativePath)
	return parent != "." && parent != "s" && sessionTempBlobName.MatchString(path.Base(relativePath))
}

func (k *KopiaCLI) RepoQuarantineTempBlob(ctx context.Context, repo, relativePath string) (RepoQuarantineResult, error) {
	if strings.TrimSpace(repo) == "" || !validSessionTempPath(relativePath) {
		return RepoQuarantineResult{}, fmt.Errorf("repo quarantine temp: one exact s/...f.tmp path and repository name are required")
	}
	confirmation := "quarantine-orphan-temp:" + repo + ":" + relativePath
	out, err := k.Runner.Run(ctx, "repo", "quarantine-orphan-temp-blob", "--name", repo,
		"--relative-temp-path", relativePath, "--confirm-quarantine", confirmation, "--json")
	if err != nil {
		return RepoQuarantineResult{}, fmt.Errorf("repo quarantine temp %q: %w", repo, err)
	}
	var result RepoQuarantineResult
	if err := json.Unmarshal(out, &result); err != nil {
		return RepoQuarantineResult{}, fmt.Errorf("repo quarantine temp %q: parse json: %w", repo, err)
	}
	if result.Repository != repo || result.OriginalPath == "" || result.QuarantinePath == "" || result.FinalBlobPath == "" || !result.Committed || !result.QuarantineSameVolume || result.SourceDataDeleted || result.FinalBlobWritten {
		return RepoQuarantineResult{}, fmt.Errorf("repo quarantine temp %q: owner returned incomplete or unsafe result", repo)
	}
	return result, nil
}

func (k *KopiaCLI) RepoCacheProbe(ctx context.Context, repo, blobID string) (OrphanBlobCandidate, error) {
	if strings.TrimSpace(repo) == "" || !validOrphanQBlobID(blobID) {
		return OrphanBlobCandidate{}, fmt.Errorf("repo cache probe requires a repository and exact q-blob ID")
	}
	out, err := k.Runner.Run(ctx, "repo", "cache-probe", "--name", repo, "--blob-id", blobID, "--json")
	if err != nil {
		return OrphanBlobCandidate{}, fmt.Errorf("repo cache probe %q: %w", repo, err)
	}
	var result struct {
		Status         string              `json:"status"`
		Blob           *RepoBlobInspection `json:"blob"`
		OrphanTempBlob *struct {
			BlobID                string `json:"blob_id"`
			CandidatePath         string `json:"candidate_path"`
			CandidateBytes        int64  `json:"candidate_bytes"`
			CandidateSHA256       string `json:"candidate_sha256"`
			FinalBlobAbsent       bool   `json:"final_blob_absent"`
			SourceRepositoryRead  bool   `json:"source_repository_read_only"`
			OverlaySnapshotStatus string `json:"overlay_snapshot_status"`
		} `json:"orphan_temp_blob_candidate"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return OrphanBlobCandidate{}, fmt.Errorf("repo cache probe %q: parse JSON: %w", repo, err)
	}
	if result.Blob != nil {
		if result.Blob.BlobID != blobID || !validRepoBlobInspection(*result.Blob) {
			return OrphanBlobCandidate{}, fmt.Errorf("repo cache probe %q returned invalid typed blob classification", repo)
		}
	}
	if result.OrphanTempBlob == nil && result.Blob != nil && result.Status == "snapshot_list_failed" {
		return OrphanBlobCandidate{BlobID: blobID, Blob: result.Blob}, nil
	}
	if result.Status != "snapshot_list_failed" || result.OrphanTempBlob == nil {
		return OrphanBlobCandidate{}, fmt.Errorf("repo cache probe %q found no exact orphan q-blob candidate", repo)
	}
	candidate := result.OrphanTempBlob
	if candidate.BlobID != blobID || candidate.CandidateBytes <= 0 || !validSHA256(candidate.CandidateSHA256) || !candidate.FinalBlobAbsent || !candidate.SourceRepositoryRead || candidate.OverlaySnapshotStatus != "snapshot_list_passed" {
		return OrphanBlobCandidate{}, fmt.Errorf("repo cache probe %q returned incomplete or unsafe q-blob evidence", repo)
	}
	return OrphanBlobCandidate{
		BlobID: candidate.BlobID, CandidatePath: candidate.CandidatePath,
		CandidateBytes: candidate.CandidateBytes, CandidateSHA256: candidate.CandidateSHA256,
		FinalBlobAbsent: candidate.FinalBlobAbsent, SourceRepositoryRead: candidate.SourceRepositoryRead,
		OverlaySnapshotStatus: candidate.OverlaySnapshotStatus,
		Blob:                  result.Blob,
	}, nil
}

func validRepoBlobInspection(blob RepoBlobInspection) bool {
	switch blob.State {
	case "missing", "present", "unreadable", "unknown":
	default:
		return false
	}
	switch blob.Problem {
	case "", "io_error", "permission_denied", "other":
	default:
		return false
	}
	if blob.RelativePath != "" {
		rel := strings.ReplaceAll(blob.RelativePath, `\`, "/")
		if filepath.IsAbs(rel) || path.IsAbs(rel) || path.Clean(rel) != rel || rel == ".." || strings.HasPrefix(rel, "../") {
			return false
		}
	}
	if blob.OrphanTemp && (blob.State != "missing" || len(blob.AdjacentTemps) == 0) {
		return false
	}
	for _, temp := range blob.AdjacentTemps {
		rel := strings.ReplaceAll(temp, `\`, "/")
		if filepath.IsAbs(rel) || path.IsAbs(rel) || path.Clean(rel) != rel || rel == ".." || strings.HasPrefix(rel, "../") {
			return false
		}
	}
	return true
}

func (k *KopiaCLI) RepoOrphanBlobCommit(ctx context.Context, repo, blobID, candidateSHA256 string) (OrphanBlobRecoveryResult, error) {
	if strings.TrimSpace(repo) == "" || !validOrphanQBlobID(blobID) || !validSHA256(candidateSHA256) {
		return OrphanBlobRecoveryResult{}, fmt.Errorf("repo orphan blob recovery requires repository, exact q-blob ID, and SHA-256")
	}
	confirmation := "restore-orphan-blob:" + repo + ":" + blobID
	out, err := k.Runner.Run(ctx, "repo", "orphan-temp-blob-commit", "--name", repo,
		"--blob-id", blobID, "--candidate-sha256", candidateSHA256,
		"--confirm-commit", confirmation, "--json")
	if err != nil {
		return OrphanBlobRecoveryResult{}, fmt.Errorf("repo orphan blob recovery %q: %w", repo, err)
	}
	var result OrphanBlobRecoveryResult
	if err := json.Unmarshal(out, &result); err != nil {
		return OrphanBlobRecoveryResult{}, fmt.Errorf("repo orphan blob recovery %q: parse JSON: %w", repo, err)
	}
	if result.Repository != repo || result.BlobID != blobID || result.CandidateSHA256 != candidateSHA256 || result.PublishedBytes <= 0 || !result.CandidatePreserved || result.TargetReplaced || !result.Committed {
		return OrphanBlobRecoveryResult{}, fmt.Errorf("repo orphan blob recovery %q returned incomplete or unsafe result", repo)
	}
	return result, nil
}

func (k *KopiaCLI) RepoRestoreTemp(ctx context.Context, repo, repositoryPath, relativePath, checksum string) (RepoTempBlobRestore, error) {
	repoPath := filepath.Clean(strings.TrimSpace(repositoryPath))
	if strings.TrimSpace(repo) == "" || !filepath.IsAbs(repoPath) || !validSHA256(checksum) {
		return RepoTempBlobRestore{}, fmt.Errorf("repo temp restore requires repository, absolute repository path, and SHA-256")
	}
	rel := strings.ReplaceAll(relativePath, `\\`, "/")
	if rel == "" || path.IsAbs(rel) || path.Clean(rel) != rel || rel == ".." || strings.HasPrefix(rel, "../") || !strings.Contains(path.Base(rel), ".f.tmp") {
		return RepoTempBlobRestore{}, fmt.Errorf("repo temp restore requires one exact repository-relative Kopia temp path")
	}
	confirmation := "restore-temp:" + repo + ":" + repoPath + ":" + rel + ":" + checksum
	out, err := k.Runner.Run(ctx, "repo", "restore-from-temp", "--name", repo, "--repository-path", repoPath,
		"--relative-temp-path", rel, "--temp-sha256", checksum, "--confirm-restore", confirmation, "--json")
	if err != nil {
		return RepoTempBlobRestore{}, fmt.Errorf("repo temp restore %q: %w", repo, err)
	}
	var result struct {
		Repository      string `json:"repository"`
		RelativePath    string `json:"relative_temp_path"`
		Source          string `json:"source_path"`
		Target          string `json:"target_path"`
		SHA256          string `json:"sha256"`
		Bytes           int64  `json:"bytes"`
		SourcePreserved bool   `json:"source_preserved"`
		TargetReplaced  bool   `json:"target_replaced"`
		Replay          bool   `json:"replay"`
		Committed       bool   `json:"committed"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return RepoTempBlobRestore{}, fmt.Errorf("repo temp restore %q: parse JSON: %w", repo, err)
	}
	expectedSource := filepath.Join(repoPath, filepath.FromSlash(rel))
	finalBase := strings.SplitN(filepath.Base(rel), ".tmp", 2)[0]
	if finalBase == filepath.Base(rel) || !strings.HasSuffix(finalBase, ".f") {
		return RepoTempBlobRestore{}, fmt.Errorf("repo temp restore %q: path does not identify a Kopia final object", repo)
	}
	expectedTarget := filepath.Join(filepath.Dir(expectedSource), finalBase)
	if result.Repository != repo || result.RelativePath != rel || result.Source != expectedSource || result.Target != expectedTarget ||
		result.SHA256 != checksum || result.Bytes <= 0 || !result.SourcePreserved || result.TargetReplaced || !result.Committed {
		return RepoTempBlobRestore{}, fmt.Errorf("repo temp restore %q returned incomplete or unsafe result", repo)
	}
	return RepoTempBlobRestore{Repository: result.Repository, RelativePath: result.RelativePath, Source: result.Source,
		Target: result.Target, SHA256: result.SHA256, Bytes: result.Bytes, SourcePreserved: result.SourcePreserved,
		TargetReplaced: result.TargetReplaced, Replay: result.Replay, Committed: result.Committed}, nil
}

func validOrphanQBlobID(value string) bool {
	if !strings.HasPrefix(value, "q") || len(value) < 2 || len(value) > 128 {
		return false
	}
	for _, r := range value {
		if !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') && r != '-' {
			return false
		}
	}
	return true
}

func validSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, r := range value {
		if !(r >= '0' && r <= '9') && !(r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
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
		var tags []string
		if raw, ok := m["tags"]; ok {
			_ = json.Unmarshal(raw, &tags)
		} else if raw, ok := m["Tags"]; ok {
			_ = json.Unmarshal(raw, &tags)
		}
		snaps = append(snaps, Snapshot{
			ID:        firstString(m, "id", "ID"),
			Source:    snapshotSource(m),
			StartTime: firstString(m, "startTime", "StartTime"),
			SizeBytes: snapshotSize(m),
			Tags:      tags,
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

// gfsBuckets are kopia's time-bucketed retention counts.
var gfsBuckets = []string{"--keep-hourly", "--keep-daily", "--keep-weekly", "--keep-monthly", "--keep-annual"}

func (k *KopiaCLI) ApplyRetention(ctx context.Context, repo, source string, policy RetentionPolicy) (int, error) {
	if strings.TrimSpace(source) == "" {
		return 0, fmt.Errorf("apply retention repo=%q: snapshot source is unknown", repo)
	}
	if policy.KeepLatest < 0 || policy.KeepHourly < 0 || policy.KeepDaily < 0 || policy.KeepWeekly < 0 || policy.KeepMonthly < 0 || policy.KeepAnnual < 0 {
		return 0, fmt.Errorf("apply retention repo=%q source=%q: retention values cannot be negative", repo, source)
	}
	if policy.KeepLatest+policy.KeepHourly+policy.KeepDaily+policy.KeepWeekly+policy.KeepMonthly+policy.KeepAnnual == 0 {
		return 0, fmt.Errorf("apply retention repo=%q source=%q: at least one retention tier must be positive", repo, source)
	}
	args := []string{"policy", "set", "--repo", repo, "--path", source, "--keep-latest", strconv.Itoa(policy.KeepLatest)}
	values := []int{policy.KeepHourly, policy.KeepDaily, policy.KeepWeekly, policy.KeepMonthly, policy.KeepAnnual}
	for i, bucket := range gfsBuckets {
		args = append(args, bucket, strconv.Itoa(values[i]))
	}
	if _, err := k.Runner.Run(ctx, args...); err != nil {
		return 0, fmt.Errorf("policy set repo=%q source=%q: %w", repo, source, err)
	}
	// kopia expires only while creating the next snapshot of a source, so a
	// policy set after this run's snapshot would not bound anything until the
	// following run. Expire explicitly so the bound holds now.
	if _, err := k.Runner.Run(ctx, "snapshot", "expire", "--repo", repo, "--path", source, "--delete"); err != nil {
		return 0, fmt.Errorf("snapshot expire repo=%q source=%q: %w", repo, source, err)
	}
	snaps, err := k.SnapshotList(ctx, repo, source)
	if err != nil {
		return 0, err
	}
	return len(snaps), nil
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
	return Snapshot{ID: id, Source: snapshotSource(m), StartTime: firstString(m, "startTime", "StartTime"), SizeBytes: snapshotSize(m)}, nil
}

// snapshotSource renders a manifest's source block as kopia's canonical
// user@host:/path identity, or "" when the manifest omits any part of it.
func snapshotSource(m map[string]json.RawMessage) string {
	raw, ok := m["source"]
	if !ok {
		return ""
	}
	var src map[string]json.RawMessage
	if json.Unmarshal(raw, &src) != nil {
		return ""
	}
	user := firstString(src, "userName", "UserName")
	host := firstString(src, "host", "Host")
	path := firstString(src, "path", "Path")
	if user == "" || host == "" || path == "" {
		return ""
	}
	return user + "@" + host + ":" + path
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
