## Steer focus: UI Design System Migration

Prioritize migrating `scenarios/{{TARGET}}/ui/` to a **token-driven, primitive-owned design system** so major theme refreshes become low-risk and repeatable.

Your goal is to reduce style coupling, preserve behavior, and converge on a UI where visual identity can be changed by updating tokens and primitive variants rather than rewriting surfaces.

Required reading:
- `docs/agent-system/SKILL_AUTHORING.md`
- `prompt-manager skill read react-coherence react-stability experience-architecture-audit`
- `prompt-manager skill read visited-tracker-tools`

---

### **1. Scope Boundaries**

**In scope:**
- Token system design and migration (`shared/theme` ownership)
- Primitive/composite ownership (`shared/ui/primitives`, `shared/ui/composites`)
- Surface migration from ad-hoc styles to design-system contracts
- Theme-refresh readiness gates and migration burn-down tracking

**Out of scope:**
- New product capabilities or backend/API changes
- Major user-flow redesigns unrelated to design-system migration
- One-off visual tweaks that bypass tokens/primitives
- Breaking selectors without explicit coordination

---

### **2. Migration Intent Contract (Required Before Code Changes)**

Capture a short migration brief first:

1. **Intent:** what should feel different (e.g., cleaner, calmer, denser, more editorial).
2. **References:** target direction(s) and non-goals.
3. **Constraints:** flows, selectors, accessibility, and behavior that must stay stable.
4. **Scope:**
   - Token refresh only
   - Token + primitive refresh
   - Token + primitive + layout refresh

If this brief does not exist, create/update `scenarios/{{TARGET}}/docs/internal/EXPERIENCE-AUDIT.md` with it before major migration work.

---

### **3. Decision Model: Pick the Correct Migration Shape**

| Condition | Migration Shape |
|----------|------------------|
| Existing app already uses semantic tokens + primitives widely | Token refresh only |
| Tokens exist but surfaces still rely on legacy class contracts | Token + primitive refresh |
| Surfaces contain heavy ad-hoc palette/layout contracts | Full phased migration |

Use this rule:
- If changing theme requires touching many surface files, you are **not** design-system ready.

---

### **4. Baseline Audit (Run at Start of Each Loop)**

```bash
# 1) Style coupling inventory
rg "#[0-9a-fA-F]{3,8}|rgb\(|hsl\(" scenarios/{{TARGET}}/ui/src --type tsx --type css

# 2) Primitive and token ownership visibility
find scenarios/{{TARGET}}/ui/src/shared -maxdepth 3 -type f | sort

# 3) Existing primitive contracts
rg "cva\(" scenarios/{{TARGET}}/ui/src --type tsx

# 4) Legacy style contract usage (rename pattern as needed)
rg "legacy-|old-|ko-|app-" scenarios/{{TARGET}}/ui/src --type tsx --type css

# 5) State/complexity hotspots that increase migration risk
rg "useState\(" scenarios/{{TARGET}}/ui/src --type tsx -c | sort -t: -k2 -nr | head -20
```

Record findings in `scenarios/{{TARGET}}/docs/internal/COHERENCE-NOTES.md`.

---

### **5. Ownership Architecture (Target End State)**

```
shared/theme/                 # Tokens + themes + typography + motion
  tokens.css
  themes/
  typography.ts
  motion.ts

shared/ui/primitives/         # Button, Card, Input, Badge, Tabs
shared/ui/composites/         # PanelHeader, EmptyState, ToolbarRow, etc.

surfaces/*                    # Feature/page composition only
```

Ownership rules:
1. `shared/theme` owns visual language values.
2. `shared/ui/primitives` owns base interaction and variant contracts.
3. `shared/ui/composites` owns repeated multi-part patterns.
4. `surfaces/*` may assemble, but should not invent new base primitives.

---

### **6. Phased Migration Sequence (Recommended)**

1. **Stabilize Tokens**
- Create or normalize semantic tokens (color, surface, border, radius, space, shadow, motion).
- Ensure old UI still renders via compatibility mapping.

2. **Normalize Primitives**
- Standardize `Button/Card/Input/Badge/Tabs` around semantic variants.
- Remove repeated primitive-like class clusters from surfaces.

3. **Migrate High-Traffic Surfaces First**
- Prioritize app shell, navigation, dashboard, and primary workflows.

4. **Migrate Remaining Surfaces**
- Complete secondary flows, edge states, and modals.

5. **Retire Legacy Contracts**
- Remove deprecated classes only after usage reaches zero.

---

### **7. Coexistence and Deprecation Rules**

Temporary dual styling is allowed only when all three are true:
1. Old and new contracts are clearly named.
2. Deprecation intent is documented in `COHERENCE-NOTES.md`.
3. Removal criteria are tracked (usage count, target phase, owner).

Do not leave permanent mixed contracts.

---

### **8. Quality Gates Per Phase**

Must verify each phase:
1. WCAG AA contrast for core text and actions.
2. Keyboard navigation and focus visibility are intact.
3. Desktop and mobile layouts remain stable.
4. Loading/error/empty states remain legible.
5. Selectors and automation hooks remain stable unless coordinated.
6. `pnpm lint`, `pnpm type-check`, and scenario tests pass.

---

### **9. Documentation Alignment (Required)**

When visual language changes, documentation must reflect the new contract in the same migration stream.

Required updates:
1. Update `scenarios/{{TARGET}}/PRD.md` if visual identity/branding language changed.
2. Update `scenarios/{{TARGET}}/README.md` UI descriptions (and screenshots if present) so they match the shipped interface.
3. Update `scenarios/{{TARGET}}/docs/internal/EXPERIENCE-AUDIT.md` with the migration brief and post-migration flow/readability outcomes.
4. Update `scenarios/{{TARGET}}/docs/internal/COHERENCE-NOTES.md` with old-vs-new style contract notes and remaining debt.

Rule:
- If screenshots, branding language, or UX claims are stale after migration, the migration is not complete.

---

### **10. Convergence Scorecard (Use Every Loop)**

Track these indicators in `COHERENCE-NOTES.md`:

| Indicator | Target |
|----------|--------|
| Surface files with raw palette values | 0 |
| Legacy class contract references in migrated surfaces | 0 |
| Core primitives migrated to semantic variants | 100% |
| High-traffic surfaces migrated | 100% |
| Deprecated contract backlog with no owner | 0 |

Definition of done:
- Theme changes can be shipped by editing tokens and primitive variants with minimal surface churn.

---

### **11. Anti-Gaming Rules**

Avoid superficial migration churn:
- Do not claim migration progress from file moves/renames alone.
- Do not replace one ad-hoc class soup with another.
- Do not pad progress metrics without reducing legacy-contract usage.

Real progress means fewer style touchpoints in surfaces and stronger primitive/token ownership.

---

### **12. Memory Loop with visited-tracker**

Use visited-tracker for systematic migration loops:

```bash
# Find least-reviewed UI files for this migration stream
visited-tracker least-visited --location scenarios/{{TARGET}}/ui --tag ui-design-system-migration --limit 10

# Mark files as reviewed after migration work
visited-tracker visit <file-path> --location scenarios/{{TARGET}}/ui --tag ui-design-system-migration --note "migrated to primitive contracts"
```

Also keep these docs current:
- `scenarios/{{TARGET}}/docs/internal/COHERENCE-NOTES.md`
- `scenarios/{{TARGET}}/docs/internal/EXPERIENCE-AUDIT.md`

---

### **13. Output Expectations**

You may update:
- Theme tokens and theme maps
- UI primitives/composites and their variant contracts
- Surface markup to consume primitives instead of legacy style contracts
- Internal migration tracking docs

You must:
- Preserve behavior and scenario workflows
- Keep selectors stable or explicitly coordinate changes
- Keep or improve test reliability
- Leave a clear migration status trail (what moved, what remains, what is blocked)

Avoid:
- One-shot rewrites without phased checkpoints
- Mixing long-term contracts across parallel style systems
- Hardcoded scenario names (always use `{{TARGET}}`)
