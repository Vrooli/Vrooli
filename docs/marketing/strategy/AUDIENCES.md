# Audiences

Personas the marketing-crew targets. Advertisers read this for writing register; researcher proposes updates via `audience-update` decisions when ≥3 converging scans support revision.

**Write rule:** operator-curated via accepted `audience-update` decisions. Researcher proposes; does not edit directly.

## Persona: Indie developer (subscription)

**Key:** `indie-dev`

**Who:** solo developers and small-team technical founders building AI-integrated products.

**What they care about:**
- Reducing setup friction across multi-service AI workflows.
- Not managing N separate cloud integrations, credential stores, identity layers.
- Working examples over marketing promises.

**Register:**
- Technical vocabulary. Comfortable with terms like "agent," "scenario," "heartbeat," "backlog."
- Appreciates honesty about trade-offs and rough edges.
- Skeptical of SaaS marketing gloss.

**Preferred channels:** X/Twitter (threads), technical blogs, GitHub.

**Revisit marker:** revisit after 12 heartbeats or when researcher reports ≥3 converging observations.

---

## Persona: Small team lead (subscription)

**Key:** `small-team-lead`

**Who:** technical leads at 2-10 person teams evaluating platform choices for AI-adjacent products.

**What they care about:**
- Team-scale operational concerns: shared identity, shared state, shared infrastructure.
- Predictable pricing as team grows.
- Ability to self-host if a vendor relationship deteriorates (not planning to self-host today, wants the option).

**Register:**
- Pragmatic. Wants trade-off clarity, not enthusiasm.
- Appreciates contractual-style feature specificity (what's included, what's not, what's imminent).

**Preferred channels:** technical blogs, LinkedIn (secondary), vendor comparison sites.

**Revisit marker:** revisit after 12 heartbeats.

---

## Persona: OSS contributor (open-source)

**Key:** `oss-contributor`

**Who:** developers interested in AI-infrastructure projects they can read, extend, or contribute to — either for employer use-cases or personal building-in-public.

**What they care about:**
- Architecture clarity: can I understand this codebase? What's the extension model?
- Visible momentum: is this project active? What shipped recently?
- Low-friction entry points: where do I start?
- Agent-driven development is interesting-in-itself.

**Register:**
- Deeply technical; appreciates detail.
- Builder-to-builder voice. First person, specific, honest about struggles.
- Story-shaped: dev logs of "here's what we shipped and how agents did it" land harder than "join us."

**Preferred channels:** X/Twitter (dev logs), GitHub (READMEs, issue threads), technical blogs (longer-form).

**Revisit marker:** revisit after 12 heartbeats.

---

## Persona: Cloud-hosted evaluator (subscription)

**Key:** `cloud-evaluator` (placeholder)

**Who:** prospective buyers evaluating the eventual cloud / hosted-tier subscription delivery.

**Status:** placeholder. This persona is held in the subscription lane's scope for now. Split into its own persona only if researcher reports material differences in what cloud evaluators prioritize vs `small-team-lead`. Until then, `subscription-advertiser` treats cloud/hosted narrative as a delivery-story variant within the existing subscription framing.

**Revisit marker:** revisit when researcher produces ≥3 scans distinguishing this audience from `small-team-lead`, OR when cloud-hosted tier is officially activated.

---

## Notes on persona discipline

- Every advertiser draft names a target persona key in `campaign-drafts.jsonl`.
- Marketing-contrarian checks register match: is the draft's register consistent with the named persona?
- Researcher's `audience-scan-*` entries reference personas by key for cross-linking.
- Personas are canon. Proposed revisions require ≥3 converging scans (`audience-update` decision discipline).
