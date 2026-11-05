package target

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	inferenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference/inference_v1connect"
)

func startJ12Bundle(t *testing.T) (*ElectronSession, string) {
	t.Helper()
	artifact := os.Getenv("J12_APPIMAGE")
	if artifact == "" {
		t.Skip("J12_APPIMAGE selects the real bundled Web Console artifact for this operated journey")
	}
	t.Setenv("J9_APPIMAGE", artifact)
	return startJ9Bundle(t)
}

func TestJ12_GatewayStartsWithNoRepoOrResourceCLIs(t *testing.T) {
	_, home := startJ12Bundle(t)
	port := runtimePort(t, home, "ai-gateway")
	client := inferenceconnect.NewInferenceServiceClient(http.DefaultClient, fmt.Sprintf("http://127.0.0.1:%d", port))
	request := connect.NewRequest(&inferencev1.RunRequest{Source: "Answer ok", Instruction: "Return a JSON object containing answer ok", SchemaJson: `{"type":"object","properties":{"answer":{"type":"string"}},"required":["answer"],"additionalProperties":false}`, Role: "extract.structured", MaxOutputTokens: 32})
	request.Header().Set("X-Vrooli-AI-Source", "local")
	response, err := client.Run(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.GetError() != nil {
		t.Fatalf("real embedded Gateway failed: %v", response.Msg.GetError())
	}
	var result map[string]string
	if err = json.Unmarshal([]byte(response.Msg.GetValueJson()), &result); err != nil || result["answer"] != "ok" {
		t.Fatalf("real model output %s: %v", response.Msg.GetValueJson(), err)
	}
	// startJ9Bundle isolates HOME, clears VROOLI_ROOT and permits only system
	// PATH entries. Check the artifact's real process environment as well.
	entries, err := os.ReadDir("/proc")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	configuredDoor := false
	for _, entry := range entries {
		if _, err := strconv.Atoi(entry.Name()); err != nil {
			continue
		}
		data, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "environ"))
		if err != nil || !bytes.Contains(data, []byte("HOME="+home+"\x00")) {
			continue
		}
		cmd, _ := os.ReadFile(filepath.Join("/proc", entry.Name(), "cmdline"))
		if !bytes.Contains(cmd, []byte("ai-gateway")) {
			continue
		}
		found = true
		for _, env := range bytes.Split(data, []byte{0}) {
			if string(env) == "LPBS_AI_DOOR_URL=https://vrooli.com" {
				configuredDoor = true
			}
			if bytes.HasPrefix(env, []byte("PATH=")) {
				for _, dir := range strings.Split(strings.TrimPrefix(string(env), "PATH="), ":") {
					for _, name := range []string{"resource-ollama", "resource-openrouter", "resource-orcarouter"} {
						if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
							t.Fatalf("bundled Gateway PATH exposes %s", name)
						}
					}
				}
			}
			if bytes.HasPrefix(env, []byte("VROOLI_ROOT=")) && string(env) != "VROOLI_ROOT=" {
				t.Fatalf("bundled Gateway has repository root")
			}
		}
	}
	if !configuredDoor {
		t.Fatal("downloaded Gateway did not receive its configured public AI door")
	}
	if !found {
		t.Fatal("real embedded Gateway process was not found in isolated HOME")
	}
	t.Log("real AppImage Gateway answered through real Ollama with no repository or resource CLIs")
}

type fullGateway struct {
	base string
	pid  int
}

func startFullGateway(t *testing.T) fullGateway {
	t.Helper()
	dir, err := filepath.Abs("../../../ai-gateway/api")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "test", "./handlers/inference", "-run", "^TestJourneyEmbeddedGatewayServer$", "-count=1", "-v", "-timeout", "10m")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off", "VROOLI_JOURNEY_TYPED_ONLY=1", "VROOLI_JOURNEY_CREDENTIALS="+filepath.Join(t.TempDir(), "authority.enc"))
	output, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	result := fullGateway{}
	t.Cleanup(func() {
		if result.base != "" {
			resp, err := http.Post(result.base+"/__journey/stop", "text/plain", nil)
			if err == nil {
				resp.Body.Close()
			}
		}
		if err := cmd.Wait(); err != nil {
			t.Errorf("real full Gateway helper: %v: %s", err, stderr.String())
		}
	})
	scanner := bufio.NewScanner(output)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "EMBEDDED_GATEWAY_URL=") {
			result.base = strings.TrimPrefix(line, "EMBEDDED_GATEWAY_URL=")
		}
		if strings.HasPrefix(line, "EMBEDDED_GATEWAY_PID=") {
			result.pid, _ = strconv.Atoi(strings.TrimPrefix(line, "EMBEDDED_GATEWAY_PID="))
		}
		if result.base != "" && result.pid > 0 {
			go io.Copy(io.Discard, output)
			return result
		}
	}
	t.Fatalf("full Gateway did not start: %s", stderr.String())
	return result
}

func TestJ12_FullServerInRangeWins(t *testing.T) {
	_, home := startJ12Bundle(t)
	embedded := runtimePort(t, home, "ai-gateway")
	full := startFullGateway(t)
	u, err := url.Parse(full.base)
	if err != nil {
		t.Fatal(err)
	}
	port, _ := strconv.Atoi(u.Port())
	// This is a full-server peer identity fixture, consumed by the runtime's
	// production peer codec and range evaluator. Its endpoint is a real Gateway.
	record := map[string]any{"schema_version": 1, "scenario": "ai-gateway", "instance": "", "tier": 1, "version": "0.0.1", "provides": []string{"api"}, "owner_pid": full.pid, "started_at": time.Now().UTC(), "ports": map[string]int{"api": port}}
	peerDir := filepath.Join(home, ".vrooli", "peers")
	if err = os.MkdirAll(peerDir, 0o700); err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(record)
	if err = os.WriteFile(filepath.Join(peerDir, "ai-gateway.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	if got := runtimePort(t, home, "ai-gateway"); got != port || got == embedded {
		t.Fatalf("runtime chose %d; full=%d embedded=%d", got, port, embedded)
	}
	web := runtimePort(t, home, "web-console")
	response, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d/api/v1/ai/generate", web), "application/json", strings.NewReader(`{"prompt":"Return the shell command printf ok"}`))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode != 200 {
		t.Fatalf("real Web Console AI request %d: %s", response.StatusCode, body)
	}
	response, err = http.Get(full.base + "/__journey/evidence")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var evidence struct {
		Calls int `json:"calls"`
	}
	if err = json.NewDecoder(response.Body).Decode(&evidence); err != nil {
		t.Fatal(err)
	}
	if evidence.Calls != 1 {
		t.Fatalf("Web Console used full Gateway %d times, want once", evidence.Calls)
	}
	if err = os.Remove(filepath.Join(peerDir, "ai-gateway.json")); err != nil {
		t.Fatal(err)
	}
	if got := runtimePort(t, home, "ai-gateway"); got != embedded {
		t.Fatalf("runtime did not return to embedded Gateway: %d", got)
	}
	t.Log("real consumer used full Gateway while present and runtime returned to embedded Gateway after peer removal")
}

// The only network substitute is the upstream router. TLS, proxying, the
// native encrypted Secret Service and the launched Gateway remain real.
func TestJ12_OwnKeyWorksInLaunchedBundle(t *testing.T) {
	if os.Getenv("J12_APPIMAGE") == "" {
		t.Skip("J12_APPIMAGE selects the real downloaded artifact")
	}
	t.Setenv("VROOLI_CREDENTIAL_BACKEND", "native")
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(os.Getenv("HOME"), ".config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(os.Getenv("HOME"), ".local", "share"))
	runtimeDir := t.TempDir()
	if err := os.Chmod(runtimeDir, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_RUNTIME_DIR", runtimeDir)
	bus := exec.Command("dbus-daemon", "--session", "--nofork", "--print-address=1", "--address=unix:path="+filepath.Join(runtimeDir, "bus"))
	output, err := bus.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var busErrors bytes.Buffer
	bus.Stderr = &busErrors
	if err = bus.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = bus.Process.Kill(); _ = bus.Wait() })
	scanner := bufio.NewScanner(output)
	if !scanner.Scan() {
		t.Fatalf("isolated Secret Service bus: %s", busErrors.String())
	}
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", scanner.Text())
	go io.Copy(io.Discard, output)

	key := "isolated-J12-own-key-never-live"
	certKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "isolated J12 router"}, DNSNames: []string{"openrouter.ai"}, NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &certKey.PublicKey, certKey)
	if err != nil {
		t.Fatal(err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	cert, err := tls.X509KeyPair(certPEM, pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(certKey)}))
	if err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	router := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != "openrouter.ai" || r.URL.Path != "/api/v1/chat/completions" {
			t.Errorf("unexpected router target %s%s", r.Host, r.URL.Path)
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer "+key {
			t.Error("launched Gateway did not resolve the native own-key credential")
			w.WriteHeader(401)
			return
		}
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[{"message":{"content":"{\"answer\":\"ok\"}"}}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`)
	}))
	router.TLS = &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}
	router.StartTLS()
	t.Cleanup(router.Close)
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodConnect || r.Host != "openrouter.ai:443" {
			http.Error(w, "unexpected proxy destination", 403)
			return
		}
		upstream, err := net.Dial("tcp", router.Listener.Addr().String())
		if err != nil {
			t.Error(err)
			w.WriteHeader(502)
			return
		}
		downstream, buffer, err := w.(http.Hijacker).Hijack()
		if err != nil {
			upstream.Close()
			t.Error(err)
			return
		}
		fmt.Fprint(buffer, "HTTP/1.1 200 Connection Established\r\n\r\n")
		buffer.Flush()
		go func() { defer downstream.Close(); defer upstream.Close(); io.Copy(upstream, buffer) }()
		go func() { defer downstream.Close(); defer upstream.Close(); io.Copy(downstream, upstream) }()
	}))
	t.Cleanup(proxy.Close)
	trust := filepath.Join(t.TempDir(), "router-ca.pem")
	if err := os.WriteFile(trust, certPEM, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SSL_CERT_FILE", trust)
	t.Setenv("HTTPS_PROXY", proxy.URL)
	t.Setenv("NO_PROXY", "localhost,127.0.0.1")

	// The real encrypted native service is ready before app launch, so no
	// auto-activated locked keyring can race fixture initialization.
	keyringHome := os.Getenv("HOME")
	daemon := exec.Command("gnome-keyring-daemon", "--foreground", "--components=secrets", "--unlock")
	daemon.Stdin = strings.NewReader("isolated encrypted J12 keyring passphrase\n")
	var daemonErrors bytes.Buffer
	daemon.Stderr = &daemonErrors
	if err := daemon.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = daemon.Process.Kill(); _ = daemon.Wait() })
	readyCtx, readyCancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer readyCancel()
	if output, err := exec.CommandContext(readyCtx, "gdbus", "wait", "--session", "--timeout", "10", "org.freedesktop.secrets").CombinedOutput(); err != nil {
		t.Fatalf("isolated Secret Service readiness: %v: %s; %s", err, output, daemonErrors.String())
	}
	authority, err := credentialauthority.Default()
	if err != nil {
		t.Fatal(err)
	}
	identity, err := credentialauthority.ParseIdentity("vrooli/openrouter")
	if err != nil {
		t.Fatal(err)
	}
	// Provision through the real Authority after the native session is ready.
	if err := authority.Put(identity, "api-key", key); err != nil {
		t.Fatalf("isolated native credential provisioning: %v; %s", err, daemonErrors.String())
	}
	t.Cleanup(func() { _ = authority.Delete(identity, "api-key") })
	report, err := authority.KeyringInspect(filepath.Join(keyringHome, ".local", "share", "keyrings", "login.keyring"))
	if err != nil || report.Format != "encrypted" {
		t.Fatalf("native fixture is not an encrypted keyring: format=%s err=%v", report.Format, err)
	}
	_, home := startJ12Bundle(t)
	port := runtimePort(t, home, "ai-gateway")
	client := inferenceconnect.NewInferenceServiceClient(http.DefaultClient, fmt.Sprintf("http://127.0.0.1:%d", port))
	req := connect.NewRequest(&inferencev1.RunRequest{Source: "Return ok", Instruction: "Return answer ok", SchemaJson: `{"type":"object","properties":{"answer":{"type":"string"}},"required":["answer"],"additionalProperties":false}`, Role: "extract.structured", MaxOutputTokens: 32})
	req.Header().Set("X-Vrooli-AI-Source", "own-key")
	ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
	defer cancel()
	response, err := client.Run(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.GetError() != nil {
		t.Fatalf("launched own-key inference: %v", response.Msg.GetError())
	}
	if response.Msg.GetValueJson() != `{"answer":"ok"}` || calls.Load() != 1 {
		t.Fatalf("own-key result=%s router calls=%d", response.Msg.GetValueJson(), calls.Load())
	}
	t.Log("actual AppImage used its native encrypted credential authority and canonical OpenRouter HTTPS endpoint; only upstream router was fake")
}
