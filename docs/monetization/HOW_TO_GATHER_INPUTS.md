# How to gather operator inputs

The `financial-tracker` needs operator-provided data to compute runway, default-alive gap, and time allocation. This doc explains, per field, how to gather each value. The data lands in [`scenarios/prompt-manager/store/teams/monetization/shared/operator-inputs.json`](../../scenarios/prompt-manager/store/teams/monetization/shared/operator-inputs.json), which the operator edits directly.

## Editing convention

- Edit `operator-inputs.json` directly. This file is **operator state, not canonical plan** — it does not go through the decision process.
- When updating a field, set `value`, refresh `updatedAt` to the current UTC timestamp, set `status` to `current`, and (where relevant) record `source` pointing at where the value came from.
- Also refresh `lastUpdatedAt` at the top of the file.
- Leave fields at `pending-operator` only if you haven't gathered them yet. Leave fields at `not-applicable-pre-launch` until the relevant phase begins.

## Status vocabulary

- `pending-operator` — the operator can provide this today but hasn't yet. Financial-tracker surfaces these as "Inputs needed" in its HANDOFF.
- `current` — the field has a fresh value within the staleness window.
- `stale` — value exists but `updatedAt` is older than the staleness threshold in `stalenessPolicy`. Financial-tracker flags stale fields each heartbeat.
- `not-applicable-pre-launch` — the field genuinely doesn't apply at the current phase. Financial-tracker does NOT nag on these.

---

## Fields

### Cash on hand

**What it is:** total liquid money across all accounts.

**How to gather:** sum balances across your bank account(s) and any other readily-available accounts (cash, operator-accessible funds). Ignore illiquid holdings.

**Update cadence:** monthly at minimum, or whenever a material transaction happens (large expense, incoming funds, etc.).

**Flag when populated:** `estimate` if approximate, `measured` if precise.

**Automation future:** none planned — personal treasury is out of scope for agents.

---

### Monthly AI / API cost

**What it is:** last-30-day spend across all AI/API providers.

**How to gather:** log into each provider's billing dashboard and record last-30-day spend. Typical providers include OpenRouter, Anthropic, OpenAI, Google (Gemini), ElevenLabs, Deepgram, plus any embedding/vector/image APIs in use. Sum across providers.

**Update cadence:** monthly.

**Flag when populated:** `measured` (providers give exact numbers).

**Automation future:** the API gateway (Tier 2 prereq, see [TIERS.md](TIERS.md)) will aggregate this automatically once built — tagged `REPLACES-MANUAL` in financial-tracker's data sources.

---

### Monthly infrastructure cost

**What it is:** recurring infra-only spend.

**How to gather:** VPS bills, CDN (Cloudflare, etc.), DNS, domain registrar annual renewals amortized monthly, any persistent storage / databases / object storage, monitoring/logging, backups.

**Update cadence:** monthly.

**Flag when populated:** `measured` (bills are exact).

**Automation future:** `scenario-to-cloud` will expose per-scenario cost aggregation — see `TELEMETRY_ROADMAP.md` Gap 3. Tagged `REPLACES-MANUAL` in financial-tracker.

---

### Monthly third-party SaaS cost

**What it is:** recurring SaaS spend that isn't AI/API or infra.

**How to gather:** check Stripe dashboard for Stripe fees; add email/transactional providers (Postmark, Resend, SendGrid), analytics tools, any notification or status services, any business-side SaaS subs in active use.

**Update cadence:** monthly.

**Flag when populated:** `measured`.

---

### Monthly tooling cost

**What it is:** dev-tooling and productivity spend.

**How to gather:** IDE subscriptions (Cursor, JetBrains), CI/CD minutes (GitHub Actions), design tools, any dev-assist subs, monitoring tools used by operator personally.

**Update cadence:** monthly.

**Flag when populated:** `measured`.

---

### Time allocation

**What it is:** rough percentage of operator working time spent on product / services / ops during the last 7 days.

**How to gather:** self-estimate at the end of the week. Round to 5% increments. Don't overthink precision — the point is trend, not precision. A typical split might be 70 / 0 / 30 (product / services / ops) pre-launch; post-launch with services active it might shift to 50 / 20 / 30.

**Categories:**
- **product** — building scenarios, features, capabilities; compounding work
- **services** — doing paid done-for-you work for clients; non-compounding unless productized
- **ops** — recurring overhead (support, deploys, meetings, admin)

**Update cadence:** weekly. (Staleness window is 7 days, so the field effectively requires weekly updates to stay `current`.)

**Flag when populated:** `estimate`.

**Automation future:** optional time-tracking integration if the operator adopts a tool; otherwise stays self-reported indefinitely.

---

### Services revenue

**What it is:** revenue from active services lines, broken out by line (lead-gen, done-for-you, consulting).

**How to gather:** per-line monthly revenue. For lead-gen, sum per-lead fees received this month. For done-for-you, sum engagement billings this month. For consulting, sum consulting invoices this month.

**Pre-launch state:** `not-applicable-pre-launch` until at least one services line is promoted to `active` per [REVENUE_LINES.md](REVENUE_LINES.md).

**Update cadence:** monthly once a services line is active.

**Flag when populated:** `measured`.

---

### Services time

**What it is:** total operator hours spent on active services engagements this window (default 14 days).

**How to gather:** if actively running services engagements, tally hours from whatever tracking you use (time-tracker, self-recall, calendar). The tracker uses this to compute time share alongside the broader `timeAllocation` — this is the finer-grained number specifically attributable to services clients.

**Pre-launch state:** `not-applicable-pre-launch` until services lines are active.

---

### Subscription MRR

**Do not fill manually.** This is auto-populated by `financial-tracker` from LPBS/Stripe events once Tier 1 ships and subscriptions begin. See [TELEMETRY_ROADMAP.md](TELEMETRY_ROADMAP.md) Gap 2.

---

## Fast-path first-run procedure

The very first time you populate this file:

1. Open `operator-inputs.json`.
2. Walk the `pending-operator` fields top-to-bottom. Fill what you can in one sitting.
3. For any field you can't readily fill, leave at `pending-operator` and move on. The financial-tracker will surface remaining gaps in its first HANDOFF — that's the team's first useful output.
4. Set `lastUpdatedAt` at the top to current time.
5. Save.

On subsequent updates you usually only touch 1-3 fields (e.g., weekly time allocation; monthly burn refresh).

## What NOT to put in this file

- Anything in the canonical docs (those are team-curated canon via decisions, not operator state).
- Anything auto-telemetry can provide (MRR, per-scenario usage, infra aggregation once those capabilities land — leave them as `not-applicable` / `pending-telemetry` so the tracker substitutes cleanly when telemetry arrives).
- Narrative or strategic content. Keep this file purely numeric and source-tagged.
