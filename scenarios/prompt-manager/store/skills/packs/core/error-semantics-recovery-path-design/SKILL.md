## Steer focus: Error Semantics & Recovery Path Design

Prioritize **making errors meaningful, structured, and recoverable** for both users and agents.

Your goal is to turn “something went wrong” into **clear categories with explicit recovery paths**, without breaking existing behavior.

Do **not** regress tests, weaken protections, or introduce unrelated features. All changes must maintain or improve completeness and reliability.

Required reading:
- `prompt-manager skill read visited-tracker-tools knowledge-observatory-tools`

---

### **1. Clarify Error Domains & Categories**

* Read the `error-semantics` doc for `{{TARGET}}` using `knowledge-observatory-tools` to understand existing error categories and recovery paths.

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

#### Principles for Good Error Categories

When deriving categories, apply these criteria:

1. **Mutually exclusive**: Each error should map to exactly one category. If you're unsure where an error belongs, the categories need refinement.

2. **Recovery-distinct**: Two categories should have different recovery paths. If "network timeout" and "service unavailable" both mean "retry after backoff," they might be one category.

3. **User-recognizable**: Category names should make sense to users, not just developers. "Validation error" is good; "schema mismatch" is too internal.

4. **Aim for 4-8 categories**: Fewer than 4 means overly generic (everything is "error"). More than 8 means hard to maintain consistency.

5. **Stable across the codebase**: The same category should mean the same thing everywhere. If "not found" sometimes means "resource deleted" and sometimes "URL typo," split them.

---

### **1.5 Error Handling Audit**

Before improving error semantics, understand the current state:

#### Step 1: Inventory Error Origins
```bash
# Where are errors created/thrown?
rg "throw |new Error|Error\(" --type ts --type go -l

# Where are errors caught/handled?
rg "catch|\.catch\(|if.*err != nil" --type ts --type go -l

# What error types/shapes exist?
rg "type.*Error|interface.*Error|class.*Error" --type ts --type go
```

#### Step 2: Assess Error Message Quality
```bash
# Find generic/unhelpful messages
rg "Something went wrong|An error occurred|Unknown error|Error:" --type ts

# Find messages that might leak internals
rg "stack|stackTrace|internal|debug" --type ts -i
```

#### Step 3: Find Recovery Path Gaps
```bash
# Error UI without recovery actions
rg "error|Error" --type tsx -A 5 | rg -v "retry|try again|go back|refresh"

# Catch blocks that swallow errors
rg "catch.*\{" --type ts -A 2 | rg -B 2 "^\s*\}"
```

**Red flags:**
- [ ] Many different error shapes (strings, objects, classes) mixed together
- [ ] Generic messages without specific guidance
- [ ] Error UIs that are dead-ends (no recovery action)
- [ ] Silent failures (empty catch blocks)
- [ ] Same error message for different failure modes

---

### **2. Normalize & Structure Error Representations**

* Where possible, replace ad-hoc error handling with **structured, consistent forms**, such as:
  * clearly named error types or codes
  * tagged results / status objects
  * well-defined error payload shapes for APIs or internal flows
* Ensure similar failures produce **the same kind of error shape**, not a mixture of strings, booleans, and thrown exceptions.
* Avoid leaking low-level details (stack traces, internal IDs, sensitive values) into user- or client-facing errors.

Do not introduce a brand-new global error framework; prefer **incremental normalization** of existing patterns.

#### Documenting Error Categories

When defining error categories, add protective comments explaining the design:

```typescript
/**
 * ╔════════════════════════════════════════════════════════════════╗
 * ║  ERROR CATEGORIES - Read before modifying                      ║
 * ║                                                                ║
 * ║  Each category has a specific recovery path. Changing these    ║
 * ║  affects UI error states and automated retry logic.            ║
 * ║                                                                ║
 * ║  To add a new category:                                        ║
 * ║  1. Ensure it has a distinct recovery path                     ║
 * ║  2. Update error UI components to handle it                    ║
 * ║  3. Add to API error mapping if client-facing                  ║
 * ╚════════════════════════════════════════════════════════════════╝
 */
```

This prevents future agents from creating ad-hoc error types that fragment the error handling system.

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

#### Criteria for Choosing Recovery Paths

When deciding what recovery action fits an error category, consider:

| Question | If YES → | If NO → |
|----------|----------|---------|
| Can the user fix the cause? | Guide them to fix it (show what's wrong, keep their input) | Don't blame them |
| Is the cause likely transient? | Retry automatically (with backoff) or offer retry button | Don't suggest retry |
| Is partial progress valuable? | Preserve it, let user continue from checkpoint | Clean abort is fine |
| Does the user need to know? | Show clear message with next steps | Log only, don't interrupt |
| Can an agent reasonably handle this? | Return structured error with machine-readable hints | User escalation path |

**Common anti-patterns:**
- Suggesting "retry" for validation errors (user needs to fix input, not retry)
- Showing technical details for transient failures (user can't fix server issues)
- Silent failures that leave user wondering what happened
- Generic "contact support" when user could self-serve

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

### **7. Memory Management with Visited Tracker**

Use the `visited-tracker-tools` skill for tracking visited files, with LOCATION set to `scenarios/{{TARGET}}` and TAG set to `error-semantics`.

---

### **8. Documentation**

Use `knowledge-observatory-tools` to read the current `error-semantics` doc for `{{TARGET}}`, then update it with your findings (error categories discovered/refined, recovery paths, error patterns improved and remaining).

---

### **9. Output Expectations**

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
