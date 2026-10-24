package recoverypoint

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/vrooli/binaryfetch"
)

// Runner executes one tool with argv and an explicit environment. It is the
// only way a provider reaches a process: no shell string is ever built.
type Runner interface {
	Run(ctx context.Context, tool string, argv []string, env []string) ([]byte, error)
}

// CaptureResult is what a provider produced for one binding.
type CaptureResult struct {
	ArtifactPath string
	Inventory    Inventory
}

// Provider captures and restores one kind of binding through its owner
// (database-native tooling, object inventory). Every method receives the
// binding so a provider never keeps per-binding state.
type Provider interface {
	// Mode names the consistency mode the provider guarantees.
	Mode() string
	// Capture writes the provider artifact beneath stageDir and returns its
	// inventory.
	Capture(ctx context.Context, b Binding, stageDir string) (CaptureResult, error)
	// EnsureClean refuses (restore_target_not_clean) when target already
	// holds data. It is called for every binding before any restore writes.
	EnsureClean(ctx context.Context, b Binding, target string) error
	// Restore loads the artifact into target.
	Restore(ctx context.Context, b Binding, artifactPath, target string) error
	// Discard removes what Restore wrote into target after a later binding
	// failed, so a failed multi-binding restore leaves no half-restored data
	// where the provider can undo it. Providers that cannot undo return an
	// error naming the residue.
	Discard(ctx context.Context, b Binding, target string) error
	// Inventory measures target after a restore.
	Inventory(ctx context.Context, b Binding, target string) (Inventory, error)
}

// Registry maps Binding.Provider to its implementation.
type Registry map[string]Provider

// Lookup returns the provider for a binding or a typed provider_unavailable.
func (r Registry) Lookup(b Binding) (Provider, error) {
	p, ok := r[b.Provider]
	if !ok || p == nil {
		return nil, newError(CodeProviderUnavailable, "binding %s: no provider registered for %q", b.ID, b.Provider).withDetail("binding", b.ID).withDetail("provider", b.Provider)
	}
	return p, nil
}

// Boundary enters a declared consistency boundary for one binding and
// returns the record plus a release function.
type Boundary interface {
	Enter(ctx context.Context, b Binding, mode string) (ConsistencyRecord, func(context.Context) error, error)
}

// ApplicationHooks is the default boundary: it runs the binding's declared
// quiesce hook when one exists and records the outcome honestly when none
// does. A database-native provider without a hook is snapshot_safe; any
// other provider without a hook is not_declared.
type ApplicationHooks struct {
	Runner Runner
}

// Enter implements Boundary.
func (h ApplicationHooks) Enter(ctx context.Context, b Binding, mode string) (ConsistencyRecord, func(context.Context) error, error) {
	record := ConsistencyRecord{Binding: b.ID, Mode: mode, Token: newToken()}
	release := func(context.Context) error { return nil }
	if b.Quiesce == nil {
		if mode == ModeDatabaseNative {
			record.WriteQuiescence = QuiescenceSnapshotSafe
		} else {
			record.WriteQuiescence = QuiescenceNotDeclared
		}
		return record, release, nil
	}
	if h.Runner == nil {
		return record, release, newError(CodeProviderUnavailable, "binding %s declares a quiesce hook but no runner is available", b.ID)
	}
	if out, err := h.Runner.Run(ctx, b.Quiesce.Tool, b.Quiesce.Argv, nil); err != nil {
		return record, release, newError(CodeCaptureFailed, "binding %s: quiesce hook failed: %v: %s", b.ID, err, strings.TrimSpace(string(out)))
	}
	record.WriteQuiescence = QuiescenceQuiesced
	if b.Release != nil {
		hook := *b.Release
		release = func(ctx context.Context) error {
			if out, err := h.Runner.Run(ctx, hook.Tool, hook.Argv, nil); err != nil {
				return fmt.Errorf("binding %s: release hook failed: %v: %s", b.ID, err, strings.TrimSpace(string(out)))
			}
			return nil
		}
	}
	return record, release, nil
}

func newToken() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return ""
	}
	return hex.EncodeToString(buf[:])
}

// ---------------------------------------------------------------------------
// Object store provider: inventory + checksum over a directory binding.
// ---------------------------------------------------------------------------

// ObjectStore captures a directory binding as a tar archive with a
// deterministic inventory, and restores it only into a clean directory.
type ObjectStore struct {
	// MaxExpandedBytes bounds extraction; zero is unbounded.
	MaxExpandedBytes int64
}

// Mode implements Provider.
func (ObjectStore) Mode() string { return ModeObjectInventory }

// Capture implements Provider.
func (ObjectStore) Capture(_ context.Context, b Binding, stageDir string) (CaptureResult, error) {
	dir := filepath.Clean(b.Locator)
	if !filepath.IsAbs(dir) {
		return CaptureResult{}, newError(CodeInvalidArgument, "binding %s: object store locator must be an absolute directory", b.ID)
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return CaptureResult{}, newError(CodeCaptureFailed, "binding %s: %s is not a readable directory", b.ID, dir)
	}
	entries, err := ListFiles(dir)
	if err != nil {
		return CaptureResult{}, newError(CodeCaptureFailed, "binding %s: inventory: %v", b.ID, err)
	}
	artifact := filepath.Join(stageDir, b.ID+".tar")
	out, err := os.Create(artifact) //nolint:gosec // stage dir is owned by the capture
	if err != nil {
		return CaptureResult{}, newError(CodeCaptureFailed, "binding %s: create archive: %v", b.ID, err)
	}
	tw := tar.NewWriter(out)
	for _, e := range entries {
		if err := addFile(tw, dir, e); err != nil {
			_ = tw.Close()
			_ = out.Close()
			return CaptureResult{}, newError(CodeCaptureFailed, "binding %s: archive %s: %v", b.ID, e.Path, err)
		}
	}
	if err := tw.Close(); err != nil {
		_ = out.Close()
		return CaptureResult{}, newError(CodeCaptureFailed, "binding %s: finish archive: %v", b.ID, err)
	}
	if err := out.Close(); err != nil {
		return CaptureResult{}, newError(CodeCaptureFailed, "binding %s: close archive: %v", b.ID, err)
	}
	return CaptureResult{ArtifactPath: artifact, Inventory: InventoryOf(entries)}, nil
}

func addFile(tw *tar.Writer, root string, e FileEntry) error {
	path := filepath.Join(root, filepath.FromSlash(e.Path))
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	header.Name = e.Path
	header.Uname, header.Gname = "", ""
	header.Uid, header.Gid = 0, 0
	if err := tw.WriteHeader(header); err != nil {
		return err
	}
	f, err := os.Open(path) //nolint:gosec // path is inside the binding directory
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(tw, f)
	return err
}

// EnsureClean implements Provider.
func (ObjectStore) EnsureClean(_ context.Context, b Binding, target string) error {
	clean, err := DirIsClean(target)
	if err != nil {
		return newError(CodeRestoreFailed, "binding %s: inspect %s: %v", b.ID, target, err)
	}
	if !clean {
		return newError(CodeRestoreTargetNotClean, "binding %s: %s already holds data; restore never overwrites operator-owned data", b.ID, target).withDetail("binding", b.ID).withDetail("target", target)
	}
	return nil
}

// Restore implements Provider.
func (o ObjectStore) Restore(_ context.Context, b Binding, artifactPath, target string) error {
	if err := o.EnsureClean(context.Background(), b, target); err != nil {
		return err
	}
	if _, err := binaryfetch.ExtractArchiveBounded(artifactPath, "tar", target, binaryfetch.ExtractOptions{MaxExpandedBytes: o.MaxExpandedBytes}); err != nil {
		_ = os.RemoveAll(target)
		return newError(CodeRestoreFailed, "binding %s: extract into %s: %v", b.ID, target, err)
	}
	return nil
}

// Discard implements Provider.
func (ObjectStore) Discard(_ context.Context, _ Binding, target string) error {
	return os.RemoveAll(target)
}

// Inventory implements Provider.
func (ObjectStore) Inventory(_ context.Context, b Binding, target string) (Inventory, error) {
	inv, _, err := DirInventory(target)
	if err != nil {
		return Inventory{}, newError(CodeVerifyFailed, "binding %s: inventory %s: %v", b.ID, target, err)
	}
	return inv, nil
}

// ---------------------------------------------------------------------------
// PostgreSQL provider: database-native consistency through pg_dump/pg_restore.
// ---------------------------------------------------------------------------

// Default argv the PostgreSQL provider runs. resources/postgres/resource.json
// declares the same argv under deployment.backup; a test in
// scenario-to-cloud keeps the two in step.
var (
	PostgresDumpArgv    = []string{"--format=custom", "--no-password", "--dbname", "{database}", "--file", "{output}"}
	PostgresRestoreArgv = []string{"--no-password", "--exit-on-error", "--dbname", "{database}", "{input}"}
	PostgresListArgv    = []string{"--list", "{input}"}
	PostgresCleanQuery  = "SELECT count(*) FROM pg_catalog.pg_tables WHERE schemaname NOT IN ('pg_catalog','information_schema')"
)

// Postgres captures a database binding with pg_dump --format=custom (a
// consistent snapshot) and restores it with pg_restore into a database that
// holds no user tables. Credentials travel in the child environment supplied
// by CredentialEnv, never in argv.
type Postgres struct {
	Runner Runner
	// CredentialEnv returns PG* environment entries for the binding
	// (PGHOST, PGPORT, PGUSER, PGPASSWORD). It is resolved per call so key
	// material never rests in the provider.
	CredentialEnv func(ctx context.Context, b Binding) ([]string, error)
	// Tool names default to pg_dump, pg_restore and psql on PATH; ToolDir
	// prefixes them with the resource's native binary directory.
	ToolDir     string
	DumpTool    string
	RestoreTool string
	PSQLTool    string
}

// Mode implements Provider.
func (Postgres) Mode() string { return ModeDatabaseNative }

func (p Postgres) tool(name, override string) string {
	if override != "" {
		name = override
	}
	if p.ToolDir != "" {
		return filepath.Join(p.ToolDir, name)
	}
	return name
}

func (p Postgres) env(ctx context.Context, b Binding) ([]string, error) {
	if p.CredentialEnv == nil {
		return nil, newError(CodeProviderUnavailable, "binding %s: postgres credentials are not configured", b.ID)
	}
	env, err := p.CredentialEnv(ctx, b)
	if err != nil {
		return nil, newError(CodeProviderUnavailable, "binding %s: resolve postgres credentials: %v", b.ID, err)
	}
	return env, nil
}

func (p Postgres) run(ctx context.Context, b Binding, tool string, argv []string, subst map[string]string) ([]byte, error) {
	if p.Runner == nil {
		return nil, newError(CodeProviderUnavailable, "binding %s: no process runner is available", b.ID)
	}
	env, err := p.env(ctx, b)
	if err != nil {
		return nil, err
	}
	return p.Runner.Run(ctx, tool, Substitute(argv, subst), env)
}

// Substitute replaces whole-argument placeholders ({database}, {output},
// {input}) with their values. A placeholder is only ever a complete argv
// element, so a value can never split into extra arguments.
func Substitute(argv []string, values map[string]string) []string {
	out := make([]string, len(argv))
	for i, arg := range argv {
		if value, ok := values[arg]; ok {
			out[i] = value
			continue
		}
		out[i] = arg
	}
	return out
}

// Capture implements Provider.
func (p Postgres) Capture(ctx context.Context, b Binding, stageDir string) (CaptureResult, error) {
	artifact := filepath.Join(stageDir, b.ID+".pgdump")
	if out, err := p.run(ctx, b, p.tool("pg_dump", p.DumpTool), PostgresDumpArgv, map[string]string{"{database}": b.Locator, "{output}": artifact}); err != nil {
		return CaptureResult{}, newError(CodeCaptureFailed, "binding %s: pg_dump failed: %v: %s", b.ID, err, strings.TrimSpace(string(out)))
	}
	inv, err := p.dumpInventory(ctx, b, artifact)
	if err != nil {
		return CaptureResult{}, err
	}
	return CaptureResult{ArtifactPath: artifact, Inventory: inv}, nil
}

// dumpInventory counts table-of-contents entries of a custom-format dump.
// The dump bytes carry a timestamp, so the checksum is not comparable
// across captures.
func (p Postgres) dumpInventory(ctx context.Context, b Binding, artifact string) (Inventory, error) {
	out, err := p.run(ctx, b, p.tool("pg_restore", p.RestoreTool), PostgresListArgv, map[string]string{"{input}": artifact})
	if err != nil {
		return Inventory{}, newError(CodeCaptureFailed, "binding %s: pg_restore --list failed: %v: %s", b.ID, err, strings.TrimSpace(string(out)))
	}
	sum, err := FileSHA256(artifact)
	if err != nil {
		return Inventory{}, newError(CodeCaptureFailed, "binding %s: digest dump: %v", b.ID, err)
	}
	return Inventory{Count: countTOCEntries(out), Checksum: sum, Comparable: false}, nil
}

func countTOCEntries(listing []byte) int64 {
	var count int64
	for _, line := range bytes.Split(listing, []byte("\n")) {
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 || trimmed[0] == ';' {
			continue
		}
		count++
	}
	return count
}

// EnsureClean implements Provider: the target database must hold no user
// tables.
func (p Postgres) EnsureClean(ctx context.Context, b Binding, target string) error {
	out, err := p.run(ctx, b, p.tool("psql", p.PSQLTool), []string{"--no-password", "--tuples-only", "--no-align", "--dbname", "{database}", "--command", PostgresCleanQuery}, map[string]string{"{database}": target})
	if err != nil {
		return newError(CodeRestoreFailed, "binding %s: inspect database %s: %v: %s", b.ID, target, err, strings.TrimSpace(string(out)))
	}
	count, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return newError(CodeRestoreFailed, "binding %s: unexpected table count %q for %s", b.ID, strings.TrimSpace(string(out)), target)
	}
	if count != 0 {
		return newError(CodeRestoreTargetNotClean, "binding %s: database %s already holds %d user tables; restore never overwrites operator-owned data", b.ID, target, count).withDetail("binding", b.ID).withDetail("target", target)
	}
	return nil
}

// Restore implements Provider.
func (p Postgres) Restore(ctx context.Context, b Binding, artifactPath, target string) error {
	if err := p.EnsureClean(ctx, b, target); err != nil {
		return err
	}
	if out, err := p.run(ctx, b, p.tool("pg_restore", p.RestoreTool), PostgresRestoreArgv, map[string]string{"{database}": target, "{input}": artifactPath}); err != nil {
		return newError(CodeRestoreFailed, "binding %s: pg_restore into %s failed: %v: %s", b.ID, target, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Discard implements Provider. A partially restored database is not dropped
// automatically; the residue is named so the operator decides.
func (Postgres) Discard(_ context.Context, b Binding, target string) error {
	return fmt.Errorf("binding %s: database %s holds a partial restore and must be dropped by its owner", b.ID, target)
}

// Inventory implements Provider by re-dumping the target and counting its
// table of contents.
func (p Postgres) Inventory(ctx context.Context, b Binding, target string) (Inventory, error) {
	stage, err := os.MkdirTemp("", "recoverypoint-inventory-")
	if err != nil {
		return Inventory{}, newError(CodeVerifyFailed, "binding %s: create inventory stage: %v", b.ID, err)
	}
	defer os.RemoveAll(stage)
	probe := b
	probe.Locator = target
	result, err := p.Capture(ctx, probe, stage)
	if err != nil {
		return Inventory{}, err
	}
	return result.Inventory, nil
}
