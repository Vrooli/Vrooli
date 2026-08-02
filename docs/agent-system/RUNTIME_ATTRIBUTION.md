# Runtime Attribution

**Status:** canon. The contract for the structured-attribution layer (Pillar 3 of topic validation). Pairs with `TOPICS_SCHEMA.md` (declarations) and the prose-scan spec in `PROSE_SCAN_TARGETS.md` (Pillar 2). Read by API handlers, CLI flag definitions, validator rules, and the per-team migration cutoff.

This document defines the contract by which **every knowledge-write carries a verifiable record of who wrote it**, so the validator can diff observed reality against declared topology and surface drift the static graph cannot see.

It is the third leg of the three-pillar validation architecture; without it, a `topics.json` declaration is a plan with no audit trail. With it, the substrate becomes self-correcting: any observed write that no declaration accounts for fires a finding, and reconciliation is a routine bookkeeping change rather than archaeological investigation.

---

## Three Pillars at a glance

| Pillar | Source | What it catches | Anchor doc |
|---|---|---|---|
| **P1 — Declared graph** | `topics.json` `intake[]` / `required_read[]` / `evidence_consumed[]` / `output[]` | Cross-member declaration mismatches, dangling decision references, orphaned producers/consumers | `TOPICS_SCHEMA.md` |
| **P2 — Prose scan** | Markdown bodies in `members/`, `agents/`, `skills/`, `docs/` | Hardcoded topic-prefix references in prose that contradict declarations | [`PROSE_SCAN_TARGETS.md`](PROSE_SCAN_TARGETS.md) |
| **P3 — Runtime attribution** | `attribution` field on every post-cutoff `knowledge.jsonl` entry | Observed writers/topics not accounted for by any declaration | this file |

P1 is the plan. P2 is the prose adjacent to the plan. P3 is the receipts. Each catches a class of drift the others structurally cannot. Errors should land on the right pillar — for example, an undeclared writer is a P3 finding (`actual_writer_undeclared`), an undeclared cross-member read is a P1 finding (`unread_required`), and a stale code-block topic-prefix in a SKILL.md is a P2 finding (`prose_topic_leak`).

Detail per pillar lives in the linked anchor doc; this file is concerned only with P3.

---

## Threat model

**Honest agents, accidental drift.**

Vrooli's agents are not adversarial — they are the operator's own automation, plus the operator themselves at the CLI. The risk we defend against is *drift between what an agent claims to be doing and what it is actually doing*, where the cause is one of:

- A prose claim ("the researcher writes `audience-scan/*`") is no longer true because the agent was retired or repurposed.
- A skill the operator wrote silently writes to a topic no one declares.
- A copy-paste run by the operator from one team's CLI session into another's.
- A heartbeat-spawned agent inheriting attribution from a prior context.

These are bookkeeping failures, not attacks. The contract is therefore designed for **detection and reconciliation**, not adversarial resistance.

**Out of scope (intentional):**
- Cryptographically attested attribution. A malicious local actor with shell access could spoof any field. Defending against that would require signed identity tokens with team/member claims, plus per-call signature verification — feasible (see § Future strengthening) but outside this contract.
- Cross-tenant isolation. Vrooli is single-tenant local infrastructure today; if multi-tenant deployment becomes a goal, attribution becomes an authn surface and gets a separate workshop.

The honest-drift threat model is the load-bearing simplification that lets attribution ship as a header + JSON payload, with the verifiable-claims path (`VROOLI_AGENT_IDENTITY_TOKEN`) listed explicitly as a future strengthening rather than a blocker.

---

## The structured-attribution payload

Every post-cutoff `KnowledgeEntry` carries an `attribution` object. This is the on-disk shape (see `path:scenarios/prompt-manager/api/store/models.go`):

```jsonc
{
  "kind": "agent-member",
  "member_id": "researcher",
  "team_id": "marketing-crew",
  "run_id": "5f9c1b2a-...-c0",
  "spawn_origin": "heartbeat",
  "source_skill_id": null
}
```

| Field | Type | Required by `kind` | Meaning |
|---|---|---|---|
| `kind` | enum (see below) | always | The category of writer; drives display, validator joins, and consumer expectations. |
| `member_id` | string \| null | required for `agent-member`, `writer-skill`; otherwise null | The team-scoped member id (matches `path:store/teams/<team>/members/<id>/`). |
| `team_id` | string \| null | required for `agent-member`, `writer-skill`; otherwise null | The team id (matches the URL path on `/teams/<team>/knowledge`). |
| `run_id` | string \| null | required for `agent-member` *unless* `spawn_origin=heartbeat` (see § Env-var bridge for why); required for `investigation`; optional for `writer-skill` (when invoked from a run) | The agent-manager run UUID this write is attributed to. Allows joining a write to its run lineage. |
| `spawn_origin` | enum (see below) | always | How this writer came into existence — distinguishes heartbeat-spawned agents from operator-spawned ones, etc. |
| `source_skill_id` | string \| null | required for `writer-skill`; optional for others (when a skill mediated the write) | The skill id that performed or mediated the write. |

### `kind` enum

| Value | Who's writing | Required fields | Validator behavior |
|---|---|---|---|
| `agent-member` | A team member's agent process running under agent-manager | `member_id`, `team_id`, `run_id` (null permitted iff `spawn_origin=heartbeat` — see § Env-var bridge) | Joined to declaring topics.json; flagged if writer/topic combination is undeclared. |
| `writer-skill` | A registered writer skill (e.g., `report-bug`, `report-friction`, `morning-vision-walk`) | `source_skill_id`, `team_id` (target); `member_id` if invoked by an agent; `run_id` if invoked from a run context | Joined to skill's `writes_to[]`; writes to topics not in the skill's declared set fire `actual_writer_undeclared`. |
| `operator-direct` | A human at the CLI, not running under any agent context | none beyond `kind`; `team_id` may be set to the URL team for cross-checks | Always permitted; flagged separately on the team's `policy.flagOperatorWritesPerWeek` if set. |
| `external` | A non-Vrooli system (e.g., a webhook, a future external integration) | none beyond `kind` | Tracked, not flagged unless `team.json::policy.flagExternalWritesPerWeek` threshold is exceeded. |
| `legacy` | Set by the one-time migration on every pre-cutoff entry | none beyond `kind`; the prior freeform `by` field is preserved as `caller_note` | Skipped by `actual_writer_undeclared`; treated as read-only relative to new validators. |
| `investigation` | A bug-investigator or root-cause-analysis run reproducing or annotating an existing entry | `run_id`, optionally `member_id` | Always permitted; treated as read-only relative to topic-flow declarations. |

The list is closed; new kinds require a `meta-optimization` decision and a migration plan (analogous to `TOPICS_SCHEMA.md`'s stability gate).

### `spawn_origin` enum

| Value | What it means |
|---|---|
| `heartbeat` | Run was spawned by prompt-manager's heartbeat scheduler. |
| `operator-cli` | Run was spawned directly from an operator's terminal (`prompt-manager team knowledge-add ...`). |
| `swarm-task` | Run was spawned as part of a swarm-manager initiative or sub-task. |
| `vision-walk` | Run was spawned from morning-vision-walk's seeded inbox entries. |
| `investigation` | Run was spawned by another run's `investigate` action. |
| `legacy` | Pre-cutoff entry; origin not recorded. |
| `unknown` | Origin couldn't be determined (e.g., a writer-skill invoked outside any run context). |

Origin is informational — used by debt-curator to spot patterns ("most undeclared writes are coming from `vision-walk` spawns") rather than gating writes.

---

## Derived display: `caller`

The `KnowledgeEntry`'s `caller` field is **not** part of attribution proper — it is a **derived display string** computed by the API at write time from the structured attribution. Readers (UI, CLI list output, validator findings) use it as a human-readable label without parsing attribution.

Derivation rules:

| `kind` | `caller` format |
|---|---|
| `agent-member` | `<team_id>/<member_id>` |
| `writer-skill` | `skill:<source_skill_id>` |
| `operator-direct` | `operator` |
| `external` | `external` |
| `legacy` | `legacy:<original_by_value>` |
| `investigation` | `investigation:<run_id-prefix>` |

Storing the derived string keeps list and search responses fast (no per-entry attribution parse) and gives readers a stable label even if the underlying enum gains new values.

## Optional `caller_note`

`caller_note` is an **optional, freeform context note** the writer may attach. It does **not** carry identity meaning — it cannot contradict or override `attribution`. Use cases:

- Debugging context ("retry of failed batch 17")
- Cross-references to runs or decisions ("supersedes via dec-1234")
- Migration breadcrumbs (the legacy `by` field's value is preserved here on every pre-cutoff entry)

`caller_note` is freeform UTF-8, capped at 256 characters, and never used by validators.

---

## HTTP header: `X-Vrooli-Attribution`

Attribution flows over HTTP as a single base64-encoded JSON header:

```
X-Vrooli-Attribution: eyJraW5kIjoiYWdlbnQtbWVtYmVyIiwibWVtYmVyX2lkIjoicmVzZWFyY2hlciIsInRlYW1faWQiOiJtYXJrZXRpbmctY3JldyIsInJ1bl9pZCI6IjVmOWMxYjJhLTAwMDAtMDAwMC0wMDAwLTAwMDAwMDAwYzAiLCJzcGF3bl9vcmlnaW4iOiJoZWFydGJlYXQiLCJzb3VyY2Vfc2tpbGxfaWQiOm51bGx9
```

The header value is the standard base64 encoding (`package:encoding/base64` in Go, `base64.b64encode` in Python — no URL-safe variant) of the canonical JSON form of the attribution object.

### Naming choice

`X-Vrooli-Attribution` matches the existing `X-Vrooli-Error-Hop` header family used by prompt-manager's HTTP layer. The `X-` prefix is retained for consistency with the codebase's existing custom-header convention (`X-Dry-Run`, etc.) even though RFC 6648 deprecates it for new protocols — a single naming family is more valuable than RFC purity.

### Legacy `X-Caller-ID`

`X-Vrooli-Attribution` is the canonical attribution header. The legacy `X-Caller-ID` header is retained only as a decision-approval compatibility fallback for older callers; new code must not use it as an engagement or identity signal. Member id flows through `attribution.member_id` derived from the new header.

The mapping for callers that previously sent `X-Caller-ID`:
- `X-Caller-ID: ui-user` → `attribution.kind: "operator-direct"`
- `X-Caller-ID: <agent-id>` (agent caller) → `attribution.kind: "agent-member"` with `member_id: <agent-id>` and `team_id` derived from the URL path

### Requesting side

Every CLI / SDK / UI client that writes via prompt-manager's HTTP API MUST set the header on every mutating request. The CLI derives the header value from:

1. **`VROOLI_PROMPT_MANAGER_ATTRIBUTION` env var, when set.** This is how an agent-manager-spawned process inherits attribution from its spawner. The CLI does not re-derive — it forwards the env var's value verbatim as the header.
2. **Otherwise, `{kind: "operator-direct", spawn_origin: "operator-cli"}`.** No team or member id required.

The CLI never *constructs* `agent-member` attribution itself. If an operator at the terminal wants to write on behalf of a member (a rare admin override), they invoke a separate `prompt-manager team knowledge-add-as` flow that prompts for explicit confirmation and emits `kind: "operator-direct"` + `caller_note: "acting-as <member>"` — never falsifies `agent-member`.

### Receiving side

Every prompt-manager HTTP handler that records attributed writes (`POST /teams/{id}/knowledge`, decision-status engagement detection on `PATCH /teams/{id}/decisions/{id}`, etc.) MUST:

1. Read `X-Vrooli-Attribution` from the request.
2. Reject with HTTP 400 if the endpoint requires attribution and the header is absent.
3. Reject with HTTP 400 if the base64 doesn't decode, or the JSON doesn't parse, or the `kind` is unknown, or required fields for the `kind` are missing.
4. Cross-check team consistency: if `attribution.team_id` is set and the URL path's team id is different, reject with HTTP 400 (see § Conflict policy).
5. Populate the resulting `KnowledgeEntry`'s `attribution` field from the validated header, derive `caller`, and store both atomically.

Decision status updates use `operator-direct` attribution as the heartbeat auto-pause engagement signal. Missing attribution preserves legacy behavior but does not reset the idle clock; malformed attribution is rejected.

Read-only endpoints (`GET /teams/{id}/knowledge`, etc.) do not require the header. Idempotent metadata endpoints used by the heartbeat builder may declare themselves read-only-equivalent in their handler-level docs.

### Conflict policy

When the request carries information about *who* in two places, mismatch is rejected — never reconciled.

| Conflict | Resolution |
|---|---|
| URL team id ≠ `attribution.team_id` | HTTP 400 `team_mismatch`; the response body names both values. |
| `attribution.kind = agent-member` but `member_id` is missing | HTTP 400 `incomplete_attribution`. |
| `attribution.kind = writer-skill` but `source_skill_id` is missing | HTTP 400 `incomplete_attribution`. |
| Header is set AND a legacy `--by` flag is also passed | HTTP 400 — `--by` is no longer accepted; this is defensive against stale clients. |
| Attribution claims `member_id` that doesn't exist on the URL team | HTTP 400 `unknown_member`. |

The caller is responsible for being honest about who they are. The server does not invent or reconcile — it accepts or rejects.

---

## Env-var bridge: `VROOLI_PROMPT_MANAGER_ATTRIBUTION`

Agent processes spawned by agent-manager don't inherit shell environment from prompt-manager; they receive only what agent-manager's `Environment` map per-request injects (see `path:scenarios/agent-manager/api/internal/orchestration/phases/env.go::MergeEnvVars`). The bridge that gets attribution from prompt-manager (the spawner) into the agent's CLI process is therefore a **per-request env var**:

```
VROOLI_PROMPT_MANAGER_ATTRIBUTION=<base64-encoded JSON, same format as the HTTP header>
```

### Lifecycle

1. **Spawner** (prompt-manager's heartbeat executor; see `path:scenarios/prompt-manager/api/heartbeat/executor.go` and the helper at `path:api/heartbeat/spawn_attribution.go::buildHeartbeatAttributionEnv`) constructs an attribution object describing the agent it is about to spawn:
   ```jsonc
   {
     "kind": "agent-member",
     "member_id": "researcher",
     "team_id": "marketing-crew",
     "run_id": null,
     "spawn_origin": "heartbeat",
     "source_skill_id": null
   }
   ```
   `run_id` is **null** at construction time. Agent-manager assigns the run UUID after `CreateRun` returns, but the `Environment` map is fixed before the request lands — so the run id can't yet be in the env. The API validator (`validateAttribution` in `path:api/heartbeat/attribution.go`) permits null `run_id` for `kind=agent-member` specifically when `spawn_origin=heartbeat`. § Future strengthening describes the path for closing the gap (overlay run_id from `VROOLI_AGENT_IDENTITY_TOKEN` claims at request time).

2. **Spawner** base64-encodes the JSON and includes it in `CreateRunRequest.Environment`:
   ```go
   key, val := buildHeartbeatAttributionEnv(teamID, agentID)
   runReq.Environment = map[string]string{key: val}
   ```

3. **Agent-manager** merges `Environment` into the agent process's env via `phases.MergeEnvVars` (Custom map). Sandbox and identity vars take precedence, but no current sandbox/identity var collides with the attribution slot — they share namespace cleanly. The env-var key (`VROOLI_PROMPT_MANAGER_ATTRIBUTION`) starts with the required `VROOLI_` prefix that agent-manager's `validateCustomEnvironment` enforces.

4. **Spawned agent's CLI** (when it issues `prompt-manager team knowledge-add ...`) reads `VROOLI_PROMPT_MANAGER_ATTRIBUTION`, validates that base64 decodes and JSON parses, and forwards it verbatim as `X-Vrooli-Attribution` on the HTTP request. No re-derivation, no field-by-field reconstruction — the env var IS the header value.

5. **Spawned agent's CLI** never *constructs* `agent-member` attribution from its own context. Its only options are:
   - Forward the env var (when present).
   - Default to `operator-direct` (when not). This handles the case where an operator runs the CLI inside a sandboxed shell that happens to have inherited some other `VROOLI_*` vars — without `VROOLI_PROMPT_MANAGER_ATTRIBUTION`, the CLI presumes operator context.

### Naming choice

`VROOLI_PROMPT_MANAGER_ATTRIBUTION` follows the existing `VROOLI_<DOMAIN>_<FIELD>` pattern from `VROOLI_AGENT_MANAGER_API_BASE`, `VROOLI_AGENT_IDENTITY_TOKEN`, etc. The full prompt-manager namespace is preferred over abbreviation (`VROOLI_PM_*`) for searchability and consistency with the existing fully-spelled prefixes.

### Why a single env var instead of one per field

Field-per-env (`VROOLI_PM_ATTR_KIND=agent-member`, `VROOLI_PM_ATTR_MEMBER_ID=researcher`, ...) sounds clean but adds friction on every change: each new field requires updating spawner, env-merge, CLI parser, and tests. A single base64-encoded JSON payload is opaque to the transport layer (env, HTTP) and round-trips losslessly. The header and env-var formats stay identical, so the CLI's role is pure passthrough.

---

## Per-team `attributionValidFrom`

Each `team.json` carries an `attributionValidFrom` ISO-8601 date (e.g., `"attributionValidFrom": "2026-05-04"`). Entries written **on or after** this date MUST carry attribution conforming to this contract. Entries written before are treated as `kind: "legacy"` regardless of their on-disk shape.

### Why per-team, not global

Each team migrates its `knowledge.jsonl` independently. The cutoff is set by the migration tool to "today" at run time, with a `--cutoff-date` flag for testing. Per-team granularity means:

- A team can adopt the contract early (or late) without coupling to the others.
- Validation can run cleanly during the rollout window — pre-cutoff teams' findings are deferred until that team's cutoff lands.
- A team can roll back its cutoff (retroactively widen the legacy window) without coordinating with peers, if a migration introduces unexpected drift.

The trade-off is six independent state machines instead of one, accepted because all six teams cross the cutoff in a single migration PR.

### Validator behavior at the boundary

The runtime-attribution scanner (`ruleActualWriterUndeclared`) processes each team's entries:

- Skip entries with `at < attributionValidFrom` — these are `kind: "legacy"` by definition; their `caller` is `"legacy:<original-by-value>"`; their topic-flow declarations don't apply.
- For entries with `at >= attributionValidFrom`, verify `attribution` is structurally valid (kind known; required fields populated). Structural failures fire `attribution_malformed` (always error severity).
- Join structurally-valid entries against topics.json declarations to detect undeclared writer/topic combinations.

### Supersession crossing the cutoff

A post-cutoff entry that supersedes a pre-cutoff entry is **a new entry on the live side of the cutoff** — it gets fresh attribution, validated normally. The superseded (pre-cutoff) entry remains `legacy`; readers display the superseder's attribution. No re-validation of legacy entries, even when superseded.

---

## Worked examples

### Example 1: Heartbeat-spawned agent writes a knowledge entry

1. Prompt-manager's heartbeat executor schedules a run for `literal:marketing-crew/producer`. The helper `buildHeartbeatAttributionEnv` constructs the attribution payload:
   ```json
   {"kind":"agent-member","member_id":"researcher","team_id":"marketing-crew","run_id":null,"spawn_origin":"heartbeat","source_skill_id":null}
   ```
   `run_id` is null because agent-manager hasn't assigned the run UUID yet (it returns the UUID in the `CreateRun` response, after the `Environment` map has been read).
2. Agent-manager spawns the agent process with `VROOLI_PROMPT_MANAGER_ATTRIBUTION=<base64>` in its env.
3. The agent runs, decides to write an audience-scan, invokes `prompt-manager team knowledge-add marketing-crew --topic=audience-scan/2026-05-04/q2-creators --content="..."`.
4. The CLI reads `VROOLI_PROMPT_MANAGER_ATTRIBUTION`, sets `X-Vrooli-Attribution` to its value, posts to `/teams/marketing-crew/knowledge`.
5. The handler validates the header, confirms `team_id` matches the URL path, sees `kind=agent-member` with `spawn_origin=heartbeat` and accepts the null `run_id`, derives `caller="marketing-crew/producer"`, persists the entry.
6. The runtime-attribution scanner sees the entry, joins it against `marketing-crew/producer/topics.json::output[]`, sees `audience-scan/*` is declared, no finding fires. ✅

### Example 2: Writer-skill `report-bug` invoked by an agent

1. The agent (e.g., `literal:monetization/opportunity-scout`) runs and decides to file a bug.
2. The agent invokes the `report-bug` skill, which writes via `prompt-manager team knowledge-add scenario-qa --topic=bug-inbox/regression/cli-flag-confusion --content="..."`.
3. The skill knows it's a writer skill — its CLI invocation logic constructs **writer-skill attribution** by inheriting agent context from `VROOLI_PROMPT_MANAGER_ATTRIBUTION` and overlaying `kind=writer-skill` and `source_skill_id=report-bug`:
   ```json
   {"kind":"writer-skill","member_id":"opportunity-scout","team_id":"scenario-qa","run_id":"<inherited-run-id>","spawn_origin":"heartbeat","source_skill_id":"report-bug"}
   ```
   Note: `team_id` is the **target** team (scenario-qa, where the bug-inbox lives), not the originating team (monetization). This is correct — attribution.team_id matches the URL path's team. The originating member context flows via `member_id`.
4. The handler validates as in example 1 — but here, the scanner joins against the skill's `writes_to[]` declaration on `report-bug/skill.json`. If `bug-inbox/regression/*` is in the skill's declared write set, no finding. ✅

### Example 3: Operator at the terminal

1. Operator at their shell runs `prompt-manager team knowledge-add marketing-crew --topic=audience-scan/2026-05-04/manual --content="..." --caller-note="hand-curated from yesterday's email"`.
2. CLI sees no `VROOLI_PROMPT_MANAGER_ATTRIBUTION` env var, constructs:
   ```json
   {"kind":"operator-direct","spawn_origin":"operator-cli"}
   ```
3. Sets `X-Vrooli-Attribution`, posts. Handler accepts, derives `caller="operator"`, stores `caller_note="hand-curated from yesterday's email"`.
4. The scanner sees `kind=operator-direct`; doesn't try to join against topics.json declarations. If `team.json::policy.flagOperatorWritesPerWeek` is set and exceeded, fire a separate finding. ✅

### Example 4: Conflict — agent tries to write to another team

1. Agent's CLI inherits `attribution.team_id="marketing-crew"` from its env.
2. Operator-side bug: the agent invokes `prompt-manager team knowledge-add monetization --topic=...`. URL path team is `monetization`; attribution claims `marketing-crew`.
3. Handler rejects with HTTP 400 `team_mismatch`. The agent's run sees the error in its run log; the operator sees it on the next heartbeat surface. The fix is on the agent side (don't write cross-team without producing a `writer-skill`-style attribution overlay) — never on the server side (don't silently accept).

---

## Future strengthening

These are paths the contract explicitly leaves open. Each requires its own workshop and decision.

### Run-id resolution via VROOLI_AGENT_IDENTITY_TOKEN

Today, attribution leaves `run_id` null on heartbeat-spawned writes because agent-manager assigns the UUID after `CreateRun` returns. The strengthening path closes that gap **without** a re-issue or post-spawn env-injection mechanism:

Agent-manager already issues `VROOLI_AGENT_IDENTITY_TOKEN` to every spawned run, with `claims.RunID` populated (see `path:scenarios/agent-manager/api/internal/identity/`). The CLI's attribution forwarder can read the token, decode-only its JWT body (no signature verification needed for self-identification), extract `claims.RunID`, and overlay it into the attribution payload at request time:

- If `VROOLI_PROMPT_MANAGER_ATTRIBUTION` carries `run_id=null` AND `VROOLI_AGENT_IDENTITY_TOKEN` is set, the CLI parses the token's `RunID` claim and substitutes it into the header value.
- If both env vars are missing, the CLI falls back to `operator-direct` as today.

The contract change is small: the CLI's "pure passthrough" property (the env var IS the header value) is relaxed to "passthrough with run_id overlay" — but only for the run_id field, only when source-of-truth (the signed token) is available. Cleaner than re-issuing or injecting post-spawn.

Cost: one decode-only JWT parse per CLI invocation; one new code path in `path:cli/internal/attribution`; updated tests. No agent-manager change. No on-disk data shape change.

When this lands, the validator's per-kind rule reverts to the strict form (`agent-member` always requires `run_id`); the `spawn_origin=heartbeat` exemption is removed.

### Cryptographic verification via VROOLI_AGENT_IDENTITY_TOKEN

A separate (and complementary) strengthening: the same token's `Meta` map can carry `team_id` and `member_id` when the spawner names them. Prompt-manager's API would then *cross-verify* `X-Vrooli-Attribution` against the token's signed claims:

- `attribution.team_id` MUST equal `claims.Meta["team_id"]` when both are set.
- `attribution.member_id` MUST equal `claims.Meta["member_id"]` when both are set.

This converts the threat model from "honest agents, accidental drift" to "honest agents, accidental drift, **with cryptographic detection of intentional drift**." A malicious local actor can still spoof the header on a direct API call, but cannot spoof a token-bearing request without compromising agent-manager's signing key.

This requires changes in three places (agent-manager Meta plumbing, prompt-manager handler verification, CLI token-forwarding pattern), so it lands as a separate workshop. The current contract is forward-compatible: when this workshop ships, no on-disk data shape changes.

### Per-skill attribution-derivation registry

Today, each writer skill knows it's a writer and constructs `kind=writer-skill` attribution in its CLI invocation logic. As the writer-skill set grows (currently three: `report-bug`, `report-friction`, `morning-vision-walk`), a registry-based derivation — `skill.json::attribution_template` — could centralize the logic and remove per-skill boilerplate.

This is deferred until the writer-skill set is large enough to make the registry's complexity worthwhile (rough threshold: five+ writer skills, all sharing the same attribution-derivation shape).

### UI display of attribution

The Knowledge tab in the prompt-manager UI currently renders `caller`. Surfacing `attribution.kind` and `attribution.spawn_origin` as filter dimensions would let operators slice "show me everything written by writer-skills last week" or "show me operator-direct writes this month for review." This is a UI enhancement separately tracked in the swarm-manager UI backlog.

---

## On-disk shape

The full `KnowledgeEntry` shape, with attribution, is defined in `path:scenarios/prompt-manager/api/store/models.go::KnowledgeEntry`. Reproduced here for cross-reference (the Go struct is canonical):

```jsonc
{
  "id": "knw-...",
  "at": "2026-05-04T15:32:11Z",
  "topic": "audience-scan/2026-05-04/q2-creators",
  "content": "...",
  "source": "<optional source-link>",
  "supersedes": "<optional superseded-id>",
  "caller": "marketing-crew/producer",
  "caller_note": null,
  "attribution": {
    "kind": "agent-member",
    "member_id": "researcher",
    "team_id": "marketing-crew",
    "run_id": "5f9c1b2a-...-c0",
    "spawn_origin": "heartbeat",
    "source_skill_id": null
  }
}
```

`by` is removed. `caller` is derived (computed at write time, persisted for read efficiency, never accepted as input). `caller_note` is the freeform optional context. `attribution` is the structured truth.

---

## Components and entry points

- **Storage shape** — `KnowledgeEntry.Caller`, `CallerNote`, `Attribution`: `path:scenarios/prompt-manager/api/store/models.go`.
- **Migration tool** — `path:scenarios/prompt-manager/api/cmd/migrate-knowledge-attribution/` (sets `kind=legacy`, populates `caller_note` from prior `by` value, sets per-team `attributionValidFrom`).
- **API handler validation** — `path:scenarios/prompt-manager/api/heartbeat/handlers.go::AddKnowledge` plus `path:scenarios/prompt-manager/api/heartbeat/attribution.go` (header decode, `validateAttribution`, conflict policy).
- **CLI attribution forwarding** — `path:scenarios/prompt-manager/cli/internal/attribution` (env-var read, header set, `--caller-note` flag).
- **Heartbeat executor propagation** — `path:scenarios/prompt-manager/api/heartbeat/spawn_attribution.go::buildHeartbeatAttributionEnv` and the call site in `executor.go` that injects it into `CreateRunRequest.Environment`.
- **Validator rule** — `path:scenarios/prompt-manager/api/memberflow/runtime_attribution.go::ruleActualWriterUndeclared` (consumes `attributionValidFrom`, joins post-cutoff entries against declarations).
- **Findings telemetry artifact** — `path:scenarios/prompt-manager/cli/graph/findings_artifact.go` (stable `schema_version: 1` shape; opt-in via `prompt-manager graph topics --findings-out=<path>`).
