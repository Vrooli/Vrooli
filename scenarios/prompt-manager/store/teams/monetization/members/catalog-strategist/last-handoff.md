### Catalog deltas since last heartbeat
- First heartbeat — no prior catalog-strategist snapshot to diff against (`fixed`, per knowledge-list).
- No scenario transitions detected: `web-console`, `git-control-tower`, `agent-manager`, `workspace-sandbox`, `swarm-manager`, `prompt-manager` all remain as mapped in `scenario-sku-map.json` (`measured` via `swarm-manager overview` — all parent initiatives still active, none completed).

### Triggered candidates
No candidate triggers fired this heartbeat.

- `lifestyle` bundle — needs ≥50 paying business-bundle subs + 2 lifestyle scenarios deployable standalone. Current subs: 0 (`fixed`, business.md:88). **No-fire.**
- `property-services` — needs ≥50 paying subs + 3 prospect requests OR validated lead-gen opportunity. opportunities.jsonl is empty (`measured`). **No-fire.**
- `elder-care`, `family-with-kids` — both gated on lifestyle being active. **No-fire.**
- Tier 2 (self-hosted) — needs paying subs + onboarding account sign-in + license/entitlement gateway. None exist (`estimate`, no scenario evidence for license gateway; no onboarding sign-in flow in `vrooli-onboarding` scenario). **No-fire.**
- Tier 3 (hosted cloud) — gated on Tier 2 active. **No-fire.**
- Tier 4 (hardware) — `north-star`, requires operator initiation. **No-fire.**

### Tier readiness
- **Tier 2 (self-hosted, candidate):** 3 prereqs unmet. No paying subs (blocks part-a), no account-sign-in onboarding flow visible (`estimate`), no license/entitlement gateway scenario exists (`estimate` — `landing-page-business-suite` hosts Stripe assurance per `Desktop Monetization Assurance` initiative 2/2 completed, but entitlement-gateway semantics not declared). **Unchanged since docs were written.**
- **Tier 3 (hosted cloud, candidate):** gated on Tier 2. `scenario-to-cloud` scenario exists but its per-account full-runtime provisioning maturity isn't structurally queryable (`pending-telemetry` — matches `REPLACES-MANUAL` note in HEARTBEAT.md about a future `scenario-to-cloud readiness` query). **Unchanged.**
- **Tier 4 (hardware, north-star):** no operator initiation. **Unchanged.**

### Headliner watch (business bundle)
- Current headliners: `web-console`, `git-control-tower` (both `in-progress`, neither `shipped`) (`fixed`, sku-map.json:28,36).
- Nearest promotion candidate: `agent-manager` → `amplifier`-to-future-`headliner` path is the closest meaningful motion. It's the explicit amplifier for GCT; once stabilized via `Agent Sandbox Audit Foundation` (0/5) and `Protected Agent Sandboxing` (0/3), GCT's standalone appeal jumps and `swarm-manager`'s blocked-by list clears (`measured` via initiative list). Estimated gap: both initiatives entirely unstarted in completion terms. **No catalog-promotion decision raised — trigger hasn't fired; headliners are a human-promoted bar.**

### Mapping proposals
No mapping changes this heartbeat. All 6 sku-map entries remain accurate against observed scenario state.

- Noted for future heartbeats (NOT raised as a decision today — guardrail #1 is "default focus is active SKUs"): the sku-map covers only headliner/amplifier/depth scenarios explicitly sold as business-bundle content. Infrastructure scenarios that Vrooli uses internally (`landing-page-business-suite`, `command-center`, `scenario-to-cloud`, `deployment-manager`, etc.) are intentionally excluded — they are platform, not product. No change proposed.

### Current bottleneck
`agent-manager` stabilization is the single most load-bearing block — it both amplifies `git-control-tower` (boosting headliner appeal without adding a scenario) AND unblocks `swarm-manager`'s future-headliner path. The two relevant initiatives — `Agent Sandbox Audit Foundation` (0/5) and `Protected Agent Sandboxing` (0/3) — are both entirely unstarted in completion terms.

### Decisions raised this heartbeat
0 decisions. No triggers fired, no role changes detected, no SKUs to retire, no services lines active (services-* contexts N/A pre-launch).

### Knowledge entry written
- topic: `catalog-snapshot-2026-04-23` (id `knw-1776969102742586321`, no prior `catalog-snapshot-*` to supersede — first entry).