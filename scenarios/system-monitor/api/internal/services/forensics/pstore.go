package forensics

import (
	"errors"
	"io/fs"
	"os"
	"strings"
	"time"
)

// PstoreEntry classifies a single artifact under /sys/fs/pstore.
type PstoreEntry struct {
	Name     string    `json:"name"`
	Kind     string    `json:"kind"` // dmesg, console, pmsg, ftrace, unknown
	Size     int64     `json:"size"`
	Modified time.Time `json:"modified"`
}

// PstoreReport is the data payload for /api/v1/forensics/pstore.
type PstoreReport struct {
	Path    string        `json:"path"`
	Entries []PstoreEntry `json:"entries"`
}

// Pstore returns the current pstore artifact list. When /sys/fs/pstore is
// unavailable (kernel feature missing, EACCES, non-Linux host) the envelope
// reports available=false with a reason.
func (s *Service) Pstore() Envelope {
	const key = "pstore"
	now := s.now()
	if cached, ok := s.cache.get(key, now); ok {
		return cached
	}

	env := s.computePstore(now)
	s.cache.set(key, env, now)
	return env
}

func (s *Service) computePstore(now time.Time) Envelope {
	env := Envelope{GeneratedAt: now}

	if s.fs.ReadDirFn == nil {
		env.Reason = "filesystem reader not configured"
		return env
	}

	entries, err := s.fs.ReadDirFn(s.PstoreDir)
	if err != nil {
		switch {
		case errors.Is(err, fs.ErrNotExist):
			env.Reason = "pstore directory not present (kernel pstore not configured)"
		case errors.Is(err, fs.ErrPermission), errors.Is(err, os.ErrPermission):
			env.Reason = "pstore directory not readable (permission denied)"
		default:
			env.Reason = "pstore read failed: " + err.Error()
		}
		return env
	}

	report := PstoreReport{Path: s.PstoreDir, Entries: []PstoreEntry{}}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		entry := PstoreEntry{Name: name, Kind: classifyPstore(name)}
		if s.fs.StatFn != nil {
			full := s.PstoreDir + "/" + name
			if info, statErr := s.fs.StatFn(full); statErr == nil {
				entry.Size = info.Size()
				entry.Modified = info.ModTime().UTC()
			}
		}
		report.Entries = append(report.Entries, entry)
	}

	env.Available = true
	env.Data = report
	return env
}

func classifyPstore(name string) string {
	switch {
	case strings.HasPrefix(name, "dmesg-"):
		return "dmesg"
	case strings.HasPrefix(name, "console-"):
		return "console"
	case strings.HasPrefix(name, "pmsg-"):
		return "pmsg"
	case strings.HasPrefix(name, "ftrace-"):
		return "ftrace"
	default:
		return "unknown"
	}
}
