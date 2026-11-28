package gocache

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// pinJournalHeader versions the journal format. The reclaimer refuses to
// delete while a live session writes a format it does not understand.
const pinJournalHeader = "vrooli-gocache-pins v1\n"

const pinJournalSuffix = ".pins"

// pinJournal is a session's append-only record of the IDs it has handed out.
// The session holds an exclusive lock on the journal for its whole life; the
// lock, not the file's existence, is what marks the session as live.
type pinJournal struct {
	file *os.File
	path string
	line []byte
}

// createPinJournal must run under the shared coordination lock so the
// reclaimer can never observe the journal between creation and locking and
// mistake it for a dead session's.
func createPinJournal(sessionsDir string) (*pinJournal, error) {
	var suffix [8]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		return nil, err
	}
	path := filepath.Join(sessionsDir, fmt.Sprintf("%d-%s%s", os.Getpid(), hex.EncodeToString(suffix[:]), pinJournalSuffix))
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fmt.Errorf("create pin journal: %w", err)
	}
	locked, err := tryLock(file, true)
	if err != nil || !locked {
		file.Close()
		os.Remove(path)
		if err == nil {
			err = errors.New("journal lock was not granted")
		}
		return nil, fmt.Errorf("lock pin journal: %w", err)
	}
	if _, err := file.WriteString(pinJournalHeader); err != nil {
		file.Close()
		os.Remove(path)
		return nil, fmt.Errorf("write pin journal header: %w", err)
	}
	return &pinJournal{file: file, path: path}, nil
}

// pin appends IDs with one write so a reader under the exclusive
// coordination lock always sees whole records.
func (j *pinJournal) pin(ids ...ID) error {
	j.line = j.line[:0]
	for _, id := range ids {
		j.line = hex.AppendEncode(j.line, id[:])
		j.line = append(j.line, '\n')
	}
	n, err := j.file.Write(j.line)
	if err == nil && n != len(j.line) {
		err = io.ErrShortWrite
	}
	return err
}

// close ends the session's pins. cmd/go sends close only after it no longer
// needs any output, so the journal is removed before the lock is released.
func (j *pinJournal) close() error {
	if j == nil {
		return nil
	}
	removeErr := os.Remove(j.path)
	closeErr := j.file.Close()
	return errors.Join(removeErr, closeErr)
}

// pinSnapshot is the reclaimer's incremental view of every live session's
// pins. It must be refreshed while the coordination lock is held
// exclusively, which also makes stale-journal removal safe.
type pinSnapshot struct {
	journals map[string]*journalPins
	// unreadable names a live journal whose format is unknown; while it is
	// set, nothing may be deleted.
	unreadable string
}

type journalPins struct {
	offset int64
	ids    map[ID]struct{}
}

func newPinSnapshot() *pinSnapshot {
	return &pinSnapshot{journals: map[string]*journalPins{}}
}

// refresh reads new records from every live journal, removes journals whose
// session has ended, and forgets pins of sessions that closed.
func (p *pinSnapshot) refresh(sessionsDir string) error {
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		return fmt.Errorf("list build-cache sessions: %w", err)
	}
	p.unreadable = ""
	seen := make(map[string]bool, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, pinJournalSuffix) {
			continue
		}
		live, err := p.refreshJournal(filepath.Join(sessionsDir, name), name)
		if err != nil {
			return err
		}
		if live {
			seen[name] = true
		}
	}
	for name := range p.journals {
		if !seen[name] {
			delete(p.journals, name)
		}
	}
	return nil
}

func (p *pinSnapshot) refreshJournal(path, name string) (bool, error) {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("open pin journal %s: %w", name, err)
	}
	defer file.Close()
	acquired, err := tryLock(file, true)
	if err != nil {
		return false, fmt.Errorf("probe pin journal %s: %w", name, err)
	}
	if acquired {
		// The owning session has ended without closing. Its pins are void and
		// the journal is garbage. Removal is safe: sessions create journals
		// only under the shared coordination lock this caller holds exclusively.
		removeErr := os.Remove(path)
		_ = unlock(file)
		if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			return false, fmt.Errorf("remove ended session journal %s: %w", name, removeErr)
		}
		return false, nil
	}
	journal := p.journals[name]
	if journal == nil {
		journal = &journalPins{ids: map[ID]struct{}{}}
	}
	data, err := readFrom(file, journal.offset)
	if err != nil {
		return false, fmt.Errorf("read pin journal %s: %w", name, err)
	}
	if journal.offset == 0 {
		if len(data) < len(pinJournalHeader) && bytes.HasPrefix([]byte(pinJournalHeader), data) {
			// A live session between creating and heading its journal holds
			// no pins yet.
			p.journals[name] = journal
			return true, nil
		}
		if !bytes.HasPrefix(data, []byte(pinJournalHeader)) {
			p.unreadable = name
			return true, nil
		}
		data = data[len(pinJournalHeader):]
		journal.offset = int64(len(pinJournalHeader))
	}
	for {
		newline := bytes.IndexByte(data, '\n')
		if newline < 0 {
			break
		}
		id, ok := idFromHex(string(data[:newline]))
		if !ok {
			p.unreadable = name
			return true, nil
		}
		journal.ids[id] = struct{}{}
		journal.offset += int64(newline + 1)
		data = data[newline+1:]
	}
	p.journals[name] = journal
	return true, nil
}

func readFrom(file *os.File, offset int64) ([]byte, error) {
	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return nil, err
	}
	return io.ReadAll(file)
}

func (p *pinSnapshot) pinned(id ID) bool {
	for _, journal := range p.journals {
		if _, ok := journal.ids[id]; ok {
			return true
		}
	}
	return false
}

func (p *pinSnapshot) liveSessions() int { return len(p.journals) }
