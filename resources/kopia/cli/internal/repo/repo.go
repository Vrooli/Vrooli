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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"resource-kopia/cli/internal/cmdutil"
	"resource-kopia/cli/internal/credentials"
	"resource-kopia/cli/internal/env"
	"resource-kopia/cli/internal/kexec"
	"resource-kopia/cli/internal/registry"
	"resource-kopia/cli/internal/repoctx"

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
	identity, err := kopiaregistry.PassphraseIdentity(name)
	if err != nil {
		return err
	}
	if s.Credentials == nil {
		return fmt.Errorf("credential authority is unavailable")
	}
	value, err := s.Credentials.Resolve(identity, kopiaregistry.PassphraseField)
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
