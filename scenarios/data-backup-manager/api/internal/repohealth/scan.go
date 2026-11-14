package repohealth

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// FS is the read-only filesystem surface the scan needs. Production uses the
// operating system; tests inject faults, because an I/O error cannot be
// produced in a temp directory.
type FS interface {
	// Names lists the names in a directory without stat-ing them, so one
	// unreadable entry cannot stop the listing of its siblings.
	Names(dir string) ([]string, error)
	Lstat(p string) (fs.FileInfo, error)
}

// OSFS is the production FS.
type OSFS struct{}

func (OSFS) Names(dir string) ([]string, error) {
	f, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return f.Readdirnames(-1)
}

func (OSFS) Lstat(p string) (fs.FileInfo, error) { return os.Lstat(p) }

// Scan is the result of one read-only walk of a repository directory.
type Scan struct {
	Root       string        `json:"root"`
	StartedAt  time.Time     `json:"started_at"`
	Duration   time.Duration `json:"duration"`
	Files      int64         `json:"files"`
	Bytes      int64         `json:"bytes"`
	TempFiles  int64         `json:"readable_temp_files"`
	Unreadable []Entry       `json:"unreadable"`
}

// Scanner walks a repository directory. It never modifies anything.
type Scanner struct {
	FS      FS
	Workers int
	Now     func() time.Time
}

func (s *Scanner) fsys() FS {
	if s != nil && s.FS != nil {
		return s.FS
	}
	return OSFS{}
}

// Filesystem is the FS the scanner reads, for callers that must re-check one
// path against the same view.
func (s *Scanner) Filesystem() FS { return s.fsys() }

func (s *Scanner) now() time.Time {
	if s != nil && s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

// Walk sums readable bytes and collects every entry that cannot be read. A
// directory that cannot be listed is itself reported as unreadable (class
// other), because everything beneath it is then unaccounted for.
func (s *Scanner) Walk(ctx context.Context, root string) (Scan, error) {
	root = filepath.Clean(root)
	fsys := s.fsys()
	started := s.now()
	if _, err := fsys.Lstat(root); err != nil {
		return Scan{}, fmt.Errorf("repository directory is not readable: %w", err)
	}
	workers := 8
	if s != nil && s.Workers > 0 {
		workers = s.Workers
	}

	var (
		mu         sync.Mutex
		result     = Scan{Root: root, StartedAt: started}
		wg         sync.WaitGroup
		queue      = make(chan string, 1<<16)
		pending    sync.WaitGroup
		firstError error
	)
	record := func(rel string, err error) {
		mu.Lock()
		result.Unreadable = append(result.Unreadable, NewEntry(rel, err))
		mu.Unlock()
	}
	var enqueue func(dir string)
	enqueue = func(dir string) {
		pending.Add(1)
		select {
		case queue <- dir:
		default:
			// The queue is bounded; fall back to a goroutine rather than block a
			// worker on its own backlog.
			go func() { queue <- dir }()
		}
	}
	process := func(dir string) {
		defer pending.Done()
		if ctx.Err() != nil {
			return
		}
		rel := relTo(root, dir)
		names, err := fsys.Names(dir)
		if err != nil {
			if rel == "." {
				mu.Lock()
				if firstError == nil {
					firstError = err
				}
				mu.Unlock()
				return
			}
			record(rel, err)
			return
		}
		for _, name := range names {
			full := filepath.Join(dir, name)
			info, err := fsys.Lstat(full)
			if err != nil {
				record(path.Join(rel, name), err)
				continue
			}
			if info.IsDir() {
				enqueue(full)
				continue
			}
			mu.Lock()
			result.Files++
			result.Bytes += info.Size()
			if c, _ := ClassifyName(path.Join(rel, name)); c == ClassLeftoverTemp {
				result.TempFiles++
			}
			mu.Unlock()
		}
	}
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for dir := range queue {
				process(dir)
			}
		}()
	}
	enqueue(root)
	pending.Wait()
	close(queue)
	wg.Wait()

	if err := ctx.Err(); err != nil {
		return Scan{}, err
	}
	if firstError != nil {
		return Scan{}, fmt.Errorf("repository directory cannot be listed: %w", firstError)
	}
	sort.Slice(result.Unreadable, func(i, j int) bool { return result.Unreadable[i].RelPath < result.Unreadable[j].RelPath })
	result.Duration = s.now().Sub(started)
	return result, nil
}

func relTo(root, dir string) string {
	rel, err := filepath.Rel(root, dir)
	if err != nil {
		return filepath.ToSlash(dir)
	}
	return filepath.ToSlash(rel)
}
