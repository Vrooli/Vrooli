package repo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/vrooli/vrooli/resources/kopia/cli/internal/cmdutil"
	"github.com/vrooli/vrooli/resources/kopia/cli/internal/registry"
)

type DamageClass string

const (
	DamageLeftoverTemp   DamageClass = "leftover_temp"
	DamageRepositoryBlob DamageClass = "repository_blob"
	DamageOther          DamageClass = "other"
)

type DamageBlobKind string

const (
	DamageSessionMarker DamageBlobKind = "session_marker"
	DamageDataPack      DamageBlobKind = "data_pack"
	DamageMetadataPack  DamageBlobKind = "metadata_pack"
	DamageIndex         DamageBlobKind = "index"
	DamageBlobOther     DamageBlobKind = "other"
)

type DamageProblem string

const (
	DamageIO           DamageProblem = "io_error"
	DamagePermission   DamageProblem = "permission_denied"
	DamageProblemOther DamageProblem = "other"
)

type DamageEntry struct {
	RelativePath string         `json:"relative_path"`
	Class        DamageClass    `json:"class"`
	BlobKind     DamageBlobKind `json:"blob_kind"`
	Problem      DamageProblem  `json:"problem"`
	Orphan       bool           `json:"orphan_temp"`
	Detail       string         `json:"detail,omitempty"`
}

type DamageScan struct {
	Root       string        `json:"root"`
	StartedAt  time.Time     `json:"started_at"`
	Duration   time.Duration `json:"duration"`
	Files      int64         `json:"files"`
	Bytes      int64         `json:"bytes"`
	TempFiles  int64         `json:"readable_temp_files"`
	Unreadable []DamageEntry `json:"unreadable"`
}

type damageScanFS interface {
	Names(string) ([]string, error)
	Lstat(string) (fs.FileInfo, error)
}

type osDamageScanFS struct{}

func (osDamageScanFS) Names(dir string) ([]string, error) {
	f, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return f.Readdirnames(-1)
}

func (osDamageScanFS) Lstat(p string) (fs.FileInfo, error) { return os.Lstat(p) }

var (
	// Kopia writes temporary content objects as <blob>.f.tmp, with an optional
	// random hex suffix on implementations that use a unique temp name. The
	// suffixless form is a real on-disk shape (and can remain after interruption).
	damageTempName = regexp.MustCompile(`^[0-9a-z]+(?:-[0-9a-z]+)*\.f\.tmp(?:\.[0-9a-f]{8,64})?$`)
	damageBlobName = regexp.MustCompile(`^[0-9a-z]+(?:-[0-9a-z]+)*\.f$`)
)

func classifyDamagePath(rel string) (DamageClass, DamageBlobKind) {
	rel = path.Clean(strings.ReplaceAll(rel, `\`, "/"))
	base := path.Base(rel)
	kind := damageBlobKind(rel)
	switch {
	case damageTempName.MatchString(base):
		return DamageLeftoverTemp, kind
	case damageBlobName.MatchString(base):
		return DamageRepositoryBlob, kind
	default:
		return DamageOther, kind
	}
}

func damageBlobKind(rel string) DamageBlobKind {
	parts := strings.Split(rel, "/")
	if len(parts) < 2 {
		return DamageBlobOther
	}
	switch parts[0] {
	case "p":
		return DamageDataPack
	case "q":
		return DamageMetadataPack
	case "s":
		return DamageSessionMarker
	case "x", "n":
		return DamageIndex
	default:
		return DamageBlobOther
	}
}

func damageProblem(err error) DamageProblem {
	switch {
	case errors.Is(err, syscall.EIO) || strings.Contains(strings.ToLower(fmt.Sprint(err)), "input/output error"):
		return DamageIO
	case errors.Is(err, fs.ErrPermission):
		return DamagePermission
	default:
		return DamageProblemOther
	}
}

func scanFilesystemRepository(ctx context.Context, root string, filesystem damageScanFS, now func() time.Time) (DamageScan, error) {
	root = filepath.Clean(root)
	if filesystem == nil {
		filesystem = osDamageScanFS{}
	}
	if now == nil {
		now = time.Now
	}
	started := now().UTC()
	if _, err := filesystem.Lstat(root); err != nil {
		return DamageScan{}, fmt.Errorf("repository directory is not readable: %w", err)
	}
	const workerCount = 8
	var result DamageScan
	result.Root, result.StartedAt = root, started
	var mu sync.Mutex
	var pending sync.WaitGroup
	var workers sync.WaitGroup
	queue := make(chan string, 1<<16)
	var firstError error
	push := func(dir string) {
		pending.Add(1)
		select {
		case queue <- dir:
		default:
			go func() { queue <- dir }()
		}
	}
	record := func(rel string, err error, orphan bool) {
		class, kind := classifyDamagePath(rel)
		problem := DamageProblemOther
		detail := "entry is unreadable"
		if err != nil {
			problem, detail = damageProblem(err), damageProblemDetail(err)
		}
		mu.Lock()
		result.Unreadable = append(result.Unreadable, DamageEntry{RelativePath: path.Clean(filepath.ToSlash(rel)), Class: class, BlobKind: kind, Problem: problem, Orphan: orphan, Detail: detail})
		mu.Unlock()
	}
	process := func(dir string) {
		defer pending.Done()
		if ctx.Err() != nil {
			return
		}
		rel, err := filepath.Rel(root, dir)
		if err != nil {
			rel = filepath.ToSlash(dir)
		} else {
			rel = filepath.ToSlash(rel)
		}
		names, err := filesystem.Names(dir)
		if err != nil {
			if rel == "." {
				mu.Lock()
				if firstError == nil {
					firstError = err
				}
				mu.Unlock()
				return
			}
			record(rel, err, false)
			return
		}
		nameSet := make(map[string]bool, len(names))
		for _, name := range names {
			nameSet[name] = true
		}
		for _, name := range names {
			full := filepath.Join(dir, name)
			info, err := filesystem.Lstat(full)
			entryRel := path.Join(rel, name)
			if err != nil {
				class, _ := classifyDamagePath(entryRel)
				finalName := tempFinalName(name)
				record(entryRel, err, class == DamageLeftoverTemp && finalName != "" && !nameSet[finalName])
				continue
			}
			if info.IsDir() {
				push(full)
				continue
			}
			mu.Lock()
			result.Files++
			result.Bytes += info.Size()
			if class, _ := classifyDamagePath(entryRel); class == DamageLeftoverTemp {
				result.TempFiles++
			}
			mu.Unlock()
		}
	}
	for i := 0; i < workerCount; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for dir := range queue {
				process(dir)
			}
		}()
	}
	push(root)
	pending.Wait()
	close(queue)
	workers.Wait()
	if err := ctx.Err(); err != nil {
		return DamageScan{}, err
	}
	if firstError != nil {
		return DamageScan{}, fmt.Errorf("repository directory cannot be listed: %w", firstError)
	}
	sort.Slice(result.Unreadable, func(i, j int) bool { return result.Unreadable[i].RelativePath < result.Unreadable[j].RelativePath })
	result.Duration = now().UTC().Sub(started)
	return result, nil
}

func tempFinalName(name string) string {
	class, _ := classifyDamagePath(name)
	if class != DamageLeftoverTemp {
		return ""
	}
	return name[:strings.Index(name, ".tmp")]
}

func damageProblemDetail(err error) string {
	switch damageProblem(err) {
	case DamageIO:
		return "input/output error"
	case DamagePermission:
		return "permission denied"
	default:
		return "entry is unreadable"
	}
}

func (s Service) DamageScan(ctx context.Context, args []string) error {
	flags := cmdutil.NewFlagSet("repo damage-scan")
	name := flags.String("name", "", "Registered filesystem repository name")
	repoPath := flags.String("path", "", "Explicit absolute Kopia filesystem repository root for a read-only scan")
	jsonOut := flags.Bool("json", false, "Emit typed damage scan as JSON")
	if err := cmdutil.Parse(flags, args); err != nil {
		return err
	}
	if strings.TrimSpace(*name) == "" == (strings.TrimSpace(*repoPath) == "") {
		return fmt.Errorf("provide exactly one of --name or --path")
	}
	var root, label string
	if strings.TrimSpace(*repoPath) != "" {
		if !filepath.IsAbs(*repoPath) || filepath.Clean(*repoPath) != *repoPath {
			return fmt.Errorf("--path must be a clean absolute repository path")
		}
		marker, err := os.Lstat(filepath.Join(*repoPath, "kopia.repository.f"))
		if err != nil || !marker.Mode().IsRegular() {
			return fmt.Errorf("--path is not a Kopia filesystem repository root")
		}
		root, label = *repoPath, *repoPath
	} else {
		if err := cmdutil.RequireName(*name); err != nil {
			return err
		}
		entry, found, err := s.Registry.Get(*name)
		if err != nil {
			return err
		}
		if !found || entry.Backend != registry.BackendFilesystem || strings.TrimSpace(entry.Path) == "" {
			return fmt.Errorf("repository %q is not a registered filesystem repository", *name)
		}
		root, label = entry.Path, *name
	}
	result, err := scanFilesystemRepository(ctx, root, nil, nil)
	if err != nil {
		return err
	}
	if *jsonOut {
		return json.NewEncoder(s.out()).Encode(result)
	}
	fmt.Fprintf(s.out(), "Repository: %s\nFiles: %d\nBytes: %d\nReadable temp files: %d\nUnreadable entries: %d\n", label, result.Files, result.Bytes, result.TempFiles, len(result.Unreadable))
	for _, entry := range result.Unreadable {
		fmt.Fprintf(s.out(), "- %s: %s %s (%s)\n", entry.RelativePath, entry.Class, entry.Problem, entry.Detail)
	}
	return nil
}
