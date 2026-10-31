package agents

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	agentsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/agents"
	agentsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/agents/agents_v1connect"

	"switchboard/internal/authoring"
	"switchboard/internal/trust"
)

// Appearance is the colour triple a prompt-manager descriptor carries, so an
// agent looks the same everywhere it appears.
type Appearance struct {
	Body   string `json:"body"`
	Head   string `json:"head"`
	Accent string `json:"accent"`
}

// Profile is the read-only projection of a prompt-manager agent descriptor.
// Switchboard never stores it; it is fetched by reference on every read.
type Profile struct {
	ID          string          `json:"id"`
	DisplayName string          `json:"display_name"`
	Description string          `json:"description"`
	Status      string          `json:"status"`
	Appearance  Appearance      `json:"appearance"`
	Tags        []string        `json:"tags"`
	Grant       CapabilityGrant `json:"-"`
	GrantSource string          `json:"-"`
}

// GrantView is the console projection of a capability grant.
type GrantView struct {
	Scopes          []string `json:"scopes"`
	ProgramBindings []string `json:"program_bindings"`
	OwnerOnly       []string `json:"owner_only"`
	Source          string   `json:"source"`
}

func (p Profile) GrantView() GrantView {
	view := GrantView{Scopes: nonNil(p.Grant.Scopes), ProgramBindings: nonNil(p.Grant.ProgramBindings), OwnerOnly: []string{}, Source: p.GrantSource}
	for _, s := range view.Scopes {
		if trust.OwnerOnly(s) {
			view.OwnerOnly = append(view.OwnerOnly, s)
		}
	}
	return view
}

func nonNil(in []string) []string {
	if in == nil {
		return []string{}
	}
	return in
}

// DefaultGrant is what an agent may do when its descriptor declares nothing:
// read only. Nothing is ever widened silently.
var DefaultGrant = CapabilityGrant{Scopes: []string{"read"}}

var ErrProfileNotFound = errors.New("agent profile not found")

// Roster reads agent profiles from prompt-manager over its typed Connect
// surface. It is the only way switchboard learns about an agent.
type Roster struct {
	BaseURL string
	Client  connect.HTTPClient
	TTL     time.Duration

	cache   []Profile
	fetched time.Time
	err     error
}

func NewRoster(baseURL string) *Roster {
	return &Roster{BaseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"), Client: &http.Client{Timeout: 5 * time.Second}, TTL: 10 * time.Second}
}

func (r *Roster) client() (agentsconnect.AgentsServiceClient, error) {
	if r == nil || r.BaseURL == "" {
		return nil, fmt.Errorf("prompt-manager URL is unavailable")
	}
	c := r.Client
	if c == nil {
		c = http.DefaultClient
	}
	return agentsconnect.NewAgentsServiceClient(c, r.BaseURL), nil
}

// List returns every agent profile. Results are cached briefly so a console
// page that fans out several reads does not hammer prompt-manager.
func (r *Roster) List(ctx context.Context) ([]Profile, error) {
	if r != nil && r.TTL > 0 && r.cache != nil && time.Since(r.fetched) < r.TTL {
		return r.cache, r.err
	}
	client, err := r.client()
	if err != nil {
		return nil, err
	}
	resp, err := client.ListAgents(ctx, connect.NewRequest(&agentsv1.ListAgentsRequest{}))
	if err != nil {
		return nil, fmt.Errorf("prompt-manager list agents: %w", err)
	}
	out := make([]Profile, 0, len(resp.Msg.GetAgents()))
	for _, a := range resp.Msg.GetAgents() {
		out = append(out, profileFrom(a))
	}
	r.cache, r.fetched, r.err = out, time.Now(), nil
	return out, nil
}

// Get returns one profile by id. A missing id is ErrProfileNotFound so callers
// can render the broken reference with its reason instead of dropping it.
func (r *Roster) Get(ctx context.Context, id string) (Profile, error) {
	client, err := r.client()
	if err != nil {
		return Profile{}, err
	}
	resp, err := client.GetAgent(ctx, connect.NewRequest(&agentsv1.GetAgentRequest{Id: id}))
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			return Profile{}, ErrProfileNotFound
		}
		return Profile{}, fmt.Errorf("prompt-manager get agent: %w", err)
	}
	return profileFrom(resp.Msg), nil
}

// GrantFor satisfies dispatch.GrantResolver. It fails closed: an unreachable
// or unknown profile gets the default read grant, never a wider one.
func (r *Roster) GrantFor(ctx context.Context, agentID string) trust.Grant {
	p, err := r.Get(ctx, agentID)
	if err != nil {
		return trust.Grant{Scopes: DefaultGrant.Scopes}
	}
	return trust.Grant{Scopes: p.Grant.Scopes}
}

// profileFrom projects the descriptor. The capability grant is read from the
// descriptor's `capabilities.requires` block: each required capability id is a
// scope the agent may exercise. An empty block means the default grant.
func profileFrom(a *agentsv1.Agent) Profile {
	p := Profile{ID: a.GetId(), DisplayName: a.GetDisplayName(), Description: a.GetDescription(), Status: a.GetStatus(), Tags: nonNil(a.GetTags())}
	if ap := a.GetAppearance(); ap != nil {
		p.Appearance = Appearance{Body: ap.GetBody(), Head: ap.GetHead(), Accent: ap.GetAccent()}
	}
	scopes := make([]string, 0)
	bindings := make([]string, 0)
	for _, cap := range a.GetCapabilities().GetRequires() {
		id := strings.TrimSpace(cap.GetCapabilityId())
		if id == "" {
			continue
		}
		if strings.HasPrefix(id, "program:") {
			bindings = append(bindings, strings.TrimPrefix(id, "program:"))
			continue
		}
		scopes = append(scopes, id)
	}
	if len(scopes) == 0 && len(bindings) == 0 {
		p.Grant, p.GrantSource = DefaultGrant, "default"
	} else {
		p.Grant, p.GrantSource = CapabilityGrant{Scopes: clean(scopes), ProgramBindings: clean(bindings)}, "descriptor"
	}
	return p
}

// WriteAgent satisfies authoring.Writer by creating the profile through
// prompt-manager's typed create RPC. Switchboard never writes a descriptor
// file itself.
func (r *Roster) WriteAgent(d authoring.Draft) error {
	client, err := r.client()
	if err != nil {
		return err
	}
	requires := make([]*agentsv1.Capability, 0, len(d.Scopes))
	for _, s := range d.Scopes {
		requires = append(requires, &agentsv1.Capability{CapabilityId: s})
	}
	input := &agentsv1.AgentInput{
		Id: d.ID, DisplayName: d.DisplayName, Description: d.Description, Status: "active",
		Tags:         []string{"switchboard"},
		Appearance:   &agentsv1.Appearance{Body: "#0E7490", Head: "#0891B2", Accent: "#A5F3FC"},
		Capabilities: &agentsv1.Capabilities{Requires: requires},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := client.CreateAgent(ctx, connect.NewRequest(&agentsv1.CreateAgentRequest{Agent: input}))
	if err != nil {
		return fmt.Errorf("prompt-manager create agent: %w", err)
	}
	r.cache = nil
	if resp.Msg.GetId() == "" {
		return fmt.Errorf("prompt-manager returned no agent id")
	}
	return nil
}

// CreatedID is a best-effort lookup of the id prompt-manager assigned to a
// draft, used to return it to the console after Confirm.
func (r *Roster) CreatedID(ctx context.Context, d authoring.Draft) string {
	if d.ID != "" {
		return d.ID
	}
	list, err := r.List(ctx)
	if err != nil {
		return ""
	}
	for _, p := range list {
		if p.DisplayName == d.DisplayName && p.Description == d.Description {
			return p.ID
		}
	}
	return ""
}

var _ authoring.Writer = (*Roster)(nil)
