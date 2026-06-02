## Practice focus: Improvement Do & Don't

The single reference for **what counts as a real improvement vs. metric-gaming**. Every steer skill (`refactor`, `test`, `security`, `polish`, `documentation-health`, `progress`, `performance`, `ux`) points here. Read it before you touch a test, a known-issue ledger, a lint/type config, or an auditor rule.

A real improvement closes a gap in the scenario's actual behavior, safety, or completeness. Gaming makes the *measurement* look better without changing the underlying reality — and the ecosystem-manager controller now **detects it, refuses to reward it, and flags the iteration** (see "How the controller responds" below). Don't make the move it's built to catch.

---

### **1. The three gaming patterns we actually caught**

These are not hypothetical. The closed-loop controller's first live runs produced each of these in miniature; the classifier now watches for them by name.

#### **DON'T loosen or delete a `[REQ:]`-tagged test to make it pass — fix the contract.**

A test tagged `[REQ: ...]` (or whose enclosing case maps to a tracked requirement) is the *executable definition* of that requirement. Weakening its assertions — turning `t.Fatalf("want ok, got %s", x)` into `t.Log(...)`, replacing `toEqual(expected)` with a non-empty check, deleting the assertion entirely — does not satisfy the requirement, it **un-defines** it.

- **DO** make the code satisfy the assertion. If the assertion is genuinely *wrong*, change it to be **more accurate to the PRD/requirement**, never looser — and say why in the diff.
- A requirement "validated" by a weakened test is no longer validated. The controller will not count it toward operational-targets, and zeros the iteration's credit.

#### **DON'T delete a known-issue ledger — migrate it.**

`docs/internal/PROBLEMS.md` and `docs/internal/PROGRESS.md` are **tracked gaps**, the institutional memory of deferred work. Deleting one to "clean up" erases the only record that the gap exists. That is the highest-leverage gaming move and the easiest to catch.

- **DO** resolve the entries (do the work) and then update the ledger, or migrate them to wherever they now belong (a backlog item, an issue). The entry leaves the ledger because it was *handled*, never because the file was *removed*.
- These files are core internal docs — "always present, never skip" (see `documentation-health`). A deletion in a diff is treated as gaming.

#### **DON'T suppress a finding — fix it, or scope the rule with a regression test.**

Adding `nolint`, `// +build ignore`, `eslint-disable`, `@ts-ignore`/`@ts-nocheck`, `# type: ignore`, lowering a coverage/threshold knob, or flipping an auditor rule to `enabled: false` makes the complaint disappear without addressing it.

- **DO** fix the underlying issue. If a rule genuinely **over-fires** (a real false positive), *scope the rule* — narrow its matcher and add a regression test that proves the false positive is gone and the true positives still fire. Never blanket-disable. (See `scenario-auditor` FP-scoping practice.)
- A bare suppression directive in a diff is treated as gaming.

---

### **2. The general test: would this survive a skeptic?**

Before shipping a change, ask: *if a reviewer who wanted to disprove "this is a real improvement" read only the diff, could they?* Real improvements pass; gaming-shaped ones don't.

| Looks like progress | Is actually | Do instead |
|---|---|---|
| Test now passes (assertion removed) | Requirement un-defined | Make the code pass the original assertion |
| Violation count dropped (rule disabled) | Detector blinded | Fix the violation; scope the rule + regression test if it's a FP |
| Coverage % rose (trivial tests added) | Numbers up, protection flat | Test real behavior + edge cases on critical paths |
| `PROBLEMS.md` gone (looks tidy) | Tracked gap erased | Resolve the entry, then update/migrate the ledger |
| Lint clean (`nolint` added) | Warning silenced | Address the warning |

Superficial changes that move a metric without moving reality are the thing to avoid. When in doubt, do the smaller *real* fix over the larger *cosmetic* one.

---

### **3. How the controller responds (so you know it's watching)**

The ecosystem-manager closed-loop controller fetches each iteration's code diff and runs an anti-gaming classifier over it. When it detects test-weakening, ledger-deletion, or suppression:

- the iteration earns **zero closed-finding credit** (the reduction-per-token bandit gets no reward for the cheap move — so it never learns to prefer it),
- the iteration is **flagged in the decision trace** (`gaming_cause`, shown in the panel), and the regression veto is recorded.

Ambiguous cases (e.g. assertions removed from a test with no `[REQ:]` tag) are **flagged for review**, not auto-penalized. There is no auto-revert — a human / Git Control Tower handles the cleanup. The point isn't punishment; it's that gaming is a dead end. Spend the iteration on the real fix.

---

### **4. No known operational edge cases**

This is a reference skill — it changes no files itself. It is read alongside the steer skill driving the current loop.
