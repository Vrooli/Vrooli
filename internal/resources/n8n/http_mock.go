package n8n

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	MaxSyntheticHTTPMockConfigBytes = 2 << 20
	maxSyntheticHTTPMockFixtures    = 128
	maxSyntheticHTTPMockBodyBytes   = 64 << 10
	maxSyntheticHTTPMockTotalBytes  = 1 << 20
	maxSyntheticHTTPMockQueryPairs  = 32
	maxSyntheticHTTPMockQueryBytes  = 2 << 10
	syntheticHTTPMockPort           = 18080
	maxHTTPMockOutputBytes          = 32 << 10
	httpMockNamePattern             = `^[a-z][a-z0-9-]{0,31}$`
	httpMockVersionPattern          = `^[0-9]+\.[0-9]+(?:\.[0-9]+)?(?:[-+][A-Za-z0-9.-]+)?$`
)

var (
	httpMockHostPattern = regexp.MustCompile(`^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+synthetic\.invalid$`)
	httpMockSegment     = regexp.MustCompile(`^[A-Za-z0-9._~!$&'()*+,;=:@-]+$`)
	httpMockQueryKey    = regexp.MustCompile(`^[A-Za-z0-9_.~-]{1,128}$`)
	httpMockCredential  = regexp.MustCompile(`(?i)\b(?:bearer|basic)\s+[A-Za-z0-9._~+/=-]{8,}|\b(?:sk-[A-Za-z0-9]{16,}|gh[pousr]_[A-Za-z0-9]{16,})\b`)
	httpMockEmail       = regexp.MustCompile(`(?i)[a-z0-9._%+-]+@[a-z0-9.-]+\.[a-z]{2,}`)
	httpMockNameRE      = regexp.MustCompile(httpMockNamePattern)
	httpMockVersionRE   = regexp.MustCompile(httpMockVersionPattern)
)

// HTTPMockFixture is a fixed response for one exact synthetic HTTP route.
// The resource owner revalidates this shape at its public control-plane edge.
type HTTPMockFixture struct {
	Host        string `json:"host"`
	Method      string `json:"method"`
	Path        string `json:"path"`
	Query       string `json:"query,omitempty"`
	Status      int    `json:"status"`
	ContentType string `json:"contentType"`
	Body        string `json:"body"`
}

type HTTPMockConfig struct {
	Name     string            `json:"name"`
	Version  string            `json:"version"`
	Fixtures []HTTPMockFixture `json:"fixtures"`
}

type HTTPMockReceipt struct {
	SandboxID          string   `json:"sandboxId"`
	CompanionRuntimeID string   `json:"companionRuntimeId"`
	NetworkAliases     []string `json:"networkAliases"`
	Port               int      `json:"port"`
	Status             string   `json:"status"`
	NoEgressVerified   bool     `json:"noEgressVerified"`
	FixtureCount       int      `json:"fixtureCount"`
	ConfigSHA256       string   `json:"configSha256"`
}

type HTTPMockStopReceipt struct {
	SandboxID          string `json:"sandboxId"`
	CompanionRuntimeID string `json:"companionRuntimeId"`
	Status             string `json:"status"`
}

const stageHTTPMockProgram = `const fs=require("fs");const p=process.argv[1];const flags=fs.constants.O_WRONLY|fs.constants.O_CREAT|fs.constants.O_TRUNC|fs.constants.O_NOFOLLOW;const fd=fs.openSync(p,flags,0o600);process.stdin.on("data",b=>{for(let o=0;o<b.length;)o+=fs.writeSync(fd,b,o,b.length-o)});process.stdin.on("error",()=>{try{fs.closeSync(fd)}catch{};process.exitCode=1});process.stdin.on("end",()=>fs.closeSync(fd));`

// This server has no network client, resolver override, proxy support, or
// forwarding code. It responds only to exact fixture matches and fails closed.
const canonicalHTTPMockQueryProgram = `function canonicalQuery(raw){if(Buffer.byteLength(raw,"utf8")>2048||raw.includes(";"))return null;if(raw==="")return "";const values=new Map();function decode(part){if(/%(?![0-9a-fA-F]{2})/.test(part))throw Error("bad escape");return decodeURIComponent(part.replace(/\+/g," "))}function encode(value){return encodeURIComponent(value).replace(/[!'()*]/g,c=>"%"+c.charCodeAt(0).toString(16).toUpperCase()).replace(/%20/g,"+")}function sensitive(key){const normalized=key.toLowerCase().replace(/[_\-.]/g,"");return ["password","passwd","secret","token","authorization","cookie","apikey","clientsecret","privatekey"].some(marker=>normalized.includes(marker))}try{for(const pair of raw.split("&")){if(pair==="")continue;const index=pair.indexOf("=");const key=decode(index<0?pair:pair.slice(0,index));const value=decode(index<0?"":pair.slice(index+1));if(!/^[A-Za-z0-9_.~-]{1,128}$/.test(key)||sensitive(key)||Buffer.byteLength(value,"utf8")>512||/[\x00\r\n]/.test(value)||/\b(?:bearer|basic)\s+[A-Za-z0-9._~+/=-]{8,}|\b(?:sk-[A-Za-z0-9]{16,}|gh[pousr]_[A-Za-z0-9]{16,})\b/i.test(value))return null;const emails=value.match(/[a-z0-9._%+-]+@[a-z0-9.-]+\.[a-z]{2,}/ig)||[];if(emails.some(email=>!email.toLowerCase().endsWith(".invalid"))||values.has(key))return null;values.set(key,value)}}catch{return null}if(values.size===0||values.size>32)return null;const canonical=[...values].sort((a,b)=>a[0]<b[0]?-1:a[0]>b[0]?1:0).map(([key,value])=>encode(key)+"="+encode(value)).join("&");return Buffer.byteLength(canonical,"utf8")<=2048?canonical:null}`

const syntheticHTTPMockServerTemplate = `const fs=require("fs"),http=require("http");${canonicalHTTPMockQueryProgram};const cfg=JSON.parse(fs.readFileSync(process.argv[1],"utf8"));const routes=new Map();for(const f of cfg.fixtures)routes.set(f.method+" "+f.host+" "+f.path+"?"+(f.query||""),f);const server=http.createServer((req,res)=>{if(req.method==="GET"&&req.url==="/__vrooli_health"){res.statusCode=204;return res.end()}const host=String(req.headers.host||"").toLowerCase().replace(/:\d+$/,"");const target=String(req.url||"");if(target.includes("#")){req.resume();res.statusCode=404;return res.end("not found")}const queryIndex=target.indexOf("?");const requestPath=queryIndex<0?target:target.slice(0,queryIndex);const requestQuery=queryIndex<0?"":canonicalQuery(target.slice(queryIndex+1));if(requestQuery===null){req.resume();res.statusCode=404;return res.end("not found")}const fixture=routes.get(req.method+" "+host+" "+requestPath+"?"+requestQuery);if(!fixture){req.resume();res.statusCode=404;res.setHeader("content-type","text/plain");return res.end("no synthetic fixture")}let size=0,tooLarge=false,responded=false;req.on("data",chunk=>{size+=chunk.length;if(size>65536&&!responded){responded=true;tooLarge=true;res.statusCode=413;res.end("request too large");req.resume()}});req.on("end",()=>{if(responded)return;if(tooLarge){res.statusCode=413;return res.end("request too large")}res.statusCode=fixture.status;res.setHeader("content-type",fixture.contentType);res.setHeader("cache-control","no-store");res.setHeader("x-content-type-options","nosniff");res.end(req.method==="HEAD"?"":fixture.body)})});server.listen(18080,"0.0.0.0");`

var syntheticHTTPMockServerProgram = strings.ReplaceAll(syntheticHTTPMockServerTemplate, "${canonicalHTTPMockQueryProgram}", canonicalHTTPMockQueryProgram)

const syntheticHTTPMockProbeProgram = `const http=require("http");let tries=0,done=false;function probe(){if(done)return;tries++;const req=http.get({host:"127.0.0.1",port:18080,path:"/__vrooli_health",timeout:500},res=>{res.resume();if(res.statusCode===204){done=true;process.stdout.write("ready")}else retry()});req.on("timeout",()=>req.destroy());req.on("error",retry)}function retry(){if(done)return;if(tries>=20){done=true;process.stderr.write("not ready");process.exitCode=1;return}setTimeout(probe,100)}probe();`

// ValidateHTTPMockConfig enforces the security properties independently of the
// Workspace Sandbox provider; no caller-supplied value becomes a command or path.
func ValidateHTTPMockConfig(config HTTPMockConfig) error {
	if !httpMockNameRE.MatchString(config.Name) || !httpMockVersionRE.MatchString(config.Version) {
		return errors.New("HTTP mock name or version is invalid")
	}
	if len(config.Fixtures) == 0 || len(config.Fixtures) > maxSyntheticHTTPMockFixtures {
		return fmt.Errorf("HTTP mock requires 1 to %d fixtures", maxSyntheticHTTPMockFixtures)
	}
	seen := map[string]struct{}{}
	total := 0
	for index := range config.Fixtures {
		fixture := &config.Fixtures[index]
		fixture.Host = strings.ToLower(strings.TrimSpace(fixture.Host))
		fixture.Method = strings.ToUpper(strings.TrimSpace(fixture.Method))
		fixture.Path = strings.TrimSpace(fixture.Path)
		fixture.Query = strings.TrimSpace(fixture.Query)
		fixture.ContentType = strings.ToLower(strings.TrimSpace(fixture.ContentType))
		if len(fixture.Host) > 253 || !httpMockHostPattern.MatchString(fixture.Host) {
			return fmt.Errorf("fixture %d host must be an exact *.synthetic.invalid hostname", index)
		}
		if !validHTTPMockMethod(fixture.Method) {
			return fmt.Errorf("fixture %d method is unsupported", index)
		}
		if !validHTTPMockPath(fixture.Path) {
			return fmt.Errorf("fixture %d path is unsafe or not exact", index)
		}
		if fixture.Path == "/__vrooli_health" {
			return fmt.Errorf("fixture %d path is reserved", index)
		}
		query, err := normalizeHTTPMockQuery(fixture.Query)
		if err != nil {
			return fmt.Errorf("fixture %d query: %w", index, err)
		}
		fixture.Query = query
		if fixture.Status < 200 || fixture.Status > 599 || fixture.Status >= 300 && fixture.Status < 400 {
			return fmt.Errorf("fixture %d status cannot redirect", index)
		}
		if fixture.ContentType == "" {
			fixture.ContentType = "application/json"
		}
		switch fixture.ContentType {
		case "application/json", "text/plain", "application/xml":
		default:
			return fmt.Errorf("fixture %d content type is not allowed", index)
		}
		if len(fixture.Body) > maxSyntheticHTTPMockBodyBytes || !utf8.ValidString(fixture.Body) || strings.ContainsRune(fixture.Body, '\x00') {
			return fmt.Errorf("fixture %d body is unsafe or exceeds %d bytes", index, maxSyntheticHTTPMockBodyBytes)
		}
		total += len(fixture.Body)
		if total > maxSyntheticHTTPMockTotalBytes {
			return fmt.Errorf("combined HTTP mock bodies exceed %d bytes", maxSyntheticHTTPMockTotalBytes)
		}
		if err := validateHTTPMockBody(*fixture); err != nil {
			return fmt.Errorf("fixture %d body: %w", index, err)
		}
		key := fixture.Method + " " + fixture.Host + " " + fixture.Path + "?" + fixture.Query
		if _, exists := seen[key]; exists {
			return fmt.Errorf("fixture %d duplicates a route", index)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func normalizeHTTPMockQuery(query string) (string, error) {
	if len(query) > maxSyntheticHTTPMockQueryBytes {
		return "", fmt.Errorf("query exceeds %d bytes", maxSyntheticHTTPMockQueryBytes)
	}
	if query == "" {
		return "", nil
	}
	values, err := url.ParseQuery(query)
	if err != nil {
		return "", errors.New("query must be valid form-encoded key/value pairs")
	}
	if len(values) == 0 || len(values) > maxSyntheticHTTPMockQueryPairs {
		return "", fmt.Errorf("query requires 1 to %d unique keys", maxSyntheticHTTPMockQueryPairs)
	}
	for key, items := range values {
		if !httpMockQueryKey.MatchString(key) || len(items) != 1 {
			return "", errors.New("query keys must be bounded identifiers with one value each")
		}
		if sensitiveHTTPMockField(key) {
			return "", fmt.Errorf("sensitive query key %q is not allowed", key)
		}
		if len(items[0]) > 512 || strings.ContainsAny(items[0], "\x00\r\n") {
			return "", errors.New("query values must be at most 512 bytes and contain no control characters")
		}
		if httpMockCredential.MatchString(items[0]) {
			return "", errors.New("query value contains a credential-shaped value")
		}
		for _, email := range httpMockEmail.FindAllString(items[0], -1) {
			if !strings.HasSuffix(strings.ToLower(email), ".invalid") {
				return "", errors.New("query value contains a non-synthetic email address")
			}
		}
	}
	canonical := values.Encode()
	if len(canonical) > maxSyntheticHTTPMockQueryBytes {
		return "", fmt.Errorf("canonical query exceeds %d bytes", maxSyntheticHTTPMockQueryBytes)
	}
	return canonical, nil
}

func validHTTPMockMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions:
		return true
	default:
		return false
	}
}

func validHTTPMockPath(path string) bool {
	if path == "" || len(path) > 2048 || path[0] != '/' || strings.ContainsAny(path, "?#%\\\x00") {
		return false
	}
	for _, segment := range strings.Split(path, "/") {
		if segment == "." || segment == ".." || segment != "" && !httpMockSegment.MatchString(segment) {
			return false
		}
	}
	return true
}

func validateHTTPMockBody(fixture HTTPMockFixture) error {
	switch fixture.ContentType {
	case "application/json":
		if !json.Valid([]byte(fixture.Body)) {
			return errors.New("invalid JSON")
		}
		var value any
		if err := json.Unmarshal([]byte(fixture.Body), &value); err != nil {
			return errors.New("invalid JSON")
		}
		if err := rejectHTTPMockSensitiveFields(value); err != nil {
			return err
		}
	case "application/xml":
		decoder := xml.NewDecoder(strings.NewReader(fixture.Body))
		decoder.Strict = true
		for {
			token, err := decoder.Token()
			if err == io.EOF {
				break
			}
			if err != nil {
				return fmt.Errorf("invalid XML: %w", err)
			}
			switch typed := token.(type) {
			case xml.StartElement:
				if sensitiveHTTPMockField(typed.Name.Local) {
					return fmt.Errorf("sensitive XML element %q is not allowed", typed.Name.Local)
				}
			case xml.Directive, xml.ProcInst:
				return errors.New("XML directives and processing instructions are not allowed")
			}
		}
	}
	if httpMockCredential.MatchString(fixture.Body) {
		return errors.New("credential-shaped value is not allowed")
	}
	for _, email := range httpMockEmail.FindAllString(fixture.Body, -1) {
		if !strings.HasSuffix(strings.ToLower(email), ".invalid") {
			return errors.New("non-synthetic email is not allowed")
		}
	}
	return nil
}

func rejectHTTPMockSensitiveFields(value any) error {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if sensitiveHTTPMockField(key) {
				return fmt.Errorf("sensitive field %q is not allowed", key)
			}
			if err := rejectHTTPMockSensitiveFields(child); err != nil {
				return err
			}
		}
	case []any:
		for _, child := range typed {
			if err := rejectHTTPMockSensitiveFields(child); err != nil {
				return err
			}
		}
	}
	return nil
}

func sensitiveHTTPMockField(key string) bool {
	normalized := strings.ToLower(strings.NewReplacer("_", "", "-", "", ".", "").Replace(key))
	for _, marker := range []string{"password", "passwd", "secret", "token", "authorization", "cookie", "apikey", "clientsecret", "privatekey"} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func parseHTTPMockConfig(data []byte) (HTTPMockConfig, []byte, string, error) {
	if len(data) == 0 || len(data) > MaxSyntheticHTTPMockConfigBytes {
		return HTTPMockConfig{}, nil, "", fmt.Errorf("HTTP mock config must be 1 to %d bytes", MaxSyntheticHTTPMockConfigBytes)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := checkUniqueJSONValue(decoder); err != nil {
		return HTTPMockConfig{}, nil, "", fmt.Errorf("invalid HTTP mock config: %w", err)
	}
	if _, err := decoder.Token(); err != io.EOF {
		return HTTPMockConfig{}, nil, "", errors.New("HTTP mock config contains trailing JSON")
	}
	decoder = json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var config HTTPMockConfig
	if err := decoder.Decode(&config); err != nil {
		return HTTPMockConfig{}, nil, "", fmt.Errorf("decode HTTP mock config: %w", err)
	}
	if err := ValidateHTTPMockConfig(config); err != nil {
		return HTTPMockConfig{}, nil, "", err
	}
	canonical, err := json.Marshal(config)
	if err != nil {
		return HTTPMockConfig{}, nil, "", fmt.Errorf("canonicalize HTTP mock config: %w", err)
	}
	digest := sha256.Sum256(canonical)
	return config, canonical, hex.EncodeToString(digest[:]), nil
}

func checkUniqueJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := map[string]struct{}{}
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return errors.New("object key is not a string")
			}
			if _, exists := seen[key]; exists {
				return fmt.Errorf("duplicate object key %q", key)
			}
			seen[key] = struct{}{}
			if err := checkUniqueJSONValue(decoder); err != nil {
				return err
			}
		}
		_, err = decoder.Token()
		return err
	case '[':
		for decoder.More() {
			if err := checkUniqueJSONValue(decoder); err != nil {
				return err
			}
		}
		_, err = decoder.Token()
		return err
	default:
		return errors.New("unexpected JSON delimiter")
	}
}

func (o *Owner) StartHTTPMock(ctx context.Context, sandboxID string, rawConfig []byte) (HTTPMockReceipt, error) {
	canonical, err := normalizeSandboxID(sandboxID)
	if err != nil {
		return HTTPMockReceipt{}, err
	}
	config, configJSON, configHash, err := parseHTTPMockConfig(rawConfig)
	if err != nil {
		return HTTPMockReceipt{}, err
	}
	if o.image() != ImageRef {
		return HTTPMockReceipt{}, fmt.Errorf("HTTP mock must use the exact pinned n8n image")
	}
	runtimeID := httpMockRuntimeIDFor(canonical, configHash)
	containerName := httpMockContainerName(canonical)
	_, n8nContainerName, err := o.ownedRunningContainer(ctx, canonical, runtimeIDFor(canonical))
	if err != nil {
		return HTTPMockReceipt{}, fmt.Errorf("HTTP mock requires its running task-owned n8n runtime: %w", err)
	}
	n8nContainer, found, err := o.inspectContainer(ctx, n8nContainerName)
	if err != nil || !found {
		if err == nil {
			err = errors.New("task-owned n8n container disappeared")
		}
		return HTTPMockReceipt{}, err
	}
	if n8nContainer.Config.Image != ImageRef {
		return HTTPMockReceipt{}, ErrForeignObject
	}
	_, taskNetwork := objectNames(canonical)
	if _, found, err := o.inspectNetwork(ctx, taskNetwork, canonical, runtimeIDFor(canonical)); err != nil || !found {
		if err == nil {
			err = errors.New("task-owned n8n network is missing")
		}
		return HTTPMockReceipt{}, err
	}
	aliases := fixtureHosts(config.Fixtures)
	if existing, found, err := o.inspectContainer(ctx, containerName); err != nil {
		return HTTPMockReceipt{}, err
	} else if found {
		if err := verifyHTTPMockContainerIdentity(existing, canonical, runtimeID); err != nil {
			return HTTPMockReceipt{}, err
		}
		if existing.Config.Labels[labelHTTPMockConfigHash] != configHash {
			return HTTPMockReceipt{}, errors.New("an HTTP mock with different fixtures is already running; stop it before replacement")
		}
		if err := o.verifyHTTPMockNetworks(ctx, canonical, existing, aliases); err != nil {
			return HTTPMockReceipt{}, err
		}
		if !existing.State.Running {
			if _, err := o.runner().Run(ctx, "start", existing.ID); err != nil {
				return HTTPMockReceipt{}, fmt.Errorf("start owned HTTP mock: %w", err)
			}
			if err := o.launchHTTPMockServer(ctx, existing.ID, httpMockConfigPath(canonical, configHash), configJSON); err != nil {
				return HTTPMockReceipt{}, err
			}
		} else if err := o.probeHTTPMock(ctx, existing.ID); err != nil {
			if launchErr := o.launchHTTPMockServer(ctx, existing.ID, httpMockConfigPath(canonical, configHash), configJSON); launchErr != nil {
				return HTTPMockReceipt{}, errors.Join(err, launchErr)
			}
		}
		return makeHTTPMockReceipt(canonical, runtimeID, aliases, len(config.Fixtures), configHash), nil
	}
	path := httpMockConfigPath(canonical, configHash)
	args := []string{"create", "--name", containerName, "--network", taskNetwork}
	for _, alias := range aliases {
		args = append(args, "--network-alias", alias)
	}
	args = append(args,
		"--read-only", "--user", "node", "--cap-drop", "ALL", "--security-opt", "no-new-privileges:true",
		"--pids-limit", "128", "--memory", "256m", "--memory-swap", "256m", "--cpus", "1",
		"--tmpfs", "/tmp:rw,noexec,nosuid,nodev,size=4m,uid=1000,gid=1000,mode=0700",
		"--label", labelResource+"=n8n", "--label", labelSandboxID+"="+canonical,
		"--label", labelRuntimeID+"="+runtimeID, "--label", labelObjectKind+"=http-mock",
		"--label", labelCompanionResource+"=n8n-http-mock", "--label", labelSandboxOwner+"="+canonical,
		"--label", labelSandboxRunID+"="+runtimeID, "--label", labelSandboxDecl+"=true",
		"--label", labelHTTPMockConfigHash+"="+configHash, "--label", labelHTTPMockAliases+"="+strings.Join(aliases, ","),
		"--expose", "18080/tcp", "--entrypoint", "node", o.image(), "-e", httpMockKeepaliveProgram)
	createdID, err := o.runner().Run(ctx, args...)
	if err != nil {
		cleanupErr := o.cleanupHTTPMockByName(context.Background(), canonical, runtimeID, configHash, containerName)
		return HTTPMockReceipt{}, errors.Join(fmt.Errorf("create task-owned HTTP mock container: %w", err), cleanupErr)
	}
	containerID := strings.TrimSpace(string(createdID))
	if !companionRuntimePattern.MatchString(containerID) {
		cleanupErr := o.cleanupHTTPMockByName(context.Background(), canonical, runtimeID, configHash, containerName)
		return HTTPMockReceipt{}, errors.Join(errors.New("Docker returned an invalid HTTP mock container ID"), cleanupErr)
	}
	if _, err := o.runner().Run(ctx, "start", containerID); err != nil {
		cleanupErr := o.cleanupHTTPMockByName(context.Background(), canonical, runtimeID, configHash, containerName)
		return HTTPMockReceipt{}, errors.Join(fmt.Errorf("start task-owned HTTP mock container: %w", err), cleanupErr)
	}
	container, found, err := o.inspectContainer(ctx, containerName)
	if err != nil || !found {
		cleanupErr := o.cleanupHTTPMockByName(context.Background(), canonical, runtimeID, configHash, containerName)
		if err == nil {
			err = errors.New("started HTTP mock container cannot be inspected")
		}
		return HTTPMockReceipt{}, errors.Join(err, cleanupErr)
	}
	if err := verifyHTTPMockContainerIdentity(container, canonical, runtimeID); err != nil {
		return HTTPMockReceipt{}, err
	}
	if err := o.verifyHTTPMockNetworks(ctx, canonical, container, aliases); err != nil {
		return HTTPMockReceipt{}, err
	}
	if err := o.launchHTTPMockServer(ctx, containerID, path, configJSON); err != nil {
		cleanupErr := o.cleanupHTTPMockByName(context.Background(), canonical, runtimeID, configHash, containerName)
		return HTTPMockReceipt{}, errors.Join(err, cleanupErr)
	}
	container, found, err = o.inspectContainer(ctx, containerName)
	if err != nil || !found {
		cleanupErr := o.cleanupHTTPMockByName(context.Background(), canonical, runtimeID, configHash, containerName)
		if err == nil {
			err = errors.New("HTTP mock container disappeared before receipt verification")
		}
		return HTTPMockReceipt{}, errors.Join(err, cleanupErr)
	}
	if err := verifyHTTPMockContainerIdentity(container, canonical, runtimeID); err != nil {
		return HTTPMockReceipt{}, err
	}
	if err := o.verifyHTTPMockNetworks(ctx, canonical, container, aliases); err != nil {
		return HTTPMockReceipt{}, err
	}
	return makeHTTPMockReceipt(canonical, runtimeID, aliases, len(config.Fixtures), configHash), nil
}

func (o *Owner) launchHTTPMockServer(ctx context.Context, containerID, configPath string, configJSON []byte) error {
	quiet, ok := o.runner().(quietCommandRunner)
	if !ok {
		return errors.New("Docker runner does not support bounded HTTP mock stdin/output")
	}
	if _, err := quiet.RunInputQuiet(ctx, configJSON, "exec", "--interactive", "--user", "node", containerID, "node", "-e", stageHTTPMockProgram, configPath); err != nil {
		return fmt.Errorf("stage HTTP mock fixture config in tmpfs: %w", err)
	}
	if _, err := o.runner().Run(ctx, "exec", "--detach", "--user", "node", containerID, "node", "-e", syntheticHTTPMockServerProgram, configPath); err != nil {
		if probeErr := o.probeHTTPMock(ctx, containerID); probeErr == nil {
			return nil
		}
		return fmt.Errorf("start fixed HTTP mock source server: %w", err)
	}
	if err := o.probeHTTPMock(ctx, containerID); err != nil {
		return err
	}
	return nil
}

func (o *Owner) StopHTTPMock(ctx context.Context, sandboxID, runtimeID string) (HTTPMockStopReceipt, error) {
	canonical, err := normalizeSandboxID(sandboxID)
	if err != nil {
		return HTTPMockStopReceipt{}, err
	}
	if !httpMockRuntimePattern(canonical, runtimeID) {
		return HTTPMockStopReceipt{}, ErrForeignObject
	}
	containerName := httpMockContainerName(canonical)
	container, found, err := o.inspectContainer(ctx, containerName)
	if err != nil || !found {
		return HTTPMockStopReceipt{SandboxID: canonical, CompanionRuntimeID: runtimeID, Status: "stopped"}, err
	}
	if err := verifyHTTPMockContainerIdentity(container, canonical, runtimeID); err != nil {
		return HTTPMockStopReceipt{}, err
	}
	aliases := parseHTTPMockAliases(container.Config.Labels[labelHTTPMockAliases])
	if err := o.verifyHTTPMockNetworks(ctx, canonical, container, aliases); err != nil {
		return HTTPMockStopReceipt{}, err
	}
	if container.State.Running {
		if _, err := o.runner().Run(ctx, "stop", "--time", "5", container.ID); err != nil {
			return HTTPMockStopReceipt{}, fmt.Errorf("stop owned HTTP mock: %w", err)
		}
	}
	container, found, err = o.inspectContainer(ctx, containerName)
	if err != nil || !found {
		return HTTPMockStopReceipt{}, err
	}
	if err := verifyHTTPMockContainerIdentity(container, canonical, runtimeID); err != nil {
		return HTTPMockStopReceipt{}, err
	}
	if err := o.verifyHTTPMockNetworks(ctx, canonical, container, aliases); err != nil {
		return HTTPMockStopReceipt{}, err
	}
	if _, err := o.runner().Run(ctx, "rm", "--volumes", container.ID); err != nil && !isNotFound(err) {
		return HTTPMockStopReceipt{}, fmt.Errorf("remove owned HTTP mock: %w", err)
	}
	return HTTPMockStopReceipt{SandboxID: canonical, CompanionRuntimeID: runtimeID, Status: "stopped"}, nil
}

const (
	labelHTTPMockConfigHash  = "com.vrooli.http-mock.config-sha256"
	labelHTTPMockAliases     = "com.vrooli.http-mock.aliases"
	httpMockKeepaliveProgram = `setInterval(()=>{},0x7fffffff)`
)

func (o *Owner) verifyHTTPMockNetworks(ctx context.Context, sandboxID string, container containerInspect, aliases []string) error {
	_, taskNetwork := objectNames(sandboxID)
	if err := verifyHTTPMockNetworksOnly(container, taskNetwork); err != nil {
		return err
	}
	if _, found, err := o.inspectNetwork(ctx, taskNetwork, sandboxID, runtimeIDFor(sandboxID)); err != nil || !found {
		if err == nil {
			err = errors.New("task-owned n8n network is missing")
		}
		return err
	}
	state, attached := container.NetworkSettings.Networks[taskNetwork]
	if !attached {
		return errors.New("HTTP mock is not attached to its task network")
	}
	if !validActualHTTPMockAliases(aliases, state.Aliases, httpMockContainerName(sandboxID)) {
		return errors.New("HTTP mock Docker aliases do not match its owner receipt")
	}
	for _, bindings := range container.NetworkSettings.Ports {
		if len(bindings) > 0 {
			return errors.New("HTTP mock unexpectedly publishes a host port")
		}
	}
	return nil
}

func verifyHTTPMockNetworksOnly(container containerInspect, taskNetwork string) error {
	for network := range container.NetworkSettings.Networks {
		if network != taskNetwork && network != "none" {
			return fmt.Errorf("%w: HTTP mock has egress-capable network %q", ErrCompanionDenied, network)
		}
	}
	return nil
}

func (o *Owner) probeHTTPMock(ctx context.Context, containerID string) error {
	probeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	quiet, ok := o.runner().(quietCommandRunner)
	if !ok {
		return errors.New("Docker runner does not support bounded HTTP mock readiness output")
	}
	output, exceeded, err := quiet.RunStdoutLimitQuiet(probeCtx, 128, "exec", containerID, "node", "-e", syntheticHTTPMockProbeProgram)
	if err != nil || exceeded || strings.TrimSpace(string(output)) != "ready" {
		if err == nil {
			err = errors.New("fixed mock health probe failed or exceeded output limit")
		}
		return fmt.Errorf("HTTP mock server is not ready: %w", err)
	}
	return nil
}

func (o *Owner) cleanupHTTPMockByName(ctx context.Context, sandboxID, runtimeID, configHash, name string) error {
	container, found, err := o.inspectContainer(ctx, name)
	if err != nil || !found {
		return err
	}
	if err := verifyHTTPMockContainerIdentity(container, sandboxID, runtimeID); err != nil {
		return err
	}
	if container.Config.Labels[labelHTTPMockConfigHash] != configHash {
		return ErrForeignObject
	}
	if err := o.verifyHTTPMockNetworks(ctx, sandboxID, container, parseHTTPMockAliases(container.Config.Labels[labelHTTPMockAliases])); err != nil {
		return err
	}
	if container.State.Running {
		if _, err := o.runner().Run(ctx, "stop", "--time", "5", container.ID); err != nil {
			return err
		}
	}
	container, found, err = o.inspectContainer(ctx, name)
	if err != nil || !found {
		return err
	}
	if err := verifyHTTPMockContainerIdentity(container, sandboxID, runtimeID); err != nil {
		return err
	}
	if err := o.verifyHTTPMockNetworks(ctx, sandboxID, container, parseHTTPMockAliases(container.Config.Labels[labelHTTPMockAliases])); err != nil {
		return err
	}
	_, err = o.runner().Run(ctx, "rm", "--volumes", container.ID)
	return err
}

func verifyHTTPMockLabels(labels map[string]string, sandboxID, runtimeID string) error {
	if labels[labelResource] != "n8n" || labels[labelSandboxID] != sandboxID || labels[labelRuntimeID] != runtimeID || labels[labelObjectKind] != "http-mock" ||
		labels[labelCompanionResource] != "n8n-http-mock" || labels[labelSandboxOwner] != sandboxID || labels[labelSandboxRunID] != runtimeID || labels[labelSandboxDecl] != "true" {
		return ErrForeignObject
	}
	digest := labels[labelHTTPMockConfigHash]
	aliases := parseHTTPMockAliases(labels[labelHTTPMockAliases])
	if !validHTTPMockDigest(digest) || runtimeID != httpMockRuntimeIDFor(sandboxID, digest) || len(aliases) == 0 || !sameStrings(aliases, fixtureAliasesOnly(aliases)) {
		return ErrForeignObject
	}
	for _, alias := range aliases {
		if !httpMockHostPattern.MatchString(alias) {
			return ErrForeignObject
		}
	}
	return nil
}

func verifyHTTPMockContainerIdentity(container containerInspect, sandboxID, runtimeID string) error {
	if err := verifyHTTPMockLabels(container.Config.Labels, sandboxID, runtimeID); err != nil {
		return err
	}
	if container.Config.Image != ImageRef {
		return fmt.Errorf("%w: HTTP mock container does not use the pinned n8n image", ErrForeignObject)
	}
	return nil
}

func validHTTPMockDigest(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func httpMockRuntimeIDFor(sandboxID, configHash string) string {
	return "n8n-httpmock:" + sandboxID + ":" + configHash
}

func httpMockRuntimePattern(sandboxID, value string) bool {
	prefix := "n8n-httpmock:" + sandboxID + ":"
	return strings.HasPrefix(value, prefix) && validHTTPMockDigest(strings.TrimPrefix(value, prefix))
}

func httpMockContainerName(sandboxID string) string {
	return "vrooli-n8n-httpmock-" + strings.ReplaceAll(sandboxID, "-", "")
}

func httpMockConfigPath(sandboxID, digest string) string {
	return "/tmp/.vrooli-http-mock-" + strings.ReplaceAll(sandboxID, "-", "")[:12] + "-" + digest[:12] + ".json"
}

func makeHTTPMockReceipt(sandboxID, runtimeID string, aliases []string, count int, digest string) HTTPMockReceipt {
	return HTTPMockReceipt{SandboxID: sandboxID, CompanionRuntimeID: runtimeID, NetworkAliases: append([]string(nil), aliases...), Port: syntheticHTTPMockPort, Status: "attached", NoEgressVerified: true, FixtureCount: count, ConfigSHA256: digest}
}

func fixtureHosts(fixtures []HTTPMockFixture) []string {
	seen := map[string]struct{}{}
	for _, fixture := range fixtures {
		seen[fixture.Host] = struct{}{}
	}
	result := make([]string, 0, len(seen))
	for host := range seen {
		result = append(result, host)
	}
	sort.Strings(result)
	return result
}

func parseHTTPMockAliases(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	sort.Strings(parts)
	return parts
}

func fixtureAliasesOnly(aliases []string) []string {
	copy := append([]string(nil), aliases...)
	sort.Strings(copy)
	unique := copy[:0]
	for _, alias := range copy {
		if len(unique) == 0 || unique[len(unique)-1] != alias {
			unique = append(unique, alias)
		}
	}
	return unique
}

func sameStrings(expected, actual []string) bool {
	if len(expected) != len(actual) {
		return false
	}
	for i := range expected {
		if expected[i] != actual[i] {
			return false
		}
	}
	return true
}

func validActualHTTPMockAliases(expected, actual []string, containerAlias string) bool {
	want := make(map[string]struct{}, len(expected))
	for _, alias := range expected {
		want[alias] = struct{}{}
	}
	got := map[string]struct{}{}
	for _, alias := range actual {
		if alias == containerAlias {
			continue
		}
		if _, allowed := want[alias]; !allowed {
			return false
		}
		got[alias] = struct{}{}
	}
	if len(got) != len(want) {
		return false
	}
	for alias := range want {
		if _, ok := got[alias]; !ok {
			return false
		}
	}
	return true
}

func HTTPMockOutputBound() int { return maxHTTPMockOutputBytes }
