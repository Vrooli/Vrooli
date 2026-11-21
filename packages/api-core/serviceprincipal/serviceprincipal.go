// Package serviceprincipal is the contract for operator-issued
// scenario-authenticator service principals: the profile the authenticator
// is configured with, the declaration a consuming scenario ships to ask for
// one, and the address its credential is kept at in the credential authority.
//
// A profile is operator authority: it fixes the audience and the complete scope
// list of every token the principal receives, and names who may issue it. A
// declaration is only a request for a profile. Nothing a scenario ships can
// grant itself authority; the operator reads what a declaration would add
// (`scenario-authenticator service-principals sync`) and applies it.
package serviceprincipal

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

const (
	// DeclarationFile is where a scenario declares the principals it consumes,
	// relative to the scenario directory.
	DeclarationFile = ".vrooli/service-principals.json"
	// CredentialField is the field of a principal's credential entry.
	CredentialField = "credential"
	// ReservedForDevelopment is the one name the authenticator supplies by code,
	// for a development environment only. It can be neither configured nor
	// declared.
	ReservedForDevelopment = "ai-gateway"
	// AudiencePrefix starts every audience the authenticator registers.
	AudiencePrefix = "scenario-authenticator:"
	// DryRunCommand shows the operator exactly which audiences and scopes the
	// declarations would add, and which credentials would be issued. It changes
	// nothing and is safe to run anywhere, including inside an agent run.
	DryRunCommand = "scenario-authenticator service-principals sync"
	// IssueCommand is the one command the operator runs to configure and issue
	// every declared service principal. Issuing needs a verified human issuer,
	// which an agent run cannot be, so no scenario issues its own.
	IssueCommand = DryRunCommand + " --apply"
)

var nameRule = regexp.MustCompile(`^[a-z][a-z0-9-]{1,62}$`)

// ValidName reports whether name can name a service principal.
func ValidName(name string) bool { return nameRule.MatchString(name) }

// Profile is one operator-owned service principal. The audience is resolved
// from the registered resource; scopes are the complete, fixed capability list
// carried by every token the principal receives. Issuers names the verified
// human accounts the operator authorizes to mint it; a realm administrator may
// also issue.
type Profile struct {
	Name     string   `json:"name"`
	Resource string   `json:"resource"`
	Scopes   []string `json:"scopes"`
	Issuers  []string `json:"issuers,omitempty"`
}

// ParseProfiles decodes the operator-owned JSON profile list. An empty value
// configures no service principals. Malformed or duplicate profiles fail
// configuration rather than being silently skipped. Principal names are unique:
// a name is the token's subject and part of its credential.
func ParseProfiles(raw string) ([]Profile, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var profiles []Profile
	if err := json.Unmarshal([]byte(raw), &profiles); err != nil {
		return nil, fmt.Errorf("invalid service principal profiles: %w", err)
	}
	seen := map[string]bool{}
	for i, p := range profiles {
		p.Name = strings.TrimSpace(p.Name)
		p.Resource = strings.TrimSpace(p.Resource)
		if !ValidName(p.Name) || p.Resource == "" || seen[p.Name] {
			return nil, fmt.Errorf("invalid service principal profile %d", i)
		}
		scopes, err := cleanScopes(p.Scopes)
		if err != nil {
			return nil, fmt.Errorf("invalid scope in service principal profile %q", p.Name)
		}
		p.Scopes = scopes
		issuers := make([]string, 0, len(p.Issuers))
		for _, issuer := range p.Issuers {
			if issuer = strings.TrimSpace(issuer); issuer == "" {
				return nil, fmt.Errorf("invalid issuer in service principal profile %q", p.Name)
			}
			issuers = append(issuers, issuer)
		}
		p.Issuers = issuers
		seen[p.Name] = true
		profiles[i] = p
	}
	return profiles, nil
}

func cleanScopes(in []string) ([]string, error) {
	scopes := make([]string, 0, len(in))
	for _, scope := range in {
		scope = strings.TrimSpace(scope)
		if scope == "" || strings.Contains(scope, "*") {
			return nil, errors.New("invalid scope")
		}
		scopes = append(scopes, scope)
	}
	return scopes, nil
}

// CredentialIdentity is the credential-authority identity of a principal's
// credential: the authenticator writes it on issuance and the consuming
// scenario reads it.
func CredentialIdentity(name, resource string) string {
	return "scenario-authenticator/service-principal/" + strings.ToLower(strings.TrimSpace(name)) + "/" + strings.ToLower(strings.TrimSpace(resource))
}

// Declaration is a consuming scenario's request for one principal: what it
// reads the credential of, the audience the token must carry, and the exact
// scopes it needs. It carries no issuers; who may issue is operator authority.
type Declaration struct {
	Name     string   `json:"name"`
	Resource string   `json:"resource"`
	Audience string   `json:"audience"`
	Scopes   []string `json:"scopes"`
}

// DeclarationFileContent is the content of DeclarationFile.
type DeclarationFileContent struct {
	Principals []Declaration `json:"principals"`
}

// ParseDeclarations decodes and validates a scenario's declaration file.
// Unknown fields are refused, so a declaration cannot carry what it is not
// allowed to ask for (issuers, say).
func ParseDeclarations(data []byte) ([]Declaration, error) {
	var file DeclarationFileContent
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&file); err != nil {
		return nil, fmt.Errorf("invalid service principal declarations: %w", err)
	}
	seen := map[string]bool{}
	out := make([]Declaration, 0, len(file.Principals))
	for _, d := range file.Principals {
		d.Name, d.Resource, d.Audience = strings.TrimSpace(d.Name), strings.TrimSpace(d.Resource), strings.TrimSpace(d.Audience)
		switch {
		case !ValidName(d.Name):
			return nil, fmt.Errorf("invalid principal name %q", d.Name)
		case d.Name == ReservedForDevelopment:
			return nil, fmt.Errorf("principal %q is supplied by the authenticator in development and cannot be declared", d.Name)
		case seen[d.Name]:
			return nil, fmt.Errorf("principal %q is declared twice; a principal name is unique", d.Name)
		case d.Resource == "":
			return nil, fmt.Errorf("principal %q has no resource", d.Name)
		case !strings.HasPrefix(d.Audience, AudiencePrefix) || len(d.Audience) == len(AudiencePrefix):
			return nil, fmt.Errorf("principal %q audience must be %s<resource>", d.Name, AudiencePrefix)
		case len(d.Scopes) == 0:
			return nil, fmt.Errorf("principal %q declares no scopes", d.Name)
		}
		scopes, err := cleanScopes(d.Scopes)
		if err != nil {
			return nil, fmt.Errorf("principal %q has an empty or wildcard scope", d.Name)
		}
		d.Scopes = scopes
		seen[d.Name] = true
		out = append(out, d)
	}
	return out, nil
}
