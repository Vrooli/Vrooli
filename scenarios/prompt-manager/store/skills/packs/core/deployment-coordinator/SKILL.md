## Practice focus: Deployment Coordinator

Intelligent deployment routing and advisory for Vrooli scenarios. Determines the deployment target, checks readiness, routes to the appropriate target-specific Tools skill, and gracefully handles unavailable targets. Adapts behavior based on whether the user wants to deploy now, learn about deployment options, or prepare for future deployment.

Required reading:
- `docs/agent-system/SKILL_AUTHORING.md`

---

### **1. When to Use This Methodology**

Use Deployment Coordinator when:
- The user wants to deploy a scenario to any target
- The user asks about deployment options or status for a scenario
- The user mentions publishing, releasing, or shipping a scenario
- The user asks about making a scenario available on a specific platform

**Do NOT use** for:
- Local development (`make start`, `vrooli scenario start`) — that's not deployment
- Building/compiling without deployment intent
- Infrastructure setup unrelated to scenario deployment (VPS provisioning, DNS registration)

**Proactive trigger:** This skill should be loaded automatically when the conversation involves deploying or publishing a scenario.

---

### **2. Deployment Target Registry**

This is the canonical list of deployment targets and their current status. Update this section as new targets come online.

| Target | Scenario | CLI Available | Skill Available | Status |
|---|---|---|---|---|
| **Cloud (VPS)** | scenario-to-cloud | Yes | `scenario-to-cloud` | **Ready** |
| **Desktop (Electron)** | scenario-to-desktop | Yes | `scenario-to-desktop` | **Ready** |
| **Android** | scenario-to-android | No | None | **Not available** |
| **iOS** | scenario-to-ios | No | None | **Not available** |
| **Browser Extension** | scenario-to-extension | No | None | **Not available** |
| **MCP Server** | scenario-to-mcp | No | None | **Not available** |

**Cross-platform preparation skill:** `cross-platform-readiness` — prepares scenarios for deployment beyond Tier 1 (local stack) by eliminating platform-specific assumptions.

**Branding readiness skill:** `brand-manager` (draft) — validates and applies brand identity (logo, favicon, colors, typography) before deployment. Use `brand-manager status --scenario <name>` to check, and `brand-manager apply` to remediate.

---

### **3. The Process**

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                     DEPLOYMENT COORDINATOR PROCESS                            │
├──────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│   ┌──────────┐     ┌──────────┐     ┌──────────┐     ┌──────────┐          │
│   │ DETECT   │ ──▶ │ ASSESS   │ ──▶ │  ROUTE   │ ──▶ │ EXECUTE  │          │
│   │  MODE    │     │ TARGET   │     │ OR ADVISE│     │ OR REPORT│          │
│   └──────────┘     └──────────┘     └──────────┘     └──────────┘          │
│                                                                              │
│   Mode detection:                                                            │
│   ├─ "Deploy X to Y"           → Action mode                                │
│   ├─ "Deploy X" (no target)    → Action mode (determine target)             │
│   ├─ "How do I deploy X?"      → Advisory mode                              │
│   ├─ "What's the deploy status?"→ Advisory mode                             │
│   └─ "Deploy to <unavailable>" → Graceful degradation                       │
│                                                                              │
└──────────────────────────────────────────────────────────────────────────────┘
```

---

### **Phase 1: Detect Mode**

**Entry criteria:** User has mentioned deployment, publishing, or releasing a scenario.

**Actions:**
1. Determine the interaction mode from the user's request:

| User Intent | Mode | Example |
|---|---|---|
| Wants deployment to happen now | **Action** | "Deploy my-scenario to cloud" |
| Wants to deploy but no target specified | **Action** (needs target) | "Deploy my-scenario" |
| Asking about options or status | **Advisory** | "How would I deploy this?" / "Is this deployed anywhere?" |
| Requesting an unavailable target | **Graceful degradation** | "Build an Android app for this" |

2. Identify the scenario name from context.

**Exit criteria:**
- [ ] Mode determined (action / advisory / graceful degradation)
- [ ] Scenario identified

---

### **Phase 2: Assess Target**

**Entry criteria:** Mode and scenario are known.

**Actions differ by mode:**

#### Action mode (target specified)
1. Look up the target in the Deployment Target Registry (Section 2).
2. If **Ready** → proceed to Phase 3 (Route).
3. If **Not available** → switch to graceful degradation mode.

#### Action mode (no target specified)
1. Ask the user which target they want. Present only **Ready** targets:
   - Cloud (VPS) — deploy to a remote server, accessible via domain
   - Desktop — package as an installable desktop application (Windows, macOS, Linux)
2. If the user picks an unavailable target, switch to graceful degradation.

#### Advisory mode
1. Check the Deployment Target Registry for all targets.
2. Gather current deployment state if possible:
   - For cloud: check if a deployment exists (`scenario-to-cloud deployment resolve`)
   - For desktop: check if builds exist (`scenario-to-desktop pipeline status` or check for artifacts)
3. Assess cross-platform readiness:
   - Has the scenario been through `cross-platform-readiness` before?
   - If this is a first deployment or a new target, suggest running it.
4. Assess branding readiness:
   - Does the scenario have a brand applied? (`brand-manager status --scenario <name>`)
   - If branding is incomplete, note it as a deployment prerequisite and suggest: `prompt-manager skill read brand-manager`

#### Graceful degradation mode
1. Acknowledge the requested target is valid but not yet available.
2. Explain what's missing (no CLI, no skill — per the registry).
3. Suggest available alternatives.
4. Offer to run `cross-platform-readiness` to prepare the scenario for when the target becomes available.

**Exit criteria:**
- [ ] Target validated against registry
- [ ] Mode confirmed or adjusted

---

### **Phase 3: Route or Advise**

**Entry criteria:** Target assessed, mode confirmed.

**Actions by mode:**

#### Action mode → Route to target skill
1. Load the target-specific Tools skill:
   ```bash
   prompt-manager skill read "<target-skill-id>"
   ```
2. For **first-time deployments** to a target, suggest cross-platform readiness first:
   ```
   This scenario hasn't been deployed to <target> before.
   Would you like to run cross-platform-readiness first to check for
   platform-specific issues, or proceed directly?
   ```
   - If user wants readiness check: `prompt-manager skill read cross-platform-readiness`
   - If user wants to proceed: continue with the target skill
3. Follow the loaded target skill's workflow.

**Target skill routing table:**

| Target | Load skill | Then follow |
|---|---|---|
| Cloud | `prompt-manager skill read scenario-to-cloud` | Cloud deployment workflow |
| Desktop | `prompt-manager skill read scenario-to-desktop` | Desktop build + deploy workflow |

#### Advisory mode → Report
1. Present deployment landscape for the scenario:
   - **Currently deployed to:** List active deployments (if any)
   - **Available targets:** Cloud, Desktop (with brief description of each)
   - **Future targets:** Android, iOS, Extension, MCP (not yet available)
   - **Cross-platform readiness:** Whether the scenario has been prepped
2. If the scenario is not deployed anywhere and has never been through readiness:
   ```
   This scenario hasn't been deployed yet. If you'd like to prepare it
   for deployment, I can run cross-platform-readiness to identify any
   platform-specific issues. Or if you're ready to deploy now, which
   target would you like?
   ```
3. Let the user decide next steps.

#### Graceful degradation → Inform and redirect
1. Report clearly:
   ```
   <Target> is a planned deployment target but isn't available yet.
   The scenario-to-<target> CLI hasn't been built.

   Available now:
   - Cloud (VPS) — accessible via domain
   - Desktop — installable app for Windows/macOS/Linux

   To prepare for <target> when it's ready, I can run
   cross-platform-readiness to make sure your scenario is portable.
   ```

**Exit criteria:**
- [ ] Target skill loaded (action mode) OR report delivered (advisory/degradation mode)

---

### **Phase 4: Execute or Report**

**Entry criteria:** Routing decision made.

**Actions:**

#### Action mode
1. Follow the loaded target skill's workflow end-to-end.
2. After completion, report the result:
   - Deployment URL/location
   - Health status
   - Any warnings or follow-up items

#### Advisory/degradation mode
1. Answer any follow-up questions from the user.
2. If the user decides to proceed with a deployment, return to Phase 2 with the chosen target.

**Exit criteria:**
- [ ] Deployment complete (action mode) OR user has the information they need (advisory mode)

---

### **4. Convergence Patterns**

#### Target Selection Decision Tree

```
User wants to deploy
├─ Target specified?
│   ├─ Yes → Is it in the registry?
│   │         ├─ Yes, status=Ready → Load target skill, deploy
│   │         ├─ Yes, status=Not available → Graceful degradation
│   │         └─ No (unknown target) → Ask for clarification
│   └─ No → Present available targets, ask user to pick
│
├─ First deployment to this target?
│   ├─ Yes → Suggest cross-platform-readiness (don't force)
│   └─ No → Proceed directly
│
└─ Multiple targets requested?
    └─ Run cross-platform-readiness first, then each target sequentially
```

#### Cross-Platform Readiness Trigger Table

| Situation | Suggest Readiness? |
|---|---|
| First deployment to any target | Yes (suggest, don't force) |
| Redeployment to existing target | No |
| User asks about deployment options generally | Yes, as a preparatory step |
| User explicitly requests a specific deployment | Only if first time to that target |
| User asks to deploy to unavailable target | Yes, as a productive alternative |
| Branding incomplete for scenario | Yes (suggest `brand-manager` for remediation) |

---

### **5. Anti-Patterns**

| Anti-Pattern | Why It Fails | Better Approach |
|---|---|---|
| **Force cross-platform-readiness on every deploy** | Unnecessary friction for redeployments | Only suggest for first-time targets |
| **Silently skip unavailable targets** | User doesn't know why nothing happened | Explicitly state it's unavailable, offer alternatives |
| **Attempt deployment without the target skill** | Missing critical workflow steps | Always load the target-specific skill first |
| **Hardcode deployment commands inline** | Duplicates target skill, drifts out of sync | Route to the target skill, let it own the commands |
| **Ask too many questions before deploying** | User said "deploy", so deploy | If target is clear and scenario is known, just do it |
| **Suggest every possible target** | Overwhelming, most aren't available | Only present Ready targets as actionable options |

---

### **6. Boundaries**

This methodology covers **deployment routing, readiness assessment, and target coordination**.

**Does NOT cover:**
- **Target-specific deployment workflows** — Those live in each target's Tools skill (scenario-to-cloud, scenario-to-desktop, etc.)
- **Cross-platform code changes** — That's cross-platform-readiness's domain
- **Local development** — Starting/stopping scenarios locally is not deployment
- **Infrastructure provisioning** — VPS setup, app store accounts, etc. are prerequisites, not part of this flow
- **CI/CD pipeline configuration** — This skill is for agent-driven deployment, not pipeline setup

---

### **7. Output Expectations**

When applying Deployment Coordinator, you **must**:

1. **Identify the mode** (action / advisory / graceful degradation) before proceeding
2. **Check the target registry** — never assume a target is available without checking
3. **Load the target skill** before executing any deployment commands
4. **Suggest cross-platform-readiness** for first-time deployments (without forcing it)
5. **Clearly communicate** when a target is unavailable, what alternatives exist, and what preparation can be done

You **should** also:
- Report deployment results (URL, health, warnings) after action mode completes
- Offer next steps after advisory mode (would you like to deploy? run readiness?)
- Keep the unavailable-target response constructive, not just "no"
