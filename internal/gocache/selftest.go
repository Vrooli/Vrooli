package gocache

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// SelfTest runs program exactly as cmd/go starts a GOCACHEPROG, against a
// private store, and proves it stores and serves an output and releases its
// session on close. Activation requires it: once GOCACHEPROG names the
// helper, every go command on the host depends on it.
func SelfTest(ctx context.Context, program string) (err error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cache, err := os.MkdirTemp("", "vrooli-gocache-selftest-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(cache)
	for i := range shardCount {
		if err := os.Mkdir(filepath.Join(cache, shardName(byte(i))), 0o755); err != nil {
			return err
		}
	}
	cmd := exec.CommandContext(ctx, program)
	cmd.Env = append(withoutEnv(os.Environ(), "GOCACHE"), "GOCACHE="+cache)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start %s: %w", program, err)
	}
	defer func() {
		_ = stdin.Close()
		if waitErr := cmd.Wait(); err == nil && waitErr != nil {
			err = fmt.Errorf("helper exited: %w: %s", waitErr, strings.TrimSpace(stderr.String()))
		}
	}()
	responses := json.NewDecoder(bufio.NewReader(stdout))
	read := func(want int64) (progResponse, error) {
		var response progResponse
		if err := responses.Decode(&response); err != nil {
			return response, fmt.Errorf("read helper response: %w (stderr: %s)", err, strings.TrimSpace(stderr.String()))
		}
		if response.ID != want {
			return response, fmt.Errorf("helper answered request %d, want %d", response.ID, want)
		}
		if response.Err != "" {
			return response, fmt.Errorf("helper refused request %d: %s", want, response.Err)
		}
		return response, nil
	}
	send := func(request progRequest, body []byte) error {
		header, err := json.Marshal(request)
		if err != nil {
			return err
		}
		frame := append(header, '\n', '\n')
		if len(body) > 0 {
			frame = append(frame, '"')
			frame = append(frame, base64.StdEncoding.EncodeToString(body)...)
			frame = append(frame, '"', '\n')
		}
		_, err = stdin.Write(frame)
		return err
	}

	hello, err := read(0)
	if err != nil {
		return err
	}
	if strings.Join(hello.KnownCommands, ",") != "get,put,close" {
		return fmt.Errorf("helper advertises %v, want get, put and close", hello.KnownCommands)
	}
	body := []byte("vrooli build-cache protocol self-test\n")
	action, output := sha256.Sum256([]byte("vrooli-gocache-selftest")), sha256.Sum256(body)
	if err := send(progRequest{ID: 1, Command: commandPut, ActionID: action[:], OutputID: output[:], BodySize: int64(len(body))}, body); err != nil {
		return err
	}
	if _, err := read(1); err != nil {
		return err
	}
	if err := send(progRequest{ID: 2, Command: commandGet, ActionID: action[:]}, nil); err != nil {
		return err
	}
	hit, err := read(2)
	if err != nil {
		return err
	}
	if hit.Miss || !bytes.Equal(hit.OutputID, output[:]) {
		return errors.New("helper did not serve the output it stored")
	}
	if served, err := os.ReadFile(hit.DiskPath); err != nil || !bytes.Equal(served, body) {
		return fmt.Errorf("helper served unreadable or wrong content at %s: %v", hit.DiskPath, err)
	}
	journals, _ := filepath.Glob(filepath.Join(cache, StateDirName, sessionsDirName, "*"+pinJournalSuffix))
	if len(journals) != 1 {
		return fmt.Errorf("helper holds %d session journals during a session, want 1", len(journals))
	}
	if err := send(progRequest{ID: 3, Command: commandClose}, nil); err != nil {
		return err
	}
	if _, err := read(3); err != nil {
		return err
	}
	if _, err := io.Copy(io.Discard, stdout); err != nil {
		return err
	}
	if journals, _ := filepath.Glob(filepath.Join(cache, StateDirName, sessionsDirName, "*"+pinJournalSuffix)); len(journals) != 0 {
		return errors.New("helper left its session journal after close")
	}
	return nil
}

func withoutEnv(environ []string, key string) []string {
	out := make([]string, 0, len(environ))
	for _, entry := range environ {
		if !strings.HasPrefix(entry, key+"=") {
			out = append(out, entry)
		}
	}
	return out
}

// DefaultCacheDir is the GOCACHE a go command started by this user would
// use: GOCACHE from this process's environment, then the go env file, then
// cmd/go's default under the user cache directory.
func DefaultCacheDir() (string, error) {
	settings := resolveGoSettings(os.Getenv, ownDirs())
	if settings.Cache == "" || settings.Cache == "off" {
		return "", errors.New("no Go build cache resolves for this user")
	}
	return settings.Cache, nil
}
