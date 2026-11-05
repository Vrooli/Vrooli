# Extended Agent Manager operations

This reference keeps lower-frequency procedures available without crowding the default usage skill. The core skill owns the first decision and routes here when one of these cases applies.

#### 4.1 Recalling an unknown run

Use this branch when you remember the subject, wording, project, date, harness, or outcome but not the run ID. Search returns retained, policy-visible events only.

| What you know | Surface and mode | One-step leaf |
|---|---|---|
| The evidence is in an Agent Manager conversation | Direct, `hybrid` | `agent-manager conversation search "<natural-language clues>" --mode hybrid` [S1] |
| You know distinctive words or a copied phrase | Direct, `text` | `agent-manager conversation search "<words or phrase>" --mode text` [S1] |
| You know a precise expert pattern | Direct, `regex` | `agent-manager conversation search '<bounded-pattern>' --mode regex` [S1] |
| You remember meaning but not wording | Direct, `semantic` | `agent-manager conversation search "<paraphrase>" --mode semantic` [S1] |
| The evidence might instead be source, docs, or distilled memory | Unified federation | `search-hub query query "<question with the remembered clues>"` [S1] |

Start with `hybrid` for partial memory. Add the narrowest known filters without inventing values: for example `--project-scopes <project>`, `--harnesses <harness>`, `--roles user,assistant`, or `--occurred-after 2026-08-01T00:00:00Z`. Use `text` when literal wording decides identity. Use `regex` only for a pattern that text search cannot express. Use `semantic` when lexical overlap is not expected; it requires the semantic substrate.

Read each result as evidence, not as a conclusion. Confirm the run label and ID, role, time, harness, source session, match explanation, and deep link. Copied text can appear in more than one run, so prefer the candidate whose surrounding provenance matches the clue. Fetch context only after selecting a stable hit:

`agent-manager conversation context <stable-hit-id> --before 3 --after 3` [S1]

Success is the intended run within the returned candidates with matching provenance. A weak or ambiguous candidate is not success. Refine one known filter or use `agent-manager.conversation-recall` when several complementary clues require a bounded multi-query merge [S3]. If output says `Degraded`, keep the lexical evidence but do not claim that semantic candidates were searched. Check `agent-manager conversation index status` before retrying [S1]. A deleted message, purged run, or policy-hidden source must remain absent; do not work around that boundary by scanning raw transcript files.

Do not paste snippets or raw conversation text into Vrooli Memory, source records, prompts, or bug reports. Capture the stable run/hit ID, evidence category, and conclusion instead. Vrooli Memory owns distilled lessons; repository providers own code and docs; Agent Manager owns attributable retained conversation evidence.

### 7. In-use settings

| Symptom | Move | Journal |
|---|---|---|
| Result parses as prose when the caller needs a label | Restart with `--classify a,b,c` | the label set and the run id |
| Result has fields but `run result` shows abstained | Restart with `--result-schema-file`; keep the schema in the caller's repo | the schema path |
| Runs of one job are not grouped in `measures get workload-breakdown` | Set one stable `--workload-key` | the key |
| `run create` retried and made a second run | Add `--idempotency-key` | both run ids |
| Workflow wait returned at the bound, execution still running | Call `execution-wait` again with a longer `--timeout-seconds`; one call, not a loop | the execution id and bound |
| Run stalls on another scenario's output | Park; name the producer and key | producer, key |

### 8. Debug order

1. `agent-manager run get <id>` — status, phase, exit code.
2. `agent-manager run report <id>` — then `investigating-agent-runs`.
3. `agent-manager run events <id> --failed`.
4. `agent-manager workflow trace <execution-id>` for a workflow node.
5. Run `vrooli scenario status agent-manager` once only when the API is unreachable
   or scenario health is part of the evidence. After starting or restarting the
   scenario, use `vrooli scenario wait agent-manager` once for that lifecycle
   operation; do not poll status or logs while it runs.

### 9. Safety

- Never run an agent binary directly to bypass a run; every agent execution is a run.
- `run approve`, `run reject`, and `run apply-investigation` change files; run them only with an operator decision in hand.
- `workflow execution-result` reveals payloads; pass `--explicitly-authorized` only when the caller may see them.
- Do not put secrets in `--prompt`; runs are retained and reported.

### 10. Troubleshooting & Edge Cases

| Symptom | Likely cause | First check | Fix |
|---|---|---|---|
| `run report` through program-runtime fails | The run id is invalid, missing, or the report service is unavailable | `program-runtime bindings describe agent-manager/run/report` and `agent-manager run report <id>` | Supply a real run UUID; retry only when the service reports a transient dependency failure |
| `run create` refuses: "use exactly one of --result-schema, --result-schema-file, or --classify" | Two result specs given | the command line | Keep one |
| `execution-wait` returns immediately with a terminal status you did not expect | The idempotency key matched an earlier execution | `workflow execution-get <id>` `created_at` | Use a new key for new work |
| `measures get <x>` shows `validity=unreliable` | Classified share is below 90 percent, or the sample is under 5 | the `validity_reason` field | Read the number as unreliable; do not quote it as a rate |
| `run wake <id>` refused | The run is not `RUN_STATUS_PARKED` | `run get <id>` | Only a parked run wakes |
| Heartbeat runs crowd `run list` | Heartbeat members create runs every cycle | `run list --tag-prefix <your prefix>` | Filter by tag or workload key |
| An owner command returns `identity_refresh_unavailable` | The ordinary run identity could not be renewed; this can be an intentional scope/expiry boundary, not an Agent Manager outage | Classify the exact command and required principal; use its supported owner credential route if one exists | Do not substitute an operation grant or retry anonymously. Mark only the identity-dependent action blocked, record the refusal, and continue independent work. `test-genie runs wait` already falls back to its inline wait if parking is unavailable; use its terminal result and do not make successful parking a prerequisite for the research itself. |

### Joined supervision reads

For delivery-goal supervision, first run
`program-runtime library run agent-manager.supervision-observation-read --input page_size=8`.
It reads one bounded Agent Manager board page and selects retained unfinished
orchestrator/worker obligations independent of runtime liveness. Preserve
unknown, review-required, blocked, stale and unavailable facts; continue only
with the returned page token. Withdrawn rows are retired. Rows without retained
orchestrator/worker assignments remain visible as admission/completion unknown.
Goal-home contents such as `activeEpoch`, open feedback or queue findings do not
establish native admission or retained ownership; `authorized_by` alone never
establishes admission.

Run `program-runtime library run agent-manager.supervision-case-read --input watch_id=<id>`
to join the latest decision, pending evidence, accepted/applied actions and assessment gaps.
Run `program-runtime library run agent-manager.supervision-experiment-read --input policy_version=<version>`
to join pinned identities, owner evaluation gates and bounded outcome coverage.
These programs do not consume cursors or write assessments. Treat a capped sample
as a sample. Use the existing usage memory loop once after the operational outcome.

An Agent Manager watch pins the immutable evaluator digest stored in its policy.
Changing the declared `supervision-evaluate` program does not update an existing
policy. Before relying on a watch, compare `agent-manager watch policy-get --json`
with the current evaluator digest from
`program-runtime programs portfolio --scenario agent-manager --json`. A missing
program row or different digest is unqualified, not proof of a match. In that
case, follow the candidate, replay, and human-reviewed promotion path, or use
an already-authorized direct supervision route. Never let this mismatch block
independent research or other work that does not require the Agent Manager watch.

### CreateRun caller identity and refusal

CreateRun requires caller proof at both the HTTP and service boundaries, before
reservation or dispatch. The shared CLI transport forwards an existing run
credential in `X-Agent-Identity-Token`; each adapter still needs qualification.
Delegated requests must name that verified run as their exact `parentRunId` and
retain qualification and execution restrictions. An existing human owner
credential belongs in `Authorization: Bearer`, where the owner verifier must
validate it as a live human with the declared `agent-manager:write` capability.
A run credential in the human channel is not human authorization. If both
channels are offered, both must validate for the same owner; scopes and expiry
are intersected. An invalid offered channel cannot be ignored.

Public CreateRun selects existing profiles. Inline defaults that would create
or change a profile must be reconciled through the existing authorized profile
owner route first; identical existing declarations remain no-ops. An accepted
public replay requires the original persisted caller, task, parent and scope
ceiling. Historical anonymous admissions are refused by the public route;
renewed proof does not renew the original run's authority expiry.

After an authentication or policy refusal, preserve the redacted request and
exact response and stop the denied action and dependent work. Do not remove an
identity header, substitute another principal, attach a run, mint credentials,
or retry anonymously. Body lineage, localhost origin, task/profile names and
harness labels do not prove authority. Use an already authorized direct session
only when the owner explicitly supports that work shape; record its actual
scope without claiming native admission.

The AUTH-01 receiving-boundary source correction is not rollout qualification.
Supported caller migration, service-level replay binding, positive verified
operator/exact-parent tests and independent evidence remain deployment gates.
Other lifecycle endpoints retain their existing contracts.

For the authority model behind these rules — who may create a run and how a bounded caller is granted that authority — read [docs/concepts/AUTHORIZATION.md](../../../../docs/concepts/AUTHORIZATION.md).
