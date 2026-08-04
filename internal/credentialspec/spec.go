// Package credentialspec holds the single credential declaration shape shared
// by resource manifests and scenario service manifests.
//
// It sits below both of them for the same reason hostreqspec does: resource
// manifests import internal/scenario, so a type both need cannot live in
// either one. Before this package existed only resources could declare a
// credential, which left every scenario-owned secret — tunnel-manager's
// Cloudflare token, for one — with no declaration at all. Undeclared meant
// invisible to `credentials list`, to `credentials doctor`, and, worst of the
// three, to `recovery export --all`: a credential nothing declares is a
// credential no backup captures.
package credentialspec

import (
	"fmt"
	"strings"
)

// DefaultField is the field name used when a descriptor omits one. A single
// unnamed value is the common case; naming it keeps the store key total.
const DefaultField = "value"

// Descriptor declares one credential without binding it to Vault, an
// environment variable, or a local file. Values are always held by the
// credential authority.
type Descriptor struct {
	// LogicalID and Field are the durable, backend-neutral name. They are the
	// only part of a descriptor the store ever sees.
	LogicalID string `json:"logical_id"`
	Field     string `json:"field,omitempty"`

	// Env is the process-scoped injection name, and it is optional on purpose.
	//
	// Declare it only when the consumer is a process Vrooli does not author —
	// a database container, a third-party CLI — which can receive a value no
	// other way. Vrooli-authored code resolves through
	// packages/credential-authority-go instead, which keeps the value out of
	// the process environment, where it would be readable at
	// /proc/<pid>/environ and inherited by every subprocess the consumer
	// spawns.
	//
	// A descriptor with no Env still participates fully in status reporting,
	// diagnosis, and recovery. It simply is not injected anywhere.
	Env string `json:"env,omitempty"`

	Required    bool   `json:"required,omitempty"`
	Label       string `json:"label,omitempty"`
	Description string `json:"description,omitempty"`
	ObtainURL   string `json:"obtain_url,omitempty"`
}

// ResolvedField returns the field this descriptor addresses, applying the
// default. Callers must not re-implement the fallback: a descriptor that
// resolved to "value" in one place and "" in another would read and write
// two different store keys for one declaration.
func (d Descriptor) ResolvedField() string {
	if field := strings.TrimSpace(d.Field); field != "" {
		return field
	}
	return DefaultField
}

// Injectable reports whether this descriptor names a process environment
// variable, and is therefore destined for a process Vrooli does not author.
func (d Descriptor) Injectable() bool { return strings.TrimSpace(d.Env) != "" }

// Declaration is the credentials block on a manifest.
type Declaration struct {
	// Descriptors is the sole credential declaration. The control plane
	// addresses values by LogicalID, never by a provider-shaped path.
	Descriptors []Descriptor `json:"descriptors,omitempty"`
}

// All returns the canonical descriptors. It intentionally does not synthesize
// descriptors from legacy fields: doing so would make legacy declarations a
// permanent runtime contract.
func (c Declaration) All() []Descriptor {
	return append([]Descriptor(nil), c.Descriptors...)
}

// Injectable returns only the descriptors bound to an environment variable,
// which is the subset an environment resolver has any business with.
func (c Declaration) Injectable() []Descriptor {
	out := make([]Descriptor, 0, len(c.Descriptors))
	for _, descriptor := range c.Descriptors {
		if descriptor.Injectable() {
			out = append(out, descriptor)
		}
	}
	return out
}

// Validate reports the declaration defects an operator cannot fix at runtime,
// so they surface when a manifest is read rather than when a scenario starts.
// Host conditions and unset values are deliberately not checked here: those
// are reported as data, never as a manifest error.
func (c Declaration) Validate(owner string) error {
	byEnv := map[string]string{}
	byKey := map[string]string{}
	for _, descriptor := range c.Descriptors {
		identity := strings.TrimSpace(descriptor.LogicalID)
		if identity == "" {
			return fmt.Errorf("%s declares a credential with no logical_id", owner)
		}
		field := descriptor.ResolvedField()
		if strings.ContainsAny(field, "/\\") {
			return fmt.Errorf("%s credential %s field %q cannot contain a path separator", owner, identity, field)
		}

		// Two descriptors addressing one store key is a declaration that
		// cannot mean two things at once, whether or not either is injected.
		key := identity + ":" + field
		if prior, duplicate := byKey[key]; duplicate {
			return fmt.Errorf("%s declares %s twice, as %s and %s", owner, key, prior, describe(descriptor))
		}
		byKey[key] = describe(descriptor)

		if !descriptor.Injectable() {
			continue
		}
		env := strings.TrimSpace(descriptor.Env)
		if prior, duplicate := byEnv[env]; duplicate {
			return fmt.Errorf("%s declares credential env %s twice, for %s and %s", owner, env, prior, identity)
		}
		byEnv[env] = identity
	}
	return nil
}

func describe(descriptor Descriptor) string {
	if env := strings.TrimSpace(descriptor.Env); env != "" {
		return env
	}
	return descriptor.ResolvedField()
}
