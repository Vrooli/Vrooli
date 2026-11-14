// Package policy owns the Ollaya resource's decision role catalog: which
// upstream answers each role, with which model, and which credentials a hosted
// upstream needs. ai-gateway reads it through `policy roles` and
// `policy resolve`; it never sees a credential value.
package policy

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	LocalityLocal  = "local"
	LocalityRemote = "remote"

	// ProtocolOllaya is the daemon's native /api/decide: the TypeSafe answer
	// shape plus timings and state_truncated.
	ProtocolOllaya = "ollaya"
	// ProtocolSystemOne is TypeSafe's hosted wire format, served unchanged by
	// TypeSafe, Eden AI and Cloudflare at different paths.
	ProtocolSystemOne = "system-one"

	// EnvelopeCloudflare wraps the System One body in Cloudflare's
	// {result, success, errors} response envelope.
	EnvelopeCloudflare = "cloudflare"

	credentialBearer  = "bearer"
	credentialAccount = "account"

	// Capability is the single capability every decision role declares, so the
	// gateway can match a decision request without reading model names.
	Capability = "decide"
)

type Policy struct {
	SchemaVersion string              `json:"schema_version"`
	Upstreams     map[string]Upstream `json:"upstreams"`
	Roles         map[string]Role     `json:"roles"`
	Models        map[string]Model    `json:"models"`
}

type Upstream struct {
	Locality    string       `json:"locality"`
	Protocol    string       `json:"protocol"`
	Endpoint    string       `json:"endpoint"`
	Envelope    string       `json:"envelope"`
	Credentials []Credential `json:"credentials"`
	Description string       `json:"description"`
}

// Credential names the process-scoped variable the control plane injects. The
// logical identity is carried so `policy roles` can tell an operator exactly
// what to provision.
type Credential struct {
	Env       string `json:"env"`
	LogicalID string `json:"logical_id"`
	Field     string `json:"field"`
	Use       string `json:"use"`
}

type Role struct {
	Upstream    string `json:"upstream"`
	Model       string `json:"model"`
	Description string `json:"description"`
}

// Model is a locally served decision model pinned by its registry manifest
// digest. Hosted models are not listed: the hosted upstream owns them.
type Model struct {
	Digest      string   `json:"digest"`
	Routes      []string `json:"routes"`
	DefaultPull bool     `json:"default_pull"`
	License     string   `json:"license"`
	DiskBytes   int64    `json:"disk_bytes"`
}

// Resolution is one role resolved against the current process environment.
// Token and AccountID are values, never serialized.
type Resolution struct {
	Role     string `json:"role"`
	Upstream string `json:"upstream"`
	Protocol string `json:"protocol"`
	Locality string `json:"locality"`
	Model    string `json:"model"`
	Envelope string `json:"envelope,omitempty"`
	// Endpoint is empty for the local upstream; the daemon URL comes from the
	// resource's exported environment at call time.
	Endpoint string `json:"endpoint,omitempty"`
	// Missing lists credential variables the upstream needs and the process
	// does not have. A non-empty list means the role is unconfigured.
	Missing   []string `json:"missing_credentials,omitempty"`
	Token     string   `json:"-"`
	AccountID string   `json:"-"`
}

func (r Resolution) Configured() bool { return len(r.Missing) == 0 }

func Load(path string) (Policy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Policy{}, fmt.Errorf("read ollaya model policy: %w", err)
	}
	var policy Policy
	if err := json.Unmarshal(data, &policy); err != nil {
		return Policy{}, fmt.Errorf("parse ollaya model policy: %w", err)
	}
	if err := policy.Validate(); err != nil {
		return Policy{}, err
	}
	return policy, nil
}

func (p Policy) Validate() error {
	if strings.TrimSpace(p.SchemaVersion) == "" || len(p.Roles) == 0 || len(p.Upstreams) == 0 {
		return errors.New("ollaya model policy requires schema_version, upstreams, and roles")
	}
	for name, upstream := range p.Upstreams {
		if err := upstream.validate(name); err != nil {
			return err
		}
	}
	for name, role := range p.Roles {
		upstream, ok := p.Upstreams[role.Upstream]
		if !ok {
			return fmt.Errorf("role %q references unknown upstream %q", name, role.Upstream)
		}
		if strings.TrimSpace(role.Model) == "" {
			return fmt.Errorf("role %q requires a model", name)
		}
		if upstream.Locality == LocalityLocal {
			if _, ok := p.Models[role.Model]; !ok {
				return fmt.Errorf("local role %q references unpinned model %q", name, role.Model)
			}
		}
	}
	for name, model := range p.Models {
		if !isDigest(model.Digest) {
			return fmt.Errorf("model %q requires a 64-character manifest digest", name)
		}
		for _, route := range model.Routes {
			if _, ok := p.Models[route]; !ok {
				return fmt.Errorf("router model %q routes to unpinned model %q", name, route)
			}
		}
	}
	return nil
}

func (u Upstream) validate(name string) error {
	switch u.Locality {
	case LocalityLocal, LocalityRemote:
	default:
		return fmt.Errorf("upstream %q locality %q must be local or remote", name, u.Locality)
	}
	switch u.Protocol {
	case ProtocolOllaya:
		if u.Locality != LocalityLocal || u.Endpoint != "" || len(u.Credentials) != 0 {
			return fmt.Errorf("upstream %q: the ollaya protocol is the local daemon only and carries no endpoint or credentials", name)
		}
	case ProtocolSystemOne:
		if u.Locality != LocalityRemote {
			return fmt.Errorf("upstream %q: system-one upstreams are hosted", name)
		}
		parsed, err := url.Parse(strings.ReplaceAll(u.Endpoint, "{account}", "account"))
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
			return fmt.Errorf("upstream %q requires an https endpoint", name)
		}
		bearer := 0
		for _, credential := range u.Credentials {
			switch credential.Use {
			case credentialBearer:
				bearer++
			case credentialAccount:
				if !strings.Contains(u.Endpoint, "{account}") {
					return fmt.Errorf("upstream %q declares an account credential but its endpoint has no {account}", name)
				}
			default:
				return fmt.Errorf("upstream %q credential use %q must be bearer or account", name, credential.Use)
			}
			if strings.TrimSpace(credential.Env) == "" {
				return fmt.Errorf("upstream %q credential requires env", name)
			}
		}
		if bearer != 1 {
			return fmt.Errorf("upstream %q requires exactly one bearer credential", name)
		}
	default:
		return fmt.Errorf("upstream %q protocol %q is unsupported", name, u.Protocol)
	}
	if u.Envelope != "" && u.Envelope != EnvelopeCloudflare {
		return fmt.Errorf("upstream %q envelope %q is unsupported", name, u.Envelope)
	}
	return nil
}

func (p Policy) RoleNames() []string {
	names := make([]string, 0, len(p.Roles))
	for name := range p.Roles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// DefaultPulls returns the pinned models pulled by `models pull` with no
// argument. Routers sort first so their tag-based target pulls are then
// overwritten by the targets' own digest-pinned pulls.
func (p Policy) DefaultPulls() []string {
	var names []string
	for name, model := range p.Models {
		if model.DefaultPull {
			names = append(names, name)
		}
	}
	sort.Slice(names, func(i, j int) bool {
		ri, rj := len(p.Models[names[i]].Routes) > 0, len(p.Models[names[j]].Routes) > 0
		if ri != rj {
			return ri
		}
		return names[i] < names[j]
	})
	return names
}

// Resolve binds a role to its upstream with credentials read from getenv. It
// fails only for an unknown role; missing credentials are reported on the
// resolution so callers can choose between listing and refusing.
func (p Policy) Resolve(roleName string, getenv func(string) string) (Resolution, error) {
	role, ok := p.Roles[strings.TrimSpace(roleName)]
	if !ok {
		return Resolution{}, fmt.Errorf("unknown decision role %q", roleName)
	}
	upstream := p.Upstreams[role.Upstream]
	resolution := Resolution{
		Role: strings.TrimSpace(roleName), Upstream: role.Upstream, Protocol: upstream.Protocol,
		Locality: upstream.Locality, Model: role.Model, Envelope: upstream.Envelope, Endpoint: upstream.Endpoint,
	}
	for _, credential := range upstream.Credentials {
		value := strings.TrimSpace(getenv(credential.Env))
		if value == "" {
			resolution.Missing = append(resolution.Missing, credential.Env)
			continue
		}
		switch credential.Use {
		case credentialBearer:
			resolution.Token = value
		case credentialAccount:
			resolution.AccountID = value
		}
	}
	if resolution.AccountID != "" {
		resolution.Endpoint = strings.ReplaceAll(resolution.Endpoint, "{account}", url.PathEscape(resolution.AccountID))
	}
	return resolution, nil
}

// PolicyPath locates model-policy.json from the source root the control plane
// exports, falling back to an explicit override for tests and operators.
func PolicyPath(getenv func(string) string) string {
	if path := strings.TrimSpace(getenv("OLLAYA_MODEL_POLICY_PATH")); path != "" {
		return path
	}
	for _, key := range []string{"VROOLI_SOURCE_ROOT", "VROOLI_CLI_SOURCE_ROOT", "RESOURCE_ROOT"} {
		if root := strings.TrimSpace(getenv(key)); root != "" {
			return filepath.Join(root, "resources", "ollaya", "model-policy.json")
		}
	}
	return filepath.Join("resources", "ollaya", "model-policy.json")
}

func isDigest(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, char := range value {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return false
		}
	}
	return true
}
