//go:build !linux

package fswatch

import (
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

// Non-Linux targets use a bounded metadata backstop. This keeps the package
// portable where the platform-native event APIs differ; the production Linux
// path uses inotify and the five-minute code-facts audit remains the final
// missed-event backstop on every platform.
type Watcher struct {
	roots    []string
	interval time.Duration
	events   chan struct{}
	done     chan struct{}
	once     sync.Once
	wg       sync.WaitGroup
}

func New(roots []string) (*Watcher, error) {
	return newWithInterval(roots, 5*time.Minute)
}

// newWithInterval exists for deterministic package tests. Production callers
// use New, whose metadata backstop is deliberately no faster than the
// five-minute repository audit.
func newWithInterval(roots []string, interval time.Duration) (*Watcher, error) {
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	w := &Watcher{roots: append([]string(nil), roots...), interval: interval, events: make(chan struct{}, 1), done: make(chan struct{})}
	w.wg.Add(1)
	go w.loop()
	return w, nil
}

func (w *Watcher) Events() <-chan struct{} { return w.events }

func (w *Watcher) loop() {
	defer w.wg.Done()
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	var previous string
	for {
		select {
		case <-w.done:
			return
		case <-ticker.C:
			current := metadataFingerprint(w.roots)
			if previous != "" && current != previous {
				select {
				case w.events <- struct{}{}:
				default:
				}
			}
			previous = current
		}
	}
}

func metadataFingerprint(roots []string) string {
	var latest time.Time
	var count int
	for _, root := range roots {
		_ = filepath.Walk(root, func(_ string, info os.FileInfo, err error) error {
			if err == nil && info != nil {
				count++
				if info.ModTime().After(latest) {
					latest = info.ModTime()
				}
			}
			return nil
		})
	}
	return latest.UTC().Format(time.RFC3339Nano) + ":" + strconv.Itoa(count)
}

func (w *Watcher) Close() error {
	if w == nil {
		return nil
	}
	w.once.Do(func() {
		close(w.done)
		w.wg.Wait()
		close(w.events)
	})
	return nil
}
