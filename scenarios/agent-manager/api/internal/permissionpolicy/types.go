// Package permissionpolicy owns Agent Manager's global portable desired
// permission intent. It does not own native configuration files, resource
// syntax, or per-run workspace-sandbox restrictions.
package permissionpolicy

import (
	"fmt"
	"sort"
)

const CurrentSchemaVersion = 1

// Catalog is the versioned, repository-owned desired state. Rules are global
// coding-agent configuration intent, not profile or run inputs.
type Catalog struct {
	SchemaVersion int      `json:"schemaVersion"`
	Metadata      Metadata `json:"metadata"`
	TargetScopes  []string `json:"targetScopes"`
	Rules         []Rule   `json:"rules"`
}

type Metadata struct {
	CatalogID string `json:"catalogId"`
	UpdatedAt string `json:"updatedAt"`
}

// Rule is a stable, auditable portable permission declaration. Resources
// translate its matcher and write their own native configuration only after a
// later explicit reconcile operation.
type Rule struct {
	ID                      string  `json:"id"`
	Action                  string  `json:"action"`
	Matcher                 Matcher `json:"matcher"`
	Rationale               string  `json:"rationale"`
	Owner                   string  `json:"owner"`
	TargetScope             string  `json:"targetScope"`
	RequiresHardEnforcement bool    `json:"requiresHardEnforcement"`
}

// Matcher is the portable resource-protocol matcher. Native syntax is owned
// by the resource adapters and must never be represented here.
type Matcher struct {
	Kind    string `json:"kind"`
	Pattern string `json:"pattern"`
}

// ResourceDocument is JSON-compatible with the shared resource `permissions
// plan|reconcile --document` v1 input. Defining the wire model locally keeps
// Agent Manager independent of CLI-app dependencies; projector code serializes
// this value and invokes the resource CLI instead of importing its internals.
type ResourceDocument struct {
	SchemaVersion string         `json:"schema_version"`
	Scope         string         `json:"scope,omitempty"`
	Rules         []ResourceRule `json:"rules"`
}

type ResourceRule struct {
	ID      string  `json:"id"`
	Action  string  `json:"action"`
	Matcher Matcher `json:"matcher"`
}

func (c *Catalog) Clone() *Catalog {
	if c == nil {
		return nil
	}
	clone := &Catalog{
		SchemaVersion: c.SchemaVersion,
		Metadata:      c.Metadata,
		TargetScopes:  append([]string(nil), c.TargetScopes...),
		Rules:         append([]Rule(nil), c.Rules...),
	}
	return clone
}

// ResourceDocument returns the resource-CLI contract for one target scope.
// The catalog's audit-only fields deliberately do not leave Agent Manager.
func (c *Catalog) ResourceDocument(scope string) (ResourceDocument, error) {
	if err := c.Validate(); err != nil {
		return ResourceDocument{}, err
	}
	if !c.hasTargetScope(scope) {
		return ResourceDocument{}, invalid("permissionPolicyCatalog.targetScopes", fmt.Sprintf("does not declare scope %q", scope))
	}
	document := ResourceDocument{
		SchemaVersion: "v1",
		Scope:         scope,
	}
	for _, rule := range c.Rules {
		if rule.TargetScope != scope {
			continue
		}
		document.Rules = append(document.Rules, ResourceRule{
			ID:      rule.ID,
			Action:  rule.Action,
			Matcher: rule.Matcher,
		})
	}
	sort.Slice(document.Rules, func(i, j int) bool { return document.Rules[i].ID < document.Rules[j].ID })
	return document, nil
}

// Scopes returns the configured target scopes in deterministic order. A scope
// is declared even when it currently has zero rules so a later reconcile can
// intentionally clear the managed subset without inventing implicit targets.
func (c *Catalog) Scopes() []string {
	if c == nil {
		return nil
	}
	scopes := append([]string(nil), c.TargetScopes...)
	sort.Strings(scopes)
	return scopes
}

func (c *Catalog) hasTargetScope(scope string) bool {
	for _, candidate := range c.TargetScopes {
		if candidate == scope {
			return true
		}
	}
	return false
}
