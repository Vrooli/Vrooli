package catalog

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const maxClassificationPrefix = 4096

var ErrNotRegularFile = errors.New("catalog discovery: not a regular file")

type FileSnapshot struct {
	Size    int64
	ModTime time.Time
	Hash    string
	Prefix  []byte
}

type FileInspector interface {
	Inspect(context.Context, string) (FileSnapshot, error)
}

type CommandProcess interface {
	Stdout() io.Reader
	Wait() error
	Close() error
}

type CommandStarter interface {
	Start(context.Context, string, string, ...string) (CommandProcess, error)
}

type GitDiscoverer struct {
	RepoRoot  string
	Roots     []string
	Starter   CommandStarter
	Inspector FileInspector
}

func (d GitDiscoverer) Open(ctx context.Context) (FileIterator, error) {
	root, roots, err := normalizeDiscoveryRoots(d.RepoRoot, d.Roots)
	if err != nil {
		return nil, err
	}
	if d.Starter == nil || d.Inspector == nil {
		return nil, fmt.Errorf("git discoverer requires command starter and file inspector")
	}
	args := []string{"ls-files", "-co", "--exclude-standard", "-z", "--"}
	args = append(args, roots...)
	process, err := d.Starter.Start(ctx, root, "git", args...)
	if err != nil {
		return nil, fmt.Errorf("start git repository discovery: %w", err)
	}
	return &gitIterator{
		repoRoot: root,
		reader:   bufio.NewReaderSize(process.Stdout(), 64*1024),
		process:  process,
		inspect:  d.Inspector,
	}, nil
}

type gitIterator struct {
	repoRoot string
	reader   *bufio.Reader
	process  CommandProcess
	inspect  FileInspector

	done    bool
	waitErr error
}

func (i *gitIterator) Next(ctx context.Context) (SourceFile, bool, error) {
	if i == nil || i.done {
		return SourceFile{}, false, i.waitErr
	}
	for {
		if err := ctx.Err(); err != nil {
			_ = i.Close()
			return SourceFile{}, false, err
		}
		raw, err := i.reader.ReadString(0)
		if errors.Is(err, io.EOF) && raw == "" {
			i.done = true
			i.waitErr = i.process.Wait()
			if i.waitErr != nil {
				return SourceFile{}, false, fmt.Errorf("git repository discovery: %w", i.waitErr)
			}
			return SourceFile{}, false, nil
		}
		if err != nil {
			_ = i.Close()
			return SourceFile{}, false, fmt.Errorf("read git repository discovery: %w", err)
		}
		path := canonicalPath(strings.TrimSuffix(raw, "\x00"))
		if path == "" || path == "." || strings.HasPrefix(path, "../") || filepath.IsAbs(path) {
			_ = i.Close()
			return SourceFile{}, false, fmt.Errorf("git returned unsafe repository path %q", path)
		}
		snapshot, err := i.inspect.Inspect(ctx, filepath.Join(i.repoRoot, filepath.FromSlash(path)))
		if errors.Is(err, os.ErrNotExist) || errors.Is(err, ErrNotRegularFile) {
			continue // tracked deletion in a dirty worktree
		}
		if err != nil {
			return SourceFile{}, false, fmt.Errorf("inspect repository file %q: %w", path, err)
		}
		classification := Classify(path, snapshot.Prefix)
		if classification.Role == RoleTransient {
			continue
		}
		return SourceFile{
			ID:         StableFileID(path),
			Path:       path,
			Language:   classification.Language.Name,
			Role:       classification.Role,
			Scope:      classification.Scope,
			Authority:  classification.Authority,
			Owner:      classification.Owner,
			Hash:       snapshot.Hash,
			Size:       snapshot.Size,
			ModTime:    snapshot.ModTime,
			Searchable: classification.Searchable,
		}, true, nil
	}
}

func (i *gitIterator) Close() error {
	if i == nil || i.process == nil || i.done {
		return nil
	}
	i.done = true
	return i.process.Close()
}

func normalizeDiscoveryRoots(repoRoot string, roots []string) (string, []string, error) {
	repoRoot = filepath.Clean(strings.TrimSpace(repoRoot))
	if repoRoot == "" || repoRoot == "." {
		return "", nil, fmt.Errorf("repository root is required")
	}
	if len(roots) == 0 {
		roots = []string{"scenarios", "packages", "cmd/vrooli", "internal", "resources"}
	}
	out := make([]string, 0, len(roots))
	seen := map[string]struct{}{}
	for _, root := range roots {
		root = canonicalPath(root)
		if root == "" || root == "." || strings.HasPrefix(root, "../") || filepath.IsAbs(root) {
			return "", nil, fmt.Errorf("invalid repository discovery root %q", root)
		}
		if _, exists := seen[root]; exists {
			continue
		}
		seen[root] = struct{}{}
		out = append(out, root)
	}
	sort.Strings(out)
	return repoRoot, out, nil
}

type OSFileInspector struct{}

func (OSFileInspector) Inspect(ctx context.Context, path string) (FileSnapshot, error) {
	file, err := os.Open(path)
	if err != nil {
		return FileSnapshot{}, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return FileSnapshot{}, err
	}
	if !info.Mode().IsRegular() {
		return FileSnapshot{}, ErrNotRegularFile
	}
	hash := sha256.New()
	prefix := &boundedPrefix{remaining: maxClassificationPrefix}
	if _, err := io.Copy(io.MultiWriter(hash, prefix), &contextReader{ctx: ctx, reader: file}); err != nil {
		return FileSnapshot{}, err
	}
	return FileSnapshot{
		Size:    info.Size(),
		ModTime: info.ModTime().UTC(),
		Hash:    "sha256:" + hex.EncodeToString(hash.Sum(nil)),
		Prefix:  prefix.bytes,
	}, nil
}

type boundedPrefix struct {
	bytes     []byte
	remaining int
}

func (w *boundedPrefix) Write(payload []byte) (int, error) {
	original := len(payload)
	if w.remaining > 0 {
		if len(payload) > w.remaining {
			payload = payload[:w.remaining]
		}
		w.bytes = append(w.bytes, payload...)
		w.remaining -= len(payload)
	}
	return original, nil
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *contextReader) Read(buffer []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(buffer)
}

type ExecCommandStarter struct{}

func (ExecCommandStarter) Start(ctx context.Context, dir, name string, args ...string) (CommandProcess, error) {
	command := exec.CommandContext(ctx, name, args...)
	command.Dir = dir
	stdout, err := command.StdoutPipe()
	if err != nil {
		return nil, err
	}
	command.Stderr = io.Discard
	if err := command.Start(); err != nil {
		return nil, err
	}
	return &execProcess{command: command, stdout: stdout}, nil
}

type execProcess struct {
	command *exec.Cmd
	stdout  io.ReadCloser
	once    sync.Once
	waitErr error
}

func (p *execProcess) Stdout() io.Reader { return p.stdout }

func (p *execProcess) Wait() error {
	p.once.Do(func() { p.waitErr = p.command.Wait() })
	return p.waitErr
}

func (p *execProcess) Close() error {
	_ = p.stdout.Close()
	if p.command.Process != nil {
		_ = p.command.Process.Kill()
	}
	return p.Wait()
}

// FilesystemDiscoverer is the fallback for external target roots that are not
// part of a Git worktree. It uses a one-item channel, so the walk cannot outrun
// the consumer and file metadata remains page-bounded.
type FilesystemDiscoverer struct {
	Root      string
	Inspector FileInspector
}

func (d FilesystemDiscoverer) Open(ctx context.Context) (FileIterator, error) {
	root := filepath.Clean(strings.TrimSpace(d.Root))
	if root == "" || root == "." || d.Inspector == nil {
		return nil, fmt.Errorf("filesystem discoverer requires root and inspector")
	}
	walkCtx, cancel := context.WithCancel(ctx)
	iterator := &walkIterator{cancel: cancel, files: make(chan walkFile, 1)}
	go iterator.walk(walkCtx, root, d.Inspector)
	return iterator, nil
}

type walkFile struct {
	file SourceFile
	err  error
}

type walkIterator struct {
	cancel context.CancelFunc
	files  chan walkFile
}

func (i *walkIterator) walk(ctx context.Context, root string, inspector FileInspector) {
	defer close(i.files)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			rel, relErr := filepath.Rel(root, path)
			if relErr != nil {
				return relErr
			}
			if rel != "." && classifyRole(canonicalPath(rel), nil, "unknown") == RoleTransient {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		snapshot, err := inspector.Inspect(ctx, path)
		if err != nil {
			return err
		}
		rel = canonicalPath(rel)
		classification := Classify(rel, snapshot.Prefix)
		file := SourceFile{
			ID: StableFileID(rel), Path: rel, Language: classification.Language.Name,
			Role: classification.Role, Scope: classification.Scope, Authority: classification.Authority,
			Owner: classification.Owner, Hash: snapshot.Hash, Size: snapshot.Size,
			ModTime: snapshot.ModTime, Searchable: classification.Searchable,
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case i.files <- walkFile{file: file}:
			return nil
		}
	})
	if err != nil && !errors.Is(err, context.Canceled) {
		i.files <- walkFile{err: err}
	}
}

func (i *walkIterator) Next(ctx context.Context) (SourceFile, bool, error) {
	select {
	case <-ctx.Done():
		return SourceFile{}, false, ctx.Err()
	case item, ok := <-i.files:
		if !ok {
			return SourceFile{}, false, nil
		}
		return item.file, item.err == nil, item.err
	}
}

func (i *walkIterator) Close() error {
	if i != nil && i.cancel != nil {
		i.cancel()
	}
	return nil
}

var (
	_ Discoverer     = GitDiscoverer{}
	_ Discoverer     = FilesystemDiscoverer{}
	_ FileInspector  = OSFileInspector{}
	_ CommandStarter = ExecCommandStarter{}
)
