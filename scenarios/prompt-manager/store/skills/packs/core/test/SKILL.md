## Steer focus: Test Suite Strengthening

> **Ladder position:** R0 and R3 (runnable & green, then features hardened — coverage). Green-and-runnable is the floor; real coverage of critical behavior is the hardening. See `prompt-manager skill read scenario-maturity-ladder` for rung context and `prompt-manager skill read improvement-do-and-dont` for what counts as a real improvement.

> **Provider & scorer:** the **unit-health** scenario is the canonical test-maturity provider behind Test Genie's `unit` phase — it owns test execution, coverage, test architecture, test quality, and the local-maturity score. This skill is the *remediation* loop, **not** the scorer. To see a scenario's current test maturity, failing/uncovered surfaces, and the next blocker, run the human report:
>
> ```bash
> unit-health validate scenario {{TARGET}}
> ```
>
> (add `--execution` to actually run the test commands; `--json` is for Test Genie/programmatic consumers, not your workflow). Fix the findings it reports; don't treat this skill's prose as the authority on whether tests are "good enough" — `unit-health` is.

> **Policy profile contract:** for react-vite-derived scenarios, `.vrooli/testing.json`
> `unit.policy_profile` declares the template unit-test contract while Code Facts
> discovers the actual API/CLI/UI surfaces. Treat `UNIT_POLICY_*`,
> `UNIT_REQUIRED_ROLE_MISSING`, `UNIT_SURFACE_UNGOVERNED`, and
> `UNIT_POLICY_PROJECTION_DRIFT` as Unit Health contract findings: fix the
> declared profile or native test projection, not Test Genie orchestration.
> Legacy `unit.languages` is compatibility-only and is not the policy source for
> new generated scenarios.

Prioritize **test quality, coverage, and reliability** across this scenario.
Do **not** break functionality or regress existing tests; all changes must maintain or improve overall completeness.

Focus on producing a **high-signal, trustworthy test suite** that accurately reflects the scenario’s operational targets and technical requirements.

Required reading:
- `prompt-manager skills read visited-tracker-tools knowledge-observatory-tools`

---

### **1. Align Tests With Operational Targets & Requirements**

* Read the `problems` doc for `{{TARGET}}` using `knowledge-observatory-tools` to understand existing test gaps.

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
* **Never weaken or delete a `[REQ:]`-tagged test to get green.** That test is the executable definition of a tracked requirement; removing its assertion un-defines the requirement (and the controller will not count it toward operational targets). Fix the code, or — only if the assertion is genuinely wrong — make it *more accurate* to the requirement, never looser. See `improvement-do-and-dont`.

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

Use `knowledge-observatory-tools` to read the current `problems` doc for `{{TARGET}}`, then update the **Test Gaps** section with your findings (critical flows lacking coverage, flaky tests, assertion quality issues, remaining coverage priorities).
