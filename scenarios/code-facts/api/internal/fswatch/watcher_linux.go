//go:build linux

package fswatch

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

// Watcher is a low-overhead native directory watcher. It watches every
// existing directory below the governed roots and adds directories created
// later, so a source edit wakes code-facts without a repository-wide poll.
type Watcher struct {
	fd     int
	events chan struct{}
	done   chan struct{}
	once   sync.Once
	wg     sync.WaitGroup
}

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
	w := &Watcher{fd: fd, events: make(chan struct{}, 1), done: make(chan struct{})}
	for _, root := range roots {
		if err := addTree(fd, root); err != nil && !os.IsNotExist(err) {
			_ = unix.Close(fd)
			return nil, err
		}
	}
	w.wg.Add(1)
	go w.loop()
	return w, nil
}

func addTree(fd int, root string) error {
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			return nil
		}
		// A single unreadable or rapidly removed directory must not disable the
		// watcher for every other governed root. The five-minute manifest audit
		// remains the recovery path for a directory that could not be watched.
		_, _ = unix.InotifyAddWatch(fd, path, unix.IN_CREATE|unix.IN_DELETE|unix.IN_MODIFY|unix.IN_MOVED_FROM|unix.IN_MOVED_TO|unix.IN_ATTRIB|unix.IN_CLOSE_WRITE)
		return nil
	})
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
		if n > 0 {
			w.signal()
		}
	}
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
