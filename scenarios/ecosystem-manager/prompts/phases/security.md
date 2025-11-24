## Steer focus: Security Hardening

Prioritize **improving the security posture** of this scenario across its UI, APIs, background jobs, and data flows.

Do **not** break functionality, regress tests, or weaken existing protections. All changes must maintain or improve overall completeness and reliability.

Focus on **practical, high-impact security improvements**, guided by these principles:

---

### **1. Protect Sensitive Data**

* Identify where sensitive data is **stored, transmitted, or logged** (e.g. credentials, tokens, keys, personal data).
* Avoid logging secrets or sensitive identifiers; ensure logs contain enough context to debug without leaking private data.
* Ensure sensitive values are **never exposed unnecessarily to the client**, to URLs, or to third-party services.
* Prefer **redaction, tokenization, or scoping** where appropriate, rather than passing raw values around.

---

### **2. Strengthen Authentication & Authorization**

* Verify that all privileged actions are **strictly access-controlled** and cannot be triggered by unauthorized users or contexts.
* Ensure permission checks are **centralized, consistent, and enforced server-side** where applicable.
* Avoid duplicating authorization logic in many places; prefer clear, reusable checks to reduce mistakes.
* Add or refine tests that prove:
  * users cannot access resources they shouldn’t
  * permission boundaries behave as intended

*(Only adjust authentication/authorization logic where it is clearly safe and improves correctness; do not introduce entirely new auth systems in this phase.)*

---

### **3. Validate Inputs & Handle Outputs Safely**

* Ensure all external inputs (user input, API payloads, webhooks, files, configuration) are **validated, sanitized, and constrained**.
* Enforce reasonable limits on size, format, and allowed values to reduce attack surface.
* Avoid blindly trusting client-side validation; ensure important checks also exist in trusted code.
* Where applicable, use **safe encoding/escaping** when rendering dynamic content, displaying errors, or constructing downstream calls.

---

### **4. Reduce Attack Surface & Abuse Vectors**

* Look for endpoints, actions, or flows that could be abused (e.g. repeated actions, heavy queries, mass operations).
* Where appropriate, consider adding:
  * lightweight **rate limiting** or backoff mechanisms
  * **safe defaults** that are secure out of the box
  * basic protections against automated abuse or misuse
* Remove or lock down dead code paths, debug endpoints, or test hooks that should no longer be accessible.

Do not introduce user-visible friction unless it meaningfully improves security for realistic threats.

---

### **5. Fail Safely & Minimize Information Leakage**

* Improve error handling so failures are:
  * **graceful** for users
  * **non-revealing** of internal details (stack traces, internal IDs, configuration)
* Ensure security-relevant failures (e.g. repeated denied access, suspicious patterns) can be **observed via logs or metrics** without exposing secrets.
* Prefer **fail-closed** behavior for security-critical checks: if a check cannot run, default to the safer option where practical.

---

### **6. Dependencies, Configuration & Secrets**

* Where the scenario uses libraries or external services, prefer **stable, non-deprecated** interfaces that have fewer known risks.
* Avoid hard-coding secrets or environment-specific values in code; rely on existing configuration mechanisms.
* If you find insecure or fragile configuration patterns, improve them in a way that:
  * is compatible with the current deployment model
  * does not require manual secrets migration in this loop

If a larger change is needed (e.g. rotating secrets, changing deployment configs), document it clearly instead of implementing risky partial changes.

---

### **7. Maintain Scenario Constraints**

* Do **not** change the scenario’s core product behavior or user-facing workflows beyond what’s necessary to improve security.
* Do **not** add new features unrelated to security hardening.
* Ensure all changes remain aligned with the scenario’s PRD, architectural goals, and test-driven requirements.
* When in doubt between a risky “big rewrite” and a safer incremental improvement, choose the **safer incremental** option and surface the larger change as a recommendation.

---

### **8. Output Expectations**

You may update:

* validation logic and guardrails
* permission checks and access patterns
* error handling and logging behavior
* configuration patterns related to security
* tests that cover authentication, authorization, and input handling
* documentation or comments that clarify security-critical assumptions

You **must**:

* keep the scenario fully functional and non-regressed
* avoid weakening existing protections
* improve the robustness of security-critical flows
* meaningfully reduce real-world risk, not just add “security theater”

Focus this loop on **practical, targeted security improvements** that make the scenario safer against realistic misuse or attack, while preserving usability and progress toward the scenario’s operational targets.

Avoid superficial changes (e.g. renaming symbols or shuffling code) that do not materially improve security.
