package gocache

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// markInterval bounds how stale a served entry's mtime may be. Recency is
// reclaim's secondary guard, so it is kept finer than cmd/go's one hour.
const markInterval = 10 * time.Minute

// cacheFileMode matches the permissions cmd/go gives cache files under a
// conventional umask, so native readers of the same user are unaffected.
const cacheFileMode = 0o644

// errMiss is a lookup that found no servable entry.
var errMiss = errors.New("cache miss")

// session is one GOCACHEPROG process: the cache of exactly one go command.
type session struct {
	store   Store
	state   *protocolState
	journal *pinJournal
	now     func() time.Time
	stderr  io.Writer

	// degraded explains why this session stopped sharing the store. A
	// degraded session serves only misses and keeps its own outputs in a
	// private directory, so it can never rely on an unpinned shared file.
	degraded string
	private  string
}

func newSession(store Store, now func() time.Time, stderr io.Writer) *session {
	s := &session{store: store, now: now, stderr: stderr}
	state, err := openState(store)
	if err != nil {
		s.degrade(err.Error())
		return s
	}
	s.state = state
	err = state.shared(func() error {
		journal, err := createPinJournal(state.sessionsDir())
		s.journal = journal
		return err
	})
	if err != nil {
		s.degrade(err.Error())
	}
	return s
}

func (s *session) degrade(reason string) {
	if s.degraded != "" {
		return
	}
	s.degraded = reason
	fmt.Fprintf(s.stderr, "vrooli-gocacheprog: build cache coordination unavailable (%s); this go command builds without shared cache reuse\n", reason)
}

// close releases the session's pins. It runs after cmd/go's close request or
// when cmd/go goes away; either way no output is needed any longer.
func (s *session) close() error {
	var errs []error
	if s.journal != nil {
		errs = append(errs, s.journal.close())
	}
	if s.state != nil {
		errs = append(errs, s.state.Close())
	}
	if s.private != "" {
		errs = append(errs, os.RemoveAll(s.private))
	}
	return errors.Join(errs...)
}

type getResult struct {
	entry    Entry
	diskPath string
}

// get serves a hit only after the action and output IDs are pinned, under
// the shared coordination lock, so reclaim cannot remove the output before
// this session's go command exits.
func (s *session) get(action ID) (getResult, error) {
	if s.degraded != "" {
		return getResult{}, errMiss
	}
	var (
		result     getResult
		actionInfo os.FileInfo
		outputInfo os.FileInfo
		pinErr     error
	)
	err := s.state.shared(func() error {
		entry, info, err := s.readEntry(action)
		if err != nil {
			return errMiss
		}
		path := s.store.entryPath(entry.Output, kindOutput)
		output, err := os.Lstat(path)
		if err != nil || !output.Mode().IsRegular() || output.Size() != entry.Size {
			// A directory output is a native executable-cache entry, which a
			// GOCACHEPROG session cannot serve.
			return errMiss
		}
		if pinErr = s.journal.pin(action, entry.Output); pinErr != nil {
			return pinErr
		}
		result = getResult{entry: entry, diskPath: path}
		actionInfo, outputInfo = info, output
		return nil
	})
	if pinErr != nil {
		s.degrade("record pin: " + pinErr.Error())
		return getResult{}, errMiss
	}
	if err != nil {
		if !errors.Is(err, errMiss) {
			s.degrade(err.Error())
		}
		return getResult{}, errMiss
	}
	s.markUsed(s.store.entryPath(action, kindAction), actionInfo)
	s.markUsed(result.diskPath, outputInfo)
	return result, nil
}

func (s *session) readEntry(action ID) (Entry, os.FileInfo, error) {
	file, err := os.Open(s.store.entryPath(action, kindAction))
	if err != nil {
		return Entry{}, nil, err
	}
	defer file.Close()
	data := make([]byte, entrySize+1)
	n, err := io.ReadFull(file, data)
	if !errors.Is(err, io.ErrUnexpectedEOF) || n != entrySize {
		return Entry{}, nil, errEntryInvalid
	}
	entry, err := decodeEntry(action, data[:n])
	if err != nil {
		return Entry{}, nil, err
	}
	info, err := file.Stat()
	if err != nil {
		return Entry{}, nil, err
	}
	return entry, info, nil
}

// markUsed refreshes an entry's last-use time, which orders reclaim.
func (s *session) markUsed(path string, info os.FileInfo) {
	now := s.now()
	if info != nil && now.Sub(info.ModTime()) < markInterval {
		return
	}
	_ = os.Chtimes(path, now, now)
}

// put stores a body under its output ID and records the action entry. The
// IDs are pinned before either file becomes visible.
func (s *session) put(action, output ID, size int64, body io.Reader) (string, error) {
	dir := s.store.shardDir(output)
	if s.degraded != "" {
		private, err := s.privateDir()
		if err != nil {
			return "", err
		}
		dir = private
	}
	temp, err := createStaged(dir, output.String()+"-d")
	if err != nil {
		return "", fmt.Errorf("stage cache output: %w", err)
	}
	staged := temp.Name()
	hash := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(temp, hash), body)
	modeErr := temp.Chmod(cacheFileMode)
	closeErr := temp.Close()
	if err := errors.Join(copyErr, modeErr, closeErr); err != nil {
		os.Remove(staged)
		return "", fmt.Errorf("stage cache output: %w", err)
	}
	if written != size {
		os.Remove(staged)
		return "", fmt.Errorf("cache output for %s is %d bytes; cmd/go declared %d", output, written, size)
	}
	if got, _ := idFromBytes(hash.Sum(nil)); got != output {
		os.Remove(staged)
		return "", fmt.Errorf("cache output content does not match output ID %s", output)
	}
	if s.degraded == "" {
		if err := s.state.shared(func() error { return s.journal.pin(action, output) }); err != nil {
			s.degrade("record pin: " + err.Error())
		}
	}
	if s.degraded != "" {
		return s.keepPrivately(staged, output)
	}
	final := s.store.entryPath(output, kindOutput)
	if err := os.Rename(staged, final); err != nil {
		os.Remove(staged)
		return "", fmt.Errorf("publish cache output: %w", err)
	}
	if err := s.writeEntry(action, Entry{Output: output, Size: size, Time: s.now()}); err != nil {
		// The output is still pinned and usable; only reuse is lost.
		fmt.Fprintf(s.stderr, "vrooli-gocacheprog: record cache entry %s: %v\n", action, err)
	}
	return final, nil
}

func (s *session) writeEntry(action ID, entry Entry) error {
	temp, err := createStaged(s.store.shardDir(action), action.String()+"-a")
	if err != nil {
		return err
	}
	staged := temp.Name()
	_, writeErr := temp.Write(encodeEntry(action, entry))
	modeErr := temp.Chmod(cacheFileMode)
	closeErr := temp.Close()
	if err := errors.Join(writeErr, modeErr, closeErr); err != nil {
		os.Remove(staged)
		return err
	}
	if err := os.Rename(staged, s.store.entryPath(action, kindAction)); err != nil {
		os.Remove(staged)
		return err
	}
	return nil
}

// keepPrivately moves a staged output into the session's private directory
// when the session can no longer pin shared files.
func (s *session) keepPrivately(staged string, output ID) (string, error) {
	private, err := s.privateDir()
	if err != nil {
		os.Remove(staged)
		return "", err
	}
	final := filepath.Join(private, output.String()+"-d")
	if filepath.Dir(staged) == private {
		if err := os.Rename(staged, final); err != nil {
			os.Remove(staged)
			return "", err
		}
		return final, nil
	}
	defer os.Remove(staged)
	if err := copyFile(staged, final); err != nil {
		return "", fmt.Errorf("keep cache output privately: %w", err)
	}
	return final, nil
}

func (s *session) privateDir() (string, error) {
	if s.private != "" {
		return s.private, nil
	}
	dir, err := os.MkdirTemp("", "vrooli-gocacheprog-")
	if err != nil {
		return "", fmt.Errorf("create private cache directory: %w", err)
	}
	s.private = dir
	return dir, nil
}

// createStaged opens a staging file beside its final name. cmd/go creates
// every shard when it opens the cache, but `go clean -cache` removes them
// while builds run, so a missing shard is recreated.
func createStaged(dir, final string) (*os.File, error) {
	temp, err := os.CreateTemp(dir, final+tempMarker+"*")
	if errors.Is(err, os.ErrNotExist) {
		if mkErr := os.MkdirAll(dir, 0o755); mkErr != nil {
			return nil, mkErr
		}
		temp, err = os.CreateTemp(dir, final+tempMarker+"*")
	}
	return temp, err
}

func copyFile(from, to string) error {
	source, err := os.Open(from)
	if err != nil {
		return err
	}
	defer source.Close()
	target, err := os.OpenFile(to, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(target, source)
	return errors.Join(copyErr, target.Close())
}
