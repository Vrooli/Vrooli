package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Config holds runtime settings for the SQLite resource.
type Config struct {
	DataRoot            string
	DatabasePath        string
	BackupPath          string
	ReplicaPath         string
	JournalMode         string
	BusyTimeout         time.Duration
	CacheSize           int
	PageSize            int
	Synchronous         string
	TempStore           string
	MmapSize            int64
	FilePermissions     os.FileMode
	CLITimeout          time.Duration
	BackupRetentionDays int
}

// Load reads environment variables and applies defaults matching the legacy Bash resource.
func Load() Config {
	dataRoot := firstNonEmpty(os.Getenv("VROOLI_DATA"), filepath.Join(userHomeDir(), ".vrooli", "data"))

	return Config{
		DataRoot:            dataRoot,
		DatabasePath:        firstNonEmpty(os.Getenv("SQLITE_DATABASE_PATH"), filepath.Join(dataRoot, "sqlite", "databases")),
		BackupPath:          firstNonEmpty(os.Getenv("SQLITE_BACKUP_PATH"), filepath.Join(dataRoot, "sqlite", "backups")),
		ReplicaPath:         firstNonEmpty(os.Getenv("SQLITE_REPLICATION_PATH"), filepath.Join(dataRoot, "sqlite", "replicas")),
		JournalMode:         firstNonEmpty(os.Getenv("SQLITE_JOURNAL_MODE"), "WAL"),
		BusyTimeout:         time.Duration(envInt("SQLITE_BUSY_TIMEOUT", 10000)) * time.Millisecond,
		CacheSize:           envInt("SQLITE_CACHE_SIZE", 2000),
		PageSize:            envInt("SQLITE_PAGE_SIZE", 4096),
		Synchronous:         firstNonEmpty(os.Getenv("SQLITE_SYNCHRONOUS"), "NORMAL"),
		TempStore:           firstNonEmpty(os.Getenv("SQLITE_TEMP_STORE"), "MEMORY"),
		MmapSize:            int64(envInt("SQLITE_MMAP_SIZE", 268435456)),
		FilePermissions:     os.FileMode(envInt("SQLITE_FILE_PERMISSIONS", 0o600)),
		CLITimeout:          time.Duration(envInt("SQLITE_CLI_TIMEOUT", 30)) * time.Second,
		BackupRetentionDays: envInt("SQLITE_BACKUP_RETENTION_DAYS", 7),
	}
}

// EnsureDirectories creates required data directories.
func (c Config) EnsureDirectories() error {
	paths := []string{c.DatabasePath, c.BackupPath, c.ReplicaPath}
	for _, p := range paths {
		if err := os.MkdirAll(p, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			return parsed
		}
	}
	return def
}

func userHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return home
}
