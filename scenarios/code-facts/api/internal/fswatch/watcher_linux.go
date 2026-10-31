//go:build linux

package fswatch

import (
	"bytes"
	"os"
	"path/filepath"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

// Watcher is a low-overhead native directory watcher. It watches every
// existing directory below the governed roots and adds directories created
// later, so a source edit wakes code-facts without a repository-wide poll.
// Directories and files rejected by Ignored are never watched and never
// produce an event, so live databases, dependency stores and build output
// cannot keep the consumer's debounce permanently armed.
type Watcher struct {
	fd     int
	events chan struct{}
	done   chan struct{}
	once   sync.Once
	wg     sync.WaitGroup

	mu   sync.Mutex
	dirs map[int32]watchedDir
}

// watchedDir remembers the directory behind a watch descriptor and the
// governed root it was discovered under, so events can be judged by their
// root-relative path rather than by wherever the repository happens to live.
type watchedDir struct {
	path string
	base string
}

const watchMask = unix.IN_CREATE | unix.IN_DELETE | unix.IN_MODIFY | unix.IN_MOVED_FROM | unix.IN_MOVED_TO | unix.IN_ATTRIB | unix.IN_CLOSE_WRITE | unix.IN_DELETE_SELF | unix.IN_MOVE_SELF

func New(roots []string) (*Watcher, error) {
	return newWithInterval(roots, 0)
}

// newWithInterval keeps the test seam consistent with the portable metadata
// watcher. Linux uses inotify regardless of the interval.
func newWithInterval(roots []string, _ time.Duration) (*Watcher, error) {
	fd, err := unix.InotifyInit1(unix.IN_NONBLOCK | unix.IN_CLOEXEC)
	if err != nil {
		return nil, err
	}
	w := &Watcher{fd: fd, events: make(chan struct{}, 1), done: make(chan struct{}), dirs: map[int32]watchedDir{}}
	for _, root := range roots {
		root = filepath.Clean(root)
		if err := w.addTree(root, filepath.Dir(root)); err != nil && !os.IsNotExist(err) {
			_ = unix.Close(fd)
			return nil, err
		}
	}
	w.wg.Add(1)
	go w.loop()
	return w, nil
}

// addTree installs watches on root and every directory below it that is not
// ignored. base is the directory paths are made relative to before Ignored
// judges them: the parent of a governed root, so "<repo>/scenarios/x/data"
// is seen as "scenarios/x/data".
func (w *Watcher) addTree(root, base string) error {
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			return nil
		}
		if path != root && Ignored(relativeTo(base, path)) {
			return filepath.SkipDir
		}
		w.addDir(path, base)
		return nil
	})
}

func relativeTo(base, path string) string {
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return path
	}
	return rel
}

func (w *Watcher) addDir(path, base string) {
	// A single unreadable or rapidly removed directory must not disable the
	// watcher for every other governed root. The five-minute manifest audit
	// remains the recovery path for a directory that could not be watched.
	wd, err := unix.InotifyAddWatch(w.fd, path, watchMask)
	if err != nil {
		return
	}
	w.mu.Lock()
	w.dirs[int32(wd)] = watchedDir{path: path, base: base}
	w.mu.Unlock()
}

func (w *Watcher) dirFor(wd int32) (watchedDir, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	dir, ok := w.dirs[wd]
	return dir, ok
}

func (w *Watcher) forget(wd int32) {
	w.mu.Lock()
	delete(w.dirs, wd)
	w.mu.Unlock()
}

func (w *Watcher) Events() <-chan struct{} { return w.events }

func (w *Watcher) signal() {
	select {
	case w.events <- struct{}{}:
	default:
	}
}

func (w *Watcher) loop() {
	defer w.wg.Done()
	buffer := make([]byte, 64*1024)
	for {
		select {
		case <-w.done:
			return
		default:
		}
		n, err := unix.Read(w.fd, buffer)
		if err == unix.EAGAIN || err == unix.EINTR {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		if err != nil {
			return
		}
		if n > 0 && w.consume(buffer[:n]) {
			w.signal()
		}
	}
}

// consume decodes one inotify read and reports whether any event touched a
// path the corpus could contain. Ignored paths are dropped here, before the
// consumer's debounce sees them.
func (w *Watcher) consume(payload []byte) bool {
	relevant := false
	const header = int(unsafe.Sizeof(unix.InotifyEvent{}))
	for offset := 0; offset+header <= len(payload); {
		event := (*unix.InotifyEvent)(unsafe.Pointer(&payload[offset]))
		nameBytes := payload[offset+header : offset+header+int(event.Len)]
		offset += header + int(event.Len)
		name := string(bytes.TrimRight(nameBytes, "\x00"))
		if event.Mask&unix.IN_Q_OVERFLOW != 0 {
			relevant = true
			continue
		}
		dir, ok := w.dirFor(event.Wd)
		if event.Mask&unix.IN_IGNORED != 0 {
			w.forget(event.Wd)
			continue
		}
		if !ok {
			continue
		}
		if event.Mask&(unix.IN_DELETE_SELF|unix.IN_MOVE_SELF) != 0 {
			relevant = true
			continue
		}
		path := dir.path
		if name != "" {
			path = filepath.Join(dir.path, name)
		}
		if Ignored(relativeTo(dir.base, path)) {
			continue
		}
		relevant = true
		if event.Mask&unix.IN_ISDIR != 0 && event.Mask&(unix.IN_CREATE|unix.IN_MOVED_TO) != 0 {
			_ = w.addTree(path, dir.base)
		}
	}
	return relevant
}

func (w *Watcher) Close() error {
	if w == nil {
		return nil
	}
	w.once.Do(func() {
		close(w.done)
		_ = unix.Close(w.fd)
		w.wg.Wait()
		close(w.events)
	})
	return nil
}
