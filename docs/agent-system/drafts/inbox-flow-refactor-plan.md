# Inbox-Flow Refactor: Plan of Action

Status: historical draft (executed). Authored 2026-05-03. Captured here as the contemporaneous plan; canon for the resulting architecture lives at `path:docs/agent-system/INTAKE_PIPELINE.md`, `TOPICS.md`, and `TOPICS_SCHEMA.md`.

> **Naming drift note (2026-05-03 post-refactor pass).** This document references `validation-queue/*` throughout. After landing, that prefix was renamed to `validation-inbox/*` to standardize external/cross-team intake naming. Treat any `validation-queue/*` mention here as the historical name; the live name is `validation-inbox/*`. See `TOPICS.md` § Known inconsistencies #1 for the resolution.

## 1. Required Reading

```bash
prompt-manager skill read decision-boundary-extraction seam-discovery-and-enforcement boundary-of-responsibility-enforcement cli-steer api-steer react-coherence
```

Plus PoR files that govern the substrate being changed:

- `path:docs/agent-system/INTAKE_PIPELINE.md`
- `path:docs/agent-system/PRIMITIVES.md`
- `path:docs/agent-system/LAYERS.md`
- `path:docs/agent-system/PROMOTION_LADDER.md`
- `path:docs/agent-system/TOPICS_SCHEMA.md` (was `drafts/topics-schema.md` at the time this plan was authored; promoted to canon as part of this refactor)
- `path:scenarios/prompt-manager/api/memberflow/schema.go` (current shape)
- `path:scenarios/prompt-manager/api/heartbeat/prompt_builder.go` (section ordering)
- `path:scenarios/prompt-manager/api/teamcontract/contract.go` (the gold-standard reference for "structural-data → generated section")

## 2. Context

Three members today drain topic-prefix inboxes via dedicated "router" skills:

| Member | Inbox | Current router skill |
|---|---|---|
| `marketing-crew/researcher` | `research-inbox/*` | `marketing-research-router` |
| `monetization/opportunity-scout` | `opportunity-inbox/*` | `monetization-opportunity-router` |
| `monetization/market-validator` | `validation-queue/*`, `monetization-benchmark-adjacent-record/*` | `market-validation-router` | <!-- validation-queue/* was renamed to validation-inbox/* on 2026-05-03 (post-refactor naming pass) -->

Each router skill conflates six concerns: (1) draining mechanics, (2) action-selection skeleton, (3) domain signal taxonomy, (4) domain thresholds and quality rules, (5) destination data schemas, (6) signal-type → method dispatch. Concerns (1) and (2) are universal procedure; (3)–(6) are domain doctrine. Mashing them inside a "skill" is the steer-skill equivalent of hardcoding a scenario path — it kills portability and makes adoption expensive (~70% of each router skill is structural restatement that would generate from `topics.json`).

This refactor splits those concerns to their long-term-correct homes so:

- **Skills** become pure judgment, no member coupling — same shape as steer skills.
- **Plan-of-record** owns each domain's taxonomy + dispatch + thresholds + schemas.
- **`topics.json`** becomes the single declarative source for per-member structural data.
- **Heartbeat builder** generates the inbox-flow section from `topics.json` + the taxonomy JSON sidecar, the same way `teamcontract.RenderMemberPolicy` does today.

Adoption beyond three members is currently expensive because of per-member prose authoring. After this refactor, adopting a fourth/fifth/etc. member is "fill in `topics.json` + cite a taxonomy"; no per-member prose is required. The user has identified ~5 candidate members to migrate next (publisher, oss-advertiser, subscription-advertiser, financial-tracker, brand-manager near-miss), making this refactor the unblock for that wave.

## 3. Goals

1. **Generated inbox-flow context.** Every member with an `intake[]` entry gets an "Inbox Flow" section in their heartbeat prompt, derived from `topics.json` + a taxonomy JSON file. No hand-authored prose carries draining mechanics, destinations, or dispatch.
2. **Portable classifier skills.** The three current router skills are renamed and rewritten so each contains pure classification judgment — no team names, no inbox prefixes, no `prompt-manager team knowledge-*` commands.
3. **Taxonomy as PoR.** Three new PoR documents (`path:docs/marketing/SIGNAL_TAXONOMY.md`, `path:docs/monetization/OPPORTUNITY_TAXONOMY.md`, `path:docs/monetization/VALIDATION_TAXONOMY.md`) hold each domain's signal types, dispatch table, evidence rules, and destination schemas. Each is paired with a parseable JSON sidecar the heartbeat builder consumes.
4. **Stricter validation.** `missing_drain_skill` is replaced by three rules: `unknown_taxonomy`, `non_portable_classifier`, `missing_destination_schema`. The `non_portable_classifier` rule is the structural guardrail that prevents the current conflation from re-emerging.
5. **Member-panel Inbox tab.** A new tab on `MemberDetailPanel` that surfaces the unrouted set Gmail-style: unrouted entries, age, source, dispatch hint, "promote to <destination>" / "drop" actions. Reuses `topics/drain-status` and a new knowledge-list backend.
6. **Zero net change in agent behavior** beyond the architectural improvements. The three existing members must continue draining their inboxes correctly; outputs to the same prefixes, same decision contexts, same handoff shape.

## 4. Non-Goals

- **Authoring new method skills.** `audience-pain-mining`, `competitor-positioning-scan`, `hook-pattern-mining`, `workflow-deconstruction`, `skill-opportunity-scan`, `channel-format-scan`, `post-type-discovery`, `benchmark-adjacent-scan` are referenced in the existing router skills' dispatch tables but are not registered skills. The taxonomy PoR will name them as the canonical method anyway and the classifier will recommend them; if the skill doesn't exist, the member applies inline guidance from the taxonomy. Authoring those skills is follow-up work.
- **Migrating additional members.** Adopting publisher / oss-advertiser / subscription-advertiser / financial-tracker / brand-manager to the new pattern is post-refactor work. This plan covers only the substrate change + the existing three adopters.
- **Replacing the team-level "Team Inbox" cross-member messaging system.** That's a different concept (gated by `ShouldInjectInbox` in the prompt builder, owned by `teamconfig`). It stays untouched.
- **Touching the docs-promotion ladder.** Existing rules in `PROMOTION_LADDER.md` apply unchanged.

## 5. Greenfield Statement

**This is greenfield work.** The three router skills are renamed (`*-router` → `*-classifier` / `*-triage`) and rewritten in place. No backwards-compatibility wrappers, no old-name aliases, no `// removed` comments, no shims pointing the old skill ID to the new one. The `drained_by_skill` field is removed from `topics.json` schema entirely (replaced by `taxonomy` + optional `classifier_skill`). Old references in `RESPONSIBILITIES.md` / `HEARTBEAT.md` / `TOOLS.md` are deleted, not commented out. After this lands, no file in the repo should reference `marketing-research-router`, `monetization-opportunity-router`, or `market-validation-router`.

## 6. Architecture Summary

### 6.1 Layer table

| Concern | Today | After |
|---|---|---|
| Inbox-draining mechanics (commands, retag-or-delete, unrouted-set invariant) | Hand-authored in each router SKILL.md + each HEARTBEAT.md + each RESPONSIBILITIES.md | Generated by `buildInboxFlowSection` from `topics.json` + universal procedure baked into the prompt builder |
| Destination prefixes | Hand-authored in router SKILL.md + duplicated in topics.json | `topics.json` `output[].prefix` + `output[].schema` is single source; rendered into Inbox Flow section |
| Decision contexts owned/consumed | `topics.json` (already) | `topics.json` (unchanged) — also surfaced into Inbox Flow section |
| Domain signal taxonomy (vocabulary) | Embedded in router SKILL.md | `path:docs/<domain>/<TAXONOMY>.md` + `path:docs/<domain>/<taxonomy>.json` sidecar |
| Default method dispatch (signal type → method skill) | Embedded in router SKILL.md | Same taxonomy JSON sidecar (single source of truth) |
| Evidence/quality rules + thresholds | Embedded in router SKILL.md | Same taxonomy PoR (human-readable side) |
| Destination front-matter schemas | Embedded in router SKILL.md | Same taxonomy PoR + JSON sidecar |
| Classification *judgment* (which signal type is this?) | Embedded in router SKILL.md | A small, portable classifier skill (no team/inbox/destination references) |

### 6.2 New `topics.json` schema

```jsonc
{
  "intake": [
    {
      "prefix": "research-inbox/*",
      "taxonomy": "marketing-research",                   // REQUIRED. Resolves to docs/marketing/signal-taxonomy.json (filename matches taxonomy id).
      "classifier_skill": "marketing-signal-classifier",  // OPTIONAL. Set when judgment is required to assign signal type.
      "source_team": null
    }
  ],
  "output": [
    {
      "prefix": "audience-scan/*",
      "destination_kind": "knowledge",
      "destination_team": null,
      "schema": "audience-scan"                           // OPTIONAL but encouraged. References taxonomy.schemas.<schema-id>.
    }
  ],
  "decisions_owned":     [...],     // unchanged
  "decisions_consumed":  [...],     // unchanged
  "raises_capability_gaps": true,   // unchanged
  "external_producers":  [...]      // unchanged
}
```

`drained_by_skill` is removed.

### 6.3 New taxonomy JSON sidecar shape

A taxonomy JSON file lives next to its human-readable PoR doc. The heartbeat builder + validator parse the JSON; the markdown is rendered from the JSON for human review (or hand-curated and kept in sync — see Risk 5). Single source for now: hand-curated JSON; human-readable PoR is regenerated on schema changes via a small CLI verb.

```jsonc
// docs/marketing/signal-taxonomy.json
{
  "schemaVersion": 1,
  "id": "marketing-research",
  "displayName": "Marketing Research Signal Taxonomy",
  "owner_team": "marketing-crew",
  "porPath": "docs/marketing/SIGNAL_TAXONOMY.md",
  "signalTypes": [
    {
      "id": "audience-pain",
      "definition": "Audience frustration, buying trigger, vocabulary",
      "defaultMethod": "audience-pain-mining",
      "defaultDestinationPrefix": "audience-scan/<slug>",
      "evidenceMinimum": "single-snapshot allowed; converging required for decision"
    },
    {
      "id": "competitor",
      "definition": "Competitor pricing, packaging, positioning",
      "defaultMethod": "competitor-positioning-scan",
      "defaultDestinationPrefix": "competitor-record/<slug>",
      "evidenceMinimum": "single-snapshot allowed"
    }
    // ...
  ],
  "evidenceRules": [
    "One source = observation; three converging independent sources = decision threshold.",
    "Single-snapshot findings carry the `light-interpretation` flag.",
    "Tool classifications are inputs, not proof."
  ],
  "actionSelection": {
    "drop":              "Weak one-off / duplicate / out of scope.",
    "observe":           "Single-snapshot fact with applicability; retag to destination prefix.",
    "promote-to-canon":  "Converging evidence; retag + raise owned-context decision.",
    "file-decision":     "Operator should decide now; raise owned-context decision.",
    "capability-gap":    "Source / tool / scenario missing; file capability-gap, leave inbox row."
  },
  "schemas": {
    "audience-scan": {
      "frontMatter": {
        "type":              "audience-scan",
        "audience_segment":  "<text>",
        "evidence_strength": "<single-snapshot|converging>",
        "honesty_flags":     "[<...>]",
        "source":            "<url-or-null>",
        "date_observed":     "<YYYY-MM-DD>"
      },
      "bodyRequiredSections": ["Observation", "Implication (light-interpretation)"]
    },
    "competitor-observation": { ... }
  },
  "honestyFlags": ["light-interpretation", "tailwind-uncited", "ai-extracted", ...]
}
```

### 6.4 Heartbeat prompt section ordering

Insert one new section between Operating Policy and Responsibilities. (Position justified: structural facts immediately follow the operating-contract structural section so prose sections read with the structural facts already loaded.)

```
1. Active Task Brief
2. Team Inbox                       (cross-member messaging — unchanged)
3. Previous Handoff
4. Storage Map
5. Team Org Context
6. Operating Policy + TEAM.md
7. Inbox Flow                       ← NEW (generated; only present when topics.json declares an intake[])
8. RESPONSIBILITIES.md              (trimmed — inbox bullets removed)
9. Agent Files
10. Heartbeat Task                  (HEARTBEAT.md trimmed — inbox steps removed)
11. Task Reminder
```

### 6.5 New "Inbox Flow" section content (template)

```markdown
# Inbox Flow

You drain one or more team-knowledge inboxes. The mechanics, destinations,
and dispatch below are generated from your topics.json + the taxonomy.
Do not paraphrase from memory; the generated text is the source of truth.

## Inbox: research-inbox/*

| | |
|---|---|
| Team        | marketing-crew |
| Prefix      | `research-inbox/*` |
| Taxonomy    | `marketing-research` (PoR: `path:docs/marketing/SIGNAL_TAXONOMY.md`) |
| Classifier  | `marketing-signal-classifier` (`prompt-manager skill read marketing-signal-classifier`) |

View unrouted entries:
```bash
prompt-manager team knowledge-list marketing-crew --topic-prefix=research-inbox/
prompt-manager team knowledge-list marketing-crew --topic-prefix=research-inbox/<signal-type>/
```

External producers: `vision-walk`, `operator`, `bookmark-intelligence-hub`.

## Drain procedure (universal)

For each entry:
1. Apply `marketing-signal-classifier` to identify `signal_type`, `evidence_strength`, `honesty_flags`.
2. Choose the smallest useful action from the taxonomy's action-selection set:
   - **drop** — `prompt-manager team knowledge-delete marketing-crew <id>`
   - **observe** — `prompt-manager team knowledge-update marketing-crew <id> --topic="<destination-prefix>"`
   - **promote-to-canon** — same as observe; pair with a decision when evidence converges
   - **file-decision** — raise an owned context; delete the inbox row if the artifact lives elsewhere
   - **capability-gap** — file `capability-gap` decision; leave the inbox entry until the gap is closed
3. After routing, the entry must no longer carry a `research-inbox/*` topic. The inbox view is the unrouted set.

## Destinations

| Prefix | Schema | Cross-team to |
|---|---|---|
| `audience-scan/<slug>` | audience-scan | — |
| `competitor-record/<slug>` | competitor-observation | — |
| `hook-record/<slug>` | hook | — |
| `monetization-benchmark-adjacent-record/<slug>` | monetization-benchmark-adjacent | monetization |

Schemas: `path:docs/marketing/SIGNAL_TAXONOMY.md#schemas`.

## Decisions

| Context | Role |
|---|---|
| audience-update | own / propose |
| channel-strategy-update | own / propose |
| post-type-proposal | own / propose |
| hook-candidate-promotion | own / propose |
| capability-gap | consume; permitted to raise |

## Default dispatch (from taxonomy)

If the classifier returns a typed signal, the default method skill is:

| signal_type | method skill |
|---|---|
| audience-pain | `audience-pain-mining` |
| competitor | `competitor-positioning-scan` |
| ... | ... |

When a method skill does not yet exist in the registry, follow inline guidance
in the taxonomy PoR until one is authored.
```

## 7. Phased Implementation Steps

### Phase A — Schema and validation (server-side foundation)

**Goal**: extend `topics.json` schema and validation; nothing visible to agents yet.

**Files**:
- `path:scenarios/prompt-manager/api/memberflow/schema.go`
  - Add `Taxonomy string` and `ClassifierSkill string` (optional) to `IntakeEntry`.
  - Add `Schema string` (optional) to `OutputEntry`.
  - **Remove** `DrainedBySkill` from `IntakeEntry`. Greenfield — no shim.
  - Update `validateIntake` to require `Taxonomy`; allow empty `ClassifierSkill`.
  - Add `validPrefix` semantics check unchanged.
- `path:scenarios/prompt-manager/api/memberflow/loader.go` — no behavior change; struct change is wire-compatible by JSON tag.
- `path:scenarios/prompt-manager/api/memberflow/taxonomy.go` (new file)
  - Load `path:docs/<domain>/<taxonomy-id>.json` from a registry rooted at repo root.
  - Schema struct: `Taxonomy{ID, DisplayName, OwnerTeam, PoRPath, SignalTypes[], EvidenceRules[], ActionSelection{}, Schemas{}, HonestyFlags[]}`.
  - `LoadTaxonomy(repoRoot, id) (*Taxonomy, error)`.
  - `LoadAllTaxonomies(repoRoot) (map[string]*Taxonomy, error)`.
- `path:scenarios/prompt-manager/api/memberflow/validation.go`
  - **Remove** `ruleMissingDrainSkill` and the `LoadSkillIDs` dependency for that rule.
  - Add `ruleUnknownTaxonomy` (error): `intake[].taxonomy` does not resolve via `LoadTaxonomy`.
  - Add `ruleNonPortableClassifier` (error): when `intake[].classifier_skill` is set, the skill's SKILL.md must NOT contain any of: `*-inbox/`, `validation-queue/`, `prompt-manager team knowledge-list`, `prompt-manager team knowledge-update`, `prompt-manager team knowledge-delete`, the literal team names, or any `<inbox-name>/<signal-type>/<slug>` topic-prefix references. Implement as content grep against the skill registry.
  - Add `ruleMissingDestinationSchema` (warning): when `output[].schema` is set, it must resolve under the *producing member's* taxonomy's `schemas{}` map.
  - Update `ValidationOptions` to take `RepoRoot` (already present) and a list of taxonomy IDs (loaded once and passed in).

**Tests**:
- `schema_test.go`: extend with cases asserting the new fields validate correctly.
- `taxonomy_test.go` (new): load + parse + reject malformed taxonomy JSON.
- `validation_test.go`: extend with positive + negative cases for each new rule.

### Phase B — Taxonomy authoring

**Files (new)**:
- `path:docs/marketing/signal-taxonomy.json` (id: `marketing-research`)
- `path:docs/marketing/SIGNAL_TAXONOMY.md` (human-readable view; cite the JSON sidecar)
- `path:docs/monetization/opportunity-taxonomy.json` (id: `monetization-opportunity`)
- `path:docs/monetization/OPPORTUNITY_TAXONOMY.md`
- `path:docs/monetization/validation-taxonomy.json` (id: `monetization-validation`)
- `path:docs/monetization/VALIDATION_TAXONOMY.md`

Content sources: lift directly from the current router SKILL.md files, recategorized by concern. Each `.json` carries: signal types + definitions + default methods + default destination prefix + evidence rules + action selection + schemas + honesty flags. Each `.md` is the human-readable narration of the JSON, plus operator-facing notes.

**Cross-team owner-of-schema rule (refinement from prior turn's contradiction)**: a destination prefix's schema is owned by the *producer's* taxonomy. So `monetization-benchmark-adjacent-record/*` schema lives in `marketing-research` (since the marketing researcher writes it), not in `monetization-validation`. The receiving member's `intake[].taxonomy` can still be `monetization-validation` for routing purposes; it doesn't redefine the schema.

**Tests**:
- `taxonomy_authoring_test.go` (new, light-weight): for each taxonomy file, assert (a) loadable, (b) all `defaultMethod` references either exist in the skill registry OR are explicitly listed in a `pendingMethodSkills` field on the taxonomy, (c) every `output[].schema` referenced from any `topics.json` resolves to a `schemas{}` entry on *some* taxonomy.

### Phase C — Heartbeat prompt builder integration

**Files**:
- `path:scenarios/prompt-manager/api/heartbeat/prompt_builder.go`
  - Add `buildInboxFlowSection(ctx, team, agentID)` analogous to `buildOperatingPolicySection`.
  - Insert in `buildSectionList` between operating-policy and responsibilities.
  - Section is omitted entirely when the member's `topics.json` has no `intake[]`.
  - Render uses `memberflow.LoadMember` + `memberflow.LoadTaxonomy` + the team's name and the universal procedure template (a `const` string in a new file `inbox_flow_template.go`).
- `path:scenarios/prompt-manager/api/heartbeat/inbox_flow.go` (new)
  - `RenderInboxFlow(memberTopics memberflow.MemberTopics, taxonomies map[string]*memberflow.Taxonomy, teamName string) string`
  - Pure function, no I/O. Mirrors `teamcontract.RenderMemberPolicy` pattern.
  - Returns empty string when intake is empty (caller skips section).

**Tests**:
- `inbox_flow_test.go` (new): table-driven, covers (a) member with one intake, (b) member with two intakes (cross-team flow), (c) member with no intake (empty render), (d) intake taxonomy resolves to dispatch table that's rendered correctly, (e) classifier-skill optional.
- `prompt_builder_test.go`: extend to assert the new section appears at position 7 when present and is absent otherwise; ordering of other sections unchanged.

### Phase D — Three classifier skills (rename + rewrite)

For each of the three router skills, perform the following atomic operation:

1. Rename directory:
   - `path:store/skills/packs/core/marketing-research-router/` → `path:store/skills/packs/core/marketing-signal-classifier/`
   - `path:store/skills/packs/core/monetization-opportunity-router/` → `path:store/skills/packs/core/monetization-signal-classifier/`
   - `path:store/skills/packs/core/market-validation-router/` → `path:store/skills/packs/core/market-validation-triage/`
2. Update `skill.json`: change `id`, `name`, `description`, `tags` (drop `router`); bump `revision` and `updatedAt`.
3. Rewrite `SKILL.md` to the trimmed, member-agnostic shape — pure judgment, ~30-50 lines each.

**SKILL.md template (apply per skill)**:

```markdown
## Tools focus: <Display Name>

Given raw <domain> input, identify the signal type, evidence strength, and
honesty flags. This skill is pure judgment — no inboxes, no teams, no
destinations. The caller is responsible for what to do with the
classification result.

### Required reading

- `path:docs/<domain>/<TAXONOMY>.md` — canonical signal types, definitions, dispatch.
- (additional domain reference docs, judgment-relevant only)

### Inputs

| Input | Required | Notes |
|---|---|---|
| `raw_items` | Yes | Each has `source`, `source_url?`, `raw_note`, optionally `producer_assigned_type`. |

### Method

For each item:
1. Read `raw_note` and any source.
2. Match against the taxonomy. If `producer_assigned_type` is present, treat as a hint, not authoritative.
3. Score `evidence_strength` ∈ {single-snapshot, converging, blocked}.
4. Set `honesty_flags` per the taxonomy's flag list.
5. Recommend a `method_skill` from the taxonomy's dispatch. Override only if clearly wrong; explain in `dispatch_reason`.

### Output

```yaml
- id: <item-id>
  signal_type: <one of taxonomy types | unknown>
  evidence_strength: <single-snapshot|converging|blocked>
  honesty_flags: [...]
  recommended_method: <method-skill-id>
  dispatch_reason: <empty unless overriding default>
  rationale: <one short paragraph>
```

This skill does not retag, delete, or promote anything. It does not file
decisions. The member's drain procedure handles those steps with this
skill's output as input.
```

**Tests**:
- `classifier_purity_test.go` (new, in memberflow or a new `skillaudit` package): for each skill in the registry whose ID matches `*-classifier` or `*-triage`, assert SKILL.md does NOT contain forbidden substrings (the same set the `non_portable_classifier` validation rule checks). Belt-and-suspenders with the validation rule, run as part of unit tests.

### Phase E — Update three members' `topics.json`

For each of the three members, update `topics.json`:

- Replace `drained_by_skill` with `taxonomy` (required) and `classifier_skill` (optional but set, since all three need judgment per the analysis).
- Add `output[].schema` references that resolve against the producer's taxonomy.

Concrete diffs:

#### marketing-crew/researcher

```jsonc
{
  "intake": [
    {
      "prefix": "research-inbox/*",
      "taxonomy": "marketing-research",
      "classifier_skill": "marketing-signal-classifier",
      "source_team": null
    }
  ],
  "output": [
    { "prefix": "audience-scan/*",                  "destination_kind": "knowledge", "destination_team": null,           "schema": "audience-scan" },
    { "prefix": "competitor-record/*",                     "destination_kind": "knowledge", "destination_team": null,           "schema": "competitor-observation" },
    { "prefix": "hook-record/*",                           "destination_kind": "knowledge", "destination_team": null,           "schema": "hook" },
    { "prefix": "monetization-benchmark-adjacent-record/*", "destination_kind": "knowledge", "destination_team": "monetization", "schema": "monetization-benchmark-adjacent" }
  ],
  "decisions_owned": ["audience-update", "channel-strategy-update", "post-type-proposal", "hook-candidate-promotion"],
  "decisions_consumed": ["capability-gap"],
  "raises_capability_gaps": true,
  "external_producers": ["vision-walk", "operator", "bookmark-intelligence-hub"]
}
```

#### monetization/opportunity-scout

```jsonc
{
  "intake": [
    {
      "prefix": "opportunity-inbox/*",
      "taxonomy": "monetization-opportunity",
      "classifier_skill": "monetization-signal-classifier",
      "source_team": null
    }
  ],
  "output": [
    { "prefix": "candidate-sku-record/*", "destination_kind": "knowledge", "destination_team": null, "schema": "candidate-sku" }
  ],
  "decisions_owned": ["catalog-promotion", "channel-activation", "services-activation"],
  "decisions_consumed": ["capability-gap"],
  "raises_capability_gaps": true,
  "external_producers": ["operator", "vision-walk"]
}
```

#### monetization/market-validator

```jsonc
{
  "intake": [
    {
      "prefix": "validation-queue/*",
      "taxonomy": "monetization-validation",
      "classifier_skill": "market-validation-triage",
      "source_team": null
    },
    {
      "prefix": "monetization-benchmark-adjacent-record/*",
      "taxonomy": "monetization-validation",
      "classifier_skill": "market-validation-triage",
      "source_team": "marketing-crew"
    }
  ],
  "output": [
    { "prefix": "monetization-benchmark-record/*", "destination_kind": "knowledge", "destination_team": null, "schema": "market-scan" }
  ],
  "decisions_owned": ["benchmark-update", "pricing-decision", "financial-model-assumption-update"],
  "decisions_consumed": ["capability-gap"],
  "raises_capability_gaps": true,
  "external_producers": ["operator", "vision-walk", "bookmark-intelligence-hub"]
}
```

### Phase F — Trim member prose

For each of the three members, remove inbox-coupled prose now generated:

- `members/<id>/HEARTBEAT.md` — remove explicit inbox triage steps, knowledge-list/update/delete commands, "run `<router-skill>`" line. Add a single bullet pointing to the generated section.
- `members/<id>/RESPONSIBILITIES.md` — same.
- `agents/researcher/TOOLS.md` — remove the bulleted `prompt-manager team knowledge-list ... --topic-prefix=research-inbox/` line and the knowledge-add/update/delete inbox-specific bullets. Keep general decision/knowledge tool access.

Concrete trim targets are in the conversation summary; the full set of files to edit is:

- `path:store/teams/marketing-crew/members/researcher/HEARTBEAT.md`
- `path:store/teams/marketing-crew/members/researcher/RESPONSIBILITIES.md`
- `path:store/agents/researcher/TOOLS.md`
- `path:store/teams/monetization/members/opportunity-scout/HEARTBEAT.md`
- `path:store/teams/monetization/members/opportunity-scout/RESPONSIBILITIES.md`
- `path:store/teams/monetization/members/market-validator/HEARTBEAT.md`
- `path:store/teams/monetization/members/market-validator/RESPONSIBILITIES.md`
- `path:store/teams/monetization/shared/TEAM.md` — remove the two "Unrouted-set invariant" rows that name router skill IDs; replace with one generic invariant row that names the taxonomy.

Greenfield: delete the prose, do not comment out. After the trim, `grep -rn 'marketing-research-router\|monetization-opportunity-router\|market-validation-router' scenarios/prompt-manager/store docs` should return zero matches.

### Phase G — Update agent-system PoR

- `path:docs/agent-system/INTAKE_PIPELINE.md` — update the "topics.json" example to show the new fields; add a paragraph on taxonomies as the dispatch single source of truth.
- `path:docs/agent-system/TOPICS_SCHEMA.md` — update schema definition to match new fields. *(Note: as of plan execution this file was promoted from `drafts/topics-schema.md` to canon; the stability gate was crossed.)*
- `path:docs/agent-system/PRIMITIVES.md` — small update to the "Inbox / synthesis" entry referencing the new `taxonomy`/`classifier_skill` shape.
- `path:docs/agent-system/README.md` — `taxonomy` enters the canon mental-model paragraph.

### Phase H — Member-panel Inbox tab (UI)

Backend dependency:
- `path:scenarios/prompt-manager/api/memberflow/handlers.go` — add `GET /teams/{id}/members/{agentId}/inbox-entries?prefix=<prefix>` returning unrouted entries from team knowledge under the prefix, paginated. Backed by the existing `KnowledgeQuery` interface (already used by `inbox_aging.go`); add `ListUnroutedFull` returning enough fields for UI display (id, topic, by, source, content preview, at).
- Routes: register in `routes_topics.go` (or wherever member-flow routes register).

Frontend:
- `path:ui/src/services/memberFlowService.ts` — add `getMemberInboxEntries(teamId, agentId, prefix, opts)` and `routeInboxEntry(teamId, agentId, id, action)` (action: `promote` with destination, `drop`).
- `path:ui/src/components/editor/MemberDetailPanel.tsx`
  - Extend `MemberDetailSection` enum with `'inbox'`.
  - Extend `ActiveTab` with `'inbox'`.
  - Conditionally render the Inbox tab only when the member has an `intake[]` declaration (fetch via `getTopicsGraph` filtered to the member or a new `getMemberTopics` endpoint).
  - Tab content: `MemberInboxTab.tsx` (new component).
- `path:ui/src/components/editor/MemberInboxTab.tsx` (new)
  - For each `intake[]` prefix on the member: header row (prefix, age summary, count, piling/stalled banner from `getDrainStatus`).
  - Below: a table of unrouted entries — columns: signal-type (parsed from prefix), `--by`, age, source URL (link), preview of content.
  - Inline actions per row: "Promote → \<destination select\>" (populated from `output[].prefix` matching the producer/taxonomy match) and "Drop". Both call the new `routeInboxEntry` endpoint, which under the hood invokes `prompt-manager team knowledge-update` / `knowledge-delete`.
  - Sidebar: "Out: routed to `audience-scan/*` (count), `competitor-record/*` (count), …" — same prefixes shown as "sent mail" via knowledge-list under those prefixes.
- `path:ui/src/components/editor/MemberInboxTab.test.tsx` (new) — covers (a) renders empty state, (b) renders unrouted entries, (c) promote action calls service, (d) drop action calls service, (e) hides tab when no intake declared.

UI scope guard: this is the v1 of the inbox tab; bulk operations and search are out of scope. Single-row promote/drop is enough to validate the affordance.

### Phase I — Fully retire `drained_by_skill`

After Phases A-G are merged and verified, run a final repo-wide grep:

```bash
rg "drained_by_skill" scenarios docs
rg "marketing-research-router\|monetization-opportunity-router\|market-validation-router" scenarios docs
```

Both must return zero. If not, that's an untrimmed reference to fix.

## 8. Validation Rules (additions and removals)

### Removed
- `missing_drain_skill` — superseded by `unknown_taxonomy` + `non_portable_classifier`.

### Added (memberflow/validation.go)
| Rule | Severity | Condition |
|---|---|---|
| `unknown_taxonomy` | error | `intake[].taxonomy` doesn't resolve via `LoadTaxonomy(repoRoot, id)` |
| `non_portable_classifier` | error | `intake[].classifier_skill` SKILL.md content matches forbidden patterns (inbox prefixes, knowledge CLI verbs, team names, retag/delete commands) |
| `missing_destination_schema` | warning | `output[].schema` set but doesn't resolve to a `schemas{}` entry on any taxonomy |

### Existing (kept unchanged)
- `conflicting_drain` (overlap on intake prefixes)
- `orphan_input` (intake prefix has no producer)
- `orphan_output` (output prefix has no consumer; warning only — operator-only snapshots are fine)
- `dangling_por_sink` (`destination_kind=por_file` with non-existent path)
- `stalled_drain`, `piling_inbox` (inbox-aging warnings)

## 9. Tests

All tests are automated. Per project feedback memory, no manual test checklists.

| Layer | Test file | What it asserts |
|---|---|---|
| Schema | `memberflow/schema_test.go` | New fields validate correctly; `drained_by_skill` field is gone |
| Taxonomy loader | `memberflow/taxonomy_test.go` | Load + parse + reject malformed; all three taxonomy files load cleanly |
| Validation | `memberflow/validation_test.go` | `unknown_taxonomy`, `non_portable_classifier`, `missing_destination_schema` cases (positive + negative) |
| Classifier purity | `memberflow/classifier_purity_test.go` | Each `*-classifier` / `*-triage` skill's SKILL.md fails forbidden-pattern grep |
| Heartbeat render | `heartbeat/inbox_flow_test.go` | Inbox Flow section renders correctly; empty when no intake |
| Heartbeat ordering | `heartbeat/prompt_builder_test.go` | Section appears at position 7; other ordering unchanged |
| Taxonomy authoring | `memberflow/taxonomy_authoring_test.go` | Each `topics.json` taxonomy reference resolves; each `output[].schema` resolves |
| API endpoint | `memberflow/handlers_test.go` | New `GET /teams/.../inbox-entries` returns unrouted entries; pagination works |
| UI | `MemberInboxTab.test.tsx` | Render + actions cases above |
| End-to-end (server) | new `inbox_flow_e2e_test.go` | Full path: load all members, render heartbeat for each of the three target members, assert generated section content matches snapshot |

Snapshot tests for the rendered Inbox Flow section: one per current adopter, locked under `testdata/inbox_flow/<team>__<member>.golden`. Update tooling: `go test -update` regenerates.

## 10. Migration Order

The migration must keep the three current members operational throughout. Strict order:

1. **Phase A (schema + validation, server-side only)**. Deployable on its own — no agent-visible change.
2. **Phase B (taxonomy authoring)**. Three new files; no agent change yet.
3. **Phase C (heartbeat builder)**. Section renders but only for members whose `topics.json` has been migrated (`taxonomy` set). Old members still use the legacy `drained_by_skill` path... wait — Phase A removes that field. So Phase A and Phase E must land together, OR Phase A keeps the field temporarily.

   **Resolution**: keep `drained_by_skill` parsable through Phases A-D (loader accepts it but ignores it; validation does not require it). Make it a hard-required removal in Phase G, after every member has been migrated. This is the one tolerated transitional state.

   Updated sequence:
   - A1: add new fields to schema, keep parsing `drained_by_skill` field (silently ignored).
   - A2: validation: `unknown_taxonomy` is an error only when `taxonomy` is set; otherwise warning ("missing_taxonomy") so unmigrated members surface but don't break.
   - B: author taxonomies.
   - C: heartbeat builder renders Inbox Flow only when `taxonomy` is set.
   - D: rewrite + rename three classifier skills.
   - E: migrate three `topics.json` files to use `taxonomy` + `classifier_skill`.
   - F: trim three members' HEARTBEAT/RESPONSIBILITIES/TOOLS prose.
   - G: update PoR docs.
   - H: UI inbox tab.
   - I: hard-remove `drained_by_skill` from schema; promote `missing_taxonomy` to a hard error; final repo-wide grep returns zero references.

4. Each step lands as a separate commit (or PR if branch policy requires). After D-E-F land for each member, that member's heartbeat is verified in isolation (Phase J below) before moving to the next.

## 11. Cleanup & Health Verification

Per project feedback memory, every plan that touches a scenario ends with:

1. **Fix all lint, type, and unit-test issues in modified files — including pre-existing ones.**
   - Go: `cd scenarios/prompt-manager/api && gofumpt -w . && golangci-lint run && go test ./... -timeout 300s`
   - UI: `cd scenarios/prompt-manager/ui && npx tsc --noEmit && npm run lint && npm test`
2. **Restart the scenario.**
   - `vrooli scenario restart prompt-manager`
3. **Verify health.**
   - `curl -s http://localhost:<prompt-manager-port>/health` returns 200.
   - UI loads at the configured URL; member-panel routes still resolve.
   - `prompt-manager graph topics` returns 0 errors.
   - `prompt-manager graph drain-status` returns the same drain status as before the refactor (same prefixes, same counts, no missing members).
4. **Per-member heartbeat smoke (Phase J)**.
   - Run `prompt-manager team member-context <team> <member>` for each of the three migrated members. Diff the generated prompt before/after — verify (a) Inbox Flow section appears at position 7, (b) HEARTBEAT.md/RESPONSIBILITIES.md no longer carry inbox prose, (c) total length is roughly equivalent (no content loss).

## 12. Risks

1. **Cross-team schema ownership confusion.** A producer's taxonomy owns the schema for prefixes it writes, even when the consumer is on another team. The validator must follow the *producer's* taxonomy when resolving `output[].schema`. Mitigation: explicit test for cross-team flow (researcher writing `monetization-benchmark-adjacent-record/*` with schema resolved via marketing's taxonomy).
2. **Generated prompt size growth.** The Inbox Flow section adds ~60-100 lines to each member's heartbeat prompt. For prompt-budget-sensitive heartbeats, this matters. Mitigation: snapshot tests verify size; if it grows unacceptably, factor the universal procedure into a single doc the section links to rather than repeating in each render.
3. **JSON-vs-markdown sync risk in taxonomy PoR.** If the `.json` and `.md` are both hand-curated, they can drift. Mitigation: ship a `prompt-manager taxonomy render <id>` CLI verb that regenerates the `.md` from the `.json`; the `.md` becomes a derived artifact. Initial implementation may keep both hand-curated; plan a follow-up to add the renderer.
4. **`non_portable_classifier` rule false positives.** A classifier skill that legitimately mentions "audience-scan" (a destination prefix) in a *judgment* context (e.g., "if the producer assigned `audience-scan` already, treat as hint") would trip the rule. Mitigation: forbidden patterns target `*-inbox/`, `validation-queue/`, and the explicit knowledge-CLI verbs — not destination-prefix names. Verify with the rewritten classifier skills.
5. **Taxonomy registry location.** Plan places taxonomies at `path:docs/<domain>/`. An alternative is `path:scenarios/prompt-manager/store/taxonomies/<id>.json` (centralized). Trade-off: domain-colocated is more discoverable for operators reading the PoR; centralized is easier for the loader. Plan keeps domain-colocated; loader does a glob over `path:docs/**/*-taxonomy.json` (or similar). Mitigation: implement loader with explicit registry path list, easy to refactor.
6. **The MEMORY.md feedback "duplicate before extracting" rule.** Suggests holding off on broader scenario-CLI verbs until the pattern matures. The `prompt-manager taxonomy render` CLI verb (Risk 3) is exactly the kind of helper that should wait until a second taxonomy-shaped surface emerges. Mitigation: ship the renderer as a one-off Go script in this scenario only; promote later if other scenarios need it.

## 13. Open Questions

1. **Should `taxonomy` be `taxonomy_id` for clarity?** The field references a JSON file by id. Either name works; `taxonomy` is shorter and matches `decisions_owned`/`external_producers` naming convention. Default: keep `taxonomy`.
2. **Should the universal procedure be inlined in every section render, or factored into a separate `INBOX_PROCEDURE.md` PoR cited by the section?** The current plan inlines it. Inlining is slightly larger per heartbeat but self-contained; factoring saves length but adds an indirection. Defer to first-implementation feel; switch to factored if snapshots feel verbose.
3. **Should classifier skills declare their taxonomy in `skill.json`?** A new field `skill.json#metadata.taxonomy_id` would let the validator cross-check that the skill named in `intake[].classifier_skill` is the one bound to the named `taxonomy`. Adds rigor but may be overkill for three skills. Defer to follow-up when adoption widens.
4. **Where do shared classifier skills sit when two members on different teams share a taxonomy?** `monetization-validation` is consumed by exactly one member today (`market-validator`). If `validation-queue` becomes a multi-team concept later, the classifier skill stays portable (it's domain-bound, not team-bound), so this should already be handled — but confirm with a hypothetical second consumer before declaring victory.
5. **Should the UI Inbox tab show the full unrouted set across *all* of a member's intakes in one view, or one tab section per intake?** Plan does one section per intake. For members like `market-validator` with two intakes, this is two stacked sections. Reconsider after first-render screenshot.

---

This plan is greenfield, additive-then-cleanup, and respects the user's "design for the long-term-correct solution" preference. Execution should follow Phases A→I in order, with verification gates between D-E-F per member.
