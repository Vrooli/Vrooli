## Steer focus: Test Suite Strengthening

Prioritize **test quality, coverage, and reliability** across this scenario.
Do **not** break functionality or regress existing tests; all changes must maintain or improve overall completeness.

Focus on producing a **high-signal, trustworthy test suite** that accurately reflects the scenario’s operational targets and technical requirements.

Required reading:
- `prompt-manager skills read visited-tracker-tools`

---

### **1. Align Tests With Operational Targets & Requirements**

* **If `docs/internal/PROBLEMS.md` exists**, read the Test Gaps section first to understand what test gaps have been identified.

* For **UI-level validation**, e2e tests are handled by BAS workflows in `bas/` directories. See the **e2e-testing** skill for strategy and the **browser-automation-studio** skill for CLI usage. This skill focuses on unit and integration tests that complement (not duplicate) e2e coverage.

Optional reading:
- `prompt-manager skills read e2e-testing browser-automation-studio`

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

### **6. Memory Management with Visited Tracker**

Use the `visited-tracker-tools` skill for tracking visited files, with LOCATION set to `scenarios/{{TARGET}}` and TAG set to `test`.

---

### **7. Output Expectations**

You may update or add:

* unit, integration, and end-to-end tests (e2e tests handled by the bas/ workflows)
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

---

### **8. Documentation**

Update the **Test Gaps** section of `docs/internal/PROBLEMS.md` to record your findings:

* The code is the source of truth. Verify existing claims against actual code before extending.
* Correct any inaccuracies and extend with your new discoveries.
* Create the `docs/internal/` directory if needed.

Include:
* Critical flows lacking test coverage
* Flaky or brittle tests identified and their status
* Assertion quality issues found
* Test organization improvements made
* Remaining coverage priorities
