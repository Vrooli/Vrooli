## Steer focus: Error Semantics & Recovery Path Design

Prioritize **making errors meaningful, structured, and recoverable** for both users and agents.

Your goal is to turn “something went wrong” into **clear categories with explicit recovery paths**, without breaking existing behavior.

Do **not** regress tests, weaken protections, or introduce unrelated features. All changes must maintain or improve completeness and reliability.

---

### **1. Clarify Error Domains & Categories**

* Examine where the scenario can fail across:
  * UI interactions
  * background jobs / workers
  * API calls (incoming and outgoing)
  * integration points (external services, storage, tools)
* Derive a small, coherent set of **error domains** and **categories**, for example:
  * configuration/setup issues
  * user input / validation issues
  * connectivity / dependency issues
  * permission / access issues
  * internal logic or invariant violations
* Ensure each important failure path **maps to one of these categories** instead of being an ad-hoc message or raw exception.

Focus on **crisp, high-signal categories**, not an explosion of tiny types.

---

### **2. Normalize & Structure Error Representations**

* Where possible, replace ad-hoc error handling with **structured, consistent forms**, such as:
  * clearly named error types or codes
  * tagged results / status objects
  * well-defined error payload shapes for APIs or internal flows
* Ensure similar failures produce **the same kind of error shape**, not a mixture of strings, booleans, and thrown exceptions.
* Avoid leaking low-level details (stack traces, internal IDs, sensitive values) into user- or client-facing errors.

Do not introduce a brand-new global error framework; prefer **incremental normalization** of existing patterns.

---

### **3. Define Explicit Recovery Paths**

For each major error category, make the **intended recovery action** clear and consistent:

* Identify whether the right response is to:
  * retry (immediately or after backoff)
  * ask the user/agent to correct input or configuration
  * escalate (surface to an operator or higher-level scenario)
  * abort gracefully and preserve partial progress
* Encode recovery paths where appropriate by:
  * returning machine-readable hints (e.g. flags, fields, codes)
  * guiding the UI toward helpful next actions
  * making it easy for agents to choose the right follow-up behavior

Avoid speculative recovery flows; focus on **concrete, realistic failure modes** that the scenario actually encounters.

---

### **4. Improve User- & Agent-Facing Error Surfaces**

* For human users:
  * Make error messages **clear, concise, and non-technical** where possible.
  * Explain what went wrong at a high level, and suggest **what to do next**.
  * Avoid blameful or alarming wording; keep the tone neutral and supportive.
* For agents and automated flows:
  * Ensure errors are **structured** enough that agents can:
    * identify category
    * infer severity
    * choose an appropriate action (retry, adjust input, escalate, etc.)
  * Avoid encoding critical distinctions only in free-form text.

Where relevant, align status codes, UI messages, and internal error objects so they tell a **coherent story**.

---

### **5. Tie Errors to Observability & Diagnosis**

* Ensure important error categories are **visible in logs/metrics/traces** in a structured way:
  * include category, source, and key context (but not secrets)
  * avoid duplicating noise or flooding logs with redundant details
* Make it easy to correlate user/agent-visible errors with internal diagnostic signals:
  * share an identifier or code between what the user sees and what logs record
* Prefer **few, high-signal log/metric points** over many low-signal ones.

The goal is to help future agents and humans quickly answer:
> “What kind of error is this, and where should I look to debug it?”

---

### **6. Preserve Behavior While Improving Semantics**

* Keep **observable behavior** (success/failure conditions) stable unless you are clearly fixing a bug.
* If you change how errors are represented internally, ensure:
  * all call sites are updated consistently
  * tests cover the new behavior and semantics
* Do not weaken existing checks just to simplify error handling.
* When you encounter error patterns that need a broader redesign, document them clearly rather than attempting risky partial rewrites in this loop.

Favor **small, coherent improvements** over wide, speculative changes.

---

### **7. Maintain Scenario Constraints**

* Do **not** introduce new product features in this phase.
* Do **not** change core workflows or business rules except where clearly necessary to make error handling internally consistent.
* Ensure all changes respect the scenario’s PRD, operational targets, and test-driven requirements.
* Keep user experience and agent behavior at least as smooth as before, ideally more predictable under failure.

---

### **8. Output Expectations**

You may update:

* error types, result shapes, and status enums/codes
* how errors are mapped to user-facing messages and agent-facing data
* validation and error-raising points to use consistent categories
* logging/metric hooks related to error reporting
* tests that assert error categories, shapes, and recovery behavior

You **must**:

* keep the scenario fully functional and non-regressed
* improve the **clarity, consistency, and usefulness** of error semantics
* make recovery paths more explicit and predictable
* reduce “mysterious failures” and ambiguous, catch-all error messages

Focus this loop on **practical, high-impact improvements** that make errors:

* easier to understand,
* easier to recover from,
* and easier to act on for both humans and agents.

Avoid superficial changes (e.g. renaming error variables without changing semantics) that do not materially improve how the system communicates and handles failure.
