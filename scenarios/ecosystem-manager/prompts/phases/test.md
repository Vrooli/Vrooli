## Steer focus: Test Suite Strengthening

Prioritize **test quality, coverage, and reliability** across this scenario.
Do **not** break functionality or regress existing tests; all changes must maintain or improve overall completeness.

Focus on producing a **high-signal, trustworthy test suite** that accurately reflects the scenario’s operational targets and technical requirements.

---

### **1. Align Tests With Operational Targets & Requirements**

* Ensure **each operational target** has clear, meaningful test coverage through its linked technical requirements.
* Where gaps exist, add tests that validate the **actual behavior** users and systems depend on, not just internal implementation details.
* Prefer tests that verify the **full intent** of a requirement (happy path + key edge cases), rather than narrow or trivial assertions.

---

### **2. Increase Coverage Where It Matters Most**

* Identify and strengthen coverage for:

  * **Critical user journeys** and core workflows
  * Error handling and fallback behavior
  * Boundary conditions (empty inputs, large inputs, missing data, timeouts)

* Prioritize **high-impact areas** where a regression would meaningfully harm the user experience or operational reliability.
* Avoid adding low-value tests that simply increase raw counts without improving real protection.

---

### **3. Improve Assertion Quality & Signal Strength**

* Upgrade vague or weak assertions (e.g. “component renders”) to **specific, behavior-focused checks**:

  * correct outputs
  * correct side effects
  * correct UI states and transitions

* Ensure tests would **fail clearly and immediately** if the behavior they protect were broken.
* Avoid loosening tests or weakening assertions just to make them pass; tests should **enforce correctness**, not accommodate bugs.

---

### **4. Reduce Flakiness and Brittleness**

* Identify and fix sources of **flaky or timing-sensitive tests**:

  * brittle selectors
  * arbitrary timeouts
  * unnecessary reliance on network, random data, or global state

* Make tests **deterministic and repeatable** by:

  * controlling randomness
  * isolating side effects
  * mocking external dependencies only where appropriate

* Keep the balance: don’t over-mock to the point that tests no longer reflect real user-visible behavior.

---

### **5. Organize and Simplify the Test Suite**

* Improve **structure, naming, and grouping** so tests are easy to read, navigate, and extend:

  * clear test names describing intent and behavior
  * logical grouping by feature, domain, or workflow
  * shared helpers for repeated setup and assertions

* Remove or refactor **redundant, overlapping, or obsolete tests** only when you’re confident they no longer provide unique value.
* Keep test files focused and approachable so future agents (and humans) can quickly understand what’s covered and what’s missing.

---

### **6. Maintain Scenario Constraints**

* Do **not** change core business logic, APIs, or workflows except when strictly necessary to improve testability in a clean, minimal way.
* Do **not** introduce new features unrelated to test quality and coverage.
* Ensure all changes remain consistent with the scenario’s PRD, operational targets, and architectural goals.
* Preserve or improve existing completeness metrics; never “game” them by:

  * deleting valid tests,
  * weakening assertions,
  * or writing superficial tests that don’t catch real failures.

---

### **6. Memory Management with Visited Tracker**

To ensure **systematic coverage without repetition**, use `visited-tracker` to maintain perfect memory across conversation loops:

**At the start of each iteration:**
```bash
# Get 5 least-visited test files with auto-campaign creation
visited-tracker least-visited \
  --location scenarios/{{TARGET}} \
  --pattern "**/*_test.{go,ts,tsx}" \
  --tag test \
  --name "{{TARGET}} - Test Suite Strengthening" \
  --limit 5
```

**After analyzing each file:**
```bash
# Record your visit with specific notes about improvements and remaining work
visited-tracker visit <file-path> \
  --location scenarios/{{TARGET}} \
  --tag test \
  --note "<summary of test improvements made and what remains>"
```

**When a file is irrelevant to testing (generated mocks, snapshots, fixtures, etc.):**
```bash
# Mark it excluded so it doesn't resurface - this is NOT a test file
visited-tracker exclude <file-path> \
  --location scenarios/{{TARGET}} \
  --tag test \
  --reason "Not a test file - generated/snapshot/fixture/etc."
```

**When a file has comprehensive test coverage:**
```bash
# Mark it excluded so it doesn't resurface in future queries
visited-tracker exclude <file-path> \
  --location scenarios/{{TARGET}} \
  --tag test \
  --reason "All test improvements complete - comprehensive coverage, clear assertions, no flakiness"
```

**Before ending your session:**
```bash
# Add campaign note for handoff context to the next iteration
visited-tracker campaigns note \
  --location scenarios/{{TARGET}} \
  --tag test \
  --name "{{TARGET}} - Test Suite Strengthening" \
  --note "<overall progress summary, patterns observed, priority areas for next iteration>"
```

**Interpreting the response:**
- Prioritize files with **high staleness_score (>7.0)** - neglected files needing attention
- Focus on **low visit_count (0-2)** - files not yet analyzed
- Review **notes from previous visits** - understand context and remaining work
- Check **coverage_percent** - track systematic progress toward 100%

**Note format guidelines:**
- **File notes**: Be specific about what you tested and what still needs work
  - ✅ Good: "Added edge case tests, strengthened assertions, fixed flaky timeout. Still need to test error paths for API failures."
  - ❌ Bad: "Made some test improvements"
- **Campaign notes**: Provide strategic context for the next agent
  - ✅ Good: "Completed 22/58 test files (38%). Focus areas: API integration tests lack error coverage, UI tests have weak assertions"
  - ❌ Bad: "Made progress on tests"

---

### **7. Output Expectations**

You may update or add:

* unit, integration, and end-to-end tests (e2e tests handled by the test/playbooks/ workflows)
* test utilities, fixtures, and helpers
* test naming, structure, and organization
* coverage of error paths, edge cases, and critical workflows
* mock tools such as testcontainers-go

You **must**:

* keep the scenario fully functional
* avoid regressions
* improve the **trustworthiness and clarity** of the test suite
* raise the **real** protective value of the tests, not just their quantity

Focus this loop on delivering **practical, high-impact test improvements** that make the scenario safer to evolve, easier to reason about, and more accurately measured by its programmatic completeness.

**Avoid superficial tests that increase coverage numbers without meaningfully protecting behavior. Only add or modify tests when they genuinely sharpen the feedback signal and reduce the risk of unnoticed regressions.**
