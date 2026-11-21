package agentharness

import (
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// treeBudget bounds how many entries one resolver walks for one query, so a
// huge tree answers "cannot tell" instead of stalling the hook.
const treeBudget = 20000

// BirthTimeResolver vouches for a path the file system says was born after the
// session started. It is the one resolver that works for any identified agent,
// however it was launched, but it cannot tell this session's files from
// another process's, so it is corroboration only: low on its own, and it lifts
// an attributing resolver (transcript, sandbox) to medium when they agree.
type BirthTimeResolver struct {
	start time.Time
}

// NewBirthTimeResolver trusts births at or after start, but no earlier than
// the 24 hours before now.
func NewBirthTimeResolver(start, now time.Time) BirthTimeResolver {
	if start.IsZero() {
		return BirthTimeResolver{}
	}
	if floor := now.Add(-maxStartWindow); start.Before(floor) {
		start = floor
	}
	return BirthTimeResolver{start: start}
}

func (BirthTimeResolver) Name() string { return "birth-time" }

// CorroboratesOnly marks birth time as unable to say whose file it is.
func (BirthTimeResolver) CorroboratesOnly() bool { return true }

// Vouch says yes when the path, and for a recursive query everything below it,
// was born after the start.
func (r BirthTimeResolver) Vouch(q CreationQuery) (Vouch, bool) {
	if r.start.IsZero() {
		return Vouch{}, false
	}
	born, ok := birthTime(q.Path)
	if !ok || born.Before(r.start) {
		return Vouch{}, false
	}
	if q.Recursive {
		if info, err := os.Lstat(q.Path); err == nil && info.IsDir() {
			budget := treeBudget
			complete := true
			_ = filepath.WalkDir(q.Path, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					complete = false
					return fs.SkipAll
				}
				if path == q.Path {
					return nil
				}
				if budget--; budget < 0 {
					complete = false
					return fs.SkipAll
				}
				if when, known := birthTime(path); !known || when.Before(r.start) {
					complete = false
					return fs.SkipAll
				}
				return nil
			})
			if !complete {
				return Vouch{}, false
			}
		}
	}
	return Vouch{Certainty: CertaintyLow, Source: "birth-time", Detail: "created " + born.Format(time.RFC3339) + ", after the session started " + r.start.Format(time.RFC3339)}, true
}
