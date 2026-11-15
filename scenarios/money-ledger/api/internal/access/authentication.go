package access

import "github.com/vrooli/api-core/authn"

// Resource is Money Ledger's name at the identity providers: the
// scenario-authenticator audience is scenario-authenticator:money-ledger, and
// tunnel-manager resolves the Cloudflare Access binding for this route.
const Resource = "money-ledger"

// Authentication is the provider chain the gate verifies callers against:
//
//   - scenario-authenticator tokens issued for the money-ledger audience: the
//     declared service writers' principals and the owner's CLI session
//     (`--local-owner`);
//   - the Cloudflare Access assertion the tunnel adds to the owner's browser
//     requests, resolved lazily from tunnel-manager so a tunnel outage never
//     stops the API;
//   - the host's owner-binding policy, which names the verified person who
//     owns these records.
//
// getenv supplies the manifest-backed runtime bindings (VROOLI_AUTH_*,
// VROOLI_CLOUDFLARE_ACCESS_*, the owner-binding file).
func Authentication(getenv func(string) string) (authn.Config, error) {
	cfg, err := authn.FromEnvironment(func(key string) string {
		if value := getenv(key); value != "" {
			return value
		}
		// Money Ledger always verifies its own audience, so the native
		// provider honours the VROOLI_AUTH_SCENARIO_* bindings even when the
		// lifecycle declares no provider list.
		switch key {
		case "VROOLI_AUTH_PROVIDERS":
			return "scenario_authenticator"
		case "VROOLI_AUTH_SCENARIO_AUDIENCE":
			return "scenario-authenticator:" + Resource
		}
		return ""
	})
	if err != nil {
		return authn.Config{}, err
	}
	return cfg.WithScenarioAuthenticator(Resource).WithManagedCloudflare(Resource), nil
}
