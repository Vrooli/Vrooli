## Steer focus: Signal & Feedback Surface Design

Prioritize **making the scenario self-explanatory at runtime** for both humans and agents.

Your goal is to ensure that important states, transitions, and failures are surfaced through **clear, reliable signals** (logs, status fields, UI feedback, metrics, notes) so that future loops can understand what’s happening without guesswork.

Do **not** break functionality, regress tests, or introduce new observability stacks. All changes must maintain or improve completeness and reliability.

---

### **1. Clarify What Must Be Observable**

* Using the PRD, operational targets, and main workflows, identify:
  * key states the scenario can be in (idle, in-progress, degraded, blocked, etc.)
  * important transitions (start/finish of operations, retries, fallbacks, cancellations)
  * high-impact failures (data loss risk, external service problems, invariant violations)
* Treat these as the **minimum set of things that must be visible somewhere**:
  * to users (when relevant)
  * to operators / future agents (for debugging and decision-making)

---

### **2. Audit Existing Signals**

* Examine current **logs, status commands, state fields, UI feedback, and tests**:
  * what events are logged?
  * what statuses or flags exist?
  * where do users see progress, success, or failure?
* Identify gaps where:
  * important events happen silently
  * signals are ambiguous, misleading, or too low-level
  * logs are noisy without adding insight
* Note where signals are **fragmented**, forcing someone to check many places to understand a single flow.

---

### **3. Align Signals With the Scenario’s Mental Model**

* Ensure top-level signals reflect the **scenario’s purpose and mental model**:
  * states and messages should use domain language, not internal implementation jargon
  * status outputs should describe *what the scenario is doing* in terms of the user’s goals
* For each major workflow, there should be a **small, coherent set of signals** that tells the story:
  * what was attempted
  * what happened
  * whether it succeeded or failed
  * what the next step is (if applicable)

Avoid exposing internal details that don’t help users or agents make better decisions.

---

### **4. Improve Runtime Feedback for Users**

* Make sure users receive timely, clear feedback for important actions:
  * progress indicators, completion messages, or clear failure explanations
  * relevant, actionable guidance on how to recover from failures
* Avoid both extremes:
  * silent failures or invisible work
  * overwhelming the user with technical noise
* Align UX feedback with the scenario’s **most common user journeys**, so users can see:
  * what’s happening now
  * whether things are working
  * what they should do next (if anything)

---

### **5. Improve Telemetry for Operators & Agents**

* Use existing logging / status mechanisms to make internal behavior **legible**:
  * log meaningful events (start/finish of important operations, fallbacks, unusual conditions)
  * keep logs structured and consistent where possible (e.g. stable keys, clear message patterns)
* Prefer **few, high-value signals** over many low-value ones.
* Ensure status commands, internal notes, or metadata:
  * summarize the current situation compactly
  * surface the most relevant recent events
  * avoid requiring a full log scroll to understand basic health

Do not introduce new external observability systems in this phase; work within the scenario’s current capabilities.

---

### **6. Make Signals Actionable & Stable**

* Design signals so they can be **relied on by future loops**:
  * stable fields and meanings
  * clear semantics (what does this status actually imply?)
  * minimal “surprise” changes in meaning over time
* Where appropriate, update or add tests that:
  * assert that important operations emit the expected signals
  * lock in behavior around critical events, not just happy-path outcomes

Avoid changing signal formats in ways that would silently break existing consumers (other scenarios, agents, or status tooling) without updating them as well.

---

### **7. Avoid Noise, Leaks & “Signal Theater”**

* Reduce or remove:
  * noisy, repetitive logs that don’t help diagnose or understand behavior
  * vague messages that obscure what actually happened
* Ensure signals **do not leak secrets or sensitive data**:
  * no credentials, tokens, or personal data in logs or status outputs
  * redact or anonymize identifiers when necessary
* Avoid adding signals solely to “look observant.” Only add or keep signals that:
  * clarify behavior
  * support debugging
  * help future agents make better decisions

---

### **8. Maintain Scenario Constraints**

* Do **not** change core business logic or workflows beyond what’s required to expose better signals.
* Do **not** add new product features unrelated to observability or feedback.
* Keep all changes aligned with the PRD, operational targets, and test requirements.
* Prefer **incremental, localized improvements** over broad rewrites of logging or status systems.

---

### **9. Output Expectations**

You may update:

* log messages, log structure, and where key events are logged
* status outputs, metadata, and internal “health” or “phase” indicators
* UI feedback for progress, completion, and failures
* tests that validate critical signals and feedback surfaces
* small documentation comments describing what signals mean and where to look

You **must**:

* keep the scenario fully functional and non-regressed
* make important states and transitions more observable than before
* reduce guesswork for understanding what the scenario is doing
* avoid both noise and under-reporting

Focus this loop on **practical, targeted improvements to how the scenario communicates its own behavior**, so that users, operators, and future agents can quickly understand, debug, and safely extend it.
