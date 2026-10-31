//go:build linux

package target

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestOfflineElectronSessionBlocksExternalNetworkAndBridgesLoopbackAcrossRestart(t *testing.T) {
	if _, err := exec.LookPath("bwrap"); err != nil {
		t.Skip("bubblewrap is unavailable")
	}
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 is unavailable")
	}
	artifact := filepath.Join(t.TempDir(), "offline-electron-fixture")
	script := `#!/usr/bin/env python3
import http.server, json, os, signal, socket, sqlite3, sys, tempfile, threading
port = next(int(a.split('=',1)[1]) for a in sys.argv[1:] if a.startswith('--remote-debugging-port='))
renderer_port = int(os.environ['VROOLI_TEST_RENDERER_PORT'])
api_port = int(os.environ['VROOLI_TEST_API_PORT'])
db_root = os.path.join(os.path.dirname(os.environ['S2D_BUNDLED_TARGET_PATH']), 'test-database')
db_path = os.path.join(db_root, 'namespace-write.sqlite')
db = sqlite3.connect(db_path); db.execute('create table if not exists probe (value text)'); db.execute('insert into probe values (?)', ('written-in-private-network-namespace',)); db.commit(); db.close()
with open(os.environ['S2D_BUNDLED_TARGET_PATH'],'w') as f:
 json.dump({'renderer_url':'http://127.0.0.1:%d/'%renderer_port,'api_url':'http://127.0.0.1:%d'%api_port},f)
probe = os.path.join(os.environ['HOME'], 'network-probe.txt')
external = socket.socket(); external.settimeout(1); external_result = external.connect_ex(('192.0.2.1', 443)); external.close()
inner = socket.socket(); inner.bind(('127.0.0.1', 0)); inner.listen(1)
def serve_inner():
 c,_=inner.accept(); c.recv(256); c.sendall(b'HTTP/1.0 200 OK\r\nContent-Length: 2\r\n\r\nok'); c.close()
threading.Thread(target=serve_inner,daemon=True).start()
def internal_probe():
 c=socket.create_connection(('127.0.0.1', inner.getsockname()[1])); c.sendall(b'GET / HTTP/1.0\r\nHost: localhost\r\n\r\n'); result=c.recv(256).decode('latin1'); c.close(); return '200 OK' in result
class Handler(http.server.BaseHTTPRequestHandler):
 def do_GET(self):
  if self.path == '/json/list':
   body=json.dumps([{'id':'main','type':'page','url':'http://127.0.0.1:%d/'%renderer_port,'webSocketDebuggerUrl':'ws://127.0.0.1:%d/devtools/page/main'%(port)}]).encode()
   self.send_response(200); self.send_header('Content-Length',str(len(body))); self.end_headers(); self.wfile.write(body)
  elif self.path == '/local':
   body=b'host-cdp-bridge-ok'; self.send_response(200); self.send_header('Content-Length',str(len(body))); self.end_headers(); self.wfile.write(body)
  elif self.path == '/quit':
   self.send_response(200); self.end_headers(); threading.Thread(target=server.shutdown,daemon=True).start()
  else: self.send_error(404)
 def log_message(self,*args): pass
server=http.server.HTTPServer(('127.0.0.1',port),Handler)
class APIHandler(http.server.BaseHTTPRequestHandler):
 def do_GET(self):
  if self.path != '/health': self.send_error(404); return
  body=b'{"status":"ok"}'; self.send_response(200); self.send_header('Content-Length',str(len(body))); self.end_headers(); self.wfile.write(body)
 def do_POST(self):
  file_root = tempfile.mkdtemp(prefix='workflow-test-roots-')
  os.makedirs(os.path.join(file_root, 'data'))
  with open(os.path.join(file_root, 'data', 'routed-write.txt'), 'w') as probe: probe.write('private-file-root-write')
  with open(os.path.join(os.environ['HOME'], 'file-root-probe.json'), 'w') as probe: json.dump({'root':file_root}, probe)
  with open(os.path.join(os.environ['HOME'], 'api-probe.txt'),'a') as f: f.write(self.path+' '+self.headers.get('X-Vrooli-Test-Lease-ID','')+'\n')
  body=json.dumps({'active_lease_id':'offline-test-lease'}).encode()
  self.send_response(200); self.send_header('Content-Length',str(len(body))); self.end_headers(); self.wfile.write(body)
 def log_message(self,*args): pass
api_server=http.server.HTTPServer(('127.0.0.1',api_port),APIHandler)
threading.Thread(target=api_server.serve_forever,daemon=True).start()
class RendererHandler(http.server.BaseHTTPRequestHandler):
 def do_GET(self):
  body=b'offline-renderer-bridge-ok'; self.send_response(200); self.send_header('Content-Length',str(len(body))); self.end_headers(); self.wfile.write(body)
 def log_message(self,*args): pass
renderer_server=http.server.HTTPServer(('127.0.0.1',renderer_port),RendererHandler)
threading.Thread(target=renderer_server.serve_forever,daemon=True).start()
with open(probe,'a') as f: f.write('%d %s\n'%(external_result,internal_probe()))
server.serve_forever()
`
	if err := os.WriteFile(artifact, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	options := sessionLaunchOptions()
	options.ProfileID = "offline"
	rendererPort, err := allocateLoopbackPort()
	if err != nil {
		t.Fatal(err)
	}
	apiPort, err := allocateLoopbackPort()
	if err != nil {
		t.Fatal(err)
	}
	rendererURL := fmt.Sprintf("http://127.0.0.1:%d", rendererPort)
	apiURL := fmt.Sprintf("http://127.0.0.1:%d", apiPort)
	options.BundledRuntime = true
	start := func() (*ElectronSession, error) {
		return StartElectronSession(context.Background(), ElectronSessionOptions{
			ArtifactPath:   artifact,
			Launch:         options,
			Renderer:       RendererExpectation{URLPrefix: rendererURL},
			OfflineNetwork: true,
			StartupTimeout: 5 * time.Second,
			Environment:    map[string]string{"VROOLI_TEST_RENDERER_PORT": fmt.Sprint(rendererPort), "VROOLI_TEST_API_PORT": fmt.Sprint(apiPort)},
		})
	}
	session, err := start()
	if err != nil {
		t.Fatalf("start isolated Electron fixture: %v", err)
	}
	root := session.networkRoot
	dbPath := filepath.Join(root, "test-database", "namespace-write.sqlite")
	if session.Target().GetTestDatabaseRoot() != filepath.Join(root, "test-database") {
		t.Fatalf("target database root = %q, want private session path", session.Target().GetTestDatabaseRoot())
	}
	if data, err := os.ReadFile(dbPath); err != nil || len(data) == 0 {
		t.Fatalf("SQLite database written inside the private namespace is not visible to the host cleanup owner: bytes=%d err=%v", len(data), err)
	}
	rootInfo, err := os.Stat(root)
	if err != nil || rootInfo.Mode().Perm() != 0o700 {
		t.Fatalf("private offline root mode = %v, err=%v", rootInfo, err)
	}
	socketInfo, err := os.Stat(filepath.Join(root, "cdp.sock"))
	if err != nil || socketInfo.Mode().Perm() != 0o600 {
		t.Fatalf("private CDP socket mode = %v, err=%v", socketInfo, err)
	}
	rendererSocketInfo, err := os.Stat(filepath.Join(root, "renderer.sock"))
	if err != nil || rendererSocketInfo.Mode().Perm() != 0o600 {
		t.Fatalf("private renderer socket mode = %v, err=%v", rendererSocketInfo, err)
	}
	endpoint, err := url.Parse(session.Endpoint())
	if err != nil || endpoint.Hostname() != "127.0.0.1" || session.profileDir != filepath.Join(root, "profile") {
		t.Fatalf("host endpoint/profile escaped the private loopback boundary: endpoint=%q profile=%q err=%v", session.Endpoint(), session.profileDir, err)
	}
	checkProbe := func() {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(session.profileDir, "network-probe.txt"))
		if err != nil {
			t.Fatalf("read network probe: %v", err)
		}
		lines := strings.Fields(string(data))
		if len(lines) < 2 || lines[len(lines)-2] == "0" || lines[len(lines)-1] != "True" {
			t.Fatalf("isolated and internal connectivity probe = %q, expected external denial and positive inner loopback", data)
		}
	}
	checkBridge := func(endpoint string) {
		t.Helper()
		response, err := http.Get(endpoint + "/local")
		if err != nil {
			t.Fatalf("host CDP bridge request: %v", err)
		}
		body, readErr := io.ReadAll(response.Body)
		_ = response.Body.Close()
		if readErr != nil || response.StatusCode != http.StatusOK || string(body) != "host-cdp-bridge-ok" {
			t.Fatalf("bridged response = status %d body %q readErr=%v", response.StatusCode, body, readErr)
		}
	}
	checkAPIAndRoutingBridge := func(session *ElectronSession) {
		t.Helper()
		if strings.HasPrefix(session.Target().GetApiUrl(), apiURL) {
			t.Fatalf("host API URL unexpectedly uses private bundled port: %q", session.Target().GetApiUrl())
		}
		if session.Target().GetRendererUrl() != rendererURL+"/" {
			t.Fatalf("target renderer URL = %q, want exact inner renderer URL %q", session.Target().GetRendererUrl(), rendererURL+"/")
		}
		listener, err := net.Listen("tcp4", fmt.Sprintf("127.0.0.1:%d", rendererPort))
		if err != nil {
			t.Fatalf("target unexpectedly published a host renderer bridge: %v", err)
		}
		_ = listener.Close()
		request, err := http.NewRequest(http.MethodPost, session.Target().GetApiUrl()+"/api/v1/routing/install", strings.NewReader(`{"lease_id":"offline-test-lease"}`))
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("X-Vrooli-Test-Mode", "1")
		request.Header.Set("X-Vrooli-Test-Lease-ID", "offline-test-lease")
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatalf("host-to-offline-bundled-API routing request: %v", err)
		}
		defer response.Body.Close()
		var payload map[string]string
		if err := json.NewDecoder(response.Body).Decode(&payload); err != nil || response.StatusCode != http.StatusOK || payload["active_lease_id"] != "offline-test-lease" {
			t.Fatalf("bundled API routing response status=%d payload=%v err=%v", response.StatusCode, payload, err)
		}
		if session.Target().GetApiUrl() == "" {
			t.Fatal("target API endpoint is empty")
		}
		fileRootEvidence, err := os.ReadFile(filepath.Join(session.profileDir, "file-root-probe.json"))
		if err != nil {
			t.Fatalf("read target file-root probe: %v", err)
		}
		var fileRoot struct {
			Root string `json:"root"`
		}
		if err := json.Unmarshal(fileRootEvidence, &fileRoot); err != nil || !pathWithin(session.networkRoot, fileRoot.Root) {
			t.Fatalf("target file root escaped writable session tmp root: root=%q evidence=%s err=%v", fileRoot.Root, fileRootEvidence, err)
		}
		if data, err := os.ReadFile(filepath.Join(fileRoot.Root, "data", "routed-write.txt")); err != nil || string(data) != "private-file-root-write" {
			t.Fatalf("target file routing write was not visible to owner cleanup: data=%q err=%v", data, err)
		}
	}
	checkProbe()
	checkBridge(session.Endpoint())
	checkAPIAndRoutingBridge(session)
	firstEndpoint := session.Endpoint()
	quitRequest, _ := http.NewRequest(http.MethodGet, firstEndpoint+"/quit", nil)
	_, _ = http.DefaultClient.Do(quitRequest)
	waitCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	err = session.WaitForExit(waitCtx)
	cancel()
	if err != nil {
		t.Fatalf("wait for first offline process: %v", err)
	}
	if _, err := http.Post(session.Target().GetApiUrl()+"/api/v1/routing/install", "application/json", strings.NewReader(`{}`)); err == nil {
		t.Fatal("API bridge remained usable after its isolated service exited")
	}
	restarted, err := session.Restart(context.Background())
	if err != nil {
		t.Fatalf("restart offline Electron session: %v", err)
	}
	if restarted.networkRoot != root || restarted.Endpoint() == firstEndpoint {
		t.Fatalf("restart did not retain private root with fresh bridge: root=%q endpoint=%q prior=%q", restarted.networkRoot, restarted.Endpoint(), firstEndpoint)
	}
	checkProbe()
	checkBridge(restarted.Endpoint())
	checkAPIAndRoutingBridge(restarted)
	if err := restarted.Close(context.Background()); err != nil {
		t.Fatalf("close restarted offline session: %v", err)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("private offline root remains after close: %q err=%v", root, err)
	}
	if _, err := http.Get(fmt.Sprintf("%s/local", firstEndpoint)); err == nil {
		t.Fatal("prior CDP bridge remained open after restart")
	}
	if _, err := os.Stat(dbPath); !os.IsNotExist(err) {
		t.Fatalf("private namespace SQLite database remains after owner cleanup: %q err=%v", dbPath, err)
	}
}

func TestOfflineAPIBridgeUsesEphemeralHostPortWhenTargetPortIsOccupied(t *testing.T) {
	occupied, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer occupied.Close()
	port := occupied.Addr().(*net.TCPAddr).Port
	root, err := os.MkdirTemp("", "s2d-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)
	bridge, err := startUnixPortBridge(filepath.Join(root, "api.sock"), 0, port)
	if err != nil {
		t.Fatalf("API bridge failed despite free ephemeral host port: %v", err)
	}
	defer bridge.Close()
	if bridge.Port() == port {
		t.Fatalf("API bridge reused occupied target port %d as host port", port)
	}
	if bridge.targetPort != port {
		t.Fatalf("API bridge relay target port = %d, want bundled private port %d", bridge.targetPort, port)
	}
}

func TestOfflineBundledAPIEndpointRequiresMatchingRendererAndPrivateOrigin(t *testing.T) {
	root := t.TempDir()
	descriptor := filepath.Join(root, "bundled-target.json")
	write := func(rendererURL, apiURL string) {
		t.Helper()
		data, err := json.Marshal(map[string]string{"renderer_url": rendererURL, "api_url": apiURL})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(descriptor, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("http://127.0.0.1:41001/", "http://127.0.0.1:41002")
	port, err := offlineBundledAPIPort(descriptor, root, "http://127.0.0.1:41001/projects")
	if err != nil || port != 41002 {
		t.Fatalf("validated bundled API port = %d, err=%v", port, err)
	}
	for _, test := range []struct {
		name       string
		renderer   string
		api        string
		wantReason string
	}{
		{name: "renderer mismatch", renderer: "http://127.0.0.1:41003/", api: "http://127.0.0.1:41002", wantReason: "does not match"},
		{name: "external API", renderer: "http://127.0.0.1:41001/", api: "http://192.0.2.1:41002", wantReason: "loopback"},
		{name: "API path", renderer: "http://127.0.0.1:41001/", api: "http://127.0.0.1:41002/api", wantReason: "bare loopback"},
	} {
		t.Run(test.name, func(t *testing.T) {
			write(test.renderer, test.api)
			_, err := offlineBundledAPIPort(descriptor, root, "http://127.0.0.1:41001/")
			if err == nil || !strings.Contains(err.Error(), test.wantReason) {
				t.Fatalf("descriptor accepted or wrong error: %v", err)
			}
		})
	}
}
