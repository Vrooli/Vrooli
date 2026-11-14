// Package realm holds the multi-tenant realm primitive's identity constants.
//
// A realm is the tenant boundary: every account belongs to exactly one realm,
// and every token is minted with that realm's audience (`aud`). A token minted
// for realm A is rejected by realm B's verifier — cross-realm acceptance is
// forbidden (DECISIONS: "even the single default realm issues aud-scoped
// tokens"). At P0 there is exactly one realm — `default` — but the aud-scoping
// is enforced now so the single-tenant path generalizes to SaaS for free.
//
// The shared constants here are the one place both this scenario and its
// relying parties (device-sync-hub) agree on the realm-qualified audience
// string. Changing DefaultAudience silently breaks every RP's aud check.
package realm

const (
	// DefaultID is the row id / slug of the single realm that exists at P0.
	DefaultID = "default"

	// DefaultName is the human label for the default realm.
	DefaultName = "Default"

	// Issuer is the JWT `iss` claim every token carries. FROZEN: device-sync-hub
	// pins this exact string (auth.AuthScenarioSlug). Never change it.
	Issuer = "scenario-authenticator"

	// DefaultAudience is the `aud` claim minted for accounts in the default
	// realm and the value relying parties must check. Realm-qualified so it
	// generalizes when realms become explicit (P1): aud becomes the realm id.
	// FROZEN as a cross-scenario contract — documented in both scenarios'
	// DECISIONS/INTEGRATIONS.
	DefaultAudience = "scenario-authenticator:default"
)

// AudienceFor returns the audience string for a realm id. At P0 only the
// default realm exists; this centralizes the realm-id→aud mapping so the
// issuance and verification sides can never drift.
func AudienceFor(realmID string) string {
	if realmID == "" || realmID == DefaultID {
		return DefaultAudience
	}
	return Issuer + ":" + realmID
}
