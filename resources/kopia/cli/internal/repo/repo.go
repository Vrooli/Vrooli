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
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"resource-kopia/cli/internal/cmdutil"
	"resource-kopia/cli/internal/env"
	"resource-kopia/cli/internal/kexec"
	"resource-kopia/cli/internal/registry"
	"resource-kopia/cli/internal/repoctx"
	"resource-kopia/cli/internal/vault"
	"strings"
)

// Service wires the dependencies the repository commands need.
type Service struct {
	Runner   kexec.Runner
	Vault    vault.Vault
	Registry *registry.Registry
	Resolver repoctx.Resolver
	Env      env.Runtime
	Out      io.Writer
}

func (s Service) out() io.Writer {
	if s.Out != nil {
		return s.Out
	}
	return os.Stdout
}

// Create provisions a new kopia repository (== Vrooli destination). Encryption
// is left at kopia's secure default; the passphrase is generated-and-stored in
// vault on first creation.
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
		accessKey  = fs.String("access-key", "", "S3 backend: access key id (stored in vault; else read from vault)")
		secretKey  = fs.String("secret-access-key", "", "S3 backend: secret access key (stored in vault)")
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

	// Passphrase: generate-then-store on first creation; never empty/default.
	passphrase, err := vault.EnsurePassphrase(ctx, s.Vault, *name)
	if err != nil {
		return err
	}
	secretEnv := map[string]string{repoctx.EnvPassword: passphrase}

	entry := registry.Entry{
		Name:       *name,
		ConfigFile: cfg,
		CacheDir:   cache,
	}

	verbArgs := []string{"--config-file", cfg, "repository", "create"}
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
	verbArgs = append(verbArgs, "--cache-directory", cache)

	out, err := s.Runner.Run(ctx, kexec.Call{Args: verbArgs, Env: secretEnv})
	if err != nil {
		return err
	}
	if err := s.Registry.Upsert(entry); err != nil {
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
		accessKey  = fs.String("access-key", "", "S3 backend: access key id (stored in vault)")
		secretKey  = fs.String("secret-access-key", "", "S3 backend: secret access key (stored in vault)")
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

// Stats prints content/size statistics for a repository (storage-usage views).
func (s Service) Stats(ctx context.Context, args []string) error {
	return s.simpleRepoCommand(ctx, args, "repo stats", []string{"content", "stats"}, true)
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
// flag-provided creds in vault, otherwise reads existing creds from vault.
func (s Service) resolveS3Creds(ctx context.Context, name, accessKey, secretKey string) (vault.S3Credentials, error) {
	if strings.TrimSpace(accessKey) != "" || strings.TrimSpace(secretKey) != "" {
		creds := vault.S3Credentials{AccessKeyID: accessKey, SecretAccessKey: secretKey}
		if !creds.Valid() {
			return vault.S3Credentials{}, fmt.Errorf("both --access-key and --secret-access-key are required together")
		}
		if err := vault.PutS3Credentials(ctx, s.Vault, name, creds); err != nil {
			return vault.S3Credentials{}, err
		}
		return creds, nil
	}
	creds, found, err := vault.S3CredentialsFor(ctx, s.Vault, name)
	if err != nil {
		return vault.S3Credentials{}, err
	}
	if !found || !creds.Valid() {
		return vault.S3Credentials{}, fmt.Errorf("no S3 credentials for repository %q; pass --access-key/--secret-access-key or store them in vault", name)
	}
	return creds, nil
}

// registerForConnect upserts a registry entry from connect-time backend flags,
// ensuring the passphrase exists and storing any provided S3 creds.
func (s Service) registerForConnect(ctx context.Context, name, backend, path, bucket, endpoint, prefix, region string, disableTLS bool, accessKey, secretKey string) error {
	if _, err := vault.RequirePassphrase(ctx, s.Vault, name); err != nil {
		return fmt.Errorf("cannot connect %q: %w", name, err)
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
