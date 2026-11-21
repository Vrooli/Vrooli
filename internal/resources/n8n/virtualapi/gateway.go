// Package virtualapi implements the task-private HTTPS API companion contract.
// The gateway has no outbound client or proxy path: an unrecognized request is
// rejected locally, and every successful response comes from a pinned fixture.
package virtualapi

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	MaxHosts             = 32
	MaxRoutes            = 512
	MaxBodyBytes         = 1 << 20
	MaxRequestBytes      = 64 << 10
	MaxServerConfigBytes = 12 << 20
	maxQueryPairs        = 32
	maxQueryBytes        = 2 << 10
)

var (
	credentialFixtureValue = regexp.MustCompile(`(?i)\b(?:bearer|basic)\s+[A-Za-z0-9._~+/=-]{8,}|\b(?:sk-[A-Za-z0-9]{16,}|gh[pousr]_[A-Za-z0-9]{16,})\b`)
	fixtureEmail           = regexp.MustCompile(`(?i)[a-z0-9._%+-]+@[a-z0-9.-]+\.[a-z]{2,}`)
)

// Route is one exact, synthetic response. Host, method, path and query form
// the complete match key; no prefix or wildcard matching is supported.
type Route struct {
	Host        string `json:"host"`
	Method      string `json:"method"`
	Path        string `json:"path"`
	Query       string `json:"query,omitempty"`
	Status      int    `json:"status"`
	ContentType string `json:"contentType"`
	Body        []byte `json:"body"`
}

// Contract is a complete task profile. Hosts are explicit DNS names and must
// correspond to the aliases installed only in the task's private network.
type Contract struct {
	TaskID string   `json:"taskId"`
	Hosts  []string `json:"hosts"`
	Routes []Route  `json:"routes"`
}

// Receipt contains no fixture data or key material. CACertPEM is the public
// task trust anchor; its SHA-256 lets the owner verify it was not substituted.
type Receipt struct {
	TaskID             string   `json:"taskId"`
	Aliases            []string `json:"aliases"`
	Port               int      `json:"port"`
	CACertPEM          []byte   `json:"caCertPem"`
	CAFingerprint      string   `json:"caFingerprintSha256"`
	ProfileFingerprint string   `json:"profileFingerprintSha256"`
	NoOutboundDial     bool     `json:"noOutboundDialInHandler"`
	NoEgressVerified   bool     `json:"noEgressVerified"`
	NetworkPolicy      string   `json:"requiredNetworkPolicy"`
	Status             string   `json:"status"`
}

type (
	routeKey struct{ host, method, path, query string }
	Gateway  struct {
		mu          sync.RWMutex
		taskID      string
		routes      map[routeKey]Route
		aliases     []string
		ca          *x509.Certificate
		caKey       *ecdsa.PrivateKey
		leaf        *tlsCertificate
		caPEM       []byte
		profileHash string
		closed      bool
		hits        map[routeKey]uint64
	}
)

// tlsCertificate keeps key material private to the gateway lifecycle.
type tlsCertificate struct{ certificatePEM, keyPEM []byte }

// ContainerServerConfig is the bounded, task-local payload consumed by the
// fixed Node adapter. It contains the leaf key only; the task CA private key
// is never exported. The bytes are intended for mode-0600 tmpfs staging.
type ContainerServerConfig struct {
	TaskID         string   `json:"taskId"`
	Hosts          []string `json:"hosts"`
	Routes         []Route  `json:"routes"`
	CertificatePEM string   `json:"certificatePem"`
	PrivateKeyPEM  string   `json:"privateKeyPem"`
}

// ContainerServerProgram is fixed owner code. It imports only Node built-ins,
// serves TLS on :443, and has no network client, proxy, resolver, or forwarding
// path. The host process owns staging/removal of its config in task tmpfs.
const ContainerServerProgram = `const fs=require("node:fs"),https=require("node:https"),tls=require("node:tls"),crypto=require("node:crypto");
const configPath=process.argv[2],stat=fs.statSync(configPath);if(!stat.isFile()||stat.size<1||stat.size>12582912)throw Error("gateway config outside size bound");
const cfg=JSON.parse(fs.readFileSync(configPath,"utf8"));
if(!Array.isArray(cfg.hosts)||!Array.isArray(cfg.routes)||cfg.hosts.length<1||cfg.hosts.length>32||cfg.routes.length<1||cfg.routes.length>512)throw Error("invalid gateway config");
const hostPattern=/^(?=.{1,253}$)(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$/;
for(const h of cfg.hosts)if(typeof h!=="string"||h!==h.toLowerCase()||!hostPattern.test(h))throw Error("invalid exact hostname");
const hosts=new Set(cfg.hosts.map(x=>String(x).toLowerCase()));
const routes=new Map(),hits=[];let total=0;
let aggregate=0;const methods=new Set(["GET","HEAD","POST","PUT","PATCH","DELETE","OPTIONS"]),types=new Set(["application/json","text/plain","application/xml"]);
function canonicalQuery(raw){if(typeof raw!=="string"||Buffer.byteLength(raw)>2048)throw Error("invalid query");if(!raw)return "";const p=new URLSearchParams(raw);if([...p].length>32)throw Error("too many query pairs");const entries=[...p].map(([k,v])=>[k,v]);for(const [k,v] of entries){if(!k||/[^\x20-\x7e]/.test(k+v)||/[\x00-\x1f\x7f*~]/.test(k+v)||/(?:token|secret|password|authorization|credential|api[_-]?key)/i.test(k)||/(?:bearer\s|sk-[A-Za-z0-9]|@(?:gmail|outlook|yahoo)\.)/i.test(v))throw Error("unsafe query value");}entries.sort((a,b)=>a[0]<b[0]?-1:a[0]>b[0]?1:a[1]<b[1]?-1:a[1]>b[1]?1:0);return new URLSearchParams(entries).toString();}
for(const f of cfg.routes){if(!hosts.has(f.host)||!methods.has(f.method)||typeof f.path!=="string"||!f.path.startsWith("/")||f.path.length>2048||/[?#%\\\x00]/.test(f.path)||f.path.includes("..")||!Number.isInteger(f.status)||f.status<200||f.status>599||(f.status>=300&&f.status<400)||!types.has(f.contentType)||typeof f.body!=="string"||!/^(?:[A-Za-z0-9+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?$/.test(f.body))throw Error("invalid exact fixture");f.query=canonicalQuery(f.query||"");const key=f.method+" "+f.host+" "+f.path+"?"+f.query;if(routes.has(key))throw Error("duplicate route");const body=Buffer.from(f.body,"base64");if(body.length>1048576)throw Error("response too large");aggregate+=body.length;if(aggregate>8388608)throw Error("aggregate fixture size too large");const route={...f,body};routes.set(key,route);hits.push({host:f.host,method:f.method,path:f.path,queryHash:crypto.createHash("sha256").update(f.query).digest("hex"),count:0});}
const keyPEM=Buffer.from(cfg.privateKeyPem,"utf8"),certPEM=Buffer.from(cfg.certificatePem,"utf8");
const contexts=new Map([...hosts].map(host=>[host,tls.createSecureContext({key:keyPEM,cert:certPEM,minVersion:"TLSv1.2"})]));
function persist(){const data=JSON.stringify({version:1,total,routes:hits});fs.writeFileSync("/tmp/virtualapi-hits.json.tmp",data,{mode:0o600});fs.renameSync("/tmp/virtualapi-hits.json.tmp","/tmp/virtualapi-hits.json");}
function safeHost(raw){return String(raw||"").toLowerCase().replace(/:443$/,"" );}
const server=https.createServer({key:keyPEM,cert:certPEM,minVersion:"TLSv1.2",SNICallback:(name,cb)=>{const context=contexts.get(String(name||"").toLowerCase());if(!context)return cb(Error("unapproved TLS hostname"));cb(null,context);}},(req,res)=>{
const sni=String(req.socket.servername||"").toLowerCase(),host=safeHost(req.headers.host);if(!hosts.has(sni)||host!==sni){req.resume();res.writeHead(404);return res.end("not found");}
const target=String(req.url||"");if(!target.startsWith("/")||target.includes("#")||target.includes("%")){req.resume();res.writeHead(404);return res.end("not found");}
const qi=target.indexOf("?"),path=qi<0?target:target.slice(0,qi);let query;try{query=canonicalQuery(qi<0?"":target.slice(qi+1));}catch{req.resume();res.writeHead(404);return res.end("no synthetic fixture");}const key=req.method+" "+host+" "+path+"?"+query,f=routes.get(key);if(!f){req.resume();res.writeHead(404);return res.end("no synthetic fixture");}
let size=0,tooLarge=false,ended=false;req.on("data",chunk=>{size+=chunk.length;if(size>65536&&!ended){ended=true;tooLarge=true;res.writeHead(413);res.end("request too large");req.resume();}});req.on("end",()=>{if(ended)return;if(tooLarge){res.writeHead(413);return res.end("request too large");}if(f.body.length>1048576){res.writeHead(500);return res.end("fixture exceeds response bound");}const ix=hits.findIndex(x=>x.method+" "+x.host+" "+x.path+"?"+x.query===key);if(ix<0){res.writeHead(404);return res.end("not found");}if(hits[ix].count<2147483647)hits[ix].count++;if(total<2147483647)total++;persist();res.writeHead(f.status,{"content-type":f.contentType,"cache-control":"no-store","x-content-type-options":"nosniff","content-length":String(f.body.length)});res.end(req.method==="HEAD"?"":f.body);});});
server.headersTimeout=10000;server.requestTimeout=30000;server.maxRequestsPerSocket=10000;persist();server.listen(443,"0.0.0.0");
`

func New(contract Contract) (*Gateway, Receipt, error) {
	canonical, err := validate(contract)
	if err != nil {
		return nil, Receipt{}, err
	}
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, Receipt{}, fmt.Errorf("generate task CA: %w", err)
	}
	now := time.Now().UTC()
	caTemplate := &x509.Certificate{SerialNumber: serial(), Subject: pkix.Name{CommonName: "Vrooli task API companion " + contract.TaskID}, NotBefore: now.Add(-time.Minute), NotAfter: now.Add(24 * time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature, MaxPathLen: 0}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		return nil, Receipt{}, fmt.Errorf("issue task CA: %w", err)
	}
	ca, err := x509.ParseCertificate(caDER)
	if err != nil {
		return nil, Receipt{}, err
	}
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})
	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, Receipt{}, fmt.Errorf("generate gateway key: %w", err)
	}
	leafTemplate := &x509.Certificate{SerialNumber: serial(), Subject: pkix.Name{CommonName: canonical[0]}, DNSNames: canonical, NotBefore: now.Add(-time.Minute), NotAfter: now.Add(12 * time.Hour), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, BasicConstraintsValid: true}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTemplate, ca, &leafKey.PublicKey, caKey)
	if err != nil {
		return nil, Receipt{}, fmt.Errorf("issue gateway certificate: %w", err)
	}
	leafKeyDER, err := x509.MarshalPKCS8PrivateKey(leafKey)
	if err != nil {
		return nil, Receipt{}, err
	}
	leaf := &tlsCertificate{pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: leafDER}), pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: leafKeyDER})}
	g := &Gateway{taskID: contract.TaskID, routes: map[routeKey]Route{}, aliases: canonical, ca: ca, caKey: caKey, leaf: leaf, caPEM: caPEM, profileHash: profileHash(contract), hits: map[routeKey]uint64{}}
	for _, r := range contract.Routes {
		r.Host = strings.ToLower(strings.TrimSpace(r.Host))
		r.Method = strings.ToUpper(strings.TrimSpace(r.Method))
		r.Path = strings.TrimSpace(r.Path)
		r.Query, _ = canonicalQuery(r.Query)
		k := routeKey{r.Host, r.Method, r.Path, r.Query}
		r.Body = append([]byte(nil), r.Body...)
		g.routes[k] = r
		g.hits[k] = 0
	}
	caSum := sha256.Sum256(caPEM)
	return g, Receipt{TaskID: contract.TaskID, Aliases: append([]string(nil), canonical...), Port: 443, CACertPEM: append([]byte(nil), caPEM...), CAFingerprint: hex.EncodeToString(caSum[:]), ProfileFingerprint: g.profileHash, NoOutboundDial: true, NoEgressVerified: false, NetworkPolicy: "task-private-internal-only", Status: "ready"}, nil
}

// ServeHTTP is intentionally suitable for a TLS listener only. It checks both
// SNI and Host against the same allowlisted hostname before considering routes.
func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.closed {
		http.Error(w, "gateway stopped", http.StatusServiceUnavailable)
		return
	}
	host := canonicalHost(r.Host)
	if r.TLS == nil || !contains(g.aliases, strings.ToLower(r.TLS.ServerName)) || host != strings.ToLower(r.TLS.ServerName) {
		http.NotFound(w, r)
		return
	}
	if r.URL == nil || r.URL.IsAbs() || r.URL.Fragment != "" || strings.Contains(r.URL.EscapedPath(), "%") {
		http.NotFound(w, r)
		return
	}
	query, err := canonicalQuery(r.URL.RawQuery)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	k := routeKey{host, strings.ToUpper(r.Method), r.URL.Path, query}
	route, ok := g.routes[k]
	if !ok {
		http.NotFound(w, r)
		return
	}
	if r.ContentLength > MaxRequestBytes {
		http.Error(w, "request too large", http.StatusRequestEntityTooLarge)
		return
	}
	if r.Body != nil {
		n, err := io.Copy(io.Discard, io.LimitReader(r.Body, MaxRequestBytes+1))
		if err != nil || n > MaxRequestBytes {
			http.Error(w, "request too large", http.StatusRequestEntityTooLarge)
			return
		}
	}
	g.hits[k]++
	w.Header().Set("Content-Type", route.ContentType)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(route.Status)
	if r.Method != http.MethodHead {
		_, _ = w.Write(route.Body)
	}
}

// ServerCertificate returns a copied leaf cert/key pair for TLS listener setup.
// The caller must stop the listener and call Close when the task ends.
func (g *Gateway) ServerCertificate() (certPEM, keyPEM []byte, err error) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.closed || g.leaf == nil {
		return nil, nil, errors.New("task gateway is closed")
	}
	return append([]byte(nil), g.leaf.certificatePEM...), append([]byte(nil), g.leaf.keyPEM...), nil
}

// ServerConfigJSON returns a bounded copy for staging in owner-managed task
// tmpfs. The public CA is intentionally omitted; only the leaf cert/key needed
// by the server are present. The private CA key never leaves this package.
func (g *Gateway) ServerConfigJSON() ([]byte, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.closed || g.leaf == nil {
		return nil, errors.New("task gateway is closed")
	}
	cfg := ContainerServerConfig{TaskID: g.taskID, Hosts: append([]string(nil), g.aliases...), CertificatePEM: string(g.leaf.certificatePEM), PrivateKeyPEM: string(g.leaf.keyPEM)}
	for _, r := range g.routes {
		r.Body = append([]byte(nil), r.Body...)
		cfg.Routes = append(cfg.Routes, r)
	}
	sort.Slice(cfg.Routes, func(i, j int) bool {
		a, b := cfg.Routes[i], cfg.Routes[j]
		if a.Host != b.Host {
			return a.Host < b.Host
		}
		if a.Method != b.Method {
			return a.Method < b.Method
		}
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		return a.Query < b.Query
	})
	data, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("encode task gateway config: %w", err)
	}
	if len(data) == 0 || len(data) > MaxServerConfigBytes {
		return nil, fmt.Errorf("task gateway config exceeds %d bytes", MaxServerConfigBytes)
	}
	return data, nil
}

// Hits returns only statically declared route identities and aggregate counts.
func (g *Gateway) Hits() map[string]uint64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	out := map[string]uint64{}
	for k, n := range g.hits {
		queryHash := sha256.Sum256([]byte(k.query))
		out[k.method+" "+k.host+k.path+"?sha256="+hex.EncodeToString(queryHash[:])] = n
	}
	return out
}

// Close destroys references to private key material, fixtures, and counters.
func (g *Gateway) Close() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.closed = true
	for k, r := range g.routes {
		for i := range r.Body {
			r.Body[i] = 0
		}
		delete(g.routes, k)
	}
	for k := range g.hits {
		delete(g.hits, k)
	}
	if g.leaf != nil {
		wipe(g.leaf.keyPEM)
		wipe(g.leaf.certificatePEM)
		g.leaf = nil
	}
	if g.caKey != nil {
		g.caKey.D.SetInt64(0)
		g.caKey = nil
	}
	g.ca = nil
	wipe(g.caPEM)
	g.caPEM = nil
}

func validate(c Contract) ([]string, error) {
	if !validTaskID(c.TaskID) {
		return nil, errors.New("task ID must be a UUID")
	}
	if len(c.Hosts) == 0 || len(c.Hosts) > MaxHosts || len(c.Routes) == 0 || len(c.Routes) > MaxRoutes {
		return nil, fmt.Errorf("contract requires 1..%d hosts and 1..%d routes", MaxHosts, MaxRoutes)
	}
	hosts := make([]string, 0, len(c.Hosts))
	seen := map[string]bool{}
	for _, raw := range c.Hosts {
		h := strings.ToLower(strings.TrimSpace(raw))
		if !validPublicHost(h) || seen[h] {
			return nil, fmt.Errorf("invalid or duplicate exact hostname %q", raw)
		}
		seen[h] = true
		hosts = append(hosts, h)
	}
	sort.Strings(hosts)
	keys := map[routeKey]bool{}
	total := 0
	for i, r := range c.Routes {
		r.Host = strings.ToLower(strings.TrimSpace(r.Host))
		r.Method = strings.ToUpper(strings.TrimSpace(r.Method))
		r.Path = strings.TrimSpace(r.Path)
		query, err := canonicalQuery(r.Query)
		if err != nil {
			return nil, fmt.Errorf("route %d query is invalid: %w", i, err)
		}
		r.Query = query
		if !seen[r.Host] {
			return nil, fmt.Errorf("route %d host is not declared", i)
		}
		if !validMethod(r.Method) || r.Path == "" || len(r.Path) > 2048 || r.Path[0] != '/' || strings.ContainsAny(r.Path, "?#%\\\x00") || strings.Contains(r.Path, "..") {
			return nil, fmt.Errorf("route %d method or exact path is invalid", i)
		}
		if r.Status < 200 || r.Status > 599 || r.Status >= 300 && r.Status < 400 {
			return nil, fmt.Errorf("route %d status is invalid or redirects", i)
		}
		if r.ContentType != "application/json" && r.ContentType != "text/plain" && r.ContentType != "application/xml" {
			return nil, fmt.Errorf("route %d content type is unsupported", i)
		}
		if len(r.Body) > MaxBodyBytes {
			return nil, fmt.Errorf("route %d body exceeds limit", i)
		}
		if err := validateSyntheticFixtureBody(r.ContentType, r.Body); err != nil {
			return nil, fmt.Errorf("route %d body is not safe synthetic fixture data: %w", i, err)
		}
		total += len(r.Body)
		if total > 8*MaxBodyBytes {
			return nil, errors.New("fixture bodies exceed aggregate limit")
		}
		k := routeKey{r.Host, r.Method, r.Path, r.Query}
		if keys[k] {
			return nil, fmt.Errorf("route %d duplicates an exact route", i)
		}
		keys[k] = true
	}
	return hosts, nil
}

func validateSyntheticFixtureBody(contentType string, body []byte) error {
	if credentialFixtureValue.Match(body) {
		return errors.New("credential-shaped value is not allowed")
	}
	for _, email := range fixtureEmail.FindAll(body, -1) {
		if !strings.HasSuffix(strings.ToLower(string(email)), ".invalid") {
			return errors.New("email address must use the reserved .invalid domain")
		}
	}
	if contentType != "application/json" {
		return nil
	}
	var value any
	if err := json.Unmarshal(body, &value); err != nil {
		return errors.New("JSON fixture body is malformed")
	}
	return rejectSensitiveFixtureFields(value)
}

func rejectSensitiveFixtureFields(value any) error {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			normalized := strings.ToLower(strings.NewReplacer("_", "", "-", "", ".", "").Replace(key))
			for _, marker := range []string{"password", "passwd", "secret", "token", "authorization", "cookie", "apikey", "clientsecret", "privatekey"} {
				if strings.Contains(normalized, marker) {
					return fmt.Errorf("sensitive field %q is not allowed", key)
				}
			}
			if err := rejectSensitiveFixtureFields(child); err != nil {
				return err
			}
		}
	case []any:
		for _, child := range typed {
			if err := rejectSensitiveFixtureFields(child); err != nil {
				return err
			}
		}
	}
	return nil
}

func validPublicHost(h string) bool {
	if len(h) > 253 || h == "" || strings.HasSuffix(h, ".") || strings.ContainsAny(h, "/:@*%\\") {
		return false
	}
	if strings.EqualFold(h, "localhost") || net.ParseIP(h) != nil {
		return false
	}
	for _, part := range strings.Split(h, ".") {
		if len(part) == 0 || len(part) > 63 || part[0] == '-' || part[len(part)-1] == '-' {
			return false
		}
		for _, ch := range part {
			if !(ch >= 'a' && ch <= 'z' || ch >= '0' && ch <= '9' || ch == '-') {
				return false
			}
		}
	}
	return strings.Contains(h, ".")
}

func validTaskID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
			continue
		}
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F') {
			return false
		}
	}
	return true
}

func validMethod(m string) bool {
	switch m {
	case "GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "OPTIONS":
		return true
	}
	return false
}

func canonicalHost(h string) string {
	h = strings.ToLower(strings.TrimSpace(h))
	if strings.HasSuffix(h, ":443") {
		return strings.TrimSuffix(h, ":443")
	}
	return h
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

func canonicalQuery(raw string) (string, error) {
	if len(raw) > maxQueryBytes {
		return "", errors.New("query exceeds byte limit")
	}
	if raw == "" {
		return "", nil
	}
	values, err := url.ParseQuery(raw)
	pairs := 0
	for _, items := range values {
		pairs += len(items)
	}
	if err != nil || pairs > maxQueryPairs {
		return "", errors.New("query is malformed or has too many keys")
	}
	for key, items := range values {
		if key == "" || unsafeQueryText(key) || !asciiQueryText(key) {
			return "", errors.New("query contains unsafe key")
		}
		if credentialLikeQueryKey(key) {
			return "", errors.New("query key resembles a credential")
		}
		for _, item := range items {
			if unsafeQueryText(item) || !asciiQueryText(item) || credentialLikeQueryValue(item) {
				return "", errors.New("query value is unsafe or credential-like")
			}
		}
		sort.Strings(items)
		values[key] = items
	}
	return values.Encode(), nil
}

func asciiQueryText(s string) bool {
	for _, r := range s {
		if r < 0x20 || r > 0x7e || r == '*' || r == '~' {
			return false
		}
	}
	return true
}

func credentialLikeQueryKey(key string) bool {
	lower := strings.ToLower(key)
	for _, marker := range []string{"token", "secret", "password", "authorization", "credential", "api_key", "api-key", "apikey"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func unsafeQueryText(s string) bool {
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			return true
		}
	}
	return false
}

func credentialLikeQueryValue(s string) bool {
	lower := strings.ToLower(s)
	return strings.Contains(lower, "bearer ") || strings.HasPrefix(lower, "sk-") || strings.Contains(lower, "@gmail.") || strings.Contains(lower, "@outlook.") || strings.Contains(lower, "@yahoo.")
}

func serial() *big.Int {
	n, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		panic(err)
	}
	return n
}

func profileHash(c Contract) string {
	c.Hosts = append([]string(nil), c.Hosts...)
	for i := range c.Hosts {
		c.Hosts[i] = strings.ToLower(strings.TrimSpace(c.Hosts[i]))
	}
	sort.Strings(c.Hosts)
	c.Routes = append([]Route(nil), c.Routes...)
	for i := range c.Routes {
		r := &c.Routes[i]
		r.Host = strings.ToLower(strings.TrimSpace(r.Host))
		r.Method = strings.ToUpper(strings.TrimSpace(r.Method))
		r.Path = strings.TrimSpace(r.Path)
		r.Query, _ = canonicalQuery(r.Query)
	}
	sort.Slice(c.Routes, func(i, j int) bool {
		a, b := c.Routes[i], c.Routes[j]
		if a.Host != b.Host {
			return a.Host < b.Host
		}
		if a.Method != b.Method {
			return a.Method < b.Method
		}
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		return a.Query < b.Query
	})
	b, _ := json.Marshal(c)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func wipe(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
