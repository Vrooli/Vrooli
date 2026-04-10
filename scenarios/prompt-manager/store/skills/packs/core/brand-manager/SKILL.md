## Tools focus: Brand Manager

Manage the full branding lifecycle for Vrooli scenarios — check status, generate assets, apply brands, and export packages using the brand-manager CLI.

> **Draft skill.** The brand-manager scenario is in the swarm-manager ideas backlog. This skill documents the planned CLI interface so the ecosystem knows how to use it once built. Commands below are planned, not yet functional.

Required reading:
- `prompt-manager skill read skill-principles`

---

### 1. When to Use This Tool

| Goal | Command |
|------|---------|
| Check branding status across all scenarios | `brand-manager status` |
| Check branding status for one scenario | `brand-manager status --scenario <name>` |
| Generate brand assets (AI-assisted) | `brand-manager generate` |
| Apply a brand to a scenario | `brand-manager apply --brand <name> --scenario <name>` |
| Partially apply specific aspects | `brand-manager apply --brand <name> --scenario <name> --only <aspect>` |
| Export a brand package | `brand-manager export --brand <name>` |
| List all brands | `brand-manager list` |
| Show brand details and history | `brand-manager show <name>` |

**In scope:**
- Brand generation (logo, favicon, icon, colors, typography, voice, copy)
- Brand storage with versioning and notes
- Assignment to scenarios (one brand can serve multiple scenarios)
- Application to scenarios (programmatic and agent-assisted)
- Branding status and validation
- Export for external use

**Out of scope:**
- Digital twin / behavioral personalization — different concern entirely
- Multi-tenant white-labeling — deployment concern, handled by scenario-to-desktop
- A/B testing or analytics — premature
- Auto-applying on brand update — application is always manual
- Scenario Auditor rule authoring — brand-manager registers rules; auditor enforces them

---

### 2. Core Concepts

**Brand** — A versioned bundle containing:
- **Identity:** display name, tagline, description, author
- **Visual assets:** logo (SVG + rasterized sizes), favicon (multi-size), app icon, og-image
- **Color system:** primary, secondary, accent, semantic colors (success/warning/error), dark/light variants — all WCAG-validated
- **Typography:** heading, body, and mono fonts with scale/weights
- **Voice:** tone descriptors and example copy snippets (for LLM consumption by other scenarios)
- **Notes:** freeform user input that influences generation (e.g., "ocean theme", "bold and playful")

**Assignment** — A record that brand X is applied to scenario Y, tracking what was applied, when, and which version.

**Asset** — An individual generated file with metadata (dimensions, format, purpose, version).

**Opt-out** — Rare and explicit. Only for scenarios that exist purely for testing (e.g., `test-scenario`, `hello-world`). Signaled via a tag or field in service.json. 99% of scenarios need branding — all are intended for monetization.

---

### 3. Command Reference

| Command | Purpose |
|---------|---------|
| `brand-manager status` | Show branding completeness for all scenarios |
| `brand-manager status --scenario <name>` | Detailed branding status for one scenario |
| `brand-manager generate` | Interactive brand generation (user provides what they know, AI fills gaps via OpenRouter) |
| `brand-manager apply --brand <name> --scenario <name>` | Apply brand to scenario (two-tier: programmatic + agent-assisted) |
| `brand-manager apply --brand <name> --scenario <name> --only <aspect>` | Partial apply (e.g., `--only logo`, `--only colors`, `--only typography`) |
| `brand-manager export --brand <name>` | Export brand package (assets + metadata) |
| `brand-manager list` | List all brands in the library |
| `brand-manager show <name>` | Show brand details, assigned scenarios, and version history |
| `brand-manager help` | Full command reference (once CLI is available) |

---

### 4. Primary Workflow

#### Step 1: Check status

```bash
brand-manager status
```

Shows a table of all scenarios with their branding completeness. Example output:

```
Scenario                          Status      Brand            Missing
landing-page-business-suite       ✓ complete  Acme Analytics   —
web-console                       ◐ partial   Vrooli Core      favicon, typography
vrooli-onboarding                 ✗ none      —                all
test-scenario                     ⊘ opted-out —                —
```

#### Step 2: Generate or import

```bash
brand-manager generate
```

Interactive wizard. The user fills in what they know (a specific primary color, a tagline, notes like "ocean theme") and leaves the rest blank. Brand Manager uses OpenRouter to generate everything that's missing — logo concepts, color palette, typography pairings, copy. The user picks from options and refines iteratively.

#### Step 3: Review

```bash
brand-manager show my-brand
```

Displays the full brand definition, assigned scenarios, and version history.

#### Step 4: Apply

```bash
brand-manager apply --brand my-brand --scenario my-scenario
```

Application decision:

```
Apply brand to scenario
├─ Standard patterns found (CSS custom properties, manifest.json, favicon dirs, service.json)?
│   └─ Programmatic application (fast, deterministic)
├─ Non-standard or complex UI integration needed?
│   └─ Agent-assisted application (agent sets things up to be programmatically validatable)
└─ Partial apply requested (--only)?
    └─ Only specified aspects applied; validation flags remaining gaps
```

Partial apply is supported — you can apply just the logo now and the color system later.

#### Step 5: Validate

```bash
brand-manager status --scenario my-scenario
```

Confirms all branding checks pass. If gaps remain, the output tells you exactly what's missing.

---

### 5. Discovery (Auto-Populate)

When a user first works with a scenario in Brand Manager, it scans the scenario's existing state — service.json, theme files, static assets, manifests — and populates a draft brand from what it finds. If the display name is just the slug or clearly not a proper name, it's flagged as needing attention, not silently accepted.

This means you never start from zero — Brand Manager bootstraps from whatever branding already exists.

---

### 6. Ecosystem Integration

**Scenario Auditor:** Brand Manager registers validation rules for each branding requirement (has logo, has favicon, has color system, has display name, has typography). These rules are always programmatically checkable — that's a hard constraint on how application works. The auditor scans and reports; it does not gate deployments directly.

**deployment-coordinator:** Checks branding readiness during Phase 2 (Assess Target). If branding is incomplete for a scenario being deployed, it suggests loading this skill for remediation.

**cross-platform-readiness:** The Red Flags Checklist includes branding items. Branding gaps are deployment red flags — a scenario without a proper display name or favicon is not ready for distribution.

**marketing-crew:** Brand Manager is the authoritative source for visual identity. Marketing content should pull brand colors, logos, and voice from Brand Manager rather than improvising.

---

### 7. Troubleshooting & Edge Cases

| Symptom | Likely Cause | Fix |
|---------|-------------|-----|
| `brand-manager` command not found | Scenario not built or not running | Start brand-manager: `cd scenarios/brand-manager && make start` |
| Generate produces low-quality assets | Notes too vague for OpenRouter | Provide specific notes (e.g., "ocean theme, bold geometric logo, dark mode primary") |
| Apply fails on non-standard scenario | Complex integration beyond programmatic patterns | Agent-assisted apply handles this; check agent output for details |
| Validation still fails after apply | Partial apply — some aspects not yet applied | Run `brand-manager status --scenario <name>` to see what's missing |
| Brand not found | Name mismatch or brand not yet created | Run `brand-manager list` to see available brands |
| Status shows opted-out for a real scenario | Incorrect opt-out tag/field in service.json | Remove the opt-out marker; only pure test scenarios should opt out |

---

### 8. Guardrails

**Do:**
- Run `brand-manager status` before applying to understand current state
- Use `brand-manager generate` rather than manually creating brand assets
- Apply brands iteratively — partial apply is supported and expected
- Validate after every apply with `brand-manager status --scenario <name>`
- Let discovery auto-populate from existing scenario state before generating new assets

**Do NOT:**
- Manually edit scenario theme files or manifests when brand-manager can apply them programmatically
- Skip validation after applying — always confirm status
- Apply a brand without the target scenario running (apply needs to read scenario state)
- Assume all scenarios need the same brand — brands are reusable but assignment is explicit
- Opt out scenarios from branding unless they are purely for testing (this is rare and intentional)

---

### 9. Output Expectations

| Command | Expected Result |
|---------|----------------|
| `status` | Human-readable table showing per-scenario branding completeness |
| `generate` | Brand created in library with all generated assets |
| `apply` | Scenario updated with branding assets; validation passes for applied aspects |
| `export` | Brand package file at specified output path |
| `list` | Table of all brands with assignment counts |
| `show` | Full brand details, assigned scenarios, version history |

---

### 10. Resource Dependencies

- **OpenRouter** — image generation (logos, favicons, icons) + LLM for copy and palette suggestions, following agent-inbox's established patterns
- **SQLite** — brand metadata, assignments, version history (portable, no server required)
