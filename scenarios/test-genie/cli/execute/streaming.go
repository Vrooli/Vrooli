package execute

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"test-genie/cli/internal/repo"
)

// LogTailer streams the newest log file in a directory.
type LogTailer struct {
	dir     string
	w       io.Writer
	offsets map[string]int64
	stop    chan struct{}
	wg      sync.WaitGroup
}

// StartLogTailer begins streaming log output from the specified directory.
func StartLogTailer(w io.Writer, dir string) (*LogTailer, error) {
	if strings.TrimSpace(dir) == "" {
		return nil, fmt.Errorf("log directory is empty")
	}
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create log directory: %w", err)
		}
	}
	t := &LogTailer{
		dir:     dir,
		w:       w,
		offsets: make(map[string]int64),
		stop:    make(chan struct{}),
	}
	t.wg.Add(1)
	go t.run()
	return t, nil
}

// Stop halts the log tailer.
func (t *LogTailer) Stop() {
	close(t.stop)
	t.wg.Wait()
}

func (t *LogTailer) run() {
	defer t.wg.Done()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-t.stop:
			return
		case <-ticker.C:
			t.tailNewest()
		}
	}
}

func (t *LogTailer) tailNewest() {
	path := t.findNewestLog()
	if path == "" {
		return
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return
	}
	prev := t.offsets[path]
	size := info.Size()
	if size == prev {
		return
	}
	if size < prev {
		prev = 0
	}
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	if _, err := f.Seek(prev, io.SeekStart); err != nil {
		return
	}
	buf := make([]byte, size-prev)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return
	}
	if n > 0 {
		rel := path
		if root := repo.Root(); root != "" {
			if relPath, err := filepath.Rel(root, path); err == nil {
				rel = relPath
			}
		}
		fmt.Fprintf(t.w, "\n[stream] %s\n%s", rel, string(buf[:n]))
	}
	t.offsets[path] = size
}

func (t *LogTailer) findNewestLog() string {
	entries, err := os.ReadDir(t.dir)
	if err != nil || len(entries) == 0 {
		return ""
	}
	type candidate struct {
		name    string
		modTime time.Time
	}
	var files []candidate
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".log") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, candidate{
			name:    filepath.Join(t.dir, entry.Name()),
			modTime: info.ModTime(),
		})
	}
	if len(files) == 0 {
		return ""
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.After(files[j].modTime)
	})
	return files[0].name
}
