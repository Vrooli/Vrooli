package virtualapi

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func contractFixture() Contract {
	return Contract{TaskID: "a3f3d4c8-08c6-42a3-a5a0-07febb9d904a", Hosts: []string{"api.notion.com", "www.googleapis.com"}, Routes: []Route{
		{Host: "api.notion.com", Method: "GET", Path: "/v1/pages/page-1", Status: 200, ContentType: "application/json", Body: []byte(`{"id":"synthetic-page"}`)},
		{Host: "www.googleapis.com", Method: "POST", Path: "/v4/spreadsheets/sheet-1/values/A1:append", Query: "valueInputOption=RAW", Status: 200, ContentType: "application/json", Body: []byte(`{"updates":{"updatedRows":1}}`)},
	}}
}

func TestVirtualGatewayTLSExactHostRouteAndTaskCleanup(t *testing.T) {
	g, receipt, err := New(contractFixture())
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.NoOutboundDial || receipt.NoEgressVerified || receipt.NetworkPolicy != "task-private-internal-only" || receipt.Status != "ready" || len(receipt.Aliases) != 2 {
		t.Fatalf("bad lifecycle receipt: %+v", receipt)
	}
	sum := sha256.Sum256(receipt.CACertPEM)
	if receipt.CAFingerprint != hex.EncodeToString(sum[:]) {
		t.Fatal("CA receipt fingerprint does not match public trust bundle")
	}
	block, _ := pem.Decode(receipt.CACertPEM)
	if block == nil {
		t.Fatal("receipt has no PEM trust anchor")
	}
	ca, err := x509.ParseCertificate(block.Bytes)
	if err != nil || !ca.IsCA {
		t.Fatalf("invalid task CA: %v", err)
	}
	certPEM, keyPEM, err := g.ServerCertificate()
	if err != nil {
		t.Fatal(err)
	}
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewUnstartedServer(g)
	server.TLS = &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}
	server.StartTLS()
	defer server.Close()
	roots := x509.NewCertPool()
	roots.AddCert(ca)
	client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}, DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "tcp", strings.TrimPrefix(server.URL, "https://"))
	}}}
	defer client.CloseIdleConnections()
	resp, err := client.Get("https://api.notion.com/v1/pages/page-1")
	if err != nil {
		t.Fatalf("valid public hostname TLS request failed: %v", err)
	}
	body := make([]byte, 128)
	n, _ := resp.Body.Read(body)
	resp.Body.Close()
	if resp.StatusCode != 200 || string(body[:n]) != `{"id":"synthetic-page"}` {
		t.Fatalf("fixture response = %d %q", resp.StatusCode, string(body[:n]))
	}
	for _, target := range []struct{ method, path, host string }{
		{http.MethodPost, "/v1/pages/page-1", "api.notion.com"},
		{http.MethodGet, "/v1/pages/page-1/extra", "api.notion.com"},
		{http.MethodGet, "/v1/pages/page-1", "attacker.example"},
	} {
		req, err := http.NewRequest(target.method, "https://api.notion.com"+target.path, nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Host = target.host
		denied, err := client.Do(req)
		if err != nil {
			t.Fatalf("deny request TLS failed: %v", err)
		}
		denied.Body.Close()
		if denied.StatusCode != http.StatusNotFound {
			t.Errorf("%s %s host=%s returned %d", target.method, target.path, target.host, denied.StatusCode)
		}
	}
	hits := g.Hits()
	if len(hits) != 2 {
		t.Fatalf("unexpected hit ledger: %#v", hits)
	}
	for k, n := range hits {
		if strings.Contains(k, "notion") && n != 1 {
			t.Fatalf("valid exact route count = %d", n)
		}
		if strings.Contains(k, "google") && n != 0 {
			t.Fatalf("unrequested route count = %d", n)
		}
	}
	g.Close()
	if _, _, err := g.ServerCertificate(); err == nil {
		t.Fatal("closed task still exposes certificate material")
	}
	if len(g.Hits()) != 0 {
		t.Fatal("closed task retained route evidence")
	}
}

func TestContractRejectsHostPathMethodAndRouteConfusion(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Contract)
	}{
		{"undeclared external route", func(c *Contract) { c.Routes[0].Host = "api.evil.example" }},
		{"wildcard host", func(c *Contract) { c.Hosts[0] = "*.notion.com" }},
		{"ip host", func(c *Contract) { c.Hosts[0] = "127.0.0.1" }},
		{"path traversal", func(c *Contract) { c.Routes[0].Path = "/v1/../admin" }},
		{"encoded path", func(c *Contract) { c.Routes[0].Path = "/v1/%2e%2e/admin" }},
		{"arbitrary method", func(c *Contract) { c.Routes[0].Method = "CONNECT" }},
		{"redirect", func(c *Contract) { c.Routes[0].Status = 302 }},
		{"duplicate route", func(c *Contract) { c.Routes = append(c.Routes, c.Routes[0]) }},
		{"credential query key", func(c *Contract) { c.Routes[0].Query = "access_token=synthetic" }},
		{"personal query value", func(c *Contract) { c.Routes[0].Query = "email=person%40gmail.com" }},
		{"excessive query pairs", func(c *Contract) { c.Routes[0].Query = strings.Repeat("a=x&", maxQueryPairs) + "a=y" }},
		{"credential field", func(c *Contract) { c.Routes[0].Body = []byte(`{"access_token":"synthetic-looking"}`) }},
		{"real address", func(c *Contract) { c.Routes[0].Body = []byte(`{"email":"person@example.com"}`) }},
		{"credential-shaped token", func(c *Contract) { c.Routes[0].Body = []byte(`{"note":"Bearer abcdefghijklmnop"}`) }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := contractFixture()
			tc.mutate(&c)
			if _, _, err := New(c); err == nil {
				t.Fatal("unsafe contract accepted")
			}
		})
	}
}

func TestQueryMatchingIsCanonicalAndReceiptsNeverExposeRawValues(t *testing.T) {
	c := Contract{TaskID: "a3f3d4c8-08c6-42a3-a5a0-07febb9d904a", Hosts: []string{"api.notion.com"}, Routes: []Route{{Host: "api.notion.com", Method: "GET", Path: "/v1/pages", Query: "z=last&a=first", Status: 200, ContentType: "application/json", Body: []byte(`{"ok":true}`)}}}
	g, _, err := New(c)
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	request := httptest.NewRequest(http.MethodGet, "/v1/pages?a=first&z=last", nil)
	request.Host = "api.notion.com"
	request.TLS = &tls.ConnectionState{ServerName: "api.notion.com"}
	response := httptest.NewRecorder()
	g.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("canonical query route returned %d", response.Code)
	}
	for identity := range g.Hits() {
		if strings.Contains(identity, "first") || strings.Contains(identity, "last") || !strings.Contains(identity, "?sha256=") {
			t.Fatalf("route receipt exposed or omitted query hash: %q", identity)
		}
	}
}

func TestProfileFingerprintIsStableAcrossHostRouteAndQueryOrder(t *testing.T) {
	a := contractFixture()
	b := contractFixture()
	a.Routes[1].Query = "z=last&a=first"
	b.Hosts[0], b.Hosts[1] = b.Hosts[1], b.Hosts[0]
	b.Routes[0], b.Routes[1] = b.Routes[1], b.Routes[0]
	b.Routes[0].Query = "a=first&z=last"
	_, receiptA, err := New(a)
	if err != nil {
		t.Fatal(err)
	}
	_, receiptB, err := New(b)
	if err != nil {
		t.Fatal(err)
	}
	if receiptA.ProfileFingerprint != receiptB.ProfileFingerprint {
		t.Fatalf("equivalent profiles have different fingerprints: %s != %s", receiptA.ProfileFingerprint, receiptB.ProfileFingerprint)
	}
}

func TestContainerConfigIsBoundedTaskScopedAndOmitsCAPrivateKey(t *testing.T) {
	g, receipt, err := New(contractFixture())
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	configJSON, err := g.ServerConfigJSON()
	if err != nil {
		t.Fatal(err)
	}
	if len(configJSON) == 0 || len(configJSON) > MaxServerConfigBytes {
		t.Fatalf("config size outside bound: %d", len(configJSON))
	}
	var cfg ContainerServerConfig
	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.TaskID != receipt.TaskID || !reflect.DeepEqual(cfg.Hosts, receipt.Aliases) || len(cfg.Routes) != 2 {
		t.Fatalf("staged config identity differs from receipt: %+v", cfg)
	}
	leafBlock, _ := pem.Decode([]byte(cfg.CertificatePEM))
	if leafBlock == nil {
		t.Fatal("leaf cert missing")
	}
	leaf, err := x509.ParseCertificate(leafBlock.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	caBlock, _ := pem.Decode(receipt.CACertPEM)
	if caBlock == nil {
		t.Fatal("CA cert missing")
	}
	ca, err := x509.ParseCertificate(caBlock.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	if err = leaf.CheckSignatureFrom(ca); err != nil {
		t.Fatalf("staged leaf is not signed by receipt CA: %v", err)
	}
	if strings.Contains(cfg.PrivateKeyPEM, "CERTIFICATE") || strings.Contains(cfg.PrivateKeyPEM, "CA PRIVATE KEY") {
		t.Fatal("server config contains non-key or CA private material")
	}
	keyBlock, _ := pem.Decode([]byte(cfg.PrivateKeyPEM))
	if keyBlock == nil {
		t.Fatal("server leaf key missing")
	}
	if keyBlock.Type != "PRIVATE KEY" {
		t.Fatalf("unexpected private key type %q", keyBlock.Type)
	}
	for _, r := range cfg.Routes {
		if len(r.Body) == 0 {
			t.Fatal("expected exact fixture body in staged server config")
		}
	}
	if strings.Contains(string(configJSON), "caKey") || strings.Contains(string(configJSON), "caPrivate") {
		t.Fatal("CA key field leaked into JSON config")
	}
}

func TestFixedNodeServerContractIsTLSOnlyExactBoundedAndNonForwarding(t *testing.T) {
	for _, required := range []string{
		`const configPath=process.argv[2]`, `fs.statSync(configPath)`, `stat.size>12582912`, `https.createServer`, `server.listen(443,"0.0.0.0")`, `SNICallback`,
		`contexts.get(String(name||"").toLowerCase())`, `host!==sni`,
		`routes.get(key)`, `size>65536`, `body.length>1048576`, `f.status>=300&&f.status<400`,
		`/tmp/virtualapi-hits.json`, `content-length`, `server.requestTimeout`,
	} {
		if !strings.Contains(ContainerServerProgram, required) {
			t.Errorf("Node adapter is missing invariant %q", required)
		}
	}
	for _, forbidden := range []string{`fetch(`, `http.request(`, `https.request(`, `net.connect(`, `dns.lookup(`, `proxy`, `Location:`} {
		if strings.Contains(ContainerServerProgram, forbidden) {
			t.Errorf("Node adapter contains outbound/redirect feature %q", forbidden)
		}
	}
}

func TestServerConfigRequiresLiveTaskAndDoesNotExposeClosedKey(t *testing.T) {
	g, _, err := New(contractFixture())
	if err != nil {
		t.Fatal(err)
	}
	g.Close()
	if _, err := g.ServerConfigJSON(); err == nil {
		t.Fatal("closed gateway still returns container config")
	}
}

// Ensure the public contract doesn't accidentally conflate a network proof
// with a locally generated receipt.
func TestLocalGatewayDoesNotClaimExternalNetworkConfinement(t *testing.T) {
	_, receipt, err := New(contractFixture())
	if err != nil {
		t.Fatal(err)
	}
	if receipt.NoEgressVerified {
		t.Fatal("local HTTP handler claimed container network isolation")
	}
	if receipt.NetworkPolicy != "task-private-internal-only" {
		t.Fatalf("network requirement missing: %q", receipt.NetworkPolicy)
	}
}
