// Package policy owns the Ollaya resource's decision role catalog: which
// upstream answers each role, with which model, and which credentials a hosted
// upstream needs. ai-gateway reads it through `policy roles` and
// `policy resolve`; it never sees a credential value.
package policy

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	sharedpolicy "github.com/vrooli/ai-provider-go/policy"
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
	Residency     Residency           `json:"residency"`
	LoadedSHA256  string              `json:"-"`
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
	Digest      string                       `json:"digest"`
	Routes      []string                     `json:"routes"`
	DefaultPull bool                         `json:"default_pull"`
	License     string                       `json:"license"`
	DiskBytes   int64                        `json:"disk_bytes"`
	Placement   *ModelPlacement              `json:"placement,omitempty"`
	Limits      *ModelLimits                 `json:"limits,omitempty"`
	Rights      *sharedpolicy.ArtifactRights `json:"rights,omitempty"`
}

// ModelPlacement declares where a local model is allowed to execute and what
// it costs there. Device "gpu" means the model must be served by the
// accelerator: a CPU load is a typed failure, not a slow success. Device "cpu"
// means the model is served from host memory and claims no VRAM. Device "any"
// serves on whichever the daemon runs on. VRAMBytes is the resident footprint
// on the accelerator and RAMBytes the resident footprint in host memory; the
// capacity broker is asked whether the relevant one fits before a model that
// is not resident is dispatched, so a model that cannot fit is skipped without
// paying a load.
type ModelPlacement struct {
	Device    string `json:"device"`
	VRAMBytes int64  `json:"vram_bytes,omitempty"`
	RAMBytes  int64  `json:"ram_bytes,omitempty"`
	// KeepAlive is how long the daemon keeps the model resident after its last
	// request (a Go duration such as "2m"). Empty applies the policy's
	// residency block. Large models declare a short one so they hand their
	// memory back soon.
	KeepAlive string `json:"keep_alive,omitempty"`
}

const (
	PlacementDeviceAny = "any"
	PlacementDeviceCPU = "cpu"
	PlacementDeviceGPU = "gpu"
)

// ModelLimits are the hard bounds a decision model enforces, so a caller can
// size digests and batches from data instead of finding out by refusal.
// ContextTokens is the state the model reads without truncation. MaxOptions is
// the most options one choice question may carry. ScoreLevels is set only for
// a model that serves exactly that many score levels (zero means the System
// One range of 2 to 10). MaxQuestions is set only when the model bounds the
// questions of one request below the daemon's own cap (zero means no
// model-specific bound).
type ModelLimits struct {
	ContextTokens int `json:"context_tokens"`
	MaxOptions    int `json:"max_options"`
	ScoreLevels   int `json:"score_levels,omitempty"`
	MaxQuestions  int `json:"max_questions,omitempty"`
}

// RequiresGPU reports whether the model may only run on the accelerator.
func (m Model) RequiresGPU() bool {
	return m.Placement != nil && m.Placement.Device == PlacementDeviceGPU
}

// IsRouter reports whether the entry only forwards to other pinned models.
func (m Model) IsRouter() bool { return len(m.Routes) > 0 }

// EffectiveLimits returns the limits a caller can rely on for a model. A
// router can answer with any of its targets, so it promises only what every
// target does: the smallest context, option and question bounds.
func (p Policy) EffectiveLimits(name string) *ModelLimits {
	model, ok := p.Models[name]
	if !ok {
		return nil
	}
	if !model.IsRouter() {
		return model.Limits
	}
	var out *ModelLimits
	for _, route := range model.Routes {
		target := p.EffectiveLimits(route)
		if target == nil {
			return nil
		}
		if out == nil {
			clone := *target
			out = &clone
			continue
		}
		out.ContextTokens = minPositive(out.ContextTokens, target.ContextTokens)
		out.MaxOptions = minPositive(out.MaxOptions, target.MaxOptions)
		out.MaxQuestions = minPositive(out.MaxQuestions, target.MaxQuestions)
		if out.ScoreLevels != target.ScoreLevels {
			out.ScoreLevels = 0
		}
	}
	return out
}

// EffectivePlacement returns where a model runs and what it needs. A router
// needs whatever its most demanding target needs, and is GPU-only only when
// every target is.
func (p Policy) EffectivePlacement(name string) *ModelPlacement {
	model, ok := p.Models[name]
	if !ok {
		return nil
	}
	if !model.IsRouter() {
		return model.Placement
	}
	out := &ModelPlacement{Device: PlacementDeviceGPU}
	for _, route := range model.Routes {
		target := p.EffectivePlacement(route)
		if target == nil {
			return nil
		}
		if target.Device != PlacementDeviceGPU {
			out.Device = PlacementDeviceAny
		}
		if target.VRAMBytes > out.VRAMBytes {
			out.VRAMBytes = target.VRAMBytes
		}
		if target.RAMBytes > out.RAMBytes {
			out.RAMBytes = target.RAMBytes
		}
	}
	return out
}

func minPositive(a, b int) int {
	switch {
	case a <= 0:
		return b
	case b <= 0:
		return a
	case b < a:
		return b
	}
	return a
}

// Resolution is one role resolved against the current process environment.
// Token and AccountID are values, never serialized.
type Resolution struct {
	Role                   string                                `json:"role"`
	Upstream               string                                `json:"upstream"`
	Protocol               string                                `json:"protocol"`
	Locality               string                                `json:"locality"`
	Model                  string                                `json:"model"`
	Rights                 *sharedpolicy.ArtifactRights          `json:"rights,omitempty"`
	Limits                 *ModelLimits                          `json:"limits,omitempty"`
	Placement              *ModelPlacement                       `json:"placement,omitempty"`
	RuntimeIdentity        *sharedpolicy.RuntimeArtifactIdentity `json:"runtime_identity,omitempty"`
	ObservedRuntimeDigest  string                                `json:"observed_runtime_digest,omitempty"`
	ExpectedArtifactDigest string                                `json:"expected_artifact_digest,omitempty"`
	Envelope               string                                `json:"envelope,omitempty"`
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
	sum := sha256.Sum256(data)
	policy.LoadedSHA256 = hex.EncodeToString(sum[:])
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
		if err := model.validateServing(name); err != nil {
			return err
		}
		for _, route := range model.Routes {
			if _, ok := p.Models[route]; !ok {
				return fmt.Errorf("router model %q routes to unpinned model %q", name, route)
			}
		}
	}
	return p.Residency.validate(p)
}

// validateServing requires every servable model to say where it runs and what
// it can read; a router declares neither because it serves its targets'.
func (m Model) validateServing(name string) error {
	if m.Placement != nil {
		switch m.Placement.Device {
		case PlacementDeviceAny:
		case PlacementDeviceGPU:
			if m.Placement.VRAMBytes <= 0 {
				return fmt.Errorf("model %q requires placement.vram_bytes when placement.device is gpu", name)
			}
		case PlacementDeviceCPU:
			if m.Placement.RAMBytes <= 0 || m.Placement.VRAMBytes != 0 {
				return fmt.Errorf("model %q requires placement.ram_bytes and no vram_bytes when placement.device is cpu", name)
			}
		default:
			return fmt.Errorf("model %q placement.device %q must be %q, %q or %q", name, m.Placement.Device, PlacementDeviceAny, PlacementDeviceCPU, PlacementDeviceGPU)
		}
		if m.Placement.VRAMBytes < 0 || m.Placement.RAMBytes < 0 {
			return fmt.Errorf("model %q placement footprints cannot be negative", name)
		}
		if m.Placement.KeepAlive != "" {
			if d, err := time.ParseDuration(m.Placement.KeepAlive); err != nil || d <= 0 {
				return fmt.Errorf("model %q placement.keep_alive %q must be a positive duration", name, m.Placement.KeepAlive)
			}
		}
	}
	if m.IsRouter() {
		return nil
	}
	if m.Placement == nil {
		return fmt.Errorf("model %q must declare placement", name)
	}
	if m.Limits == nil || m.Limits.ContextTokens <= 0 || m.Limits.MaxOptions < 2 || m.Limits.MaxOptions > 255 ||
		m.Limits.ScoreLevels < 0 || m.Limits.ScoreLevels == 1 || m.Limits.ScoreLevels > 10 || m.Limits.MaxQuestions < 0 {
		return fmt.Errorf("model %q must declare limits with context_tokens > 0 and max_options between 2 and 255", name)
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
	if upstream.Locality == LocalityLocal {
		model := p.Models[role.Model]
		resolution.Rights = model.Rights
		resolution.Limits = p.EffectiveLimits(role.Model)
		resolution.Placement = p.EffectivePlacement(role.Model)
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

// PolicyPath locates model-policy.json. An explicit override wins; then the
// resource directory the control plane exports as RESOURCE_ROOT; then the
// repository source roots. The first candidate that exists is used, so a
// process whose environment names only one of them still finds the policy.
func PolicyPath(getenv func(string) string) string {
	if path := strings.TrimSpace(getenv("OLLAYA_MODEL_POLICY_PATH")); path != "" {
		return path
	}
	var candidates []string
	if root := strings.TrimSpace(getenv("RESOURCE_ROOT")); root != "" {
		candidates = append(candidates, filepath.Join(root, "model-policy.json"))
	}
	for _, key := range []string{"VROOLI_SOURCE_ROOT", "VROOLI_CLI_SOURCE_ROOT", "OLLAYA_CLI_SOURCE_ROOT"} {
		if root := strings.TrimSpace(getenv(key)); root != "" {
			candidates = append(candidates, filepath.Join(root, "resources", "ollaya", "model-policy.json"))
		}
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return filepath.Join("resources", "ollaya", "model-policy.json")
}

// RightsContextPath resolves only the scenario's canonical operator-owned
// context file. The caller cannot choose a context path per inference request.
func RightsContextPath(getenv func(string) string) string {
	path, _ := sharedpolicy.ResolveRightsContextPath(getenv)
	return path
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
