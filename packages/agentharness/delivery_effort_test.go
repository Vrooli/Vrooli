package agentharness

import (
	"bytes"
	"encoding/json"
	"github.com/vrooli/cli-core/agentcatalog"
	"os"
	"testing"
)

func TestCodingPolicyResolveDeliveryCarriesActualDeclaredEffort(t *testing.T) {
	for _, value := range []string{"medium", "high"} {
		t.Run(value, func(t *testing.T) {
			path := writeCodingCatalog(t, "codex")
			raw, e := os.ReadFile(path)
			if e != nil {
				t.Fatal(e)
			}
			var c agentcatalog.CodingRoleCatalog
			if e = json.Unmarshal(raw, &c); e != nil {
				t.Fatal(e)
			}
			r := c.Roles["code.default"]
			r.Models = []agentcatalog.RoleModel{{Model: r.Model, Effort: &value}}
			c.Roles["code.delivery"] = r
			raw, e = json.Marshal(c)
			if e != nil {
				t.Fatal(e)
			}
			if e = os.WriteFile(path, raw, 0600); e != nil {
				t.Fatal(e)
			}
			var out bytes.Buffer
			cfg := CodingPolicyConfig{Runner: "codex", CatalogPath: path, Stdout: &out, Posture: EnforcementPosture{Permissions: "intent_only"}}
			if e = codingPolicyResolve(cfg, []string{"--role", "code.delivery", "--json"}); e != nil {
				t.Fatal(e)
			}
			var got codingRoleResponse
			if e = json.Unmarshal(out.Bytes(), &got); e != nil || got.Effort != value || got.Model != r.Model {
				t.Fatal("lost concrete model effort", e)
			}
			out.Reset()
			if e = codingPolicyResolve(cfg, []string{"--role", "code.default", "--json"}); e != nil {
				t.Fatal(e)
			}
			if bytes.Contains(out.Bytes(), []byte(`"effort"`)) {
				t.Fatal("non-delivery response changed")
			}
		})
	}
}
func TestDeliveryDeclaredEffortRefusesMissingAmbiguousInvalidEvidence(t *testing.T) {
	medium, high, invalid := "medium", "high", " HIGH "
	for _, r := range []agentcatalog.CodingRole{
		{Model: "m"},
		{Model: "m", Models: []agentcatalog.RoleModel{{Model: "other", Effort: &medium}}},
		{Model: "m", Models: []agentcatalog.RoleModel{{Model: "m"}}},
		{Model: "m", Models: []agentcatalog.RoleModel{{Model: "m", Effort: &invalid}}},
		{Model: "m", Models: []agentcatalog.RoleModel{{Model: "m", Effort: &medium}, {Model: "m", Effort: &high}}},
	} {
		if _, e := deliveryDeclaredEffort("code.delivery", r); e == nil {
			t.Fatal("ambiguous/missing effort accepted")
		}
	}
}
