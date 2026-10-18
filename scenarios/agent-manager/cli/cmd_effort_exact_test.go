package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliutil"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	accountsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts"
	accountsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts/accounts_v1connect"
)

func TestExactReconcileSiteWireAndProtectedCredentialContinuity(t *testing.T) {
	for _, mode := range []string{"same", "changed", "changed operation", "refused", "legacy same", "wrong protocol", "unavailable", "redirect307", "redirect308"} {
		t.Run(mode, func(t *testing.T) {
			const token = "disposable-protected-test-leaf"
			const operationToken = "disposable-protected-operation"
			requests := 0
			transport := exactCLIRoundTripper(func(r *http.Request) (*http.Response, error) {
				if err := cliutil.ApplyAgentIdentity(r); err != nil {
					return nil, err
				}

				requests++
				if r.Header.Get("X-Agent-Identity-Token") != token || r.Header.Get("Authorization") != "" {
					t.Error("credential attribution changed")
				}
				wantOperation := operationToken
				if mode == "legacy same" {
					wantOperation = ""
				}
				if r.Header.Get(cliutil.HeaderAgentOperationToken) != wantOperation {
					t.Errorf("operation credential changed: got %q, want %q", r.Header.Get(cliutil.HeaderAgentOperationToken), wantOperation)
				}

				if strings.Contains(r.URL.Path, "exact-challenge") {
					return exactCLIResponse(r, `{"executor":"fixture"}`), nil
				}
				if r.Header.Get("X-Keyring-Exact-Proof") != base64.RawURLEncoding.EncodeToString([]byte(`{"v":1}`)) {
					t.Error("Site wire not carried unchanged")
				}
				if strings.HasPrefix(mode, "redirect") {
					response := exactCLIResponse(r, `{"effortRef":"must-not-confirm","revision":"must-not-confirm"}`)
					response.StatusCode = 307
					if mode == "redirect308" {
						response.StatusCode = 308
					}
					response.Header.Set("Location", "http://localhost:18801/leak")
					return response, nil
				}
				if mode == "unavailable" {
					response := exactCLIResponse(r, `{"code":"unavailable","message":"generic proxy failure"}`)
					response.StatusCode = 503
					return response, nil
				}
				return exactCLIResponse(r, `{"effortRef":"effort:fixture","revision":"2"}`), nil
			})
			client := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{Client: &http.Client{Transport: transport}}), func() cliutil.APIBaseOptions { return cliutil.APIBaseOptions{Override: "http://127.0.0.1:18800"} }, func() string { return "operator-must-not-be-used" })

			a := &App{services: &Services{Watches: &WatchService{api: client}}}
			signed := false
			resolver := func(context.Context) (cliutil.AgentCredentials, error) {
				if signed {
					if mode == "changed" {
						return cliutil.AgentCredentials{BaselineToken: "another-leaf", OperationToken: operationToken}, nil
					}
					if mode == "changed operation" {
						return cliutil.AgentCredentials{BaselineToken: token, OperationToken: "another-operation"}, nil
					}
					if mode == "refused" {
						return cliutil.AgentCredentials{}, fmt.Errorf("broker refused")
					}
				}
				if mode == "legacy same" {
					return cliutil.AgentCredentials{LegacyToken: token}, nil
				}
				return cliutil.AgentCredentials{BaselineToken: token, OperationToken: operationToken}, nil
			}
			protocol := "keyring.exact-action.v1"
			if mode == "wrong protocol" {
				protocol = "wrong"
			}
			input := fmt.Sprintf(`{"protocol":%q,"bytes":%q,"signature":%q}`, protocol, base64.RawURLEncoding.EncodeToString([]byte(`{"v":1}`)), base64.RawURLEncoding.EncodeToString(make([]byte, 64)))
			var output bytes.Buffer
			err := a.exactReconcile(&pb.ReconcileEffortMetadataRequest{}, "fixture-grant", &afterChallengeReader{Reader: strings.NewReader(input), after: func() { signed = true }}, &output, resolver)
			if mode == "same" || mode == "legacy same" {
				if err != nil || requests != 2 {
					t.Fatal(err, requests)
				}
			} else if strings.HasPrefix(mode, "redirect") {
				if err == nil || requests != 2 || !strings.Contains(err.Error(), "recover the original receipt") || !strings.Contains(err.Error(), "same executor and original request identity") || strings.Contains(output.String(), "must-not-confirm") {
					t.Fatal("redirect represented as confirmed result or lost reconciliation", err, requests, output.String())
				}
			} else if mode == "unavailable" {
				if err == nil || requests != 2 || strings.Contains(err.Error(), "uncommitted") || !strings.Contains(err.Error(), "outcome is unknown") {
					t.Fatal("generic unavailable response misrepresented", err, requests)
				}
			} else if err == nil || requests != 1 {
				t.Fatal("custody/protocol rejection submitted effect", err, requests)
			}
			if strings.Contains(output.String(), token) {
				t.Fatal("bearer printed")
			}
		})
	}
}

type afterChallengeReader struct {
	io.Reader
	after func()
}

func (r *afterChallengeReader) Read(p []byte) (int, error) { r.after(); return r.Reader.Read(p) }

type exactCLIRoundTripper func(*http.Request) (*http.Response, error)

func (f exactCLIRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func exactCLIResponse(r *http.Request, body string) *http.Response {
	return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}
}

// Disposable harness exercises the production launcher and compiled main CLI.
// Credentials come only from LaunchCodingAgentResult, never fixture injection.
func TestExactCLIJoinedProcess(t *testing.T) {
	if os.Getenv("KEYRING_DISPOSABLE_JOINED_CHILD") != "1" {
		t.Skip("native fixture only")
	}
	var config exactNativeHarnessConfig
	if json.NewDecoder(os.Stdin).Decode(&config) != nil {
		t.Fatal("fixture input")
	}
	binary := os.Getenv("KEYRING_JOINED_MAIN_BINARY")
	if binary == "" {
		t.Fatal("compiled main required")
	}
	config.Binary = binary
	// Fixed before any authority lookup, with no host broker/owner access.
	fixtureHome := t.TempDir()
	t.Setenv("HOME", fixtureHome)
	t.Setenv("TMPDIR", fixtureHome)
	t.Setenv(cliutil.EnvIdentityToken, "")
	if strings.HasPrefix(config.Mode, "setup-") {
		action := strings.TrimPrefix(config.Mode, "setup-")
		file := filepath.Join(fixtureHome, "public.json")
		if os.WriteFile(file, config.Request, 0600) != nil {
			t.Fatal("public request")
		}
		args := []string{"--api-base", config.Base, "effort", "exact-setup", action, "--request-file", file}
		if action == "status" {
			args = []string{"--api-base", config.Base, "effort", "exact-setup", action, "--grant", config.Grant}
		}
		cmd := exec.Command(binary, args...)
		cmd.Env = []string{"PATH=/usr/bin:/bin", "HOME=" + fixtureHome, "TMPDIR=" + fixtureHome, "VROOLI_CLI_ARTIFACT_MODE=1", "AGENT_MANAGER_API_TOKEN=" + config.Token}
		var output, stderr bytes.Buffer
		cmd.Stdout = &output
		cmd.Stderr = &stderr
		err := cmd.Run()
		if err != nil {
			t.Fatalf("ordinary main setup refused: %s", stderr.String())
		}
		fmt.Printf("EXACT_SETUP_RECEIPT:%s\n", bytes.TrimSpace(output.Bytes()))
		return
	}
	config.Token = ""
	raw, _ := json.Marshal(config)
	// The real launcher spawns a disposable harness process and privately delivers
	// AttachRun identity in its ordinary child environment. RunChild is not replaced.
	launcher := os.Getenv("KEYRING_JOINED_COMMAND_BINARY")
	if launcher == "" {
		t.Fatal("compiled command wrapper required")
	}
	harness, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(launcher, "--command", "--", harness, "-test.run=^TestExactNativeHarness$", "-test.count=1")
	cmd.Env = []string{
		"PATH=/usr/bin:/bin", "HOME=" + fixtureHome, "TMPDIR=" + fixtureHome,
		cliutil.AgentManagerLauncherBaseEnv + "=" + config.Base, "KEYRING_DISPOSABLE_JOINED_CHILD=1",
		"KEYRING_SITE_SIGN_HELPER=" + os.Getenv("KEYRING_SITE_SIGN_HELPER"),
	}
	cmd.Stdin = io.MultiReader(bytes.NewReader(append(raw, '\n')), os.Stdin)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("actual protected command wrapper failed: %v", err)
	}
}

type exactNativeHarnessConfig struct {
	Base, Token, Grant, Owner, Issuer string
	Key                               []byte
	Mode                              string
	Request                           json.RawMessage
	Deadline                          int64
	Binary                            string
}

func TestExactNativeHarness(t *testing.T) {
	if os.Getenv("KEYRING_DISPOSABLE_JOINED_CHILD") != "1" {
		t.Skip("native fixture only")
	}
	var config exactNativeHarnessConfig
	decoder := json.NewDecoder(os.Stdin)
	if decoder.Decode(&config) != nil {
		t.Fatal("harness request")
	}
	token, err := cliutil.ResolveAgentCredential(context.Background())
	if err != nil || token == "" {
		t.Fatal("launcher-delivered custody required", err)
	}
	file := filepath.Join(t.TempDir(), "public-request.json")
	if os.WriteFile(file, config.Request, 0600) != nil {
		t.Fatal("request")
	}
	operation := "exact-reconcile"
	if config.Mode == "read" {
		operation = "exact-receipt"
	}
	iterations := 1
	if config.Mode == "mutate" {
		iterations = 2
	}
	var firstExecutor string
	var receipt []byte
	for turn := 0; turn < iterations; turn++ {
		cmd := exec.Command(config.Binary, "--api-base", config.Base, "effort", operation, "--request-file", file, "--grant", config.Grant)
		// Identity is inherited directly from the production launcher. No new token,
		// alternative resolver, socket namespace change or owner credential is supplied.
		cmd.Env = append(os.Environ(), "VROOLI_CLI_ARTIFACT_MODE=1")
		input, err := cmd.StdinPipe()
		if err != nil {
			t.Fatal(err)
		}
		output, err := cmd.StdoutPipe()
		if err != nil {
			t.Fatal(err)
		}
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if cmd.Start() != nil {
			t.Fatal("ordinary main start")
		}
		reader := bufio.NewReader(output)
		challenge, err := reader.ReadBytes('\n')
		if err != nil {
			t.Fatal("ordinary main challenge", stderr.String())
		}
		var parsed struct {
			Executor string `json:"executor"`
		}
		if json.Unmarshal(challenge, &parsed) != nil || parsed.Executor == "" {
			t.Fatal("typed challenge")
		}
		signer := &exactJoinedSigner{output: bytes.NewBuffer(bytes.TrimSpace(challenge)), key: config.Key, owner: config.Owner, issuer: config.Issuer, deadline: config.Deadline}
		_, signErr := io.Copy(input, signer)
		_ = input.Close()
		currentReceipt, readErr := io.ReadAll(reader)
		runErr := cmd.Wait()
		if signErr != nil || readErr != nil || runErr != nil {
			t.Fatal("ordinary main signed action", signErr, readErr, runErr, stderr.String())
		}
		if bytes.Contains(currentReceipt, []byte(token)) {
			t.Fatal("credential exposed")
		}
		if turn == 0 {
			firstExecutor = parsed.Executor
			receipt = currentReceipt
		} else if parsed.Executor != firstExecutor || !bytes.Equal(bytes.TrimSpace(receipt), bytes.TrimSpace(currentReceipt)) {
			t.Fatal("ordinary executable replay changed identity or receipt")
		}
	}
	fmt.Printf("EXACT_BOOTSTRAP_RUN:%s\nEXACT_MAIN_RECEIPT:%s\n", firstExecutor, bytes.TrimSpace(receipt))
	// Keep the actual native harness alive for mounted replay/refusal checks.
	// Parent releases this private pipe after assertions; normal launcher detach follows.
	if _, err := io.Copy(io.Discard, io.MultiReader(decoder.Buffered(), os.Stdin)); err != nil {
		t.Fatal("harness completion", err)
	}
}

type exactJoinedSigner struct {
	output        *bytes.Buffer
	key           ed25519.PrivateKey
	owner, issuer string
	deadline      int64
	reader        *bytes.Reader
}

func (s *exactJoinedSigner) Read(p []byte) (int, error) {
	if s.reader == nil {
		var c struct {
			V         int             `json:"v"`
			Purpose   string          `json:"purpose"`
			Instance  string          `json:"instance"`
			Audience  string          `json:"audience"`
			Grant     string          `json:"grant"`
			Epoch     int             `json:"epoch"`
			Executor  string          `json:"executor"`
			Operation string          `json:"operation"`
			Target    string          `json:"target"`
			Payload   string          `json:"payload"`
			Request   string          `json:"request"`
			IssuedAt  int64           `json:"issuedAt"`
			Action    json.RawMessage `json:"action"`
		}
		if json.Unmarshal(bytes.TrimSpace(s.output.Bytes()), &c) != nil {
			return 0, fmt.Errorf("typed challenge required")
		}
		if os.Getenv("KEYRING_SITE_SIGN_HELPER") != "" {
			helper := os.Getenv("KEYRING_SITE_SIGN_HELPER")
			config, _ := json.Marshal(struct {
				Challenge json.RawMessage `json:"challenge"`
				Owner     string          `json:"owner"`
				Issuer    string          `json:"issuer"`
				Key       []byte          `json:"key"`
				Deadline  int64           `json:"deadline"`
			}{bytes.TrimSpace(s.output.Bytes()), s.owner, s.issuer, s.key, s.deadline})
			cmd := exec.Command("/home/matthalloran8/.nvm/versions/node/v22.23.2/bin/node", "--experimental-transform-types", "--experimental-loader", filepath.Join(filepath.Dir(helper), "loader.mjs"), helper)
			cmd.Env = []string{"PATH=/usr/bin:/bin"}
			cmd.Stdin = bytes.NewReader(config)
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			if err := cmd.Run(); err != nil {
				return 0, fmt.Errorf("actual Site signer refused fixture challenge: %s", stderr.String())
			}
			s.reader = bytes.NewReader(stdout.Bytes())
			return s.reader.Read(p)
		}
		return 0, fmt.Errorf("actual configured Site signer helper required")
	}
	return s.reader.Read(p)
}

func TestExactReceiptProtocolAndCustodyRefuseBeforeReadSubmission(t *testing.T) {
	for _, mode := range []string{"same", "changed", "refused", "mutation proof"} {
		t.Run(mode, func(t *testing.T) {
			const token = "disposable-read-leaf"
			calls := 0
			signed := false
			transport := exactCLIRoundTripper(func(r *http.Request) (*http.Response, error) {
				if err := cliutil.ApplyAgentIdentity(r); err != nil {
					return nil, err
				}
				calls++
				if r.Header.Get("X-Agent-Identity-Token") != token || r.Header.Get("Authorization") != "" {
					t.Error("reader attribution changed")
				}
				return exactCLIResponse(r, `{"fixture":true}`), nil
			})
			api := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{Client: &http.Client{Transport: transport}}), func() cliutil.APIBaseOptions { return cliutil.APIBaseOptions{Override: "http://127.0.0.1:18800"} }, nil)
			a := &App{services: &Services{Watches: &WatchService{api: api}}}
			protocol := "keyring.receipt-read.v1"
			if mode == "mutation proof" {
				protocol = "keyring.exact-action.v1"
			}
			input := fmt.Sprintf(`{"protocol":%q,"bytes":%q,"signature":%q}`, protocol, base64.RawURLEncoding.EncodeToString([]byte(`{"v":1}`)), base64.RawURLEncoding.EncodeToString(make([]byte, 64)))
			resolve := func(context.Context) (cliutil.AgentCredentials, error) {
				if signed && mode == "changed" {
					return cliutil.AgentCredentials{LegacyToken: "different-leaf"}, nil
				}
				if signed && mode == "refused" {
					return cliutil.AgentCredentials{}, fmt.Errorf("authority refused")
				}
				return cliutil.AgentCredentials{LegacyToken: token}, nil
			}
			var out bytes.Buffer
			err := a.exactReceipt(exactReceiptRequest{}, "read-grant", &afterChallengeReader{Reader: strings.NewReader(input), after: func() { signed = true }}, &out, resolve)
			if mode == "same" {
				if err != nil || calls != 2 {
					t.Fatal(err, calls)
				}
			} else if err == nil || calls != 1 {
				t.Fatal("read custody confusion submitted", err, calls)
			}
			if strings.Contains(out.String(), token) {
				t.Fatal("reader token printed")
			}
		})
	}
}

func TestExactSetupCLIUsesExistingOperatorAndTypedActions(t *testing.T) {
	for _, action := range []string{"stage", "activate", "revoke", "status"} {
		t.Run(action, func(t *testing.T) {
			requests := 0
			transport := exactCLIRoundTripper(func(r *http.Request) (*http.Response, error) {
				requests++
				if r.URL.Path != "/api/v1/keyring/authority/setup" || r.Header.Get("Authorization") != "Bearer disposable-existing-operator" {
					t.Error("setup lost existing operator proof")
				}
				if action == "status" {
					if r.Method != "GET" || r.URL.Query().Get("grant") != "fixture-grant" {
						t.Error("status contract")
					}
				} else {
					var body struct {
						Action string `json:"action"`
					}
					if json.NewDecoder(r.Body).Decode(&body) != nil || body.Action != action || r.Method != "POST" {
						t.Error("typed action")
					}
				}
				return exactCLIResponse(r, `{"grant":"fixture-grant","state":"fixture"}`), nil
			})
			client := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{Client: &http.Client{Transport: transport}}), func() cliutil.APIBaseOptions { return cliutil.APIBaseOptions{Override: "http://fixture.invalid"} }, func() string { return "disposable-existing-operator" })
			a := &App{services: &Services{Watches: &WatchService{api: client}}}
			file := filepath.Join(t.TempDir(), "setup.json")
			if err := os.WriteFile(file, exactSetupPublicJSON(t, action), 0600); err != nil {
				t.Fatal(err)
			}
			args := []string{action, "--request-file", file}
			if action == "status" {
				args = []string{action, "--grant", "fixture-grant"}
			}
			if err := a.effortExactSetup(args); err != nil || requests != 1 {
				t.Fatal("supported setup CLI", err, requests)
			}
			if action != "status" && action != "activate" {
				if err := a.effortExactSetup([]string{"activate", "--request-file", file}); action != "activate" && (err == nil || requests != 1) {
					t.Fatal("mismatched action submitted")
				}
			}
		})
	}
}

// These fixtures use a private Unix listener and disposable response strings.
// They exercise the actual exchange client and ordinary CLI dispatch, never a
// host credential provider, installed command or native account mutation.
type exactSetupOwnerFixture struct {
	accountsconnect.UnimplementedAccountsServiceHandler
	calls   atomic.Int32
	refusal connect.Code
	empty   bool
}

func (f *exactSetupOwnerFixture) ExchangeMachinePrincipal(_ context.Context, req *connect.Request[accountsv1.ExchangeMachinePrincipalRequest]) (*connect.Response[accountsv1.LoginResponse], error) {
	f.calls.Add(1)
	if req.Msg.MachineId == "" || req.Msg.Resource != "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("existing binding contract changed"))
	}
	if f.refusal != 0 {
		return nil, connect.NewError(f.refusal, fmt.Errorf("synthetic owner refusal"))
	}
	access := "synthetic-owner-access"
	if f.empty {
		access = ""
	}
	return connect.NewResponse(&accountsv1.LoginResponse{
		Tokens: &accountsv1.TokenPair{AccessToken: access, RefreshToken: "synthetic-unused-refresh"},
	}), nil
}

func newExactSetupOwnerApp(t *testing.T, fixture *exactSetupOwnerFixture, handler http.Handler) (*App, string, string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)

	t.Setenv(cliutil.EnvIdentityToken, "")
	t.Setenv("AGENT_MANAGER_API_TOKEN", "synthetic-ordinary-operator")
	t.Setenv("VROOLI_CLI_ARTIFACT_MODE", "1")
	socketDir, err := os.MkdirTemp("/tmp", "keyring-cli-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(socketDir) })
	t.Setenv("TMPDIR", socketDir)
	socket := filepath.Join(socketDir, "owner.sock")
	t.Setenv("VROOLI_AUTH_SOCKET", socket)
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	path, exchange := accountsconnect.NewAccountsServiceHandler(fixture)
	mux := http.NewServeMux()
	mux.Handle(path, exchange)
	server := &http.Server{Handler: mux, ReadHeaderTimeout: time.Second}
	done := make(chan struct{})
	go func() { defer close(done); _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close(); <-done })
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			fmt.Fprint(w, "{\"status\":\"healthy\"}")
			return
		}
		handler.ServeHTTP(w, r)
	}))
	t.Cleanup(api.Close)
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	return app, api.URL, home
}

func exactSetupPublicJSON(t *testing.T, action string) []byte {
	t.Helper()
	req := exactSetupRequest{Action: action, Grant: "fixture-grant", ExpectedEpoch: 1, PolicyDigest: strings.Repeat("0", 64)}
	if action == "stage" {
		req.Grant, req.PolicyDigest, req.ExpectedEpoch = "", "", 0
		req.Policy = &exactSetupPolicy{
			ID: "fixture-grant", Owner: "fixture-owner", Instance: "fixture-instance", Issuer: "fixture-issuer",
			KeyVersion: 1, Epoch: 1, Deadline: time.Now().Add(time.Hour).Unix(),
			EffortRef: "effort:fixture", AuthorizedBy: "fixture-owner", AuthorityRef: "fixture-approval",
			Profile: "disposable-thin-fixture-v1", PublicKey: make([]byte, 32), Operations: []string{"effort.reconcile"},
		}
	}
	if action == "activate" {
		req.KeyProof = base64.RawURLEncoding.EncodeToString(make([]byte, 64))
	}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func serveExactSetupBroker(t *testing.T, mode string) {
	t.Helper()
	path := cliutil.AgentCredentialSocket()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if mode == "broker-unavailable" {
		return
	}
	listener, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{ReadHeaderTimeout: time.Second, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if mode == "broker-refused" {
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprint(w, "{\"code\":\"synthetic-refusal\"}")
			return
		}
		fmt.Fprint(w, "{\"token\":\"synthetic-broker-agent\"}")
	})}
	done := make(chan struct{})
	go func() { defer close(done); _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close(); <-done })
}

func exactSetupOwnerArgs(t *testing.T, action string) []string {
	t.Helper()
	if action == "status" {
		return []string{action, "--grant", "fixture-grant", "--local-owner"}
	}
	file := filepath.Join(t.TempDir(), "public.json")
	if err := os.WriteFile(file, exactSetupPublicJSON(t, action), 0600); err != nil {
		t.Fatal(err)
	}
	return []string{action, "--request-file", file, "--local-owner"}
}

func runExactSetupOwner(t *testing.T, app *App, base string, args []string) (string, error) {
	t.Helper()
	var err error
	output := captureStdout(t, func() error {
		err = app.Run(append([]string{"--api-base", base, "effort", "exact-setup"}, args...))
		return nil
	})
	if strings.Contains(output, "synthetic-owner-access") || strings.Contains(output, "synthetic-unused-refresh") || err != nil && strings.Contains(err.Error(), "synthetic-owner-access") {
		t.Fatal("owner credential escaped into output")
	}
	return output, err
}

func TestExactSetupLocalOwnerTopLevelTransport(t *testing.T) {
	for _, action := range []string{"stage", "activate", "revoke", "status"} {
		t.Run(action, func(t *testing.T) {
			fixture := &exactSetupOwnerFixture{}
			var calls atomic.Int32
			app, base, home := newExactSetupOwnerApp(t, fixture, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Path != "/api/v1/keyring/authority/setup" || r.Header.Get("Authorization") != "Bearer synthetic-owner-access" || r.Header.Get("X-Agent-Identity-Token") != "" || r.Header.Get(cliutil.HeaderInvocationCommand) != "effort exact-setup" {
					t.Error("owner setup lost request-local authentication or provenance")
				}
				if action == "status" {
					if r.Method != "GET" || r.URL.Query().Get("grant") != "fixture-grant" {
						t.Error("status contract changed")
					}
				} else {
					var input struct{ Action string }
					if r.Method != "POST" || json.NewDecoder(r.Body).Decode(&input) != nil || input.Action != action {
						t.Error("typed setup action changed")
					}
				}
				fmt.Fprint(w, "{\"grant\":\"fixture-grant\",\"state\":\"fixture\"}")
			}))
			output, err := runExactSetupOwner(t, app, base, exactSetupOwnerArgs(t, action))
			if err != nil || fixture.calls.Load() != 1 || calls.Load() != 1 || !strings.Contains(output, "fixture-grant") {
				t.Fatal("ordinary setup route did not use one explicit exchange and one request", err)
			}
			if err := filepath.WalkDir(home, func(path string, entry os.DirEntry, err error) error {
				if err != nil || entry.IsDir() {
					return err
				}
				data, err := os.ReadFile(path)
				if bytes.Contains(data, []byte("synthetic-owner-access")) || bytes.Contains(data, []byte("synthetic-unused-refresh")) {
					t.Error("owner credential persisted")
				}
				return err
			}); err != nil {
				t.Fatal(err)
			}
			if app.core.Config.Token == "synthetic-owner-access" {
				t.Fatal("request-local exchange changed ordinary configuration")
			}
		})
	}
}

func TestExactSetupLocalOwnerTopLevelRefusals(t *testing.T) {
	for _, mode := range []string{"identified", "broker-identified", "broker-refused", "broker-unavailable", "permission-refused", "missing-access", "api-refused"} {
		t.Run(mode, func(t *testing.T) {
			fixture := &exactSetupOwnerFixture{empty: mode == "missing-access"}
			if mode == "permission-refused" {
				fixture.refusal = connect.CodePermissionDenied
			}
			var calls atomic.Int32
			app, base, _ := newExactSetupOwnerApp(t, fixture, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				http.Error(w, "synthetic scope refusal", http.StatusForbidden)
			}))
			if mode == "identified" {
				t.Setenv(cliutil.EnvIdentityToken, "synthetic-agent-leaf")
			}
			if strings.HasPrefix(mode, "broker-") {
				serveExactSetupBroker(t, mode)
			}
			output, err := runExactSetupOwner(t, app, base, exactSetupOwnerArgs(t, "stage"))
			wantExchange, wantCalls := int32(1), int32(0)
			if mode == "identified" || strings.HasPrefix(mode, "broker-") {
				wantExchange = 0
			}
			if mode == "api-refused" {
				wantCalls = 1
			}
			if err == nil || fixture.calls.Load() != wantExchange || calls.Load() != wantCalls || strings.Contains(output, "fixture-grant") {
				t.Fatal("refusal issued, fell back, retried or confirmed setup", mode, err)
			}
		})
	}
}

func TestExactSetupLocalOwnerValidatesTopLevelBeforeExchange(t *testing.T) {
	fixture := &exactSetupOwnerFixture{}
	var calls atomic.Int32
	app, base, _ := newExactSetupOwnerApp(t, fixture, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1) }))
	public := filepath.Join(t.TempDir(), "public.json")
	for _, input := range []string{"{", "{\"action\":\"activate\"}", "{\"action\":\"stage\",\"unknown\":true}", "{\"action\":\"stage\",\"policy\":\"invalid\"}", "{\"action\":\"stage\",\"policy\":{\"unknown\":true}}", "{\"action\":\"stage\"}", strings.Repeat("x", 16001)} {
		if err := os.WriteFile(public, []byte(input), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := runExactSetupOwner(t, app, base, []string{"stage", "--request-file", public, "--local-owner"}); err == nil {
			t.Fatal("malformed public request accepted")
		}
	}
	for _, args := range [][]string{
		{"unsupported", "--request-file", public, "--local-owner"},
		{"stage", "--local-owner"},
		{"status", "--local-owner"},
		{"status", "--grant", "fixture-grant", "--request-file", public, "--local-owner"},
		{"stage", "--request-file", public, "--grant", "fixture-grant", "--local-owner"},
		{"stage", "--request-file", public, "--unknown-flag", "--local-owner"},
	} {
		if _, err := runExactSetupOwner(t, app, base, args); err == nil {
			t.Fatal("unsupported arguments accepted")
		}
	}
	if fixture.calls.Load() != 0 || calls.Load() != 0 {
		t.Fatal("invalid input reached exchange or native setup")
	}
}

func TestExactSetupLocalOwnerTopLevelRefusesRedirects(t *testing.T) {
	for _, action := range []string{"stage", "activate", "revoke", "status"} {
		for _, changedHost := range []bool{false, true} {
			for _, code := range []int{301, 302, 303, 307, 308} {
				t.Run(fmt.Sprintf("%s/changed-host=%t/%d", action, changedHost, code), func(t *testing.T) {
					var destinationCalls atomic.Int32
					dest := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						destinationCalls.Add(1)
						t.Error("redirect destination received a request")
					}))
					defer dest.Close()
					location := dest.URL
					if changedHost {
						location = strings.Replace(location, "127.0.0.1", "localhost", 1)
					}
					fixture := &exactSetupOwnerFixture{}
					var originalCalls atomic.Int32
					app, base, _ := newExactSetupOwnerApp(t, fixture, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						originalCalls.Add(1)
						if r.Header.Get("Authorization") != "Bearer synthetic-owner-access" {
							t.Error("original request lost owner authentication")
						}
						w.Header().Set("Location", location)
						w.WriteHeader(code)
						fmt.Fprint(w, "{\"grant\":\"must-not-confirm\"}")
					}))
					output, err := runExactSetupOwner(t, app, base, exactSetupOwnerArgs(t, action))
					if err == nil || fixture.calls.Load() != 1 || originalCalls.Load() != 1 || destinationCalls.Load() != 0 || strings.Contains(output, "must-not-confirm") {
						t.Fatal("redirect leaked, retried or confirmed setup")
					}
					if !strings.Contains(err.Error(), "original grant status") || action != "status" && !strings.Contains(err.Error(), "mutation may have committed") {
						t.Fatal("redirect lost uncertain original outcome")
					}
				})
			}
		}
	}
}

func TestExactSetupLocalOwnerTopLevelRefusesUnfollowable3xx(t *testing.T) {
	for _, action := range []string{"stage", "activate", "revoke", "status"} {
		for _, code := range []int{300, 301, 302, 303, 304, 305, 306, 307, 308, 399} {
			for _, emptyLocation := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%d/empty-location=%t", action, code, emptyLocation), func(t *testing.T) {
					fixture := &exactSetupOwnerFixture{}
					var calls atomic.Int32
					app, base, _ := newExactSetupOwnerApp(t, fixture, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						calls.Add(1)
						if r.URL.Path != "/api/v1/keyring/authority/setup" || r.Header.Get("Authorization") != "Bearer synthetic-owner-access" {
							t.Error("unfollowable response test lost original owner request")
						}
						if emptyLocation {
							w.Header().Set("Location", "")
						}
						w.WriteHeader(code)
						fmt.Fprint(w, "{\"grant\":\"must-not-confirm\"}")
					}))
					output, err := runExactSetupOwner(t, app, base, exactSetupOwnerArgs(t, action))
					if err == nil || fixture.calls.Load() != 1 || calls.Load() != 1 || strings.Contains(output, "must-not-confirm") {
						t.Fatal("unfollowable3xx confirmed setup or retried")
					}
					if !strings.Contains(err.Error(), "original grant status") || action != "status" && !strings.Contains(err.Error(), "mutation may have committed") {
						t.Fatal("unfollowable3xx lost original mutation uncertainty")
					}
				})
			}
		}
	}
}
