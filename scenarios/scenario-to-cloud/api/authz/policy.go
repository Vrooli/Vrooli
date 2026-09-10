package authz

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/api-core/identity"
)

// Target is the minimal identity of a deployment the boundary needs to bind a
// grant to. It never carries the locator (host, port, user): the boundary must
// be able to refuse without learning where the workload lives.
type Target struct {
	DeploymentID string
	Environment  string
	// Key is the deployment's target key as persisted (machine:<id> or
	// host:<host>); it is compared against policy grants and logged, never
	// returned to a refused caller.
	Key string
}

// Grant is one principal's target binding.
type Grant struct {
	// Environments the principal may act on; "*" grants every environment.
	Environments []string `json:"environments,omitempty"`
	// Machines (target keys) the principal may act on; "*" grants every target.
	Machines []string `json:"machines,omitempty"`
}

// PolicyDocument is the operator-owned policy file shape.
type PolicyDocument struct {
	// Principals maps a verified subject to its grant. Subjects are the
	// provider subjects (osuser:<uid> for personal_local, the JWT subject for
	// shared providers).
	Principals map[string]Grant `json:"principals"`
	// Revoked lists subjects whose grants are withdrawn; a revoked subject is
	// refused at the effect boundary even when its credential still verifies.
	Revoked []string `json:"revoked,omitempty"`
}

// Policy binds principals to targets. Absent a policy file the personal_local
// owner is unrestricted and every other principal is restricted (no targets).
type Policy struct {
	path string

	mu       sync.Mutex
	loadedAt time.Time
	modTime  time.Time
	doc      PolicyDocument
	err      error
}

// DefaultPolicyPath is ~/.vrooli/scenario-to-cloud/authz-policy.json.
func DefaultPolicyPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".vrooli", Scenario, "authz-policy.json")
}

// NewPolicy returns a policy backed by the file at path (may be absent).
func NewPolicy(path string) *Policy {
	return &Policy{path: strings.TrimSpace(path)}
}

// NewStaticPolicy returns an in-memory policy for tests and embedding.
func NewStaticPolicy(doc PolicyDocument) *Policy {
	return &Policy{doc: doc, loadedAt: time.Now()}
}

// Path returns the backing file (empty for static policies).
func (p *Policy) Path() string {
	if p == nil {
		return ""
	}
	return p.path
}

// document returns the current policy document, re-reading the file when its
// modification time changed so revocations take effect without a restart.
func (p *Policy) document() (PolicyDocument, error) {
	if p == nil {
		return PolicyDocument{}, nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.path == "" {
		return p.doc, nil
	}
	info, err := os.Stat(p.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			p.doc, p.err, p.modTime = PolicyDocument{}, nil, time.Time{}
			return p.doc, nil
		}
		return PolicyDocument{}, fmt.Errorf("stat authz policy: %w", err)
	}
	if !p.loadedAt.IsZero() && info.ModTime().Equal(p.modTime) {
		return p.doc, p.err
	}
	raw, err := os.ReadFile(p.path)
	if err != nil {
		p.err = fmt.Errorf("read authz policy: %w", err)
		return PolicyDocument{}, p.err
	}
	var doc PolicyDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		p.err = fmt.Errorf("decode authz policy: %w", err)
		return PolicyDocument{}, p.err
	}
	p.doc, p.err, p.modTime, p.loadedAt = doc, nil, info.ModTime(), time.Now()
	return p.doc, nil
}

// Revoked reports whether the subject has been revoked by the operator.
func (p *Policy) Revoked(principal identity.Principal) (bool, error) {
	doc, err := p.document()
	if err != nil {
		return true, err
	}
	for _, subject := range doc.Revoked {
		if strings.EqualFold(strings.TrimSpace(subject), strings.TrimSpace(principal.Subject)) {
			return true, nil
		}
	}
	return false, nil
}

// grantFor returns the grant and whether the principal is unrestricted. A
// personal_local owner without a named grant is unrestricted; every other
// unnamed principal has no target grant at all.
func (p *Policy) grantFor(principal identity.Principal) (Grant, bool, error) {
	doc, err := p.document()
	if err != nil {
		return Grant{}, false, err
	}
	if grant, ok := doc.Principals[strings.TrimSpace(principal.Subject)]; ok {
		return grant, grant.unrestricted(), nil
	}
	if principal.Source == identity.SourcePersonalLocal {
		return Grant{Environments: []string{"*"}, Machines: []string{"*"}}, true, nil
	}
	return Grant{}, false, nil
}

// Unrestricted reports whether the principal may address any target,
// including targets the boundary cannot resolve ahead of the handler.
func (p *Policy) Unrestricted(principal identity.Principal) (bool, error) {
	_, unrestricted, err := p.grantFor(principal)
	return unrestricted, err
}

// Allows reports whether the principal's grant names the target's environment
// or its target key.
func (p *Policy) Allows(principal identity.Principal, target Target) (bool, error) {
	grant, unrestricted, err := p.grantFor(principal)
	if err != nil {
		return false, err
	}
	if unrestricted {
		return true, nil
	}
	return grant.covers(target), nil
}

func (g Grant) unrestricted() bool {
	return contains(g.Environments, "*") || contains(g.Machines, "*")
}

func (g Grant) covers(target Target) bool {
	if target.Environment != "" && contains(g.Environments, target.Environment) {
		return true
	}
	if target.Key != "" && contains(g.Machines, target.Key) {
		return true
	}
	return false
}

func contains(values []string, want string) bool {
	want = strings.TrimSpace(want)
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), want) {
			return true
		}
	}
	return false
}
