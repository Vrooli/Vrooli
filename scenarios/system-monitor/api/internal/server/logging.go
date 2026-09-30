package server

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/api-core/storage"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/config"
)

type rotatingFileWriter struct {
	path        string
	maxBytes    int64
	maxBackups  int
	maxAgeDays  int
	file        *os.File
	currentSize int64
	mu          sync.Mutex
}

func newRotatingFileWriter(path string, maxSizeMB, maxBackups, maxAgeDays int) (*rotatingFileWriter, error) {
	if maxSizeMB <= 0 {
		maxSizeMB = 100
	}
	if maxBackups < 0 {
		maxBackups = 0
	}
	if maxAgeDays < 0 {
		maxAgeDays = 0
	}

	rw := &rotatingFileWriter{
		path:       path,
		maxBytes:   int64(maxSizeMB) * 1024 * 1024,
		maxBackups: maxBackups,
		maxAgeDays: maxAgeDays,
	}

	if err := rw.openFile(); err != nil {
		return nil, err
	}

	return rw, nil
}

func (rw *rotatingFileWriter) Write(p []byte) (int, error) {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	if rw.file == nil {
		if err := rw.openFile(); err != nil {
			return 0, err
		}
	}

	if rw.currentSize+int64(len(p)) > rw.maxBytes {
		if err := rw.rotate(); err != nil {
			return 0, err
		}
	}

	n, err := rw.file.Write(p)
	if n > 0 {
		rw.currentSize += int64(n)
	}
	return n, err
}

func (rw *rotatingFileWriter) openFile() error {
	file, err := storage.OpenAppendFile(rw.path, 0o664)
	if err != nil {
		return err
	}

	info, err := file.Stat()
	if err != nil {
		file.Close()
		return err
	}

	rw.file = file
	rw.currentSize = info.Size()
	return nil
}

func (rw *rotatingFileWriter) rotate() error {
	if rw.file != nil {
		rw.file.Close()
	}

	timestamp := time.Now().Format("20060102-150405")
	rotatedPath := fmt.Sprintf("%s.%s", rw.path, timestamp)

	if err := storage.RenameFile(rw.path, rotatedPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	rw.cleanup()
	return rw.openFile()
}

func (rw *rotatingFileWriter) cleanup() {
	matches, err := filepath.Glob(rw.path + ".*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "log rotation glob error: %v\n", err)
		return
	}

	base := filepath.Base(rw.path)
	cutoff := time.Duration(rw.maxAgeDays) * 24 * time.Hour
	var backups []string

	for _, match := range matches {
		if !strings.HasPrefix(filepath.Base(match), base+".") {
			continue
		}

		info, statErr := os.Stat(match)
		if statErr != nil {
			fmt.Fprintf(os.Stderr, "log rotation stat error: %v\n", statErr)
			continue
		}

		if rw.maxAgeDays > 0 && time.Since(info.ModTime()) > cutoff {
			if removeErr := storage.RemoveFile(match); removeErr != nil {
				fmt.Fprintf(os.Stderr, "log rotation remove error: %v\n", removeErr)
			}
			continue
		}

		backups = append(backups, match)
	}

	sort.Slice(backups, func(i, j int) bool {
		iInfo, _ := os.Stat(backups[i])
		jInfo, _ := os.Stat(backups[j])
		if iInfo == nil || jInfo == nil {
			return backups[i] > backups[j]
		}
		return iInfo.ModTime().After(jInfo.ModTime())
	})

	if rw.maxBackups == 0 {
		for _, path := range backups {
			if err := storage.RemoveFile(path); err != nil {
				fmt.Fprintf(os.Stderr, "log rotation remove error: %v\n", err)
			}
		}
		return
	}

	for i := rw.maxBackups; i < len(backups); i++ {
		if err := storage.RemoveFile(backups[i]); err != nil {
			fmt.Fprintf(os.Stderr, "log rotation remove error: %v\n", err)
		}
	}
}

func setupLogging(cfg *config.Config) {
	if cfg.Logging.FilePath == "" || cfg.Logging.FilePath == "/var/log/system-monitor.log" {
		if resolved, err := resolveScenarioLogPath(cfg.Server.ServiceName); err == nil {
			cfg.Logging.FilePath = resolved
		} else {
			log.Printf("Failed to resolve scenario log path, using stdout: %v", err)
			cfg.Logging.Output = "stdout"
		}
	}

	if cfg.Logging.Output == "stdout" && os.Getenv("VROOLI_LIFECYCLE_MANAGED") == "true" {
		cfg.Logging.Output = "file"
	}

	if cfg.Logging.Output == "file" {
		writer, err := newRotatingFileWriter(cfg.Logging.FilePath, cfg.Logging.MaxSize, cfg.Logging.MaxBackups, cfg.Logging.MaxAge)
		if err != nil {
			log.Printf("Failed to configure log rotation, using stdout: %v", err)
		} else {
			log.SetOutput(writer)
		}
	}

	if cfg.Logging.Format == "json" {
		log.SetFlags(log.Ldate | log.Ltime | log.LUTC | log.Lshortfile)
	} else {
		log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	}
}

func resolveScenarioLogPath(serviceName string) (string, error) {
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return "", fmt.Errorf("create storage resolver: %w", err)
	}
	return resolver.Path(
		storage.Options{ScenarioID: "system-monitor"},
		storage.ClassLogs,
		filepath.Join(serviceName, "api.log"),
	)
}
