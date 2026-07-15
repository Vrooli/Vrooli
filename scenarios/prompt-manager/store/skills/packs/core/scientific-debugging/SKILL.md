## Practice focus: Scientific Debugging

Apply the **scientific method to debugging**: generate falsifiable hypotheses, design experiments (tests) to validate them, and systematically narrow down to the root cause. This methodology produces regression tests and documented findings that prevent recurrence.

Required reading:
- `docs/scenario-qa/methods/investigation/scientific-debugging.md` — strategic-canon home: when this technique applies, when it backfires, what the qa-contrarian challenges.
- `docs/agent-system/SKILL_AUTHORING.md`

Optional reading:
- `prompt-manager skill read skill-authoring-practice`

---

### **1. When to Use This Methodology**

Use Scientific Debugging when:
- A bug's cause is not immediately obvious
- Initial fixes didn't work or made things worse
- The bug involves multiple interacting components
- You need to explain the root cause to others
- You want to prevent similar bugs in the future

**Do NOT use** for:
- Typos or obvious one-line fixes
- Well-understood, documented error conditions
- Issues where the fix is already known

**Always start with Phase 0 (Prior-Art Check)** when the bug targets a
scenario, even for bugs that look "new". If Phase 0 surfaces a likely
recurrence, the rest of the methodology is short-circuited — see Phase 0
for details.

---

### **2. The Process**

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                        SCIENTIFIC DEBUGGING PROCESS                          │
├──────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│   ┌──────────┐     ┌─────────┐     ┌─────────────┐     ┌─────────┐          │
│   │ PRIOR    │ ──▶ │ OBSERVE │ ──▶ │ HYPOTHESIZE │ ──▶ │  TEST   │ ──▶ ...  │
│   │ ART      │     └─────────┘     └─────────────┘     └─────────┘          │
│   │ (PHASE 0)│                                                               │
│   └──────────┘                                                               │
│                                                                              │
│   Phase 0 has three exits:                                                   │
│     • No prior art      → continue to OBSERVE                                │
│     • Related prior art → read it, then OBSERVE with priors in mind          │
│     • Likely recurrence → reopen / spawn-from prior fix; STOP investigating  │
│       fresh; the prior investigation transcript is now your evidence.        │
│                                                                              │
└──────────────────────────────────────────────────────────────────────────────┘
```

---

### **Phase 0: Prior-Art Check** (mandatory)

**Entry criteria:** A bug has been reported against a known scenario, or a
debugging task is about to begin.

**Why this phase exists:** Bugs recur. Without a deliberate prior-art check, an
agent will re-investigate something the system has already solved, miss
related context that would shorten the new investigation, or open a duplicate
fix. Skipping this phase wastes effort and corrodes institutional memory.

**Actions** (start with the federated pass; the two scoped passes add depth, they are not redundant):

1. **Identify the affected scenario.** If multiple scenarios are involved,
   pick the most-affected one for the scoped passes; the federated pass covers
   the rest.

2. **Pass 1 — Federated recall (start here).** One semantic query across the
   records corpus (the recursive-learning memory). Each `record` hit carries
   *how* a prior agent solved it — its trigger and approach — not just a link.

   ```bash
   search-hub query "<one-sentence symptom>" --type record
   ```

   Widen to `--type record,doc` to also pull in any written explanation of the
   subsystem. This is the AGENTS.md §4 "Recall prior work first" beat applied to
   debugging.

3. **Pass 2 — Scenario-local fix history (depth / fallback).** List every prior
   fix targeting this scenario, including archived ones — scoped deeper than the
   federated pass, and the fallback when search-hub is unavailable:

   ```bash
   swarm-manager scenarios fixes --name "<scenario>" --all --search "<symptom keywords>"
   ```

   Without `--search`, scope by recency:

   ```bash
   swarm-manager scenarios fixes --name "<scenario>" --all --limit 20
   ```

4. **Pass 3 — Direct fix-corpus semantic recall (fallback).** The records-only
   AI search that search-hub federates; use it directly when search-hub is down,
   or to scope by `--kind fix` and target scenario. Archived items are included
   by default when `--kind fix` is set.

   ```bash
   swarm-manager ai-search query "<one-sentence symptom>" --kind fix --target-scenario "<scenario>"
   ```

   If AI search reports `fallback: unavailable`, run `swarm-manager ai-search
   status` to confirm the index is reachable, then proceed with pass 2 only
   and note the gap in the investigation log.

**Required output** — Phase 0 must conclude with exactly one of:

- **No prior art.** Document the queries you ran (so a reviewer can confirm
  the search was real) and proceed to Phase 1: Observe.
- **Related prior art.** Link each related fix item by name, summarize how it
  was resolved in one or two sentences, and proceed to Phase 1 with those
  priors as starting hypotheses or constraints.
- **Likely recurrence.** Link the prior fix item, recommend reopening or
  spawning a follow-up fix linked via `spawned_from`, and STOP. Do not start
  a fresh investigation — the prior investigation transcript is now your
  evidence base. Re-enter Phase 1 only if the recurrence hypothesis is
  ruled out by inspection.

**Exit criteria:**

- [ ] Both pass 1 (scenario-local CLI) and pass 2 (semantic) commands have
      been executed and their outputs captured.
- [ ] One of the three required outputs above is on the record.
- [ ] If "Likely recurrence", the existing fix item is linked and a decision
      to reopen / spawn-from is documented before any new code change.

**Tool contract:** All prior-art lookups go through the scenario CLI. Do not
substitute raw HTTP, `grep`, or filesystem walks — those bypass the same
indexes the rest of the system uses, miss archived items, and produce
inconsistent results across investigations. If a needed query is missing from
the CLI, add it to the CLI first; do not work around.

---

### **Phase 1: Observe**

**Entry criteria:** A bug or unexpected behavior has been reported or discovered.

**Actions:**
1. **Reproduce the bug** — Confirm you can trigger it consistently
2. **Gather symptoms** — Collect error messages, logs, stack traces, screenshots
3. **Identify the delta** — What changed? When did it start? What was different before?
4. **Define the expected vs actual behavior** — Be precise

**Exit criteria:**
- [ ] Bug is reproducible (or documented as intermittent with conditions)
- [ ] Symptoms are documented
- [ ] Expected behavior is clearly defined

**Artifacts:**
- Bug reproduction steps
- Collected evidence (logs, errors, screenshots)

---

### **Phase 2: Hypothesize**

**Entry criteria:** Bug is observed and documented.

**Actions:**
1. **Generate multiple hypotheses** — List at least 2-3 possible causes
2. **Prioritize by likelihood** — Use evidence to rank hypotheses
3. **Make each hypothesis falsifiable** — Define what would prove it wrong
4. **Consider the "Five Whys"** — Dig deeper than surface causes

**Hypothesis Template:**
```markdown
### Hypothesis [N]: [Brief description]

**Claim:** [Specific, testable statement about the cause]

**If true, we would expect:**
- [Observable consequence 1]
- [Observable consequence 2]

**If false, we would see:**
- [Evidence that would disprove this]

**Test:** [How to validate/invalidate this hypothesis]

**Likelihood:** [High/Medium/Low] because [reasoning]
```

**Exit criteria:**
- [ ] At least 2 hypotheses generated
- [ ] Each hypothesis is falsifiable
- [ ] Hypotheses are prioritized

**Artifacts:**
- Documented hypotheses with test plans

---

### **Phase 3: Test**

**Entry criteria:** Hypotheses are documented with test plans.

**Actions:**
1. **Start with highest-likelihood hypothesis**
2. **Design a minimal test** that would confirm or reject it
3. **Execute the test** — Add logging, write a test case, use a debugger
4. **Record results** — What did you observe?

**Test Design Guidelines:**

| Test Type | When to Use | Example |
|-----------|-------------|---------|
| Add logging | Tracing data flow | Log values at key points |
| Write unit test | Isolated component | Test function with specific inputs |
| Add assertions | Validate assumptions | Assert expected state at checkpoints |
| Binary search | Large codebase | Comment out half the code |
| Minimal reproduction | Complex system | Smallest code that triggers bug |

**Exit criteria:**
- [ ] Test executed
- [ ] Results recorded
- [ ] Hypothesis confirmed or rejected

**Artifacts:**
- Test code or logging additions
- Test results documentation

---

### **Phase 4: Analyze**

**Entry criteria:** Test results are available.

**Actions:**
1. **Evaluate results against hypothesis**
   - If confirmed: Proceed to Fix phase
   - If rejected: Return to Hypothesize with new information
2. **Update understanding** — What did you learn?
3. **Check for secondary effects** — Could this cause other issues?

**Decision Table:**

| Test Result | Hypothesis Status | Next Action |
|-------------|-------------------|-------------|
| Evidence supports hypothesis | Confirmed | Proceed to Fix |
| Evidence contradicts hypothesis | Rejected | Generate new hypothesis |
| Evidence is inconclusive | Needs refinement | Design better test |
| Unexpected evidence found | New information | Incorporate into new hypothesis |

**Exit criteria:**
- [ ] Root cause identified (hypothesis confirmed)
- [ ] OR new hypothesis generated (return to Phase 2)

**Artifacts:**
- Analysis notes
- Updated hypothesis status

---

### **Phase 5: Fix**

**Entry criteria:** Root cause is identified and confirmed.

**Actions:**
1. **Write a failing test first** — Captures the bug as a regression test
2. **Implement the fix** — Address the root cause, not symptoms
3. **Run the failing test** — Confirm it now passes
4. **Run full test suite** — Ensure no regressions

**Fix Checklist:**
- [ ] Fix addresses the root cause, not just symptoms
- [ ] Failing test written BEFORE the fix
- [ ] Test passes AFTER the fix
- [ ] Full test suite still passes
- [ ] No new warnings or errors introduced

**Exit criteria:**
- [ ] Test that reproduces bug now passes
- [ ] Full test suite passes
- [ ] Fix is minimal and focused

**Artifacts:**
- Regression test
- Fix implementation

---

### **Phase 6: Verify**

**Entry criteria:** Fix is implemented and tests pass.

**Actions:**
1. **Manual verification** — Reproduce original bug steps, confirm fixed
2. **Edge case testing** — Test boundary conditions
3. **Document the root cause** — Explain WHY it happened
4. **Check for similar patterns** — Could this bug exist elsewhere?

**Root Cause Documentation Template:**
```markdown
## Root Cause Analysis

**Bug:** [Brief description]

**Symptom:** [What users/systems observed]

**Root Cause:** [Technical explanation of WHY]

**Fix:** [What was changed]

**Prevention:** [How to prevent similar bugs]

**Related Areas:** [Other code that might have same issue]
```

**Exit criteria:**
- [ ] Bug confirmed fixed via manual testing
- [ ] Root cause documented
- [ ] Related code checked for similar issues

**Artifacts:**
- Root cause documentation
- PR/commit with detailed explanation

---

### **3. Convergence Patterns**

#### **The Five Whys**

Keep asking "why" until you reach the root cause:

```
Why did the app timeout?
  → Waiting for wrong token file

Why wrong token file?
  → Template used default path instead of config

Why default instead of config?
  → Config wasn't passed to template generator

Why wasn't it passed?
  → Generate stage didn't extract it from manifest

ROOT CAUSE: Missing extraction logic in stage_generate.go
```

#### **Hypothesis Prioritization Matrix**

| Factor | High Priority | Low Priority |
|--------|---------------|--------------|
| Evidence | Strong evidence points here | No direct evidence |
| Recency | Code recently changed | Code unchanged for months |
| Complexity | Simple, likely failure point | Complex, many safeguards |
| History | Similar bugs before | Never failed here |

---

### **4. Anti-Patterns**

| Anti-Pattern | Why It Fails | Better Approach |
|--------------|--------------|-----------------|
| **Shotgun debugging** | Random changes obscure cause | Systematic hypothesis testing |
| **Fix without understanding** | Symptom returns or moves | Find root cause first |
| **Single hypothesis fixation** | Confirmation bias | Generate multiple hypotheses |
| **Skipping the test** | No regression protection | Write failing test BEFORE fix |
| **Fixing symptoms** | Underlying issue remains | Ask "why" until root cause |
| **Not documenting** | Knowledge lost | Document root cause in PR |

---

### **5. Boundaries**

This methodology covers **functional debugging** (code doesn't work as expected).

**Does NOT cover:**
- **Performance debugging** — Different methodology (profiling, measurement)
- **Security incident response** — Requires containment before analysis
- **Data corruption recovery** — Requires backup/restore procedures
- **Intermittent/race conditions** — May need specialized tools

---

### **6. Output Expectations**

When applying Scientific Debugging, you **must** produce:

1. **Documented hypotheses** — At least 2, with test plans
2. **Regression test** — Failing test that passes after fix
3. **Root cause documentation** — Explains WHY, not just WHAT
4. **Fix** — Addresses root cause, not symptoms

You **should** also:
- Check for similar patterns elsewhere in codebase
- Update relevant documentation if the bug revealed a gap
- Consider if the methodology itself could be improved

---

### **7. Write a record (recursive-learning loop)**

After the fix ships, write a swarm-manager record so future agents can find your investigation:

```bash
swarm-manager records create --kind fix --scenario "<name>" \
  --trigger "<one-line symptom>" \
  --approach "<root cause + what was built>" \
  --ruled-out "<hypothesis A>" --ruled-out "<hypothesis B>" \
  --evidence "<validation results: suites run, baseline diffs, live checks>" \
  --commit <sha> --files <path> [--files <path>] \
  --outcome shipped [--backlog-ref kind/name]
```

**Required when** the fix was non-trivial: required >15 min of investigation, OR touched >2 files, OR involved a hypothesis that was ruled out (you generated ≥2 hypotheses per §1 — that's the trigger).

**Optional** for one-line trivial fixes (typo, obvious off-by-one).

If you closed a swarm-manager backlog item via `backlog review-decide --accept`, a stub record was auto-created. Fill it with `swarm-manager records edit --id <stub-id> ...` instead of creating a new one.

Records are the write-side of the prior-fix lookup you ran in Phase 0. Skipping this step means the next agent debugging the same class of bug starts from zero.

**Quality bar:** Another engineer should be able to understand:
- What the bug was
- Why it happened
- Why the fix is correct
- How to avoid similar bugs
