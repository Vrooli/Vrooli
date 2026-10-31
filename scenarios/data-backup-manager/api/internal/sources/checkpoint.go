package sources

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"unicode/utf8"
)

const checkpointVersion = "dbm-workspace-v1"

// CheckpointManifest describes the supported fidelity profile, not an assertion
// that concurrent writers were paused. Stable scans detect observed drift;
// they cannot establish an atomic point-in-time view of a live filesystem.
type CheckpointManifest struct {
	Version     string            `json:"version"`
	Consistency string            `json:"consistency"`
	Profile     string            `json:"profile"`
	Entries     []CheckpointEntry `json:"entries"`
}
type CheckpointEntry struct {
	Path     string `json:"path"`
	Kind     string `json:"kind"`
	Mode     uint32 `json:"mode"`
	Size     int64  `json:"size,omitempty"`
	Modified int64  `json:"modified_ns,omitempty"`
	Digest   string `json:"sha256,omitempty"`
	Link     string `json:"link,omitempty"`
}

type checkpointCapturer struct{}

func (*checkpointCapturer) Kind() SourceKind { return KindWorkspace }

// CheckpointInspector is the capture-time evidence seam. Tests can inject
// movement without touching any production source or backup repository.
type CheckpointInspector func(context.Context, string) (CheckpointManifest, error)

func (*checkpointCapturer) Capture(ctx context.Context, spec CaptureSpec) (Artifact, error) {
	return captureCheckpoint(ctx, spec, InspectCheckpoint)
}

func captureCheckpoint(ctx context.Context, spec CaptureSpec, inspect CheckpointInspector) (Artifact, error) {
	if !filepath.IsAbs(spec.Locator) || !filepath.IsAbs(spec.StageDir) {
		return Artifact{}, errors.New("workspace and staging roots must be absolute")
	}
	source, err := filepath.EvalSymlinks(spec.Locator)
	if err != nil {
		return Artifact{}, err
	}
	stage, err := filepath.EvalSymlinks(spec.StageDir)
	if err != nil {
		return Artifact{}, err
	}
	if within(source, stage) || within(stage, source) {
		return Artifact{}, errors.New("workspace and staging roots must be disjoint")
	}
	before, err := inspect(ctx, source)
	if err != nil {
		return Artifact{}, err
	}
	root := filepath.Join(stage, "workspace")
	if err = os.Mkdir(root, 0700); err != nil {
		return Artifact{}, err
	}
	tree := filepath.Join(root, "tree")
	if _, err = copyTreeContext(ctx, source, tree); err != nil {
		return Artifact{}, err
	}
	after, err := inspect(ctx, source)
	if err != nil {
		return Artifact{}, err
	}
	copied, err := inspect(ctx, tree)
	if err != nil {
		return Artifact{}, err
	}
	if !reflect.DeepEqual(before, after) || !reflect.DeepEqual(before, copied) {
		return Artifact{}, errors.New("workspace changed or metadata could not be preserved; checkpoint rejected")
	}
	data, err := json.Marshal(before)
	if err != nil {
		return Artifact{}, err
	}
	f, err := os.OpenFile(filepath.Join(root, "manifest.json"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return Artifact{}, err
	}
	_, err = f.Write(data)
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return Artifact{}, err
	}
	if closeErr != nil {
		return Artifact{}, closeErr
	}
	var size int64
	for _, e := range before.Entries {
		size += e.Size
	}
	return Artifact{Path: root, Bytes: size + int64(len(data))}, nil
}

func within(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	return err == nil && (rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)))
}

// InspectCheckpoint hashes regular files and inventories directories and link
// targets without following links. Unsupported metadata is refused, not lost.
func InspectCheckpoint(ctx context.Context, path string) (CheckpointManifest, error) {
	m := CheckpointManifest{Version: checkpointVersion, Consistency: "observed-stable-not-atomic", Profile: "posix-basic-v1"}
	root, err := os.OpenRoot(path)
	if err != nil {
		return m, err
	}
	defer root.Close()
	err = fs.WalkDir(root.FS(), ".", func(p string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if e = ctx.Err(); e != nil {
			return e
		}
		if len(m.Entries) >= 1000000 {
			return errors.New("checkpoint exceeds one million entries")
		}
		if !utf8.ValidString(p) {
			return errors.New("checkpoint paths must be valid UTF-8")
		}
		info, e := root.Lstat(p)
		if e != nil {
			return e
		}
		if e = checkpointMetadata(filepath.Join(path, p), info); e != nil {
			return e
		}
		// Linked worktrees/submodules and alternate object stores need an explicit
		// multi-root contract. Refuse them instead of creating incomplete Git backups.
		if filepath.Base(p) == ".git" && !info.IsDir() {
			return errors.New("external Git directory requires a multi-root checkpoint")
		}
		if strings.HasSuffix(filepath.ToSlash(p), "objects/info/alternates") {
			return errors.New("Git alternate object storage requires a multi-root checkpoint")
		}
		ent := CheckpointEntry{Path: p, Mode: uint32(info.Mode().Perm()), Modified: info.ModTime().UnixNano()}
		switch {
		case info.IsDir():
			ent.Kind = "directory"
		case info.Mode()&os.ModeSymlink != 0:
			ent.Kind = "symlink"
			ent.Mode = 0
			ent.Modified = 0
			ent.Link, e = root.Readlink(p)
			if e != nil {
				return e
			}
			if !utf8.ValidString(ent.Link) {
				return errors.New("link target is not UTF-8")
			}
		case info.Mode().IsRegular():
			ent.Kind = "file"
			ent.Size = info.Size()
			f, e := root.Open(p)
			if e != nil {
				return e
			}
			opened, e := f.Stat()
			if e != nil {
				f.Close()
				return e
			}
			if !os.SameFile(info, opened) {
				f.Close()
				return errors.New("file changed during inventory")
			}
			h := sha256.New()
			_, e = io.Copy(h, &contextReader{ctx: ctx, r: f})
			end, se := f.Stat()
			ce := f.Close()
			if e != nil {
				return e
			}
			if se != nil {
				return se
			}
			if ce != nil {
				return ce
			}
			if end.Size() != info.Size() || !end.ModTime().Equal(info.ModTime()) {
				return errors.New("file changed during inventory")
			}
			ent.Digest = fmt.Sprintf("%x", h.Sum(nil))
		default:
			return fmt.Errorf("unsupported checkpoint entry: %s", p)
		}
		m.Entries = append(m.Entries, ent)
		return nil
	})
	return m, err
}

type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func (r *contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.r.Read(p)
}

// VerifyCheckpoint compares a restored envelope against its capture manifest.
// Legacy filesystem snapshots cannot be promoted into this stronger guarantee.
func VerifyCheckpoint(ctx context.Context, path string) (CheckpointManifest, error) {
	var want CheckpointManifest
	root, err := os.OpenRoot(path)
	if err != nil {
		return want, err
	}
	defer root.Close()
	entries, err := fs.ReadDir(root.FS(), ".")
	if err != nil {
		return want, err
	}
	if len(entries) != 2 {
		return want, errors.New("checkpoint envelope must contain only manifest.json and tree")
	}
	for _, name := range []string{"manifest.json", "tree"} {
		info, e := root.Lstat(name)
		if e != nil {
			return want, e
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return want, errors.New("checkpoint envelope contains a symlink")
		}
	}
	f, err := root.Open("manifest.json")
	if err != nil {
		return want, err
	}
	data, err := io.ReadAll(io.LimitReader(f, 64*1024*1024+1))
	f.Close()
	if err != nil {
		return want, err
	}
	if len(data) > 64*1024*1024 {
		return want, errors.New("checkpoint manifest exceeds size budget")
	}
	if err = json.Unmarshal(data, &want); err != nil {
		return want, err
	}
	if want.Version != checkpointVersion || want.Profile != "posix-basic-v1" || want.Consistency != "observed-stable-not-atomic" {
		return want, errors.New("unsupported checkpoint contract")
	}
	got, err := InspectCheckpoint(ctx, filepath.Join(path, "tree"))
	if err != nil {
		return want, err
	}
	if !reflect.DeepEqual(want, got) {
		return want, errors.New("checkpoint manifest mismatch: bytes, paths, types, or metadata changed")
	}
	return want, nil
}

func (*checkpointCapturer) Restore(ctx context.Context, spec RestoreSpec) error {
	want, err := VerifyCheckpoint(ctx, spec.ArtifactPath)
	if err != nil {
		return err
	}
	if err = requireEmptyDestination(spec.Target); err != nil {
		return err
	}
	source, err := filepath.EvalSymlinks(spec.ArtifactPath)
	if err != nil {
		return err
	}
	if within(source, spec.Target) || within(spec.Target, source) {
		return errors.New("restore and checkpoint roots must be disjoint")
	}
	if _, err = copyTreeContext(ctx, filepath.Join(spec.ArtifactPath, "tree"), spec.Target); err != nil {
		return err
	}
	got, err := InspectCheckpoint(ctx, spec.Target)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(want, got) {
		return errors.New("restored workspace does not match capture manifest")
	}
	return nil
}
