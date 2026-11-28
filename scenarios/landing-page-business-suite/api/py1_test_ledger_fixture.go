package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type py1LedgerFixtureConfig struct {
	URL              string `json:"url"`
	FixtureID        string `json:"fixture_id"`
	AdapterID        string `json:"adapter"`
	BookID           string `json:"book"`
	RevenueAccountID string `json:"service_order_revenue"`
}

type py1LedgerFixtureLease struct {
	config py1LedgerFixtureConfig
	cmd    *exec.Cmd
	client *http.Client
	once   sync.Once
}

// startLocalPY1LedgerFixture launches the dedicated Money Ledger fixture from
// the checked-out API source. Its subprocess inherits only toolchain/runtime
// location variables; credentials, agent identity, and API authentication are
// excluded. The fixture publishes endpoint and scope IDs, never tokens.
func startLocalPY1LedgerFixture(ctx context.Context) (*py1LedgerFixtureLease, error) {
	root, err := py1SourceRoot()
	if err != nil {
		return nil, errors.New("Vrooli source root is unavailable for the local PY1 ledger fixture")
	}
	apiDir := filepath.Join(root, "scenarios", "money-ledger", "api")
	if _, err := os.Stat(filepath.Join(apiDir, "go.mod")); err != nil {
		return nil, errors.New("local Money Ledger PY1 fixture source is unavailable")
	}
	goBinary, err := exec.LookPath("go")
	if err != nil {
		return nil, errors.New("Go is unavailable for the local Money Ledger PY1 fixture")
	}
	cmd := exec.Command(goBinary, "run", "./cmd/py1-test-ledger-fixture")
	cmd.Dir = apiDir
	cmd.Env = py1LedgerFixtureEnvironment(os.Environ())
	cmd.Stderr = io.Discard
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, errors.New("could not capture local fixture identity")
	}
	if err := cmd.Start(); err != nil {
		return nil, errors.New("could not start local Money Ledger PY1 fixture")
	}
	lease := &py1LedgerFixtureLease{cmd: cmd, client: &http.Client{Timeout: 5 * time.Second}}
	lineCh := make(chan []byte, 1)
	readErrCh := make(chan error, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 1024), 64*1024)
		if !scanner.Scan() {
			readErrCh <- scanner.Err()
			return
		}
		lineCh <- bytes.Clone(scanner.Bytes())
		for scanner.Scan() {
			// The fixture emits no further lines. Drain bounded output without
			// retaining it so the child cannot block on a full stdout pipe.
		}
		readErrCh <- scanner.Err()
	}()
	startupCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	select {
	case line := <-lineCh:
		if err := json.Unmarshal(line, &lease.config); err != nil {
			lease.Close()
			return nil, errors.New("local fixture emitted an invalid identity record")
		}
	case <-readErrCh:
		lease.Close()
		return nil, errors.New("local fixture exited before publishing its identity")
	case <-startupCtx.Done():
		lease.Close()
		return nil, errors.New("local fixture startup timed out")
	}
	if !strings.HasPrefix(lease.config.URL, "http://127.0.0.1:") || !strings.HasPrefix(lease.config.FixtureID, "money-ledger-test-sqlite:") || lease.config.AdapterID == "" || lease.config.BookID == "" || lease.config.RevenueAccountID == "" {
		lease.Close()
		return nil, errors.New("local fixture identity is incomplete or not loopback scoped")
	}
	if err := lease.verifyIdentity(startupCtx); err != nil {
		lease.Close()
		return nil, err
	}
	return lease, nil
}

func (l *py1LedgerFixtureLease) verifyIdentity(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(l.config.URL, "/")+"/__journey/identity", nil)
	if err != nil {
		return errors.New("could not create local fixture identity request")
	}
	response, err := l.client.Do(req)
	if err != nil {
		return errors.New("local Money Ledger fixture is not reachable")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return errors.New("local Money Ledger fixture rejected identity verification")
	}
	var identity struct {
		FixtureID        string `json:"fixture_id"`
		AdapterID        string `json:"adapter_id"`
		BookID           string `json:"book_id"`
		RevenueAccountID string `json:"revenue_account_id"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 16*1024)).Decode(&identity); err != nil || identity.FixtureID != l.config.FixtureID || identity.AdapterID != l.config.AdapterID || identity.BookID != l.config.BookID || identity.RevenueAccountID != l.config.RevenueAccountID {
		return errors.New("local Money Ledger fixture identity changed or did not match its startup record")
	}
	return nil
}

func (l *py1LedgerFixtureLease) Close() {
	if l == nil || l.cmd == nil {
		return
	}
	l.once.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(l.config.URL, "/")+"/__journey/stop", nil)
		if err == nil {
			if response, requestErr := l.client.Do(req); requestErr == nil {
				response.Body.Close()
			}
		}
		done := make(chan struct{})
		go func() {
			_ = l.cmd.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-ctx.Done():
			_ = l.cmd.Process.Kill()
			<-done
		}
	})
}

func py1SourceRoot() (string, error) {
	candidates := []string{strings.TrimSpace(os.Getenv("VROOLI_SOURCE_ROOT"))}
	if executable, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Dir(executable))
	}
	if working, err := os.Getwd(); err == nil {
		candidates = append(candidates, working)
	}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		abs, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		for dir := abs; ; dir = filepath.Dir(dir) {
			if _, err := os.Stat(filepath.Join(dir, "scenarios", "money-ledger", "api", "go.mod")); err == nil {
				return dir, nil
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
		}
	}
	return "", errors.New("source root not found")
}

func py1LedgerFixtureEnvironment(entries []string) []string {
	allowed := map[string]struct{}{
		"PATH": {}, "HOME": {}, "USER": {}, "LOGNAME": {}, "TMPDIR": {}, "TMP": {}, "TEMP": {}, "LANG": {}, "LC_ALL": {},
		"GOROOT": {}, "GOPATH": {}, "GOCACHE": {}, "GOMODCACHE": {}, "GOPROXY": {}, "GOSUMDB": {}, "GOTOOLCHAIN": {}, "GOAMD64": {},
		"VROOLI_ROOT": {}, "VROOLI_SOURCE_ROOT": {}, "VROOLI_HOME": {}, "VROOLI_STATE_ROOT": {}, "VROOLI_CONFIG_ROOT": {}, "VROOLI_DATA_ROOT": {},
		"VROOLI_STORAGE_ROOT": {}, "VROOLI_LOGS_ROOT": {}, "VROOLI_BIN": {}, "VROOLI_API_PORT": {}, "VROOLI_HOST_LIFECYCLE_BASE": {},
		"HTTP_PROXY": {}, "HTTPS_PROXY": {}, "NO_PROXY": {}, "ALL_PROXY": {}, "SSL_CERT_FILE": {}, "SSL_CERT_DIR": {},
	}
	result := make([]string, 0, len(allowed)+1)
	for _, entry := range entries {
		name, _, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		if _, ok := allowed[name]; ok {
			result = append(result, entry)
		}
	}
	return append(result, "GOWORK=off")
}
