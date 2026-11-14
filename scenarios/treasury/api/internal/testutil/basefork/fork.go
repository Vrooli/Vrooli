// Package basefork owns the opt-in, disposable local Base fork used by Treasury
// integration rehearsals. Every write is confined to its loopback Anvil child.
package basefork

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/sha3"
)

const USDC = "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913"

type Fork struct {
	URL, Upstream, BlockNumber, BlockHash string
	Accounts                              []string
	client                                *http.Client
	cmd                                   *exec.Cmd
	done                                  chan error
	logs                                  *os.File
	stopOnce                              sync.Once
	stopMu                                sync.Mutex
	stopReceipt                           StopReceipt
}

// AcquireSharedFacilitatorHarness serializes opt-in tests that use the one
// lifecycle-managed local facilitator identity and its fixed loopback ports.
// Go's package test binaries can run concurrently, so an in-process mutex is
// not sufficient. The held loopback listener is the cross-process lease; it
// is released automatically when the test ends.
func AcquireSharedFacilitatorHarness(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(12 * time.Minute)
	var listener net.Listener
	for time.Now().Before(deadline) {
		candidate, err := net.Listen("tcp", "127.0.0.1:18499")
		if err == nil {
			listener = candidate
			break
		}
		timer := time.NewTimer(250 * time.Millisecond)
		select {
		case <-t.Context().Done():
			timer.Stop()
			t.Fatalf("wait for shared facilitator integration lease: %v", t.Context().Err())
		case <-timer.C:
		}
	}
	if listener == nil {
		t.Fatal("timed out waiting for shared facilitator integration lease")
	}
	t.Cleanup(func() { _ = listener.Close() })
}

// Identity is the immutable chain identity the rehearsal binds to a Treasury
// instance. URL is always a loopback endpoint owned by the Go test process.
type Identity struct {
	URL         string   `json:"url"`
	Upstream    string   `json:"upstream"`
	ChainID     string   `json:"chain_id"`
	BlockNumber string   `json:"fork_block_number"`
	BlockHash   string   `json:"fork_block_hash"`
	AnvilPID    int      `json:"anvil_pid"`
	Accounts    []string `json:"test_accounts,omitempty"`
}

// StopReceipt records whether this fixture's child exited and its loopback
// endpoint stopped serving after the owner interrupted it.
type StopReceipt struct {
	PID                    int       `json:"anvil_pid"`
	StoppedAt              time.Time `json:"stopped_at"`
	InterruptSent          bool      `json:"interrupt_sent"`
	KilledAfterTimeout     bool      `json:"killed_after_timeout"`
	ProcessExited          bool      `json:"process_exited"`
	ProcessWaitError       string    `json:"process_wait_error,omitempty"`
	EndpointReachableAfter bool      `json:"endpoint_reachable_after_stop"`
	EndpointProbeError     string    `json:"endpoint_probe_error,omitempty"`
	Clean                  bool      `json:"clean"`
}

func (f *Fork) Identity() Identity {
	return Identity{URL: f.URL, Upstream: f.Upstream, ChainID: "0x2105", BlockNumber: f.BlockNumber, BlockHash: f.BlockHash, AnvilPID: f.cmd.Process.Pid, Accounts: append([]string(nil), f.Accounts...)}
}

// Stop interrupts the owned child, waits within ctx, kills it if the bounded
// owner wait expires, and checks that its loopback JSON-RPC listener is gone.
// Repeated calls return the same receipt and never signal a reused process ID.
func (f *Fork) Stop(ctx context.Context) StopReceipt {
	if ctx == nil {
		ctx = context.Background()
	}
	f.stopOnce.Do(func() {
		receipt := StopReceipt{PID: f.cmd.Process.Pid}
		if err := f.cmd.Process.Signal(os.Interrupt); err == nil {
			receipt.InterruptSent = true
		}
		select {
		case waitErr := <-f.done:
			receipt.ProcessExited = true
			if waitErr != nil {
				receipt.ProcessWaitError = waitErr.Error()
			}
		case <-ctx.Done():
			receipt.KilledAfterTimeout = true
			_ = f.cmd.Process.Kill()
			if waitErr := <-f.done; waitErr != nil {
				receipt.ProcessWaitError = waitErr.Error()
			}
			receipt.ProcessExited = true
		}
		if f.logs != nil {
			_ = f.logs.Close()
		}
		probeCtx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
		defer cancel()
		var chainID string
		probeErr := f.Call(probeCtx, "eth_chainId", []any{}, &chainID)
		receipt.EndpointReachableAfter = probeErr == nil
		if probeErr != nil {
			receipt.EndpointProbeError = probeErr.Error()
		}
		receipt.StoppedAt = time.Now().UTC()
		receipt.Clean = receipt.ProcessExited && !receipt.KilledAfterTimeout && !receipt.EndpointReachableAfter
		f.stopMu.Lock()
		f.stopReceipt = receipt
		f.stopMu.Unlock()
	})
	f.stopMu.Lock()
	defer f.stopMu.Unlock()
	return f.stopReceipt
}

// Start requires a deliberate live-test opt-in because a fork reads a public
// RPC and starts a subprocess. The upstream is never used for a write method.
func Start(t *testing.T) *Fork {
	return StartAt(t, 0)
}

// StartAt starts the disposable fork on a requested loopback port. Port zero
// retains the default ephemeral-port behavior; fixed ports let a separately
// supervised local resource bind to the fork without exposing any public RPC.
func StartAt(t *testing.T, requestedPort int) *Fork {
	t.Helper()
	if os.Getenv("TREASURY_BASE_FORK_TEST") != "1" {
		t.Skip("set TREASURY_BASE_FORK_TEST=1 for the disposable Base fork rehearsal")
	}
	upstream := strings.TrimSpace(os.Getenv("TREASURY_BASE_FORK_RPC"))
	if upstream == "" {
		upstream = "https://base-rpc.publicnode.com"
	}
	parsed, err := url.Parse(upstream)
	if err != nil || parsed.Scheme != "https" || parsed.User != nil {
		t.Fatal("fork upstream must be a credential-free HTTPS RPC")
	}
	client := &http.Client{Timeout: 20 * time.Second}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	var block struct{ Number, Hash string }
	if err := call(ctx, client, upstream, "eth_getBlockByNumber", []any{"finalized", false}, &block); err != nil {
		t.Fatalf("read upstream fork identity: %v", err)
	}
	number, ok := new(big.Int).SetString(strings.TrimPrefix(block.Number, "0x"), 16)
	if !ok || number.Sign() <= 0 || block.Hash == "" {
		t.Fatal("upstream returned no finalized fork identity")
	}
	_, source, _, _ := runtime.Caller(0)
	ui := filepath.Clean(filepath.Join(filepath.Dir(source), "../../../../ui"))
	module, err := filepath.EvalSymlinks(filepath.Join(ui, "node_modules/@foundry-rs/anvil"))
	if err != nil {
		t.Fatalf("Anvil is not provisioned through Treasury's governed dev dependencies: %v", err)
	}
	platform := runtime.GOOS + "-" + runtime.GOARCH
	if runtime.GOOS == "windows" {
		platform = "win32-" + runtime.GOARCH
	}
	binary := filepath.Join(filepath.Dir(module), "anvil-"+platform, "bin", "anvil")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	if requestedPort < 0 || requestedPort > 65535 {
		t.Fatalf("invalid loopback fork port %d", requestedPort)
	}
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", requestedPort))
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()
	logs, err := os.Create(filepath.Join(t.TempDir(), "anvil.log"))
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(binary, "--host", "127.0.0.1", "--port", strconv.Itoa(port), "--chain-id", "8453", "--fork-url", upstream, "--fork-block-number", number.String(), "--accounts", "3", "--threads", "2", "--silent")
	cmd.Stdout, cmd.Stderr = logs, logs
	cmd.Env = append(os.Environ(), "NO_COLOR=1")
	if err := cmd.Start(); err != nil {
		_ = logs.Close()
		t.Fatalf("start local fork: %v", err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	var fork *Fork
	t.Cleanup(func() {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer stopCancel()
		receipt := fork.Stop(stopCtx)
		if !receipt.Clean {
			t.Errorf("local Base fork cleanup was not clean: %+v", receipt)
		}
		if t.Failed() {
			if data, err := os.ReadFile(logs.Name()); err == nil {
				if len(data) > 8192 {
					data = data[len(data)-8192:]
				}
				t.Logf("Anvil process output: %s", data)
			}
		}
	})
	fork = &Fork{URL: "http://127.0.0.1:" + strconv.Itoa(port), Upstream: upstream, BlockNumber: block.Number, BlockHash: block.Hash, client: client, cmd: cmd, done: done, logs: logs}
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		var chainID string
		if err := fork.Call(ctx, "eth_chainId", []any{}, &chainID); err == nil {
			if chainID != "0x2105" {
				t.Fatalf("unexpected fork chain %s", chainID)
			}
			break
		}
		select {
		case <-ctx.Done():
			t.Fatalf("local fork did not become ready: %v (logs %s)", ctx.Err(), logs.Name())
		case err := <-done:
			done <- err
			t.Fatalf("local fork exited: %v (logs %s)", err, logs.Name())
		case <-ticker.C:
		}
	}
	if err := fork.Call(ctx, "eth_accounts", []any{}, &fork.Accounts); err != nil || len(fork.Accounts) != 3 {
		t.Fatalf("local test accounts unavailable: %v", err)
	}
	t.Logf("Base fork: upstream=%s block=%s hash=%s local=%s", upstream, number, block.Hash, fork.URL)
	return fork
}

func (f *Fork) Call(ctx context.Context, method string, params []any, result any) error {
	raw, err := f.CallRaw(ctx, method, params)
	if err != nil {
		return err
	}
	if result == nil {
		return nil
	}
	return json.Unmarshal(raw, result)
}

// CallRaw returns the exact JSON-RPC result body from the owned loopback fork.
// Rehearsals use it when the unprojected node evidence must be retained.
func (f *Fork) CallRaw(ctx context.Context, method string, params []any) (json.RawMessage, error) {
	u, err := url.Parse(f.URL)
	if err != nil || u.Scheme != "http" || u.Hostname() != "127.0.0.1" {
		return nil, fmt.Errorf("fork mutations require the owned loopback node")
	}
	return callRaw(ctx, f.client, f.URL, method, params)
}

func call(ctx context.Context, client *http.Client, endpoint, method string, params []any, result any) error {
	raw, err := callRaw(ctx, client, endpoint, method, params)
	if err != nil {
		return err
	}
	if result == nil {
		return nil
	}
	return json.Unmarshal(raw, result)
}

func callRaw(ctx context.Context, client *http.Client, endpoint, method string, params []any) (json.RawMessage, error) {
	payload, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	var envelope struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 16<<20)).Decode(&envelope); err != nil {
		return nil, err
	}
	if envelope.Error != nil {
		return nil, fmt.Errorf("%s: RPC %d: %s", method, envelope.Error.Code, envelope.Error.Message)
	}
	if len(envelope.Result) == 0 {
		return nil, fmt.Errorf("%s: JSON-RPC response omitted result", method)
	}
	return append(json.RawMessage(nil), envelope.Result...), nil
}

// Mint configures a local-only minter on the fork's real USDC implementation.
// It uses no live key or live issuance: Anvil alone unlocks the forked role.
func (f *Fork) Mint(ctx context.Context, recipient, amount string) (string, error) {
	var encoded string
	if err := f.Call(ctx, "eth_call", []any{map[string]string{"to": USDC, "data": calldata("masterMinter()")}, "latest"}, &encoded); err != nil {
		return "", err
	}
	if len(encoded) != 66 {
		return "", fmt.Errorf("USDC master minter returned invalid address")
	}
	minter := "0x" + encoded[len(encoded)-40:]
	if err := f.Call(ctx, "anvil_impersonateAccount", []any{minter}, nil); err != nil {
		return "", err
	}
	defer func() { _ = f.Call(context.WithoutCancel(ctx), "anvil_stopImpersonatingAccount", []any{minter}, nil) }()
	if err := f.Call(ctx, "anvil_setBalance", []any{minter, "0xde0b6b3a7640000"}, nil); err != nil {
		return "", err
	}
	quantity, ok := new(big.Int).SetString(amount, 10)
	if !ok || quantity.Sign() <= 0 || quantity.BitLen() > 256 {
		return "", fmt.Errorf("invalid test mint amount")
	}
	if _, err := f.Send(ctx, minter, USDC, calldata("configureMinter(address,uint256)", addressWord(f.Accounts[0]), quantity.Text(16))); err != nil {
		return "", err
	}
	return f.Send(ctx, f.Accounts[0], USDC, calldata("mint(address,uint256)", addressWord(recipient), quantity.Text(16)))
}

func (f *Fork) Transfer(ctx context.Context, sender, recipient, amount string) (string, error) {
	quantity, ok := new(big.Int).SetString(amount, 10)
	if !ok || quantity.Sign() <= 0 || quantity.BitLen() > 256 {
		return "", fmt.Errorf("invalid test transfer amount")
	}
	return f.Send(ctx, sender, USDC, calldata("transfer(address,uint256)", addressWord(recipient), quantity.Text(16)))
}

func (f *Fork) Send(ctx context.Context, sender, recipient, data string) (string, error) {
	var hash string
	err := f.Call(ctx, "eth_sendTransaction", []any{map[string]string{"from": sender, "to": recipient, "data": data, "gas": "0x100000"}}, &hash)
	if err != nil {
		return "", err
	}
	// eth_sendTransaction may return while automining is still queued. Mine an
	// explicit local block before reading so an unknown pending hash is never
	// delegated to the upstream provider's archive lookup.
	if err := f.Call(ctx, "evm_mine", []any{}, nil); err != nil {
		return "", err
	}
	var receipt struct{ Status string }
	if err := f.Call(ctx, "eth_getTransactionReceipt", []any{hash}, &receipt); err != nil {
		return "", err
	}
	if receipt.Status != "0x1" {
		return "", fmt.Errorf("local transaction %s failed", hash)
	}
	return hash, nil
}

func calldata(signature string, words ...string) string {
	hash := sha3.NewLegacyKeccak256()
	_, _ = hash.Write([]byte(signature))
	encoded := "0x" + hex.EncodeToString(hash.Sum(nil)[:4])
	for _, word := range words {
		encoded += strings.Repeat("0", 64-len(word)) + word
	}
	return encoded
}

func addressWord(address string) string { return strings.TrimPrefix(strings.ToLower(address), "0x") }
