// Package legacymigrate contains one-time storage migrations for web-console
// paths that predate the runtime-home layout. Keeping the file and candidate
// mechanics here prevents startup construction from becoming a second owner
// of migration policy.
package legacymigrate

import (
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
)

// DefaultStateFiles is the complete set of web-console State-class artifacts
// that may have been left in the pre-runtime-home location.
var DefaultStateFiles = []string{
	"hook-token.txt",
	"tts-config.json",
	"tts-hook-config.json",
	"tts-summarize-config.json",
	"voice-config.json",
	"speaker-verification-config.json",
	"wakeword-template.json",
}

// MigrateDatabase copies a legacy database and any SQLite sidecars only when
// the canonical destination is absent. It is idempotent and never overwrites
// an existing canonical database.
func MigrateDatabase(dbPath string) {
	if _, err := os.Stat(dbPath); err == nil {
		return
	} else if !errors.Is(err, os.ErrNotExist) {
		return
	}

	for _, legacy := range DatabaseCandidates() {
		if legacy == dbPath {
			continue
		}
		if _, err := os.Stat(legacy); err != nil {
			continue
		}
		if err := CopyFileIfExists(legacy, dbPath); err != nil {
			log.Printf("legacy-db migration: copy %s: %v", legacy, err)
			continue
		}
		for _, suffix := range []string{"-wal", "-shm"} {
			if err := CopyFileIfExists(legacy+suffix, dbPath+suffix); err != nil {
				log.Printf("legacy-db migration: copy %s: %v", legacy+suffix, err)
			}
		}
		log.Printf("legacy-db migration: relocated %s -> %s", legacy, dbPath)
		return
	}
}

// DatabaseCandidates lists pre-runtime-home database locations, most-specific
// first.
func DatabaseCandidates() []string {
	var out []string
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		out = append(out, filepath.Join(xdg, "vrooli", "web-console", "web-console.db"))
	}
	if home, err := os.UserHomeDir(); err == nil {
		out = append(out, filepath.Join(home, ".local", "share", "vrooli", "web-console", "web-console.db"))
	}
	return out
}

// MigrateStateFiles migrates each named state file using resolveCanonical to
// preserve the host application's storage resolver without importing it here.
func MigrateStateFiles(resolveCanonical func(name string) string, names []string) {
	for _, name := range names {
		MigrateStateFile(resolveCanonical(name), name)
	}
}

// MigrateStateFile copies one state artifact when its canonical destination is
// absent. The source mode is preserved; sensitive files such as the hook token
// therefore remain owner-readable only when that was their source mode.
func MigrateStateFile(canonicalPath, name string) {
	if _, err := os.Stat(canonicalPath); err == nil {
		return
	} else if !errors.Is(err, os.ErrNotExist) {
		return
	}
	for _, legacy := range StateCandidates(name) {
		if legacy == canonicalPath {
			continue
		}
		info, err := os.Stat(legacy)
		if err != nil {
			continue
		}
		if err := CopyFileWithMode(legacy, canonicalPath, info.Mode().Perm()); err != nil {
			log.Printf("legacy-state migration: copy %s: %v", legacy, err)
			continue
		}
		log.Printf("legacy-state migration: relocated %s -> %s", legacy, canonicalPath)
		return
	}
}

// StateCandidates lists pre-runtime-home State-class locations, most-specific
// first. These live below XDG_STATE_HOME, not XDG_DATA_HOME.
func StateCandidates(name string) []string {
	var out []string
	if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" {
		out = append(out, filepath.Join(xdg, "vrooli", "web-console", name))
	}
	if home, err := os.UserHomeDir(); err == nil {
		out = append(out, filepath.Join(home, ".local", "state", "vrooli", "web-console", name))
	}
	return out
}

// CopyFileWithMode copies src to dst with an explicit mode and creates the
// destination's parent directory.
func CopyFileWithMode(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close() //nolint:errcheck
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

// CopyFileIfExists copies src to dst, creating the destination's parent
// directory. A missing source is a no-op so SQLite sidecars can be attempted
// without a separate existence race.
func CopyFileIfExists(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	defer in.Close() //nolint:errcheck
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}
