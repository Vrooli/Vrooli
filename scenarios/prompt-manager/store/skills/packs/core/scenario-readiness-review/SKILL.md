## Tools focus: Scenario Readiness Review

Assess whether a scenario's uncommitted changes are ready to commit by synthesizing signals from code quality, standards compliance, test results, change provenance, and git state through the git-control-tower API.

---

### **1. When to Use / When Not to Use**

| Use when | Don't use when |
|----------|----------------|
| Assessing if a scenario's changes are ready for commit | You need to actually commit changes (human responsibility) |
| Producing a readiness briefing for the director | Debugging a specific bug (use scientific-debugging) |
| Reviewing quality of recent agent work on a scenario | Building new features (use scenario-feature team) |
| Auditing what agent runs contributed to current changes | Managing deployments (use scenario-to-cloud) |
| Checking scenario health before making priority decisions | Running tests directly (use test-genie) |

---

### **2. Guardrails — READ FIRST**

**Absolute prohibitions — no exceptions:**

| Endpoint | Why it's prohibited |
|----------|-------------------|
| `POST /repo/commit` | Committing is a human responsibility |
| `POST /repo/push` | Pushing is a human responsibility |
| `POST /repo/pull` | Modifies repository state |
| `POST /repo/stage` | Modifies staging area |
| `POST /repo/unstage` | Modifies staging area |
| `POST /repo/discard` | Destroys uncommitted work |
| `POST /agent/run` | Creates new agent work |
| `POST /agent/runs/{id}/approve` | Applies sandbox changes to repo |
| `POST /agent/runs/{id}/reject` | Modifies agent run state |
| `POST /repo/rules-fix` | Modifies source files |

**This tool is strictly read-only.** You may query, assess, and recommend. You may not modify repository state, approve agent changes, or trigger fixes. Even after a commit recommendation is accepted by a human, the director-swarm does not perform the commit — the human handles it.

**Allowed write-like operations** (these trigger read-only scans, not file modifications):
- `POST /repo/rules-run` — starts a standards compliance scan
- `POST /repo/tidiness-scan` — triggers a code quality scan

---

### **3. Prerequisites**

**Required scenario:**
- **git-control-tower** — must be running (`vrooli scenario status git-control-tower`)

**Integrated scenarios** (checked at runtime via `GET /capabilities`):

| Scenario | Provides | Impact if unavailable |
|----------|----------|----------------------|
| tidiness-manager | Code quality scores and issues | Skip quality dimension |
| scenario-auditor | Standards/rules compliance | Skip compliance dimension |
| test-genie | Test execution results | Skip testing dimension |
| agent-manager | Agent run history | Skip run detail in provenance |
| workspace-sandbox | Change provenance (sandbox-to-repo mapping) | Skip provenance dimension |

Partial assessments are still valuable. Always note which dimensions were unavailable rather than blocking the entire review.

---

### **4. Core Workflow**

```
Step 1: Integration health    → What can we assess?
Step 2: Git state              → What changed?
Step 3: Code quality           → Is it clean?
Step 4: Standards compliance   → Does it follow rules?
Step 5: Test results           → Does it work?
Step 6: Change provenance      → Who changed it and why?
Step 7: Synthesize assessment  → Ready or not?
Step 8: Act on classification  → Recommend commit or delegate remediation
```

#### Step 1: Check integration health

```
GET /capabilities
```

Record which integrations report healthy. This determines which subsequent steps are possible. Proceed with whatever is available.

#### Step 2: Get git state

Current file changes:
```
GET /repo/status?scenario={{SCENARIO}}
```

If no changes exist (no staged, unstaged, or untracked files), stop — there is nothing to review.

Actual diff content:
```
GET /repo/diff?scenario={{SCENARIO}}
```

Recent commit history for context:
```
GET /repo/history?scenario={{SCENARIO}}&limit=20
```

Key signals: file count, change scope (how many distinct areas are touched), whether changes are coherent or sprawling.

#### Step 3: Assess code quality

*Skip if tidiness-manager is unavailable.*

```
GET /repo/tidiness-score?scenario={{SCENARIO}}
GET /repo/tidiness-issues?scenario={{SCENARIO}}
```

Check freshness first — stale scores may not reflect current state:
```
GET /repo/tidiness-staleness?scenario={{SCENARIO}}
```

If stale, trigger a fresh scan:
```
POST /repo/tidiness-scan   (body: {"scenario": "{{SCENARIO}}"})
```

Then re-query score and issues.

Key signals: overall grade, issue count by severity, whether critical issues exist.

#### Step 4: Check standards compliance

*Skip if scenario-auditor is unavailable.*

Check for existing violations:
```
GET /repo/rules-violations?scenario={{SCENARIO}}
```

If no recent scan exists or data is stale, trigger one:
```
POST /repo/rules-run       (body: {"scenario": "{{SCENARIO}}"})
```

Poll for completion:
```
GET /repo/rules-job/{jobId}
```

Then re-query violations.

Key signals: violation count, severity distribution, whether any are blocking.

#### Step 5: Review test results

*Skip if test-genie is unavailable.*

```
GET /repo/test-executions?scenario={{SCENARIO}}
```

For the most recent execution:
```
GET /repo/test-executions/{id}
```

Key signals: pass/fail status, phase-by-phase results, failure details if any.

#### Step 6: Review change provenance

*Skip if workspace-sandbox is unavailable.*

Which agent runs produced which file changes:
```
GET /repo/provenance?scenario={{SCENARIO}}
```

Sandbox-approved pending changes:
```
GET /repo/approved-changes?scenario={{SCENARIO}}
```

For deeper context on a specific contributing agent run:
```
GET /agent/runs?scenario={{SCENARIO}}
GET /agent/runs/{id}/events
GET /agent/runs/{id}/diff
```

Key signals: what percentage of changes have traceable provenance, what the agent was asked to do vs. what it changed, whether changes align with stated intent.

#### Step 7: Synthesize readiness assessment

Combine all signals into a structured assessment:

| Dimension | Source | Ready signal | Blocker signal |
|-----------|--------|-------------|----------------|
| Changes exist | git status | Staged or unstaged changes present | No changes (nothing to review) |
| Code quality | tidiness-manager | Grade B or above, no critical issues | Grade D/F or critical issues present |
| Standards compliance | scenario-auditor | Zero blocking violations | Blocking violations present |
| Tests passing | test-genie | Most recent run passed | Tests failing or no recent run |
| Provenance clear | workspace-sandbox | High coverage, changes traceable to approved runs | Unexplained changes, no provenance |
| Change scope | git diff | Changes are coherent and focused | Sprawling, unrelated changes mixed together |

Classify the scenario:

| Classification | Criteria | Next action |
|----------------|----------|-------------|
| **Ready** | All assessed dimensions green | Proceed to commit recommendation protocol (section 5) |
| **Ready with notes** | Minor issues that don't block commit | Proceed to commit recommendation protocol with caveats |
| **Needs work** | One or more blocking issues | Delegate remediation (section 6) |
| **Not assessable** | Key integrations unavailable or data too stale | Report gaps, defer assessment |

---

### **5. Commit Recommendation Protocol**

When a scenario is classified as **Ready** or **Ready with notes**, use the prompt-manager decision system to request commit approval from the human operator.

#### Log the pending decision

```bash
prompt-manager team decision-add director-swarm \
  --by {{AGENT_ID}} \
  --decision "Recommend committing {{SCENARIO}} — {{FILE_COUNT}} files, {{QUALITY_SUMMARY}}" \
  --rationale "{{DETAILED_FINDINGS}}" \
  --context "commit-request" \
  --status "pending"
```

**Decision text must include:**
- Scenario name and uncommitted file count
- One-line quality summary (e.g., "tidiness B+, 0 violations, tests passing, 89% provenance coverage")
- Any caveats if classified as "ready with notes"

**Rationale must include:**
- Per-dimension assessment findings with specific data
- List of contributing agent runs with brief descriptions (if provenance available)
- Any risks or items the human reviewer should pay attention to
- Which dimensions were unavailable and why

#### On subsequent heartbeats, check decision status

```bash
prompt-manager team decision-list director-swarm --context commit-request
```

| Decision status | Action |
|----------------|--------|
| `pending` | No action — awaiting human review |
| `accepted` | Note in team handoff. Do NOT commit — the human handles it. |
| `rejected` | Read feedback. Adjust future assessments or delegate remediation as indicated. |

#### Supersede stale recommendations

If a scenario's state has materially changed since the last pending recommendation (new commits landed, new agent runs completed, quality score shifted), supersede the old decision:

```bash
prompt-manager team decision-add director-swarm \
  --by {{AGENT_ID}} \
  --decision "Updated: Recommend committing {{SCENARIO}} — {{FILE_COUNT}} files, {{UPDATED_SUMMARY}}" \
  --rationale "Supersedes previous assessment due to: {{WHAT_CHANGED}}" \
  --context "commit-request" \
  --status "pending" \
  --supersedes {{OLD_DECISION_ID}}
```

---

### **6. Delegation Pattern**

When a scenario is classified as **Needs work**, the operations-chief can delegate specific remediation to the appropriate team:

| Issue type | Delegate to | Example |
|-----------|------------|---------|
| Code quality issues (high severity) | scenario-refactor | "Resolve 3 critical tidiness issues in tunnel-manager before commit" |
| Standards violations | scenario-qa | "Fix blocking rule violations in prompt-manager API handlers" |
| Test failures | scenario-debug | "Investigate test-genie failures in agent-manager integration tests" |
| Missing test coverage | scenario-qa | "Add test coverage for new tunnel-manager endpoints" |
| Unexplained changes (no provenance) | Investigation | Flag in intelligence briefing for human review |

Communicate assignments through normal team channels:

```bash
prompt-manager team message-send director-swarm \
  --from operations-chief \
  --to {{TARGET_AGENT}} \
  --content "{{REMEDIATION_REQUEST_WITH_SPECIFIC_FINDINGS}}"
```

Include specific findings from the readiness review — don't just say "fix quality issues." Cite the exact issues, files, and severity.

---

### **7. Troubleshooting & Edge Cases**

| Symptom | Likely cause | First check | Action |
|---------|-------------|-------------|--------|
| `GET /capabilities` shows integration unavailable | Scenario not running | `vrooli scenario status {{name}}` | Start the scenario or skip that dimension |
| Tidiness score seems stale | No recent scan triggered | `GET /repo/tidiness-staleness` | Trigger `POST /repo/tidiness-scan`, then re-query |
| Rules scan returns no job ID | scenario-auditor overloaded | `GET /capabilities` auditor status | Wait briefly and retry, or skip dimension |
| Provenance shows very low coverage | Changes made outside sandbox workflow | Compare provenance file list to git status | Note as caveat — changes exist without tracked origin |
| No test executions found | Tests never run for this scenario | `GET /repo/test-executions` returns empty | Note as gap in assessment, recommend running tests before commit |
| Assessment dimensions give conflicting signals | e.g., tests pass but quality grade is low | Review each signal independently | Classify as "ready with notes" and cite the conflict |
| Multiple pending commit-request decisions exist | Previous ones not reviewed | `prompt-manager team decision-list --context commit-request` | Supersede stale ones, keep only the current assessment |

---

### **8. Output Expectations**

**You must produce:**
- Structured readiness assessment covering all available dimensions
- Clear classification: ready / ready with notes / needs work / not assessable
- Specific evidence for each dimension (scores, counts, pass/fail, coverage percentages)
- A pending decision via prompt-manager when recommending commit
- Honest reporting of unavailable dimensions rather than skipping them silently

**You must NOT:**
- Perform any git write operation under any circumstance
- Approve or reject agent sandbox changes
- Trigger automated fixes via rules-fix
- Skip the commit recommendation protocol when a scenario is assessed as ready
- Present partial assessments as definitive without noting gaps
- Create commit recommendations without specific evidence from the review
