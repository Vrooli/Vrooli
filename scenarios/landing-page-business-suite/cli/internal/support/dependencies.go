package support

import (
	"bytes"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	"landing-page-business-suite/cli/internal/clock"
)

type Dependencies struct {
	Core  func() *cliapp.ScenarioApp
	Clock clock.Clock
}

func (d Dependencies) Now() time.Time {
	if d.Clock == nil {
		return clock.System{}.Now()
	}
	return d.Clock.Now()
}

type EndpointDef struct {
	Name         string
	Method       string
	Path         string
	Description  string
	Root         bool
	AllowRawPath bool
	Stream       bool
}

type HealthResponse struct {
	Status     string                 `json:"status"`
	Service    string                 `json:"service"`
	Version    string                 `json:"version"`
	Timestamp  string                 `json:"timestamp"`
	Details    map[string]interface{} `json:"details"`
	Readiness  bool                   `json:"readiness"`
	Deps       map[string]string      `json:"dependencies"`
	Error      string                 `json:"error"`
	Message    string                 `json:"message"`
	Operations map[string]interface{} `json:"operations"`
}

type UsageHealthResponse struct {
	Healthy               bool   `json:"healthy"`
	DatabaseConnected     bool   `json:"database_connected"`
	ServiceAuthConfigured bool   `json:"service_auth_configured"`
	ServiceAuthMode       string `json:"service_auth_mode"`
}

type DeployReadinessCheck struct {
	Name     string `json:"name"`
	Required bool   `json:"required"`
	Passed   bool   `json:"passed"`
	Blocked  bool   `json:"blocked,omitempty"`
	Detail   string `json:"detail"`
}

type DeployReadinessReport struct {
	Ready      bool                   `json:"ready"`
	ProfileTag string                 `json:"profile_tag,omitempty"`
	ProfileID  string                 `json:"profile_id,omitempty"`
	Domain     string                 `json:"domain,omitempty"`
	Checks     []DeployReadinessCheck `json:"checks"`
	NextSteps  []string               `json:"next_steps,omitempty"`
	CheckedAt  string                 `json:"checked_at"`
}

type AdminLoginResponse struct {
	Email         string `json:"email,omitempty"`
	Authenticated bool   `json:"authenticated"`
	ResetEnabled  bool   `json:"reset_enabled"`
}

type AdminSessionConfig struct {
	APIBase   string     `json:"api_base"`
	Session   string     `json:"session"`
	Email     string     `json:"email,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type adminSessionStore struct {
	Sessions map[string]AdminSessionConfig `json:"sessions,omitempty"`
}

type OptionalString struct {
	Value string
	IsSet bool
}

type boolFlag interface {
	IsBoolFlag() bool
}

var pathParamRegex = regexp.MustCompile(`\{([^}]+)\}`)

func (o *OptionalString) String() string {
	return o.Value
}

func (o *OptionalString) Set(value string) error {
	o.Value = value
	o.IsSet = true
	return nil
}

func (d Dependencies) ScenarioApp() *cliapp.ScenarioApp {
	if d.Core == nil {
		return nil
	}
	return d.Core()
}

func (d Dependencies) EndpointCommands(defs []EndpointDef) []cliapp.Command {
	commands := make([]cliapp.Command, 0, len(defs))
	for _, def := range defs {
		commands = append(commands, d.EndpointCommand(def))
	}
	return commands
}

func (d Dependencies) EndpointCommand(def EndpointDef) cliapp.Command {
	return cliapp.Command{
		Name:        def.Name,
		NeedsAPI:    true,
		Description: def.Description,
		Run: func(args []string) error {
			return d.RunEndpoint(def, args)
		},
	}
}

func (d Dependencies) RunEndpoint(def EndpointDef, args []string) error {
	fs := flag.NewFlagSet(def.Name, flag.ContinueOnError)
	var queries cliutil.StringList
	fs.Var(&queries, "query", "Query parameters (key=value or key=value&key2=value2). Repeatable.")
	body := fs.String("body", "", "JSON body payload or @file.json")
	jsonOut := cliutil.JSONFlag(fs)
	if err := ParseFlagSetInterspersed(fs, args); err != nil {
		return err
	}

	path, argNames, err := ResolvePath(def.Path, fs.Args(), def.AllowRawPath)
	if err != nil {
		return fmt.Errorf("usage: %s%s [--query k=v] [--body @file.json] [--json]", def.Name, FormatArgUsage(argNames))
	}

	payload, err := ParseBody(*body)
	if err != nil {
		return err
	}

	query, err := ParseQueries(queries.Values())
	if err != nil {
		return err
	}

	if def.Stream {
		return d.StreamEndpoint(def, path, query, payload)
	}

	resp, err := d.Request(def, path, query, payload)
	if err != nil {
		return err
	}

	cliutil.PrintJSON(resp)
	if *jsonOut {
		return nil
	}
	return nil
}

func (d Dependencies) Request(def EndpointDef, path string, query url.Values, payload []byte) ([]byte, error) {
	if def.Root {
		return d.ScenarioApp().RequestRoot(def.Method, path, query, rawBody(payload))
	}
	if strings.HasPrefix(path, "/admin/") || path == "/admin" {
		return d.RequestAdmin(def.Method, path, query, payload)
	}
	return d.ScenarioApp().Request(def.Method, path, query, rawBody(payload))
}

func (d Dependencies) StreamEndpoint(def EndpointDef, path string, query url.Values, payload []byte) error {
	urlString, err := d.ResolveURL(path, def.Root, query)
	if err != nil {
		return err
	}

	var body io.Reader
	if payload != nil {
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequest(def.Method, urlString, body)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "text/event-stream")
	for key, value := range d.ScenarioApp().APIClient.AuthHeaders() {
		req.Header.Set(key, value)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		data, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf("read response: %w", readErr)
		}
		return cliutil.ParseAPIError(resp.StatusCode, data)
	}

	_, err = io.Copy(os.Stdout, resp.Body)
	return err
}

func (d Dependencies) ResolveURL(path string, root bool, query url.Values) (string, error) {
	core := d.ScenarioApp()
	if core == nil {
		return "", fmt.Errorf("scenario app is not initialized")
	}
	base := strings.TrimRight(strings.TrimSpace(core.APIBase()), "/")
	if root {
		base = strings.TrimRight(strings.TrimSpace(core.APIRootBase()), "/")
	}
	if base == "" {
		return "", fmt.Errorf("api base URL is empty")
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	endpoint := base + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}
	return endpoint, nil
}

func (d Dependencies) AdminSessionConfigFile() (*cliutil.ConfigFile, error) {
	core := d.ScenarioApp()
	if core == nil || core.ConfigFile == nil {
		return nil, fmt.Errorf("config not initialized")
	}
	dir := filepath.Dir(core.ConfigFile.Path)
	return cliutil.NewConfigFile(filepath.Join(dir, "admin_session.json"))
}

func (d Dependencies) CurrentAPIBase() string {
	core := d.ScenarioApp()
	if core == nil || core.APIClient == nil {
		return ""
	}
	return NormalizeAPIBase(core.APIClient.BaseURL())
}

func (d Dependencies) LoadAdminSession() (AdminSessionConfig, error) {
	cfgFile, err := d.AdminSessionConfigFile()
	if err != nil {
		return AdminSessionConfig{}, err
	}

	data, readErr := os.ReadFile(cfgFile.Path)
	if readErr != nil {
		if os.IsNotExist(readErr) {
			return AdminSessionConfig{}, nil
		}
		return AdminSessionConfig{}, fmt.Errorf("read config file: %w", readErr)
	}
	if len(data) == 0 {
		return AdminSessionConfig{}, nil
	}

	base := d.CurrentAPIBase()

	var store adminSessionStore
	if err := json.Unmarshal(data, &store); err == nil && len(store.Sessions) > 0 {
		if base == "" {
			return AdminSessionConfig{}, fmt.Errorf("api base URL is empty; configure an API base first")
		}
		cfg := store.Sessions[base]
		if strings.TrimSpace(cfg.Session) == "" {
			return AdminSessionConfig{}, nil
		}
		if cfg.ExpiresAt != nil && d.Now().After(cfg.ExpiresAt.UTC()) {
			delete(store.Sessions, base)
			_ = cfgFile.Save(store)
			return AdminSessionConfig{}, nil
		}
		cfg.APIBase = base
		return cfg, nil
	}

	var legacy AdminSessionConfig
	if err := json.Unmarshal(data, &legacy); err != nil {
		return AdminSessionConfig{}, fmt.Errorf("parse config: %w", err)
	}
	if strings.TrimSpace(legacy.Session) == "" {
		return AdminSessionConfig{}, nil
	}
	if base == "" {
		return AdminSessionConfig{}, fmt.Errorf("api base URL is empty; configure an API base first")
	}
	if legacy.APIBase != "" && !strings.EqualFold(NormalizeAPIBase(legacy.APIBase), base) {
		return AdminSessionConfig{}, nil
	}
	if legacy.ExpiresAt != nil && d.Now().After(legacy.ExpiresAt.UTC()) {
		_ = cfgFile.Save(AdminSessionConfig{})
		return AdminSessionConfig{}, nil
	}
	legacy.APIBase = base
	return legacy, nil
}

func (d Dependencies) SaveAdminSession(cfg AdminSessionConfig) error {
	cfgFile, err := d.AdminSessionConfigFile()
	if err != nil {
		return err
	}
	base := d.CurrentAPIBase()
	if base == "" {
		return fmt.Errorf("api base URL is empty; configure an API base first")
	}

	store := adminSessionStore{Sessions: map[string]AdminSessionConfig{}}
	if data, err := os.ReadFile(cfgFile.Path); err == nil && len(data) > 0 {
		var loaded adminSessionStore
		if jsonErr := json.Unmarshal(data, &loaded); jsonErr == nil && len(loaded.Sessions) > 0 {
			store = loaded
			if store.Sessions == nil {
				store.Sessions = map[string]AdminSessionConfig{}
			}
		} else {
			var legacy AdminSessionConfig
			if jsonErr := json.Unmarshal(data, &legacy); jsonErr == nil && strings.TrimSpace(legacy.Session) != "" {
				legacyBase := NormalizeAPIBase(legacy.APIBase)
				if legacyBase != "" {
					store.Sessions[legacyBase] = legacy
				}
			}
		}
	}

	cfg.APIBase = base
	cfg.UpdatedAt = d.Now().UTC()
	store.Sessions[base] = cfg
	return cfgFile.Save(store)
}

func (d Dependencies) ClearAdminSession() error {
	cfgFile, err := d.AdminSessionConfigFile()
	if err != nil {
		return err
	}
	base := d.CurrentAPIBase()
	if base == "" {
		return cfgFile.Save(AdminSessionConfig{})
	}

	data, err := os.ReadFile(cfgFile.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read config file: %w", err)
	}
	if len(data) == 0 {
		return nil
	}

	var store adminSessionStore
	if err := json.Unmarshal(data, &store); err == nil && len(store.Sessions) > 0 {
		delete(store.Sessions, base)
		return cfgFile.Save(store)
	}
	return cfgFile.Save(AdminSessionConfig{})
}

func (d Dependencies) RequestAdmin(method, pathValue string, query url.Values, payload []byte) ([]byte, error) {
	session, err := d.LoadAdminSession()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(session.Session) == "" {
		return nil, fmt.Errorf("admin session not configured. Run admin-login first")
	}

	endpoint, err := d.ResolveURL(pathValue, false, query)
	if err != nil {
		return nil, err
	}

	var body io.Reader
	if len(payload) > 0 {
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequest(method, endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if len(payload) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range d.ScenarioApp().APIClient.AuthHeaders() {
		req.Header.Set(key, value)
	}
	req.Header.Set("Cookie", fmt.Sprintf("admin_session=%s", session.Session))

	client := &http.Client{Timeout: d.ScenarioApp().HTTPClient.Timeout()}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		_ = d.ClearAdminSession()
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, cliutil.ParseAPIError(resp.StatusCode, data)
	}
	return data, nil
}

func (d Dependencies) RequestRemoteProxy(profileID, method, pathValue string, query url.Values, headers map[string]string, body []byte) ([]byte, error) {
	profileID = strings.TrimSpace(profileID)
	if profileID == "" {
		return nil, fmt.Errorf("remote profile id is required")
	}
	if strings.TrimSpace(method) == "" {
		return nil, fmt.Errorf("proxy method is required")
	}
	if strings.TrimSpace(pathValue) == "" {
		return nil, fmt.Errorf("proxy path is required")
	}

	payload := map[string]interface{}{
		"method": method,
		"path":   pathValue,
	}
	if len(query) > 0 {
		payload["query"] = FlattenQueryValues(query)
	}
	if len(headers) > 0 {
		payload["headers"] = headers
	}
	if len(body) > 0 {
		payload["body"] = json.RawMessage(body)
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode proxy payload: %w", err)
	}
	return d.RequestAdmin("POST", "/admin/remote-profiles/"+url.PathEscape(profileID)+"/proxy", nil, payloadBytes)
}

func (d Dependencies) RequestAdminJSON(profileID, method, pathValue string, payload interface{}) ([]byte, error) {
	var body []byte
	if payload != nil {
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("encode request: %w", err)
		}
		body = payloadBytes
	}
	if strings.TrimSpace(profileID) == "" {
		return d.RequestAdmin(method, pathValue, nil, body)
	}
	headers := map[string]string{}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	return d.RequestRemoteProxy(profileID, method, pathValue, nil, headers, body)
}

func (d Dependencies) ResolveRemoteProfileIDByTag(tag string) (string, error) {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return "", fmt.Errorf("--tag is required")
	}

	resp, err := d.RequestAdmin("GET", "/admin/remote-profiles", nil, nil)
	if err != nil {
		return "", err
	}

	var payload struct {
		Profiles []struct {
			ID  json.RawMessage `json:"id"`
			Tag string          `json:"tag"`
		} `json:"profiles"`
	}
	if err := json.Unmarshal(resp, &payload); err != nil {
		return "", fmt.Errorf("parse remote profiles list: %w", err)
	}

	var matchedID string
	for _, profile := range payload.Profiles {
		if strings.TrimSpace(profile.Tag) != tag {
			continue
		}
		candidateID, err := NormalizeRemoteProfileID(profile.ID)
		if err != nil {
			return "", fmt.Errorf("parse remote profile id for tag %q: %w", tag, err)
		}
		if matchedID != "" && matchedID != candidateID {
			return "", fmt.Errorf("remote profile tag %q maps to multiple ids; run remote-profiles-list --json and fix duplicates", tag)
		}
		matchedID = candidateID
	}
	if matchedID == "" {
		return "", fmt.Errorf("remote profile tag %q not found; run remote-profiles-list to inspect available tags", tag)
	}
	return matchedID, nil
}

func (d Dependencies) ParseRemoteProfileSelector(cmdName string, args []string) (string, bool, error) {
	fs := flag.NewFlagSet(cmdName, flag.ContinueOnError)
	profileIDFlag := fs.String("profile-id", "", "Remote profile id (alternative to positional <id>)")
	profileTag := fs.String("profile-tag", "", "Remote profile tag (resolves id automatically)")
	tagAlias := fs.String("tag", "", "Alias for --profile-tag")
	jsonOut := cliutil.JSONFlag(fs)
	if err := ParseFlagSetInterspersed(fs, args); err != nil {
		return "", false, err
	}
	if len(fs.Args()) > 1 {
		return "", false, fmt.Errorf("usage: %s <id> [--profile-id <id> | --profile-tag <tag>] [--json]", cmdName)
	}

	positionalProfileID := ""
	if len(fs.Args()) == 1 {
		positionalProfileID = strings.TrimSpace(fs.Args()[0])
	}
	flagProfileID := strings.TrimSpace(*profileIDFlag)
	tagValue := strings.TrimSpace(*profileTag)
	tagAliasValue := strings.TrimSpace(*tagAlias)
	if tagValue == "" && tagAliasValue != "" {
		tagValue = tagAliasValue
	}
	if tagValue != "" && tagAliasValue != "" && tagValue != tagAliasValue {
		return "", false, fmt.Errorf("use only one of --profile-tag or --tag (values differ)")
	}
	if positionalProfileID != "" && flagProfileID != "" {
		return "", false, fmt.Errorf("use either positional <id> or --profile-id, not both")
	}
	if tagValue != "" && (positionalProfileID != "" || flagProfileID != "") {
		return "", false, fmt.Errorf("use either --profile-tag/--tag or an explicit profile id (--profile-id or positional <id>)")
	}

	profileID := positionalProfileID
	if profileID == "" {
		profileID = flagProfileID
	}
	if profileID == "" && tagValue != "" {
		resolvedProfileID, err := d.ResolveRemoteProfileIDByTag(tagValue)
		if err != nil {
			return "", false, err
		}
		profileID = resolvedProfileID
	}
	if profileID == "" {
		return "", false, fmt.Errorf("usage: %s <id> [--profile-id <id> | --profile-tag <tag>] [--json]", cmdName)
	}

	return profileID, *jsonOut, nil
}

func NormalizeAPIBase(value string) string {
	return strings.TrimRight(strings.TrimSpace(value), "/")
}

func ValidateRemoteProfileAPIBase(value string) (string, error) {
	trimmed := strings.TrimRight(strings.TrimSpace(value), "/")
	parsed, err := url.ParseRequestURI(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", operationalError(cliapp.OperationalReport{
			Status: []string{"Status: INVALID INPUT"},
			Triage: []cliapp.TriageGroup{{Heading: "Input", Items: []string{"[FAIL] api_base_url: --api-base must be an absolute URL"}}},
			NextSteps: []string{
				"Use canonical form: --api-base https://<domain>/api/v1",
				"Re-run remote-profiles-create with the corrected value",
			},
		})
	}
	if !strings.HasSuffix(trimmed, "/api/v1") {
		return "", operationalError(cliapp.OperationalReport{
			Status: []string{"Status: INVALID INPUT"},
			Triage: []cliapp.TriageGroup{{Heading: "Input", Items: []string{"[FAIL] api_base_format: --api-base must end with /api/v1"}}},
			NextSteps: []string{
				"Use canonical form: --api-base https://<domain>/api/v1",
				"Re-run remote-profiles-create with the corrected value",
			},
		})
	}
	return trimmed, nil
}

func NormalizeRemoteProfileID(raw json.RawMessage) (string, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return "", fmt.Errorf("missing id")
	}

	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		asString = strings.TrimSpace(asString)
		if asString == "" {
			return "", fmt.Errorf("missing id")
		}
		return asString, nil
	}

	var asNumber json.Number
	if err := json.Unmarshal(raw, &asNumber); err == nil {
		s := strings.TrimSpace(asNumber.String())
		if s == "" {
			return "", fmt.Errorf("missing id")
		}
		return s, nil
	}

	return "", fmt.Errorf("unsupported id format %q", trimmed)
}

func ResolveSecretArg(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", nil
	}
	if strings.HasPrefix(trimmed, "@") {
		data, err := cliutil.ReadFileString(strings.TrimPrefix(trimmed, "@"))
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(data), nil
	}
	return trimmed, nil
}

func ParseFlagSetInterspersed(fs *flag.FlagSet, args []string) error {
	return fs.Parse(ReorderInterspersedArgs(fs, args))
}

func ReorderInterspersedArgs(fs *flag.FlagSet, args []string) []string {
	flags := make([]string, 0, len(args))
	positionals := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			positionals = append(positionals, args[i+1:]...)
			break
		}
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			positionals = append(positionals, arg)
			continue
		}

		name := ParseFlagTokenName(arg)
		flagDef := fs.Lookup(name)
		if flagDef == nil {
			flags = append(flags, arg)
			continue
		}

		flags = append(flags, arg)
		if strings.Contains(arg, "=") || IsBoolFlag(flagDef) {
			continue
		}
		if i+1 < len(args) {
			i++
			flags = append(flags, args[i])
		}
	}

	return append(flags, positionals...)
}

func ParseFlagTokenName(arg string) string {
	token := strings.TrimLeft(arg, "-")
	if token == "" {
		return ""
	}
	if idx := strings.IndexByte(token, '='); idx >= 0 {
		return token[:idx]
	}
	return token
}

func IsBoolFlag(f *flag.Flag) bool {
	if f == nil {
		return false
	}
	bf, ok := f.Value.(boolFlag)
	return ok && bf.IsBoolFlag()
}

func ResolvePath(template string, args []string, allowRaw bool) (string, []string, error) {
	matches := pathParamRegex.FindAllStringSubmatch(template, -1)
	argNames := make([]string, 0, len(matches))
	for _, match := range matches {
		argNames = append(argNames, match[1])
	}
	if len(args) < len(argNames) {
		return "", argNames, fmt.Errorf("missing arguments")
	}
	if len(args) > len(argNames) {
		return "", argNames, fmt.Errorf("too many arguments")
	}
	path := template
	for i, name := range argNames {
		replacement := strings.TrimSpace(args[i])
		if replacement == "" {
			return "", argNames, fmt.Errorf("empty %s", name)
		}
		if !allowRaw {
			replacement = url.PathEscape(replacement)
		}
		path = strings.Replace(path, "{"+name+"}", replacement, 1)
	}
	return path, argNames, nil
}

func FormatArgUsage(names []string) string {
	if len(names) == 0 {
		return ""
	}
	return " " + strings.Join(names, " ")
}

func ParseBody(value string) ([]byte, error) {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return nil, nil
	}
	if raw == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("read stdin: %w", err)
		}
		raw = strings.TrimSpace(string(data))
	} else if strings.HasPrefix(raw, "@") {
		path := strings.TrimSpace(strings.TrimPrefix(raw, "@"))
		if path == "" {
			return nil, fmt.Errorf("body file path is empty")
		}
		data, err := cliutil.ReadFileString(path)
		if err != nil {
			return nil, fmt.Errorf("read body file: %w", err)
		}
		raw = strings.TrimSpace(data)
	}
	if raw == "" {
		return nil, fmt.Errorf("body is empty")
	}
	var decoded interface{}
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return nil, fmt.Errorf("body must be valid JSON: %w", err)
	}
	return []byte(raw), nil
}

func ParseQueries(values []string) (url.Values, error) {
	if len(values) == 0 {
		return nil, nil
	}
	query := url.Values{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		parsed, err := url.ParseQuery(value)
		if err != nil {
			return nil, fmt.Errorf("invalid query %q: %w", value, err)
		}
		for key, vals := range parsed {
			for _, v := range vals {
				query.Add(key, v)
			}
		}
	}
	if len(query) == 0 {
		return nil, nil
	}
	return query, nil
}

func FlattenQueryValues(values url.Values) map[string]string {
	if len(values) == 0 {
		return nil
	}
	flat := make(map[string]string)
	for key, vals := range values {
		if len(vals) == 0 {
			continue
		}
		flat[key] = vals[0]
	}
	if len(flat) == 0 {
		return nil
	}
	return flat
}

func ParseKeyValuePairs(values []string) (map[string]string, error) {
	if len(values) == 0 {
		return nil, nil
	}
	pairs := map[string]string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		var key string
		var val string
		if strings.Contains(value, "=") {
			parts := strings.SplitN(value, "=", 2)
			key = strings.TrimSpace(parts[0])
			val = strings.TrimSpace(parts[1])
		} else if strings.Contains(value, ":") {
			parts := strings.SplitN(value, ":", 2)
			key = strings.TrimSpace(parts[0])
			val = strings.TrimSpace(parts[1])
		} else {
			return nil, fmt.Errorf("invalid pair %q: expected key=value", value)
		}
		if key == "" {
			return nil, fmt.Errorf("invalid pair %q: empty key", value)
		}
		pairs[key] = val
	}
	if len(pairs) == 0 {
		return nil, nil
	}
	return pairs, nil
}

func ComputeSHA512(filePath string) (string, error) {
	// #nosec G304 -- caller supplies an explicit local CLI file path for hashing.
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("open file for SHA512: %w", err)
	}
	defer f.Close()
	h := sha512.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("compute SHA512: %w", err)
	}
	return base64.StdEncoding.EncodeToString(h.Sum(nil)), nil
}

func NormalizeDownloadPlatform(raw string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch value {
	case "windows", "win":
		return "windows", nil
	case "mac", "macos", "osx":
		return "mac", nil
	case "linux":
		return "linux", nil
	case "":
		return "", fmt.Errorf("platform is required (windows, mac, linux)")
	default:
		return "", fmt.Errorf("unsupported platform %q (use windows, mac, or linux)", raw)
	}
}

func ResolveContentType(pathValue, override string) string {
	trimmed := strings.TrimSpace(override)
	if trimmed != "" {
		return trimmed
	}
	ext := strings.ToLower(filepath.Ext(pathValue))
	if ext != "" {
		if guessed := mime.TypeByExtension(ext); guessed != "" {
			return guessed
		}
	}
	return "application/octet-stream"
}

func DeriveCookieExpiry(cookie *http.Cookie) *time.Time {
	if cookie == nil {
		return nil
	}
	if !cookie.Expires.IsZero() {
		expiry := cookie.Expires.UTC()
		return &expiry
	}
	if cookie.MaxAge > 0 {
		expiry := time.Now().Add(time.Duration(cookie.MaxAge) * time.Second).UTC()
		return &expiry
	}
	return nil
}

func FindCookie(cookies []*http.Cookie, name string) *http.Cookie {
	for _, cookie := range cookies {
		if cookie != nil && cookie.Name == name {
			return cookie
		}
	}
	return nil
}

func BuildMultipartForm(filePath string, fieldName string, extraFields map[string]string) (*bytes.Buffer, string, error) {
	// #nosec G304 -- caller supplies an explicit local CLI file path for upload.
	file, err := os.Open(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile(fieldName, filepath.Base(filePath))
	if err != nil {
		return nil, "", fmt.Errorf("create form file: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, "", fmt.Errorf("read file: %w", err)
	}
	for key, value := range extraFields {
		if strings.TrimSpace(value) == "" {
			continue
		}
		if err := writer.WriteField(key, strings.TrimSpace(value)); err != nil {
			return nil, "", fmt.Errorf("write %s: %w", key, err)
		}
	}
	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("close writer: %w", err)
	}
	return &buf, writer.FormDataContentType(), nil
}

func RenderOperationalError(report cliapp.OperationalReport) error {
	return operationalError(report)
}

func rawBody(payload []byte) interface{} {
	if payload == nil {
		return nil
	}
	return json.RawMessage(payload)
}

func operationalError(report cliapp.OperationalReport) error {
	var buf bytes.Buffer
	if err := cliapp.RenderOperationalReport(&buf, report); err != nil {
		return err
	}
	return errors.New(strings.TrimSpace(buf.String()))
}
