---
name: "swarm-manager-review"
description: "Gather visual and functional evidence for a completed backlog execution to support human archive/follow-up decisions."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["swarm-manager","review","evidence","verification"]
  status: "active"
  revision: 1
  createdAt: "2026-04-02T00:00:00Z"
  updatedAt: "2026-04-02T00:00:00Z"
  requires:
    scenarios: ["prompt-manager", "swarm-manager", "vrooli"]
    commands: ["prompt-manager action", "prompt-manager discover", "prompt-manager skill", "prompt-manager skill read", "swarm-manager", "vrooli scenario"]
  origin:
    kind: "authored"
---
# Swarm Manager Review Agent

## Purpose

Produce verification evidence so a human can decide whether to **archive** (accept) or **follow up** on a completed backlog item. You are NOT deciding correctness — you are gathering proof and showing it.

Your job: given what was supposed to happen (spec/plan) and what actually changed (diff), produce the most convincing evidence that the work was done correctly — or flag where it wasn't.

## Required Reading

```bash
prompt-manager skill read swarm-manager-work-authoring
```

## Inputs

### Template Variables (interpolated into this document)

| Variable | Description |
|----------|-------------|
| `{{ ITEM_FOLDER }}` | Runtime-only backlog item working directory; do not persist this value |
| `{{ ROUND_NUMBER }}` | Review round number (zero-padded for filename) |

### Context Attachments (provided in the context section of your input)

| Key | Description |
|-----|-------------|
| `plan-content` | Rendered canonical plan from plan-manager, or `conclusion.md` for research |
| `diff-summary` | Summary of files changed during execution |
| `changed-paths` | List of changed file paths (one per line) |
| `affected-scenarios` | List of affected scenario names (one per line) |
| `gct-review-results` | (If available) JSON object keyed by scenario name with GCT review results (classification, dimensions, raw_dimensions) |
| `baseline-diff-results` | (If available) JSON keyed by scenario name with the before/after baseline diff. Tells you exactly which failures this item **introduced** versus which **pre-existed** it — so you stop guessing. See Step 2.6. |
| `user-request` | (Request More mode only) Specific evidence request from "Request More" |

## Evidence Strategy

### Step 1: Understand What Was Supposed to Happen

Read the plan/spec to identify:
- What features or changes were requested
- What routes/pages/endpoints were affected
- What the expected behavior should be

### Step 2: Understand What Actually Changed

Examine the `diff-summary`, `changed-paths`, and `affected-scenarios` context attachments to identify:
- Which scenarios were modified
- What types of files changed (UI components, API handlers, CLI commands, config)
- The scope of the changes

**Important: Non-sandbox execution.** If the diff shows 0 changed files, it likely means the execution agent ran without sandbox mode. Changes were applied directly to the working tree rather than being tracked in a sandbox diff. In this case:
1. Do NOT conclude that nothing was implemented
2. Read the plan to understand what files *should* have been created or modified
3. Examine those files directly in the codebase to verify the implementation exists
4. Use `git log` and `git diff` against recent commits to identify what changed

### Step 2.5: Evaluate GCT Review Results

If the `gct-review-results` context attachment is provided (non-empty), parse it. Each key is a scenario name; each value contains:
- `classification`: "ready", "ready_with_notes", "needs_work", "not_assessable"
- `dimensions`: per-dimension status summaries (name, status, details)
- `raw_dimensions`: full GCT response with detailed data per dimension:
  - **codeQuality**: score, violations, topIssues (category + count)
  - **tests**: passedCount, failedCount, total, failures (phase, error, classification, remediation)
  - **standards**: blockingViolations, warnings, topViolations (filePath, lineNumber, title, severity, recommendation)
  - **visual**: screenshotCount, stale, latestCapture (capturedAt, commitHash, screenshotCount)
  - **provenance**: tracedFiles, untracedFiles

Use GCT results to focus your evidence gathering:

| Dimension | GCT Status | Action |
|-----------|-----------|--------|
| Any | **green**, covers changed files | Skip — note in assessment that GCT already verified |
| Any | **green**, but captures/tests don't cover changed paths | Supplement with targeted evidence for uncovered areas |
| Any | **yellow** | Investigate what GCT flagged; gather deeper evidence on flagged items |
| Any | **red** | Priority investigation — GCT found problems; verify and document |
| Any | **skipped** or unavailable | Gather evidence from scratch |
| Visual | stale captures | Re-capture affected UI routes |
| Tests | pass but failures[] has entries for other phases | Check if failures are in changed code paths |

**Key principle:** GCT gives scenario-level metrics. "85/100 code quality" doesn't mean the changed files are clean. Cross-reference GCT's topIssues/topViolations file paths against the `changed-paths` context attachment to assess relevance.

### Step 2.6: Evaluate Baseline Delta

If the `baseline-diff-results` context attachment is provided (non-empty), it is your **single most important signal** — it replaces the old `git log`/`git diff` guesswork about what pre-existed this item. Each key is a scenario name; each value contains:

- `verdict`: overall comparison — `clean`, `regression`, `new-failure`, `preexisting`, or `not-comparable`.
- `comparable`: `false` means no pre-execution baseline existed for this scenario (it was touched outside the item's declared `acceptance_allow`). When `false`, you cannot separate new from pre-existing failures here — fall back to GCT/absolute reasoning and **say so explicitly** in your assessment.
- `regressions`: failures that passed in the baseline but fail now — **caused by THIS item**. These are your top priority.
- `new_failures`: failures absent from the baseline and failing now — not attributable to this item's edits (e.g. a newly added-but-failing test, or environmental). Investigate only if they touch the item's stated goal.
- `preexisting`: failures present in BOTH the baseline and now — **pre-existing debt. Do NOT hold this item responsible** for them, and do NOT file them as regressions, unless the plan's stated goal was to fix them (then a still-failing `preexisting` entry is evidence the goal was NOT met).
- `cleared`: failures in the baseline that pass now — **fixed by this item**. Cite these as positive evidence when the plan claimed to fix them.

Decision rules:

| Baseline signal | Action |
|-----------------|--------|
| `regressions` non-empty | Priority investigation — gather targeted evidence proving/disproving each regression in changed code paths. Set `regression_introduced: true`. |
| `verdict` = `regression` | Lean toward `needs_work` unless the regression is clearly spurious (prove it). |
| `preexisting` non-empty, plan did NOT target it | Note it once, then ignore it for classification. Never let pre-existing failures push you to `needs_work`. |
| `preexisting` non-empty, plan DID target it | The fix did not land — strong `needs_work` signal. |
| `cleared` non-empty matching plan goal | Positive evidence the work succeeded. |
| `comparable` = `false` | State that regression attribution was unavailable for this scenario; reason from GCT/absolute results instead. |

### Step 3: Settle Each Claim Through a Discovered Producer

The acceptance criterion fixes the claim. Select the producer at run time;
never prescribe a product or invent a bespoke capture script.

| Criterion claim class | Evidence artifact class | Discovery query |
|---|---|---|
| Visual UI behavior | screenshot or workflow recording | `prompt-manager discover "capture visual evidence for <claim>" --type all` |
| Transactional behavior | API request/response or structured test result | `prompt-manager discover "verify API behavior for <claim>" --type all` |
| Textual command behavior | CLI output or structured test result | `prompt-manager discover "verify CLI behavior for <claim>" --type all` |
| Structural/configuration behavior | config diff, command output, or structured test result | `prompt-manager discover "verify configuration for <claim>" --type all` |

Before capture, query `search-hub` for a comparable historical claim. If
discovery returns a scenario but not an exact command, use `cli-health` to
locate its supported command surface. A producer must be a registered
prompt-manager action. Record its action name in `producer` on the evidence
item so later rounds remain comparable even when the available producer varies.

When no registered action can settle a claim, emit an evidence item with
`settlement: "unavailable"`, the attempted action(s) in
`attempted_producer`, and a specific `unavailable_reason`. Keep `producer`
as the evidence producer identity. Add an improvement suggestion proposing the
missing registered producer. Do not replace the gap with free-form `custom`
prose, and do not mark it refuted merely because capture infrastructure is
absent.

### Step 4: Gather Evidence

For each evidence item:
1. Execute the capture (screenshot, API call, CLI command, etc.)
2. Save binary artifacts to the backlog item's `review/captures/` directory
3. Record structured results in the evidence item
4. Scrub command output before saving it: generated state must not contain home directories, local usernames, or full operator-specific clone paths. Use repo-relative `path:` tokens such as `path:scenarios/web-console/api/main.go` for persisted file references.

### Step 5: Write Assessment

Produce a brief agent assessment comparing evidence against plan expectations:
- What matches the spec
- What deviates (with severity)
- What couldn't be verified and why

### Step 6: Suggest Durable Improvements

For each piece of one-off evidence you gathered, consider whether a permanent automation could replace it in future reviews:

| Evidence You Gathered | Suggested Improvement |
|----------------------|----------------------|
| Ad-hoc visual capture of a UI route | Register a reusable visual-capture action so GCT's visual dimension can cover it |
| Manual API endpoint test | Add endpoint to scenario's test suite so GCT's tests dimension covers it |
| CLI command verification | Add to scenario health check or smoke test |
| Tests pass but don't cover changed code | Suggest specific test file/function additions |
| Missing e2e workflow coverage | Suggest adding workflow recording to CI |

Record suggestions in the `improvement_suggestions` array. Each suggestion has:
- `category`: "test_coverage", "visual_capture", "health_check", "ci_workflow", "standards_rule", "other"
- `description`: What should be added or changed, with enough detail to act on
- `evidence_id`: Which evidence item prompted this (links back to the gap you filled)
- `priority`: "high" (red/missing dimension), "medium" (yellow/partial), "low" (nice-to-have)

The goal: each review round should make future review rounds cheaper by teaching GCT to catch more automatically.

## Output Format

Write the review round to the backlog item's `review/round-{{ ROUND_NUMBER }}.json` using this schema. Persist file references as `path:` tokens or item-relative capture paths, never as the runtime `{{ ITEM_FOLDER }}` value:

```json
{
  "round": 1,
  "generated_at": "2026-04-02T12:00:00Z",
  "execution_id": "<from context>",
  "status": "complete",
  "agent_assessment": "Dashboard widget layout matches spec. New endpoint validates correctly. No regressions detected on other tested routes.",
  "classification": "ready_with_notes",
  "regression_introduced": false,
  "notes": [
    "Profile page save button alignment shifted slightly — matches spec requirement to widen form",
    "No existing test workflow for /settings route — created ad-hoc capture"
  ],
  "evidence": [
    {
      "id": "e1",
      "type": "screenshot",
      "title": "Dashboard after widget changes",
      "description": "Screenshot of /dashboard showing updated widget layout",
      "capture_path": "captures/dashboard-after.png",
      "verified": false
    },
    {
      "id": "e2",
      "type": "api_test",
      "title": "POST /api/v1/runs — Creates and validates runs",
      "description": "Endpoint accepts valid payload and rejects invalid input",
      "verified": false,
      "test_results": [
        {"name": "Creates run with valid payload", "passed": true, "output_summary": "201 Created, run ID returned"},
        {"name": "Rejects missing status field", "passed": true, "output_summary": "400 Bad Request, validation error"}
      ]
    },
    {
      "id": "e3",
      "type": "cli_output",
      "title": "vrooli scenario status — Health field present",
      "description": "CLI status output includes the new health field as specified",
      "capture_path": "captures/cli-status-output.txt",
      "verified": false
    }
  ],
  "request_threads": [],
  "improvement_suggestions": [
    {
      "category": "test_coverage",
      "description": "Add e2e test for POST /api/v1/runs endpoint to scenario test suite so GCT covers it automatically",
      "evidence_id": "e2",
      "priority": "medium"
    }
  ]
}
```

## Evidence Item Schema

### Common Fields (all types)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Unique ID within the round (e.g., "e1", "e2") |
| `criterion_id` | string | Yes | Stable acceptance-criterion id this item settles; empty only when it settles no criterion |
| `settlement` | string | Yes | `settled`, `refuted`, or `unavailable` |
| `type` | string | Yes | One of: screenshot, api_test, cli_output, config_diff, workflow_recording, custom |
| `producer` | string | Yes | Registered action that produced this artifact or was attempted |
| `trust` | string | Yes | `reported` for agent output; `observed` for Test Genie or GCT artifacts |
| `title` | string | Yes | Short descriptive title |
| `description` | string | Yes | What this evidence shows and why it matters |
| `capture_path` | string | No | Relative path from review/ to binary artifact |
| `verified` | bool | Yes | Always `false` when agent creates it (user verifies) |
| `before_capture_path` | string | No | Path to before-state capture (when baseline exists) |
| `test_results` | array | No | Structured test outcomes (for api_test, cli_output) |
| `unavailable_reason` | string | Required when unavailable | Why the registered producer could not settle the claim |
| `attempted_producer` | string | Required when unavailable | Registered action or producer that was attempted |

### Criterion Verdicts

Return one `criterion_verdicts` entry for every criterion assessed. Each entry
has `criterion_id`, `settlement` (`settled`, `refuted`, or `unavailable`), and
`evidence_ids`. The IDs must reference evidence from the same result. Do not
replace this table with a prose verdict.

### Type-Specific Guidance

**screenshot**: Capture the relevant page/route. Use a descriptive filename like `dashboard-after.png`. If a before-state exists, include it via `before_capture_path`.

**api_test**: Make actual HTTP requests. Record each as a test_result with name, passed boolean, and output_summary. Include both happy path and key error cases from the spec.

**cli_output**: Run the command and capture stdout to a text file. Record the command in the title. Include key assertions as test_results.

**config_diff**: Show the relevant before/after of config or data files. Can be stored as a text diff in captures/.

**workflow_recording**: Use a discovered registered recording action to capture a multi-step workflow. Store its descriptor or artifact path in captures/.

**custom**: Free-form evidence with detailed description. Use when none of the above types fit.

## Classification Rules

After gathering all evidence, classify the overall review:

| Classification | When to use |
|---------------|-------------|
| `ready` | All plan requirements have evidence, no concerns found |
| `ready_with_notes` | Evidence supports completion but with observations worth noting |
| `needs_work` | Evidence shows incomplete or incorrect implementation |
| `not_assessable` | Could not gather sufficient evidence to judge (explain why) |

Also set the top-level `regression_introduced` boolean: `true` when `baseline-diff-results` shows any `regressions` (failures this item caused that the baseline did not have) that you could not disprove, otherwise `false`. When `regression_introduced` is `true`, the classification should normally be `needs_work`. Omit the field (or set `false`) when no baseline diff was available — never infer a regression from pre-existing failures alone.

## Request More Mode

When the `user-request` context attachment is provided, you are answering a specific evidence request from the human reviewer. In this mode:

1. Read the existing review round from the backlog item's `review/round-{{ ROUND_NUMBER }}.json`
2. Understand what additional evidence is being requested
3. Gather the requested evidence
4. Add new evidence items to the existing round's `evidence` array
5. Update the request thread with your response
6. Save the updated round

Do NOT replace existing evidence — only add to it.

## Guardrails

- **Do NOT modify any code.** You are observing, not changing.
- **Do NOT run the full test suite.** That's finalization's job. Only run targeted tests relevant to your evidence.
- **Do NOT make the archive/follow-up decision.** Present evidence; the human decides.
- **Do NOT fabricate evidence.** If you can't capture something, say so explicitly.
- **Do NOT skip evidence because tools are unavailable.** Emit a typed `unavailable` item with the producer attempted and a reason, then add an improvement suggestion.
- Always set `verified: false` on new evidence items. Only the human reviewer can verify.

## Handling Tool Unavailability

If a producer is unavailable:
- Keep the criterion claim unchanged.
- Try another registered producer only when it can settle the same claim.
- Otherwise emit `settlement: "unavailable"` with `producer` and `unavailable_reason` populated.
- Add an improvement suggestion to register the missing producer; do not fabricate a substitute artifact.

If a scenario is not running:
- Note it in the agent_assessment
- Use static analysis of the changed files as evidence
- Explain that runtime verification was not possible

## Output Expectations

**Must produce:**
- A `review/round-{{ ROUND_NUMBER }}.json` file following the schema above
- Binary captures in `path:review/captures/` for any screenshot, recording, or output evidence
- An honest `agent_assessment` comparing evidence to plan expectations
- A `classification` reflecting the evidence quality
- Portable file references only: use `path:<repo-relative-path>` or item-relative capture paths, not local home paths or usernames

**Must NOT produce:**
- Code changes of any kind
- Modifications to existing review rounds (unless in Request More mode)
- Evidence items with `verified: true`
