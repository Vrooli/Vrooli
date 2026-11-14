// Package repo implements the repository lifecycle commands of resource-kopia:
// create, connect, status, stats, list, disconnect, validate. A Vrooli
// "destination" maps one-to-one to a kopia repository identified by --name.
//
// Invariants enforced here:
//   - Encryption is always on. No handler ever emits a kopia flag that disables
//     or downgrades encryption (kopia's default is AES256-GCM-HMAC-SHA256).
//   - Secrets travel only through the env overlay (KOPIA_PASSWORD, AWS_*),
//     never as argv flags.
//   - Runtime state (the registry, config files, caches) resolves outside the
//     repo tree via internal/env.
package repo

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/vrooli/resources/kopia/cli/internal/cmdutil"
	"github.com/vrooli/vrooli/resources/kopia/cli/internal/credentials"
	"github.com/vrooli/vrooli/resources/kopia/cli/internal/env"
	"github.com/vrooli/vrooli/resources/kopia/cli/internal/kexec"
	"github.com/vrooli/vrooli/resources/kopia/cli/internal/registry"
	"github.com/vrooli/vrooli/resources/kopia/cli/internal/repoctx"

	kopiaregistry "github.com/vrooli/vrooli/packages/kopiaregistry-go"
)

// Service wires the dependencies the repository commands need.
type Service struct {
	Runner      kexec.Runner
	Credentials credentials.Store
	Registry    *registry.Registry
	Resolver    repoctx.Resolver
	Env         env.Runtime
	Out         io.Writer
}

func (s Service) out() io.Writer {
	if s.Out != nil {
		return s.Out
	}
	return os.Stdout
}

// Create provisions a new kopia repository (== Vrooli destination). Encryption
// is left at kopia's secure default; the passphrase is generated-and-stored in
// the credential authority on first creation.
func (s Service) Create(ctx context.Context, args []string) error {
	fs := cmdutil.NewFlagSet("repo create")
	var (
		name       = fs.String("name", "", "Destination/repository name (required)")
		backend    = fs.String("backend", "", "Backend: filesystem | s3 (required)")
		path       = fs.String("path", "", "Filesystem backend: repository directory")
		bucket     = fs.String("bucket", "", "S3 backend: bucket name")
		endpoint   = fs.String("endpoint", "", "S3 backend: endpoint (e.g. minio:9000)")
		prefix     = fs.String("prefix", "", "S3 backend: key prefix")
		region     = fs.String("region", "", "S3 backend: region")
		disableTLS = fs.Bool("disable-tls", false, "S3 backend: disable TLS (local MinIO)")
		accessKey  = fs.String("access-key", "", "S3 backend: access key id (stored in the credential authority; else read from it)")
		secretKey  = fs.String("secret-access-key", "", "S3 backend: secret access key (stored in the credential authority)")
		jsonOut    = fs.Bool("json", false, "Emit kopia's native JSON")
	)
	if err := cmdutil.Parse(fs, args); err != nil {
		return err
	}
	if err := cmdutil.RequireName(*name); err != nil {
		return err
	}
	if err := s.Env.EnsureDirectories(); err != nil {
		return err
	}
	cfg := s.Env.RepoConfigFile(*name)
	cache := s.Env.RepoCacheDir(*name)
	if err := os.MkdirAll(filepath.Dir(cfg), 0o700); err != nil {
		return fmt.Errorf("create repo config dir: %w", err)
	}

	entry := registry.Entry{
		Name:       *name,
		ConfigFile: cfg,
		CacheDir:   cache,
	}

	verbArgs := []string{"--config-file", cfg, "repository", "create"}
	secretEnv := map[string]string{}
	switch strings.ToLower(strings.TrimSpace(*backend)) {
	case registry.BackendFilesystem:
		if strings.TrimSpace(*path) == "" {
			return fmt.Errorf("--path is required for the filesystem backend")
		}
		entry.Backend = registry.BackendFilesystem
		entry.Path = *path
		if err := os.MkdirAll(*path, 0o700); err != nil {
			return fmt.Errorf("create filesystem repo path: %w", err)
		}
		verbArgs = append(verbArgs, "filesystem", "--path", *path)
	case registry.BackendS3:
		if strings.TrimSpace(*bucket) == "" {
			return fmt.Errorf("--bucket is required for the s3 backend")
		}
		entry.Backend = registry.BackendS3
		entry.Bucket = *bucket
		entry.Endpoint = *endpoint
		entry.Prefix = *prefix
		entry.Region = *region
		entry.DisableTLS = *disableTLS
		creds, err := s.resolveS3Creds(ctx, *name, *accessKey, *secretKey)
		if err != nil {
			return err
		}
		secretEnv[repoctx.EnvAWSAccessKey] = creds.AccessKeyID
		secretEnv[repoctx.EnvAWSSecretKey] = creds.SecretAccessKey
		verbArgs = append(verbArgs, s3Args(entry)...)
	default:
		return fmt.Errorf("unsupported --backend %q (use filesystem or s3)", *backend)
	}
	passphrase, err := credentials.GeneratePassphrase()
	if err != nil {
		return err
	}
	identity, err := kopiaregistry.PassphraseIdentity(entry.Name)
	if err != nil {
		return err
	}
	if s.Credentials == nil {
		return fmt.Errorf("credential authority is unavailable")
	}
	if err := s.Credentials.Put(identity, kopiaregistry.PassphraseField, passphrase); err != nil {
		return fmt.Errorf("store repository passphrase: %w", err)
	}
	if err := credentials.ValidateStoredPassphrase(s.Credentials, identity, passphrase); err != nil {
		_ = s.Credentials.Delete(identity, kopiaregistry.PassphraseField)
		return err
	}
	secretEnv[repoctx.EnvPassword] = passphrase
	verbArgs = append(verbArgs, "--cache-directory", cache)

	out, err := s.Runner.Run(ctx, kexec.Call{Args: verbArgs, Env: secretEnv})
	if err != nil {
		_ = s.Credentials.Delete(identity, kopiaregistry.PassphraseField)
		return err
	}
	if err := s.Registry.Upsert(entry); err != nil {
		_ = s.Credentials.Delete(identity, kopiaregistry.PassphraseField)
		return err
	}
	return s.report(*jsonOut, out, fmt.Sprintf("created repository %q (%s, encryption on)", *name, entry.Backend))
}

// Connect attaches to a repository. With backend flags it (re)registers a
// pre-existing repository (e.g. for disaster recovery re-attach); without them
// it connects using the registered backend params.
func (s Service) Connect(ctx context.Context, args []string) error {
	fs := cmdutil.NewFlagSet("repo connect")
	var (
		name       = fs.String("name", "", "Repository name (required)")
		backend    = fs.String("backend", "", "Backend to (re)register before connecting: filesystem | s3")
		path       = fs.String("path", "", "Filesystem backend: repository directory")
		bucket     = fs.String("bucket", "", "S3 backend: bucket name")
		endpoint   = fs.String("endpoint", "", "S3 backend: endpoint")
		prefix     = fs.String("prefix", "", "S3 backend: key prefix")
		region     = fs.String("region", "", "S3 backend: region")
		disableTLS = fs.Bool("disable-tls", false, "S3 backend: disable TLS")
		accessKey  = fs.String("access-key", "", "S3 backend: access key id (stored in the credential authority)")
		secretKey  = fs.String("secret-access-key", "", "S3 backend: secret access key (stored in the credential authority)")
		jsonOut    = fs.Bool("json", false, "Emit kopia's native JSON")
	)
	if err := cmdutil.Parse(fs, args); err != nil {
		return err
	}
	if err := cmdutil.RequireName(*name); err != nil {
		return err
	}
	if err := s.Env.EnsureDirectories(); err != nil {
		return err
	}

	if strings.TrimSpace(*backend) != "" {
		if err := s.registerForConnect(ctx, *name, *backend, *path, *bucket, *endpoint, *prefix, *region, *disableTLS, *accessKey, *secretKey); err != nil {
			return err
		}
	}

	target, err := s.Resolver.Resolve(ctx, *name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target.Entry.ConfigFile), 0o700); err != nil {
		return fmt.Errorf("create repo config dir: %w", err)
	}

	verbArgs := []string{"--config-file", target.Entry.ConfigFile, "repository", "connect"}
	switch target.Entry.Backend {
	case registry.BackendFilesystem:
		verbArgs = append(verbArgs, "filesystem", "--path", target.Entry.Path)
	case registry.BackendS3:
		verbArgs = append(verbArgs, s3Args(target.Entry)...)
	default:
		return fmt.Errorf("repository %q has unknown backend %q", *name, target.Entry.Backend)
	}
	if target.Entry.CacheDir != "" {
		verbArgs = append(verbArgs, "--cache-directory", target.Entry.CacheDir)
	}

	out, err := s.Runner.Run(ctx, kexec.Call{Args: verbArgs, Env: target.Env})
	if err != nil {
		return err
	}
	if err := target.RetireLegacyPassphrase(); err != nil {
		return err
	}
	return s.report(*jsonOut, out, fmt.Sprintf("connected to repository %q", *name))
}

// Status prints `kopia repository status` for a repository.
func (s Service) Status(ctx context.Context, args []string) error {
	return s.simpleRepoCommand(ctx, args, "repo status", []string{"repository", "status"}, true)
}

// repoStatsSummary is the JSON shape emitted by `repo stats --json`. It reports
// the repository's true on-disk (physical) footprint: the bytes actually stored
// on the backend after kopia's content-addressed dedup and compression. This is
// the denominator for a dedup ratio (logical bytes ÷ physical bytes).
type repoStatsSummary struct {
	PhysicalBytes int64 `json:"physicalBytes"`
	BlobCount     int64 `json:"blobCount"`
}

// Stats reports storage-usage statistics for a repository. kopia's stats
// subcommands have no `--json` flag in the supported build, so we run
// `kopia blob stats --raw` (exact byte integers) and either pass kopia's native
// text through or parse it into our own stable JSON summary.
func (s Service) Stats(ctx context.Context, args []string) error {
	fs := cmdutil.NewFlagSet("repo stats")
	name := fs.String("name", "", "Repository name (required)")
	jsonOut := fs.Bool("json", false, "Emit a JSON physical-size summary")
	if err := cmdutil.Parse(fs, args); err != nil {
		return err
	}
	if err := cmdutil.RequireName(*name); err != nil {
		return err
	}
	target, err := s.Resolver.Resolve(ctx, *name)
	if err != nil {
		return err
	}
	// `blob stats` measures the physical bytes stored on the backend; `--raw`
	// makes kopia print exact integers instead of human-readable units.
	argv := append(target.GlobalArgs(), "blob", "stats", "--raw")
	out, err := s.Runner.Run(ctx, kexec.Call{Args: argv, Env: target.Env})
	if err != nil {
		return err
	}
	if err := target.RetireLegacyPassphrase(); err != nil {
		return err
	}
	if !*jsonOut {
		_, err = s.out().Write(cmdutil.EnsureTrailingNewline(out))
		return err
	}
	summary, err := parseBlobStats(out)
	if err != nil {
		return fmt.Errorf("repo stats %q: %w", *name, err)
	}
	return cmdutil.WriteJSON(s.out(), summary)
}

// parseBlobStats extracts the total physical bytes and blob count from the
// output of `kopia blob stats --raw`, whose first lines are `Count: <n>` and
// `Total: <bytes>`. Histogram rows use a lowercase "(total …)" and never match.
func parseBlobStats(raw []byte) (repoStatsSummary, error) {
	var summary repoStatsSummary
	sc := bufio.NewScanner(bytes.NewReader(raw))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		switch {
		case strings.HasPrefix(line, "Count:"):
			summary.BlobCount, _ = strconv.ParseInt(strings.TrimSpace(strings.TrimPrefix(line, "Count:")), 10, 64)
		case strings.HasPrefix(line, "Total:"):
			summary.PhysicalBytes, _ = strconv.ParseInt(strings.TrimSpace(strings.TrimPrefix(line, "Total:")), 10, 64)
		}
	}
	if err := sc.Err(); err != nil {
		return repoStatsSummary{}, fmt.Errorf("scan blob stats output: %w", err)
	}
	if summary.PhysicalBytes == 0 && summary.BlobCount == 0 {
		return repoStatsSummary{}, fmt.Errorf("could not parse physical size from blob stats output")
	}
	return summary, nil
}

// Validate runs a provider connectivity/integrity check.
func (s Service) Validate(ctx context.Context, args []string) error {
	return s.simpleRepoCommand(ctx, args, "repo validate", []string{"repository", "validate-provider"}, false)
}

// CacheProbe repeats read-only snapshot enumeration with an ephemeral Kopia
// configuration and cache. It isolates the registered cache from a fresh local
// cache without changing repository bytes or deleting the original cache.
func (s Service) CacheProbe(ctx context.Context, args []string) error {
	fs := cmdutil.NewFlagSet("repo cache-probe")
	name := fs.String("name", "", "Registered filesystem repository name (required)")
	requestedBlobID := fs.String("blob-id", "", "Optional exact q-blob ID to test in the disposable overlay")
	jsonOut := fs.Bool("json", false, "Emit a structured probe result")
	if err := cmdutil.Parse(fs, args); err != nil {
		return err
	}
	if err := cmdutil.RequireName(*name); err != nil {
		return err
	}
	entry, found, err := s.Registry.Get(*name)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("repository %q is not registered", *name)
	}
	if entry.Backend != registry.BackendFilesystem || strings.TrimSpace(entry.Path) == "" {
		return fmt.Errorf("repository %q is not a registered filesystem repository", *name)
	}
	target, err := s.Resolver.Resolve(ctx, *name)
	if err != nil {
		return err
	}
	if strings.TrimSpace(s.Env.CacheRoot) == "" {
		return fmt.Errorf("resource-kopia cache root is unavailable for isolated repository probe")
	}
	diagnosticRoot := filepath.Join(s.Env.CacheRoot, "diagnostics")
	if err := os.MkdirAll(diagnosticRoot, 0o700); err != nil {
		return fmt.Errorf("create isolated repository probe root: %w", err)
	}
	probeDir, err := os.MkdirTemp(diagnosticRoot, "repository-")
	if err != nil {
		return fmt.Errorf("create isolated repository probe directory: %w", err)
	}
	defer os.RemoveAll(probeDir)

	configPath := filepath.Join(probeDir, "repository.config")
	cachePath := filepath.Join(probeDir, "cache")
	connectArgs := []string{
		"--config-file", configPath,
		"repository", "connect", "filesystem",
		"--path", entry.Path,
		"--cache-directory", cachePath,
	}
	connectOutput, connectErr := s.Runner.Run(ctx, kexec.Call{
		Args: connectArgs, Env: target.Env, CaptureStderr: true,
	})
	result := struct {
		Repository         string                   `json:"repository"`
		Path               string                   `json:"path"`
		CacheIsolated      bool                     `json:"cache_isolated"`
		RepositoryModified bool                     `json:"repository_modified"`
		Status             string                   `json:"status"`
		ConnectOutputSHA   string                   `json:"connect_output_sha256,omitempty"`
		SnapshotListSHA    string                   `json:"snapshot_list_output_sha256,omitempty"`
		OrphanTempBlob     *orphanTempBlobCandidate `json:"orphan_temp_blob_candidate,omitempty"`
		Error              string                   `json:"error,omitempty"`
	}{
		Repository: *name, Path: entry.Path,
		CacheIsolated: true, RepositoryModified: false,
		Status: "connect_failed",
	}
	if connectErr != nil {
		result.Error = summarizeCacheProbeError(connectErr)
		return writeCacheProbeResult(s.out(), *jsonOut, result)
	}
	result.ConnectOutputSHA = digestText(connectOutput)

	listArgs := []string{
		"--config-file", configPath,
		"--log-level=info", "--no-progress",
		"snapshot", "list", "--json",
	}
	listOutput, listErr := s.Runner.Run(ctx, kexec.Call{
		Args: listArgs, Env: target.Env, CaptureStderr: true,
	})
	if listErr != nil {
		result.Status = "snapshot_list_failed"
		result.Error = summarizeCacheProbeError(listErr)
		blobID := missingBlobID(listErr.Error())
		if strings.TrimSpace(*requestedBlobID) != "" {
			blobID = strings.TrimSpace(*requestedBlobID)
		}
		if blobID != "" {
			if !validKopiaBlobID(blobID) || !strings.HasPrefix(blobID, "q") {
				return fmt.Errorf("blob ID must be an exact q-blob identifier")
			}
			candidate, overlayStatus, overlayErr := s.probeOrphanTempBlob(ctx, entry.Path, blobID, probeDir, target.Env)
			if candidate != nil {
				candidate.OverlaySnapshotStatus = overlayStatus
				if overlayErr != nil {
					candidate.OverlayError = summarizeCacheProbeError(overlayErr)
				}
				result.OrphanTempBlob = candidate
			}
		}
		return writeCacheProbeResult(s.out(), *jsonOut, result)
	}
	result.Status = "snapshot_list_passed"
	result.SnapshotListSHA = digestText(listOutput)
	return writeCacheProbeResult(s.out(), *jsonOut, result)
}

type orphanTempBlobCandidate struct {
	BlobID                string    `json:"blob_id"`
	ExpectedPath          string    `json:"expected_path"`
	CandidatePath         string    `json:"candidate_path"`
	CandidateBytes        int64     `json:"candidate_bytes"`
	CandidateModifiedAt   time.Time `json:"candidate_modified_at"`
	CandidateSHA256       string    `json:"candidate_sha256"`
	FinalBlobAbsent       bool      `json:"final_blob_absent"`
	SourceRepositoryRead  bool      `json:"source_repository_read_only"`
	OverlaySnapshotStatus string    `json:"overlay_snapshot_status"`
	OverlaySnapshotSHA    string    `json:"overlay_snapshot_output_sha256,omitempty"`
	OverlayError          string    `json:"overlay_error,omitempty"`
}

// missingBlobID extracts Kopia's exact quoted blob ID from a repository error.
func missingBlobID(message string) string {
	if !strings.Contains(message, "BLOB not found") {
		return ""
	}
	const marker = `blob "`
	start := strings.LastIndex(message, marker)
	if start < 0 {
		return ""
	}
	value := message[start+len(marker):]
	end := strings.IndexByte(value, '"')
	if end <= 0 {
		return ""
	}
	return value[:end]
}

// probeOrphanTempBlob checks whether exactly one stale filesystem temp file
// could satisfy a missing q-blob, then tests it in a temporary repository
// overlay. The registered repository and its cache are only read; the
// candidate is copied into the disposable overlay and never published.
func (s Service) probeOrphanTempBlob(ctx context.Context, repoPath, blobID, probeDir string, env map[string]string) (*orphanTempBlobCandidate, string, error) {
	if !strings.HasPrefix(blobID, "q") || strings.ContainsAny(blobID, `/\\`) {
		return nil, "", nil
	}
	relParts, err := filesystemBlobPath(repoPath, blobID)
	if err != nil {
		return nil, "", nil
	}
	expected := filepath.Join(append([]string{repoPath}, relParts...)...)
	if _, err := os.Stat(expected); err == nil {
		return nil, "", nil
	} else if !os.IsNotExist(err) {
		return nil, "", err
	}
	candidates, err := filepath.Glob(expected + ".tmp.*")
	if err != nil || len(candidates) != 1 {
		return nil, "", nil
	}
	info, err := os.Stat(candidates[0])
	if err != nil || !info.Mode().IsRegular() {
		return nil, "", nil
	}
	checksum, err := fileSHA256(candidates[0])
	if err != nil {
		return nil, "", err
	}
	candidate := &orphanTempBlobCandidate{
		BlobID: blobID, ExpectedPath: expected, CandidatePath: candidates[0],
		CandidateBytes: info.Size(), CandidateModifiedAt: info.ModTime().UTC(),
		CandidateSHA256: checksum, FinalBlobAbsent: true, SourceRepositoryRead: true,
		OverlaySnapshotStatus: "not_run",
	}
	overlay := filepath.Join(probeDir, "orphan-blob-overlay")
	if err := makeFilesystemOverlay(repoPath, overlay, relParts, candidates[0]); err != nil {
		return candidate, "overlay_setup_failed", err
	}
	configPath := filepath.Join(probeDir, "orphan-blob.config")
	cachePath := filepath.Join(probeDir, "orphan-blob-cache")
	connectArgs := []string{
		"--config-file", configPath,
		"repository", "connect", "filesystem",
		"--path", overlay,
		"--cache-directory", cachePath,
	}
	if _, err := s.Runner.Run(ctx, kexec.Call{Args: connectArgs, Env: env, CaptureStderr: true}); err != nil {
		return candidate, "connect_failed", err
	}
	listArgs := []string{"--config-file", configPath, "--log-level=info", "--no-progress", "snapshot", "list", "--json"}
	listOutput, err := s.Runner.Run(ctx, kexec.Call{Args: listArgs, Env: env, CaptureStderr: true})
	if err != nil {
		return candidate, "snapshot_list_failed", err
	}
	candidate.OverlaySnapshotStatus = "snapshot_list_passed"
	candidate.OverlaySnapshotSHA = digestText(listOutput)
	return candidate, "snapshot_list_passed", nil
}

// filesystemBlobPath derives the Kopia filesystem path from its .shards file.
// It supports the repository's default shard list and refuses unknown override
// layouts rather than guessing an object location.
func filesystemBlobPath(repoPath, blobID string) ([]string, error) {
	data, err := os.ReadFile(filepath.Join(repoPath, ".shards"))
	if err != nil {
		return nil, err
	}
	var cfg struct {
		Default             []int `json:"default"`
		MaxNonShardedLength int   `json:"maxNonShardedLength"`
		Overrides           []struct {
			Prefix string `json:"prefix"`
			Shards []int  `json:"shards"`
		} `json:"overrides"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	shards := cfg.Default
	for _, override := range cfg.Overrides {
		if strings.HasPrefix(blobID, override.Prefix) {
			return nil, fmt.Errorf("blob uses repository-specific shard override")
		}
	}
	if len(shards) == 0 || (cfg.MaxNonShardedLength > 0 && len(blobID) <= cfg.MaxNonShardedLength) {
		return []string{blobID + ".f"}, nil
	}
	remaining := blobID
	parts := make([]string, 0, len(shards)+1)
	for _, size := range shards {
		if size <= 0 || len(remaining) < size {
			return nil, fmt.Errorf("invalid shard layout")
		}
		parts = append(parts, remaining[:size])
		remaining = remaining[size:]
	}
	if remaining == "" {
		return nil, fmt.Errorf("empty blob filename after sharding")
	}
	return append(parts, remaining+".f"), nil
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// makeFilesystemOverlay symlinks all repository paths except the missing blob's
// shard branch. That branch is copied so the candidate can be added only in the
// disposable overlay. The original repository remains untouched.
func makeFilesystemOverlay(repoPath, overlay string, relParts []string, candidatePath string) error {
	if len(relParts) < 2 {
		return fmt.Errorf("sharded blob path is required for an overlay probe")
	}
	if err := os.Mkdir(overlay, 0o700); err != nil {
		return err
	}
	resolved, err := filepath.EvalSymlinks(repoPath)
	if err != nil {
		return err
	}
	rootEntries, err := os.ReadDir(resolved)
	if err != nil {
		return err
	}
	for _, entry := range rootEntries {
		src := filepath.Join(resolved, entry.Name())
		dst := filepath.Join(overlay, entry.Name())
		if entry.Name() == relParts[0] && entry.IsDir() {
			finalName := relParts[len(relParts)-1]
			if err := overlayPathBranch(src, dst, relParts[1:len(relParts)-1], finalName, candidatePath); err != nil {
				return err
			}
			continue
		}
		if err := os.Symlink(src, dst); err != nil {
			return err
		}
	}
	return nil
}

func overlayPathBranch(src, dst string, remaining []string, finalName, candidatePath string) error {
	if err := os.Mkdir(dst, 0o700); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	if len(remaining) == 0 {
		for _, entry := range entries {
			if err := os.Symlink(filepath.Join(src, entry.Name()), filepath.Join(dst, entry.Name())); err != nil {
				return err
			}
		}
		if _, err := os.Lstat(filepath.Join(dst, finalName)); err == nil {
			return fmt.Errorf("candidate final blob already exists in overlay")
		}
		return copyFile(candidatePath, filepath.Join(dst, finalName))
	}
	branch := remaining[0]
	for _, entry := range entries {
		childSrc := filepath.Join(src, entry.Name())
		childDst := filepath.Join(dst, entry.Name())
		if entry.Name() == branch && entry.IsDir() {
			if err := overlayPathBranch(childSrc, childDst, remaining[1:], finalName, candidatePath); err != nil {
				return err
			}
			continue
		}
		if err := os.Symlink(childSrc, childDst); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o400)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

// OrphanTempBlobCommit publishes one verified stale filesystem temp blob under
// its expected Kopia name. The source is preserved, the target must be absent,
// the candidate hash must match the separately reviewed value, and the
// confirmation phrase is bound to the exact repository and blob ID.
func (s Service) OrphanTempBlobCommit(ctx context.Context, args []string) error {
	fs := cmdutil.NewFlagSet("repo orphan-temp-blob-commit")
	name := fs.String("name", "", "Registered filesystem repository name (required)")
	blobID := fs.String("blob-id", "", "Exact missing Kopia q-blob ID (required)")
	sha := fs.String("candidate-sha256", "", "Expected SHA-256 of the single adjacent temp candidate (required)")
	confirm := fs.String("confirm-commit", "", "Must equal restore-orphan-blob:<repository-name>:<blob-id>")
	jsonOut := fs.Bool("json", false, "Emit a structured recovery result")
	if err := cmdutil.Parse(fs, args); err != nil {
		return err
	}
	if err := cmdutil.RequireName(*name); err != nil {
		return err
	}
	if !validKopiaBlobID(*blobID) || !strings.HasPrefix(*blobID, "q") {
		return fmt.Errorf("blob ID must be an exact q-blob identifier")
	}
	expectedConfirmation := "restore-orphan-blob:" + *name + ":" + *blobID
	if strings.TrimSpace(*confirm) != expectedConfirmation {
		return fmt.Errorf("orphan temp blob recovery requires --confirm-commit %q", expectedConfirmation)
	}
	expectedSHA := strings.ToLower(strings.TrimSpace(*sha))
	if len(expectedSHA) != sha256.Size*2 {
		return fmt.Errorf("candidate SHA-256 must contain 64 hexadecimal characters")
	}
	if _, err := hex.DecodeString(expectedSHA); err != nil {
		return fmt.Errorf("candidate SHA-256 is invalid: %w", err)
	}
	entry, found, err := s.Registry.Get(*name)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("repository %q is not registered", *name)
	}
	if entry.Backend != registry.BackendFilesystem || strings.TrimSpace(entry.Path) == "" {
		return fmt.Errorf("repository %q is not a registered filesystem repository", *name)
	}
	if _, err := s.Resolver.Resolve(ctx, *name); err != nil {
		return err
	}
	relParts, err := filesystemBlobPath(entry.Path, *blobID)
	if err != nil {
		return fmt.Errorf("resolve exact Kopia blob path: %w", err)
	}
	finalPath := filepath.Join(append([]string{entry.Path}, relParts...)...)
	if _, err := os.Lstat(finalPath); err == nil {
		return fmt.Errorf("target blob already exists; refusing to replace repository data")
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect target blob: %w", err)
	}
	candidates, err := filepath.Glob(finalPath + ".tmp.*")
	if err != nil {
		return fmt.Errorf("find adjacent temp candidates: %w", err)
	}
	if len(candidates) != 1 {
		return fmt.Errorf("expected exactly one adjacent temp candidate; found %d", len(candidates))
	}
	candidatePath := candidates[0]
	info, err := os.Stat(candidatePath)
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 {
		return fmt.Errorf("temp candidate is not a readable non-empty regular file")
	}
	candidateSHA, err := fileSHA256(candidatePath)
	if err != nil {
		return fmt.Errorf("read temp candidate: %w", err)
	}
	if candidateSHA != expectedSHA {
		return fmt.Errorf("temp candidate hash changed: got %s, expected %s", candidateSHA, expectedSHA)
	}
	staging, err := os.CreateTemp(filepath.Dir(finalPath), filepath.Base(finalPath)+".dbm-recovery-*")
	if err != nil {
		return fmt.Errorf("create same-volume staging blob: %w", err)
	}
	stagingPath := staging.Name()
	defer os.Remove(stagingPath)
	in, err := os.Open(candidatePath)
	if err != nil {
		_ = staging.Close()
		return fmt.Errorf("open temp candidate: %w", err)
	}
	_, copyErr := io.Copy(staging, in)
	inputCloseErr := in.Close()
	if copyErr != nil {
		_ = staging.Close()
		return fmt.Errorf("stage candidate blob: %w", copyErr)
	}
	if inputCloseErr != nil {
		_ = staging.Close()
		return fmt.Errorf("close temp candidate: %w", inputCloseErr)
	}
	if err := staging.Sync(); err != nil {
		_ = staging.Close()
		return fmt.Errorf("sync staging blob: %w", err)
	}
	if err := staging.Close(); err != nil {
		return fmt.Errorf("close staging blob: %w", err)
	}
	stagedSHA, err := fileSHA256(stagingPath)
	if err != nil || stagedSHA != expectedSHA {
		return fmt.Errorf("staged candidate failed hash verification")
	}
	// Link is atomic and fails if another process created the target after the
	// earlier absence check. It never replaces an existing repository object.
	if err := os.Link(stagingPath, finalPath); err != nil {
		return fmt.Errorf("publish verified candidate without replacement: %w", err)
	}
	dir, dirErr := os.Open(filepath.Dir(finalPath))
	if dirErr == nil {
		dirErr = dir.Sync()
		_ = dir.Close()
	}
	finalSHA, err := fileSHA256(finalPath)
	if err != nil || finalSHA != expectedSHA {
		return fmt.Errorf("published blob failed post-write hash verification")
	}
	result := struct {
		Repository         string `json:"repository"`
		BlobID             string `json:"blob_id"`
		CandidatePath      string `json:"candidate_path"`
		PublishedPath      string `json:"published_path"`
		CandidateSHA256    string `json:"candidate_sha256"`
		PublishedBytes     int64  `json:"published_bytes"`
		CandidatePreserved bool   `json:"candidate_preserved"`
		TargetReplaced     bool   `json:"target_replaced"`
		Committed          bool   `json:"committed"`
		DurabilityWarning  string `json:"durability_warning,omitempty"`
	}{
		Repository: *name, BlobID: *blobID, CandidatePath: candidatePath,
		PublishedPath: finalPath, CandidateSHA256: expectedSHA,
		PublishedBytes: info.Size(), CandidatePreserved: true,
		TargetReplaced: false, Committed: true,
	}
	if dirErr != nil {
		result.DurabilityWarning = "blob is visible and hash-verified, but syncing its shard directory failed: " + dirErr.Error()
	}
	if *jsonOut {
		return json.NewEncoder(s.out()).Encode(result)
	}
	fmt.Fprintf(s.out(), "Restored one orphan Kopia temp blob for %s\nBlob: %s\nPublished: %s\nCandidate preserved: yes\nTarget replaced: no\nBytes: %d\nSHA-256: %s\n", result.Repository, result.BlobID, result.PublishedPath, result.PublishedBytes, result.CandidateSHA256)
	if result.DurabilityWarning != "" {
		fmt.Fprintf(s.out(), "Warning: %s\n", result.DurabilityWarning)
	}
	return nil
}

// QuarantineOrphanTempBlob moves one exact, unpaired Kopia content temp out of
// the repository's shard namespace without deleting or reading its bytes. This
// is for a temp entry that prevents directory enumeration but cannot be stat'ed
// or read; it is deliberately not a q-blob restore and does not touch a final
// object. The sibling quarantine remains on the same filesystem so the move is
// reversible and does not copy data across devices.
func (s Service) QuarantineOrphanTempBlob(ctx context.Context, args []string) error {
	fs := cmdutil.NewFlagSet("repo quarantine-orphan-temp-blob")
	name := fs.String("name", "", "Registered filesystem repository name (required)")
	relativePath := fs.String("relative-temp-path", "", "Exact repository-relative s/...f.tmp.<suffix> path (required)")
	confirm := fs.String("confirm-quarantine", "", "Must equal quarantine-orphan-temp:<repository-name>:<relative-temp-path>")
	jsonOut := fs.Bool("json", false, "Emit a structured quarantine result")
	if err := cmdutil.Parse(fs, args); err != nil {
		return err
	}
	if err := cmdutil.RequireName(*name); err != nil {
		return err
	}
	rel := filepath.Clean(strings.TrimSpace(*relativePath))
	if rel == "." || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("relative-temp-path must stay inside the repository")
	}
	slashRel := filepath.ToSlash(rel)
	base := filepath.Base(rel)
	tempMarker := strings.LastIndex(base, ".tmp.")
	if !strings.HasPrefix(slashRel, "s/") || tempMarker <= 0 || !strings.HasSuffix(base[:tempMarker], ".f") || tempMarker+len(".tmp.") >= len(base) {
		return fmt.Errorf("only an exact s/... .f.tmp.<suffix> path can be quarantined")
	}
	wantConfirmation := "quarantine-orphan-temp:" + *name + ":" + slashRel
	if strings.TrimSpace(*confirm) != wantConfirmation {
		return fmt.Errorf("quarantine requires --confirm-quarantine %q", wantConfirmation)
	}
	entry, found, err := s.Registry.Get(*name)
	if err != nil {
		return err
	}
	if !found || entry.Backend != registry.BackendFilesystem || strings.TrimSpace(entry.Path) == "" {
		return fmt.Errorf("repository %q is not a registered filesystem repository", *name)
	}
	if _, err := s.Resolver.Resolve(ctx, *name); err != nil {
		return err
	}
	repoPath, err := filepath.EvalSymlinks(entry.Path)
	if err != nil {
		return fmt.Errorf("resolve repository path: %w", err)
	}
	repoPath, err = filepath.Abs(repoPath)
	if err != nil {
		return fmt.Errorf("resolve absolute repository path: %w", err)
	}
	shardParentPath := repoPath
	for _, component := range strings.Split(filepath.Dir(rel), string(filepath.Separator)) {
		if component == "." || component == "" {
			continue
		}
		shardParentPath = filepath.Join(shardParentPath, component)
		info, err := os.Lstat(shardParentPath)
		if err != nil {
			return fmt.Errorf("inspect exact shard directory component %q: %w", component, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("exact shard directory component %q is not a real directory", component)
		}
	}
	sourcePath := filepath.Join(repoPath, rel)
	parentPath := filepath.Dir(sourcePath)
	finalName := base[:tempMarker]
	finalPath := filepath.Join(parentPath, finalName)
	entries, err := os.ReadDir(parentPath)
	if err != nil {
		return fmt.Errorf("read exact shard directory: %w", err)
	}
	matchingTemps := 0
	sourceListed := false
	for _, item := range entries {
		if strings.HasPrefix(item.Name(), finalName+".tmp.") {
			matchingTemps++
			if item.Name() == base {
				sourceListed = true
			}
		}
	}
	if !sourceListed || matchingTemps != 1 {
		return fmt.Errorf("expected the exact temp source to be the only adjacent temp; present=%t adjacent=%d", sourceListed, matchingTemps)
	}
	if _, err := os.Lstat(finalPath); err == nil {
		return fmt.Errorf("a final blob exists beside the temp; refusing to quarantine it")
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect final blob path: %w", err)
	}
	quarantineRoot, err := os.MkdirTemp(filepath.Dir(repoPath), filepath.Base(repoPath)+"-recovery-quarantine-*")
	if err != nil {
		return fmt.Errorf("create unique same-volume quarantine root: %w", err)
	}
	quarantinePath := filepath.Join(quarantineRoot, rel)
	if err := os.MkdirAll(filepath.Dir(quarantinePath), 0o700); err != nil {
		_ = os.RemoveAll(quarantineRoot)
		return fmt.Errorf("create same-volume quarantine directory: %w", err)
	}
	if err := os.Rename(sourcePath, quarantinePath); err != nil {
		_ = os.RemoveAll(quarantineRoot)
		return fmt.Errorf("move exact temp to same-volume quarantine without replacement: %w", err)
	}
	result := struct {
		Repository           string `json:"repository"`
		OriginalPath         string `json:"original_path"`
		QuarantinePath       string `json:"quarantine_path"`
		FinalBlobPath        string `json:"final_blob_path"`
		SourceDataDeleted    bool   `json:"source_data_deleted"`
		FinalBlobWritten     bool   `json:"final_blob_written"`
		QuarantineSameVolume bool   `json:"quarantine_same_volume"`
		Committed            bool   `json:"committed"`
	}{
		Repository: *name, OriginalPath: sourcePath, QuarantinePath: quarantinePath,
		FinalBlobPath: finalPath, SourceDataDeleted: false, FinalBlobWritten: false,
		QuarantineSameVolume: true, Committed: true,
	}
	if *jsonOut {
		return json.NewEncoder(s.out()).Encode(result)
	}
	fmt.Fprintf(s.out(), "Quarantined one orphan Kopia temp for %s\nOriginal: %s\nQuarantine: %s\nFinal blob written: no\nSource data deleted: no\n", result.Repository, result.OriginalPath, result.QuarantinePath)
	return nil
}

func validKopiaBlobID(value string) bool {
	if value == "" || strings.TrimSpace(value) != value {
		return false
	}
	for _, r := range value {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-') {
			return false
		}
	}
	return true
}

func digestText(value []byte) string {
	digest := sha256.Sum256(value)
	return hex.EncodeToString(digest[:])
}

func summarizeCacheProbeError(err error) string {
	message := strings.TrimSpace(err.Error())
	if strings.Contains(message, "BLOB not found") {
		const marker = `blob "`
		if start := strings.LastIndex(message, marker); start >= 0 {
			value := message[start+len(marker):]
			if end := strings.Index(value, `"`); end > 0 {
				return fmt.Sprintf("snapshot metadata references missing blob %s (BLOB not found)", value[:end])
			}
		}
		return "repository references a missing blob (BLOB not found)"
	}
	if strings.Contains(message, "input/output error") || strings.Contains(message, "EIO") {
		return "repository storage returned an I/O error (EIO)"
	}
	if line, _, ok := strings.Cut(message, "\n"); ok {
		message = line
	}
	const maxSummaryBytes = 512
	if len(message) > maxSummaryBytes {
		message = message[:maxSummaryBytes] + "…"
	}
	return message
}

func writeCacheProbeResult(out io.Writer, jsonOut bool, result any) error {
	if jsonOut {
		return json.NewEncoder(out).Encode(result)
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(out, string(data))
	return err
}

// RepairFilesystemFormatPlan requests Kopia's non-mutating dry-run for
// recovery of the filesystem repository format blob. It is not a general
// repository or snapshot-manifest repair operation. Applying repair remains
// separate and is not supported here.
func (s Service) RepairFilesystemFormatPlan(ctx context.Context, args []string) error {
	fs := cmdutil.NewFlagSet("repo repair-format-plan")
	name := fs.String("name", "", "Registered filesystem repository name (required)")
	jsonOut := fs.Bool("json", false, "Emit a structured repair plan")
	if err := cmdutil.Parse(fs, args); err != nil {
		return err
	}
	if err := cmdutil.RequireName(*name); err != nil {
		return err
	}
	entry, found, err := s.Registry.Get(*name)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("repository %q is not registered", *name)
	}
	if entry.Backend != registry.BackendFilesystem || strings.TrimSpace(entry.Path) == "" {
		return fmt.Errorf("repository %q is not a registered filesystem repository", *name)
	}
	if strings.TrimSpace(entry.ConfigFile) == "" {
		return fmt.Errorf("repository %q has no registered Kopia configuration file", *name)
	}
	argv := []string{"--config-file", entry.ConfigFile, "--dangerous-commands=enabled", "repository", "repair", "filesystem", "--path", entry.Path, "--dry-run"}
	output, err := s.Runner.Run(ctx, kexec.Call{Args: argv})
	if err != nil {
		return fmt.Errorf("filesystem repository format-recovery dry-run %q: %w", *name, err)
	}
	if strings.TrimSpace(string(output)) == "" {
		return fmt.Errorf("filesystem repository format-recovery dry-run %q returned no reviewable plan; no changes were made", *name)
	}
	digest := sha256.Sum256(output)
	plan := struct {
		Repository  string `json:"repository"`
		Backend     string `json:"backend"`
		Path        string `json:"path"`
		DryRun      bool   `json:"dry_run"`
		PlanSHA256  string `json:"plan_sha256"`
		PlanOutput  string `json:"plan_output"`
		ApplyAction string `json:"apply_action"`
	}{
		Repository: *name, Backend: entry.Backend, Path: entry.Path,
		DryRun: true, PlanSHA256: hex.EncodeToString(digest[:]),
		PlanOutput:  strings.TrimSpace(string(output)),
		ApplyAction: "not_supported_by_this_command",
	}
	if *jsonOut {
		return json.NewEncoder(s.out()).Encode(plan)
	}
	fmt.Fprintf(s.out(), "Non-mutating filesystem repository format-recovery plan for %s\nPath: %s\nPlan SHA-256: %s\nApply: not supported by this command\n\n%s\n", plan.Repository, plan.Path, plan.PlanSHA256, plan.PlanOutput)
	return nil
}

// IndexRecoveryScan invokes Kopia's non-committing index recovery pass. It
// scans the pack blobs for recoverable index records but never commits them or
// deletes existing indexes. The resource exposes this read-only diagnostic so
// an operator can assess recoverability before a separate, reviewed repair.
func (s Service) IndexRecoveryScan(ctx context.Context, args []string) error {
	fs := cmdutil.NewFlagSet("repo index-recovery-scan")
	name := fs.String("name", "", "Registered filesystem repository name (required)")
	jsonOut := fs.Bool("json", false, "Emit a structured scan result")
	if err := cmdutil.Parse(fs, args); err != nil {
		return err
	}
	if err := cmdutil.RequireName(*name); err != nil {
		return err
	}
	entry, found, err := s.Registry.Get(*name)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("repository %q is not registered", *name)
	}
	if entry.Backend != registry.BackendFilesystem || strings.TrimSpace(entry.Path) == "" {
		return fmt.Errorf("repository %q is not a registered filesystem repository", *name)
	}
	if strings.TrimSpace(entry.ConfigFile) == "" {
		return fmt.Errorf("repository %q has no registered Kopia configuration file", *name)
	}
	target, err := s.Resolver.Resolve(ctx, *name)
	if err != nil {
		return err
	}
	argv := append(target.GlobalArgs(), "--log-level=info", "--no-progress", "--dangerous-commands=enabled", "index", "recover", "--no-commit")
	output, err := s.Runner.Run(ctx, kexec.Call{Args: argv, Env: target.Env, CaptureStderr: true})
	if err != nil {
		return fmt.Errorf("non-committing index recovery scan %q: %w", *name, err)
	}
	if strings.TrimSpace(string(output)) == "" {
		return fmt.Errorf("non-committing index recovery scan %q returned no reviewable output; no indexes were committed", *name)
	}
	digest := sha256.Sum256(output)
	result := struct {
		Repository string `json:"repository"`
		Backend    string `json:"backend"`
		Path       string `json:"path"`
		Committed  bool   `json:"committed"`
		Deleted    bool   `json:"deleted"`
		OutputSHA  string `json:"output_sha256"`
		Output     string `json:"output"`
	}{
		Repository: *name, Backend: entry.Backend, Path: entry.Path,
		Committed: false, Deleted: false,
		OutputSHA: hex.EncodeToString(digest[:]), Output: strings.TrimSpace(string(output)),
	}
	if *jsonOut {
		return json.NewEncoder(s.out()).Encode(result)
	}
	fmt.Fprintf(s.out(), "Non-committing index recovery scan for %s\nPath: %s\nCommitted: no\nDeleted: no\nOutput SHA-256: %s\n\n%s\n", result.Repository, result.Path, result.OutputSHA, result.Output)
	return nil
}

// IndexRecoveryCommit applies Kopia's recovered index records to one registered
// filesystem repository. It is deliberately separate from the read-only scan
// and requires an exact confirmation phrase bound to the registered name.
// Kopia's delete-indexes option remains unset, so this operation does not
// delete existing indexes.
func (s Service) IndexRecoveryCommit(ctx context.Context, args []string) error {
	fs := cmdutil.NewFlagSet("repo index-recovery-commit")
	name := fs.String("name", "", "Registered filesystem repository name (required)")
	confirm := fs.String("confirm-commit", "", "Must equal commit-index-recovery:<repository-name>")
	jsonOut := fs.Bool("json", false, "Emit a structured commit result")
	if err := cmdutil.Parse(fs, args); err != nil {
		return err
	}
	if err := cmdutil.RequireName(*name); err != nil {
		return err
	}
	wantConfirmation := "commit-index-recovery:" + *name
	if strings.TrimSpace(*confirm) != wantConfirmation {
		return fmt.Errorf("index recovery commit requires --confirm-commit %q", wantConfirmation)
	}
	entry, found, err := s.Registry.Get(*name)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("repository %q is not registered", *name)
	}
	if entry.Backend != registry.BackendFilesystem || strings.TrimSpace(entry.Path) == "" {
		return fmt.Errorf("repository %q is not a registered filesystem repository", *name)
	}
	if strings.TrimSpace(entry.ConfigFile) == "" {
		return fmt.Errorf("repository %q has no registered Kopia configuration file", *name)
	}
	target, err := s.Resolver.Resolve(ctx, *name)
	if err != nil {
		return err
	}
	argv := append(target.GlobalArgs(), "--log-level=info", "--no-progress", "--dangerous-commands=enabled", "index", "recover", "--commit")
	output, err := s.Runner.Run(ctx, kexec.Call{Args: argv, Env: target.Env, CaptureStderr: true})
	if err != nil {
		return fmt.Errorf("index recovery commit %q: %w", *name, err)
	}
	outputText := strings.TrimSpace(string(output))
	if outputText == "" {
		outputText = "Kopia exited successfully but did not emit a summary."
	}
	digest := sha256.Sum256([]byte(outputText))
	result := struct {
		Repository string `json:"repository"`
		Backend    string `json:"backend"`
		Path       string `json:"path"`
		Committed  bool   `json:"committed"`
		Deleted    bool   `json:"deleted"`
		OutputSHA  string `json:"output_sha256"`
		Output     string `json:"output"`
	}{
		Repository: *name, Backend: entry.Backend, Path: entry.Path,
		Committed: true, Deleted: false,
		OutputSHA: hex.EncodeToString(digest[:]), Output: outputText,
	}
	if *jsonOut {
		return json.NewEncoder(s.out()).Encode(result)
	}
	fmt.Fprintf(s.out(), "Index recovery commit for %s\nPath: %s\nCommitted: yes\nDeleted: no\nOutput SHA-256: %s\n\n%s\n", result.Repository, result.Path, result.OutputSHA, result.Output)
	return nil
}

// Disconnect detaches from a repository, leaving the registry entry so it can
// be reconnected later.
func (s Service) Disconnect(ctx context.Context, args []string) error {
	return s.simpleRepoCommand(ctx, args, "repo disconnect", []string{"repository", "disconnect"}, false)
}

// Delete removes the local resource-kopia metadata for a repository and
// deletes the credential and backend secret refs that belong to it. Backend object bytes are not
// removed; operators can reconnect later only if they re-provision the secret.
func (s Service) Delete(ctx context.Context, args []string) error {
	fs := cmdutil.NewFlagSet("repo delete")
	name := fs.String("name", "", "Repository name (required)")
	if err := cmdutil.Parse(fs, args); err != nil {
		return err
	}
	if err := cmdutil.RequireName(*name); err != nil {
		return err
	}
	entry, found, err := s.Registry.Get(*name)
	if err != nil {
		return err
	}
	if !found {
		fmt.Fprintf(s.out(), "repository %q was not registered\n", *name)
		return nil
	}
	identity, err := kopiaregistry.PassphraseIdentity(*name)
	if err != nil {
		return err
	}
	if s.Credentials == nil {
		return fmt.Errorf("credential authority is unavailable")
	}
	if err := s.Credentials.Delete(identity, kopiaregistry.PassphraseField); err != nil {
		return err
	}
	if entry.Backend == registry.BackendS3 {
		if err := credentials.DeleteS3Credentials(s.Credentials, *name); err != nil {
			return err
		}
	}
	if err := s.Registry.Remove(*name); err != nil {
		return err
	}
	if strings.TrimSpace(entry.ConfigFile) != "" {
		if err := os.Remove(entry.ConfigFile); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove repo config %s: %w", entry.ConfigFile, err)
		}
	}
	if strings.TrimSpace(entry.CacheDir) != "" {
		if err := os.RemoveAll(entry.CacheDir); err != nil {
			return fmt.Errorf("remove repo cache %s: %w", entry.CacheDir, err)
		}
	}
	fmt.Fprintf(s.out(), "deleted repository metadata %q\n", *name)
	return nil
}

// List prints the repositories this host has registered (from the registry,
// which never holds secrets).
func (s Service) List(_ context.Context, args []string) error {
	fs := cmdutil.NewFlagSet("repo list")
	jsonOut := fs.Bool("json", false, "Emit JSON")
	if err := cmdutil.Parse(fs, args); err != nil {
		return err
	}
	entries, err := s.Registry.Load()
	if err != nil {
		return err
	}
	if *jsonOut {
		return cmdutil.WriteJSON(s.out(), entries)
	}
	if len(entries) == 0 {
		_, err := fmt.Fprintln(s.out(), "No repositories registered.")
		return err
	}
	for _, e := range entries {
		dest := e.Path
		if e.Backend == registry.BackendS3 {
			dest = fmt.Sprintf("s3://%s/%s @ %s", e.Bucket, e.Prefix, e.Endpoint)
		}
		if _, err := fmt.Fprintf(s.out(), "%-24s %-12s %s\n", e.Name, e.Backend, dest); err != nil {
			return err
		}
	}
	return nil
}

// simpleRepoCommand resolves a repo by --name and runs a fixed kopia subcommand
// with the repo's global args + secret env, optionally appending --json.
func (s Service) simpleRepoCommand(ctx context.Context, args []string, cmdName string, verb []string, supportsJSON bool) error {
	fs := cmdutil.NewFlagSet(cmdName)
	name := fs.String("name", "", "Repository name (required)")
	var jsonOut *bool
	if supportsJSON {
		jsonOut = fs.Bool("json", false, "Emit kopia's native JSON")
	}
	if err := cmdutil.Parse(fs, args); err != nil {
		return err
	}
	if err := cmdutil.RequireName(*name); err != nil {
		return err
	}
	target, err := s.Resolver.Resolve(ctx, *name)
	if err != nil {
		return err
	}
	argv := append(target.GlobalArgs(), verb...)
	if supportsJSON && *jsonOut {
		argv = append(argv, "--json")
	}
	out, err := s.Runner.Run(ctx, kexec.Call{Args: argv, Env: target.Env})
	if err != nil {
		return err
	}
	if err := target.RetireLegacyPassphrase(); err != nil {
		return err
	}
	_, err = s.out().Write(cmdutil.EnsureTrailingNewline(out))
	return err
}

// resolveS3Creds returns the S3 credentials for a new repo: stores
// flag-provided creds in the credential authority, otherwise reads existing
// authority-backed creds.
func (s Service) resolveS3Creds(ctx context.Context, name, accessKey, secretKey string) (credentials.S3Credentials, error) {
	if strings.TrimSpace(accessKey) != "" || strings.TrimSpace(secretKey) != "" {
		creds := credentials.S3Credentials{AccessKeyID: accessKey, SecretAccessKey: secretKey}
		if !creds.Valid() {
			return credentials.S3Credentials{}, fmt.Errorf("both --access-key and --secret-access-key are required together")
		}
		if err := credentials.PutS3Credentials(s.Credentials, name, creds); err != nil {
			return credentials.S3Credentials{}, err
		}
		return creds, nil
	}
	creds, found, err := credentials.S3CredentialsFor(s.Credentials, name)
	if err != nil {
		return credentials.S3Credentials{}, err
	}
	if !found || !creds.Valid() {
		return credentials.S3Credentials{}, fmt.Errorf("no S3 credentials for repository %q; pass --access-key/--secret-access-key or provision them in the credential authority", name)
	}
	return creds, nil
}

// registerForConnect upserts a registry entry from connect-time backend flags,
// ensuring the passphrase exists and storing any provided S3 creds.
func (s Service) registerForConnect(ctx context.Context, name, backend, path, bucket, endpoint, prefix, region string, disableTLS bool, accessKey, secretKey string) error {
	if s.Credentials == nil {
		return fmt.Errorf("credential authority is unavailable")
	}
	value, err := credentials.ResolvePassphrase(s.Credentials, name, s.Env.RepoLegacyPasswordFile(name))
	if err != nil {
		return fmt.Errorf("cannot connect %q: read repository passphrase: %w", name, err)
	}
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("cannot connect %q: repository passphrase is not configured", name)
	}
	entry := registry.Entry{
		Name:       name,
		ConfigFile: s.Env.RepoConfigFile(name),
		CacheDir:   s.Env.RepoCacheDir(name),
	}
	switch strings.ToLower(strings.TrimSpace(backend)) {
	case registry.BackendFilesystem:
		if strings.TrimSpace(path) == "" {
			return fmt.Errorf("--path is required for the filesystem backend")
		}
		entry.Backend = registry.BackendFilesystem
		entry.Path = path
	case registry.BackendS3:
		if strings.TrimSpace(bucket) == "" {
			return fmt.Errorf("--bucket is required for the s3 backend")
		}
		entry.Backend = registry.BackendS3
		entry.Bucket = bucket
		entry.Endpoint = endpoint
		entry.Prefix = prefix
		entry.Region = region
		entry.DisableTLS = disableTLS
		if strings.TrimSpace(accessKey) != "" || strings.TrimSpace(secretKey) != "" {
			if _, err := s.resolveS3Creds(ctx, name, accessKey, secretKey); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unsupported --backend %q (use filesystem or s3)", backend)
	}
	return s.Registry.Upsert(entry)
}

// s3Args renders the kopia s3 backend flags for an entry. Never includes
// credentials — those flow through the env overlay.
func s3Args(e registry.Entry) []string {
	args := []string{"s3", "--bucket", e.Bucket}
	if strings.TrimSpace(e.Endpoint) != "" {
		args = append(args, "--endpoint", e.Endpoint)
	}
	if strings.TrimSpace(e.Prefix) != "" {
		args = append(args, "--prefix", e.Prefix)
	}
	if strings.TrimSpace(e.Region) != "" {
		args = append(args, "--region", e.Region)
	}
	if e.DisableTLS {
		args = append(args, "--disable-tls")
	}
	return args
}

func (s Service) report(jsonOut bool, raw []byte, human string) error {
	if jsonOut && len(strings.TrimSpace(string(raw))) > 0 {
		_, err := s.out().Write(cmdutil.EnsureTrailingNewline(raw))
		return err
	}
	_, err := fmt.Fprintln(s.out(), human)
	return err
}
