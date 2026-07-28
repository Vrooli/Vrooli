# Security — Channel Manager

This document records the scenario's security and privacy posture.
Update it before adding auth, user data, external APIs, payment flows,
secrets, or sensitive business data.

## Purpose Of This Document

Use this document to answer:

- What sensitive data exists?
- How is access controlled?
- Where do secrets come from?
- Which threats are known and how are they mitigated?

**This is the credential-adjacent scenario of the marketing trio.** `content-desk`
declares that no credential ever enters it and `asset-studio` handles generated
media; this scenario is the one that references platform credentials and stores
account handles. Its security posture is correspondingly the strictest of the
three, and most of it is enforced structurally rather than by policy.

## Data Sensitivity

| Data | Sensitivity | Owner | Details |
|---|---|---|---|
| Platform credential | **highest** | `vault` — **not this scenario** | Never stored, cached, or logged here. An identity row holds a vault path; the value is read at execution time and discarded. `CHANMGR-P0-002` asserts this in the suite rather than trusting review. |
| Account handle | **high** | identities | Marketing canon deliberately keeps handles out of `docs/marketing/` and routes them here. A handle plus a purpose tag reveals which public accounts a single operator controls — the association is the sensitive part, not either field alone. |
| Environment attestation | **high** | identities | Proxy region and device fingerprint references. Reveals operational setup that, if disclosed, would make the accounts trivially linkable. |
| Persona reference | medium | identities | Points at an `asset-studio` character. Links a public-facing persona to the operator. |
| Action record | medium | queue | What was done as an account and when. Not sensitive in isolation; in aggregate it is a behavioural fingerprint. |
| Metric observation | low | signals | Aggregate figures the platform already reports to the account owner. No individual audience-member data is ever stored. |

### The association is the asset

Individually these fields are unremarkable. Together they answer *"which public
accounts does this operator control, from where, and behaving how"* — which is
exactly what platform coordination detection looks for, and exactly what an
adversary would want. Treat the database as more sensitive than any single row in
it.

Practical consequences: handles and vault paths must not appear in log output at
default verbosity; no endpoint returns a credential value under any circumstance,
including debug paths, of which there are none; and exports of action history
should be treated as sensitive artefacts rather than reports.

<!-- EXAMPLE-DOMAIN:notes START -->
The shipped worked-example `notes` domain carries placeholder data only
(removed by `template-manager detemplate`):

| Data | Sensitivity | Owner | Details |
|---|---|---|---|
| Template notes data | low | notes reference | Local development data only; replace with real scenario data classification. |
| Attachment bytes | unknown | notes reference | Treat as potentially sensitive if retained in product scope. |
<!-- EXAMPLE-DOMAIN:notes END -->

## Auth And Authorization

The generated template does not include an auth provider. Add auth only
when product requirements identify protected data or user-specific
behavior. UI and CLI must not enforce business authorization locally;
authorization belongs at the API/service layer.

## Secrets

| Secret | Source | Required? | Details |
|---|---|---|---|
| Platform credentials (session cookies, tokens, passwords) | `vault`, via `resource-vault content get --path <kv> --key <field> --format raw` | yes | Read at execution time only. Never written to this scenario's storage, never returned by an endpoint, never logged. |
| `LPBS_SECRET`, gateway credentials | **not applicable** | no | This scenario never charges a credit or checks an entitlement. Metering lives in `browser-automation-studio`. See `../business/MONETIZATION.md`. |

## Threat Model

| Risk | Impact | Mitigation | Status |
|---|---|---|---|
| Credential leaks into scenario storage | The highest-severity failure available. A stored token converts a local database into a full account compromise. | Structural: vault-path-only schema, write rejection for credential-shaped values, and a schema assertion that fails the suite if a credential-shaped column appears. | designed (`CHANMGR-P0-002`) |
| Credential leaks into logs or errors | Same impact, different path, and easier to do accidentally. | No credential value crosses a function boundary that logs. Error messages from a failed vault read name the path, never the value. | designed |
| Database discloses the whole portfolio | Handles, environments, personas, and behaviour in one place makes every account linkable. | Local-only storage, no hosted tier (`Tier 3` ruled out in `MONETIZATION.md`), no export surface beyond `--json` on read verbs. | designed |
| Browser executor exfiltrates session state | A shared browser profile would let one identity's cookies reach another, linking the accounts. | Per-identity BAS session profiles are mandatory, not optional. Two identities never share a profile. | designed (`CHANMGR-P1-001`) |
| Compromised BAS dispatches unauthorized actions | Actions taken as a real account that nobody authorized. | Every action originates from a queue entry; there is no direct-execute path. Action records are permanent, so an unauthorized action is at least reconstructable. | designed (`CHANMGR-P0-006`) |
| Stale environment silently breaks the invariant | An expired proxy sends traffic from a new IP, and the account is linked or throttled with no signal. | **Not mitigated.** Attestations are point-in-time and unverifiable (D-006). Recorded in `PROBLEMS.md`. | **open gap** |
| Multi-user access to the operator console | The console exposes the full portfolio; there is no auth. | Single-operator, local-only. Acceptable today; blocking for any shared or hosted deployment. | accepted |

## Security Gaps

| Gap | Severity | Revisit Trigger |
|---|---|---|
| No auth model on the API or console | conditional — acceptable single-operator and local, blocking otherwise | Required before `CHANMGR-P2-004` (multi-operator handoff) or any non-local deployment. |
| Environment attestations are unverifiable | medium | If an environment provider exposes a programmatic check, or if drift becomes a recurring quarantine cause. See `PROBLEMS.md`. |
| No credential rotation path | low | Rotation happens in `vault`; this scenario holds a path. Becomes real only if an execution failure should trigger a rotation request. |
| No log redaction policy is implemented yet | medium | Before the first identity is created. The rule is stated here; the enforcement is not written. |

## Cross-References

- [`../concepts/DATA.md`](../concepts/DATA.md) — data ownership and retention
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — external services and secrets
- [`ERROR-HANDLING.md`](ERROR-HANDLING.md) — error response behavior
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved security debt
