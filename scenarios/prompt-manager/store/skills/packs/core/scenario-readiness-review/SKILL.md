## Tools focus: Scenario Readiness Review

Assess whether a scenario's uncommitted changes are ready to commit by querying the git-control-tower unified review endpoint for code health signals, then reviewing the actual git changes for coherence and intent.

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

**This tool is strictly read-only.** You may query, assess, and recommend. You may not modify repository state, approve agent changes, or trigger fixes.

**Note:** The unified review endpoint (`POST /review/run`) triggers read-only scans internally (tidiness, tests, rules). These do not modify source files.

---

### **3. Prerequisites**

**Required scenario:**
- **git-control-tower** — must be running (`vrooli scenario status git-control-tower`)

The review endpoint internally checks availability of integrated scenarios (tidiness-manager, scenario-auditor, test-genie, browser-automation-studio, workspace-sandbox). Partial assessments are returned when some integrations are unavailable — the `capabilities` field in the response indicates which are online.

---

### **4. Core Workflow**

```
Phase 1: Code Health    → Is the code clean, tested, and standards-compliant?
Phase 2: Git State      → What changed, and is it coherent?
Phase 3: Synthesize     → Ready or not?
Phase 4: Act            → Recommend commit or delegate remediation
```

#### Phase 1: Code Health (unified review endpoint)

Trigger a fresh review run with detail items:

```
POST /api/v1/review/run
Body: {"scenarioName": "{{SCENARIO}}", "details": 5}
```

Poll until complete:
```
GET /api/v1/review/run/{jobId}
```

When `status` is `"completed"`, the response includes a `summary` with:
- **`readiness`**: `"green"` | `"yellow"` | `"red"` — the server-authoritative classification
- **`dimensions`**: per-dimension data with top-K detail items:
  - `codeQuality`: score, violations count, `topIssues` (category + count)
  - `tests`: pass/fail counts, `failures` (phase, error, classification, remediation)
  - `standards`: blocking/warning counts, `topViolations` (file, line, title, severity, recommendation)
  - `visual`: screenshot count, `latestCapture` (timestamp, commit hash)
  - `provenance`: traced file count, `untracedFiles` (files with no provenance)
- **`capabilities`**: which integrations were available

For a quick read without triggering new checks:
```
GET /api/v1/review/summary?scenarioName={{SCENARIO}}&details=5
```

**Use the `readiness` field as the authoritative starting classification.** Do not recompute readiness from raw signals — the server applies the canonical formula.

#### Phase 2: Git State (change review)

Regardless of readiness, review what actually changed:

```
GET /api/v1/repo/status?scenario={{SCENARIO}}
```

If no changes exist (no staged, unstaged, or untracked files), stop — there is nothing to review.

```
GET /api/v1/repo/diff?scenario={{SCENARIO}}
GET /api/v1/repo/history?scenario={{SCENARIO}}&limit=20
```

Key signals: file count, change scope (how many distinct areas are touched), whether changes are coherent or sprawling.

#### Phase 3: Synthesize readiness assessment

Combine the review endpoint classification with git state observations:

| Classification | Criteria | Next action |
|----------------|----------|-------------|
| **Ready** | Readiness green + changes coherent | Proceed to commit recommendation (section 5) |
| **Ready with notes** | Readiness green/yellow + minor concerns from git review | Proceed to commit recommendation with caveats |
| **Needs work** | Readiness yellow/red with blocking issues | Delegate remediation (section 6) |
| **Not assessable** | Key integrations unavailable or no changes to review | Report gaps, defer assessment |

---

### **5. Commit Recommendation Protocol**

When a scenario is classified as **Ready** or **Ready with notes**, use the prompt-manager decision system to request commit approval.

```bash
prompt-manager team decision-add director-swarm \
  --by {{AGENT_ID}} \
  --decision "Recommend committing {{SCENARIO}} — {{FILE_COUNT}} files, readiness {{READINESS}}" \
  --rationale "{{REVIEW_SUMMARY_WITH_DETAIL_ITEMS}}" \
  --context "commit-request" \
  --status "pending"
```

**Decision text must include:** scenario name, file count, readiness classification, one-line quality summary.

**Rationale must include:** per-dimension findings with specific data from the review endpoint's detail fields, git change scope assessment, and which dimensions were unavailable.

Do NOT commit — the human handles it.

---

### **6. Delegation Pattern**

When classified as **Needs work**, delegate remediation using specific findings from the review endpoint:

| Issue type | Delegate to | Use detail field |
|-----------|------------|------------------|
| Code quality issues | swarm-manager `execute` item | `topIssues` categories and counts |
| Standards violations | scenario-qa | `topViolations` with file, line, recommendation |
| Test failures | swarm-manager `fix` item | `failures` with phase, error, remediation |
| Untraced changes | Investigation | `untracedFiles` list for human review |

Include specific findings — don't just say "fix quality issues." Cite the exact issues, files, and severity from the detail fields.

---

### **7. Troubleshooting**

| Symptom | Action |
|---------|--------|
| Review endpoint returns 404 | git-control-tower needs updating — check scenario version |
| `capabilities` shows integration unavailable | Start the missing scenario or skip that dimension |
| Readiness is yellow but all dimensions look fine | Check if screenshots are missing (required for green) |
| Review run returns 409 conflict | Another run is in progress — wait for it to complete |

---

### **8. Output Expectations**

**You must produce:**
- Structured readiness assessment using the server's classification
- Specific evidence from detail fields (top violations, test failures, untraced files)
- A pending decision via prompt-manager when recommending commit
- Honest reporting of unavailable dimensions

**You must NOT:**
- Perform any git write operation
- Recompute readiness from raw signals (use the server's `readiness` field)
- Skip the commit recommendation protocol when a scenario is assessed as ready
- Present partial assessments as definitive without noting gaps
