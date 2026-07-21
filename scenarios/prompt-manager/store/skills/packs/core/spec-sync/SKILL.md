## Steer focus: Spec Sync

Synchronize **specification artifacts with actual implementation state** in `scenarios/{{TARGET}}/`, ensuring that PRD.md, requirements/, README.md, and docs/ accurately describe what the code does today.

Your goal is to produce specifications so faithful to the current implementation that a new agent could **recreate the scenario from specs alone** with full feature parity. This is the critical precondition before archiving a scenario via the swarm-manager.

Required reading:
- `prompt-manager skill read documentation-health`
- `prompt-manager skill read visited-tracker-tools`

---

### **1. Why This Skill Exists**

Scenarios evolve through many agent sessions. Over time, specs drift from reality:
- PRD.md describes features that were descoped or changed
- requirements/ modules reference code paths that no longer exist
- README.md advertises capabilities the code doesn't actually have
- Docs describe architectures that were refactored away

This matters critically when **archiving and recreating scenarios**. The swarm-manager archive flow preserves specs (PRD.md, requirements/, docs/) but deletes the implementation. If those specs don't match the code at archive time, the recreated scenario will have **different behavior** than the original — defeating the entire purpose of archiving.

**The spec-sync skill closes the loop:** run it before archiving to guarantee that preserved specs are an accurate blueprint of the implementation being replaced.

---

### **2. Scope**

**In scope:**
- Updating PRD.md operational targets to match implemented features
- Updating requirements/ modules and status fields to match code reality
- Updating README.md to accurately describe current capabilities, setup, and usage
- Updating docs/ content that describes architecture, APIs, or behavior
- Adding missing requirements for features that exist in code but not in specs
- Removing or marking requirements for features that were never implemented
- Ensuring documentation health standards are met (per `documentation-health` skill)

**Out of scope:**
- Changing implementation code (this skill reads code, it does not modify it)
- Adding new features or fixing bugs
- Creating aspirational requirements for planned-but-unbuilt features
- Modifying other scenarios
- Changing skill files or prompt-manager configuration

---

### **3. The Sync Direction: Code Is Truth**

```
                    ┌─────────────────────┐
                    │   Implementation     │
                    │   (source of truth)  │
                    └──────────┬──────────┘
                               │
                    ┌──────────▼──────────┐
                    │   Read & Analyze     │
                    │   actual behavior    │
                    └──────────┬──────────┘
                               │
          ┌────────────────────┼────────────────────┐
          ▼                    ▼                    ▼
    ┌───────────┐      ┌─────────────┐     ┌────────────┐
    │  PRD.md   │      │requirements/│     │ README.md  │
    │  (update) │      │  (update)   │     │  (update)  │
    └───────────┘      └─────────────┘     └────────────┘
```

**The code is always the source of truth.** Specs are updated to match code, never the reverse. If the code does something the PRD doesn't mention, add it to the PRD. If the PRD describes something the code doesn't do, remove it from the PRD or mark it as not implemented.

---

### **4. Sync Process**

Follow this sequence to systematically sync all spec artifacts:

#### Phase 1: Inventory & Gap Detection

Map what exists in the scenario:

```bash
# What spec artifacts exist?
ls scenarios/{{TARGET}}/PRD.md scenarios/{{TARGET}}/README.md scenarios/{{TARGET}}/requirements/ scenarios/{{TARGET}}/docs/ 2>/dev/null

# What implementation components exist?
ls scenarios/{{TARGET}}/api/ scenarios/{{TARGET}}/ui/ scenarios/{{TARGET}}/cli/ scenarios/{{TARGET}}/store/ 2>/dev/null

# Get a high-level view of the codebase
find scenarios/{{TARGET}} -name "*.go" -o -name "*.ts" -o -name "*.tsx" -o -name "*.py" | head -50
```

**Gap Detection:** For each missing artifact, record the required action:

| Missing Artifact | Action | When |
|-----------------|--------|------|
| `PRD.md` | **STOP** — flag to user. PRD.md is a prerequisite input for this skill. | — |
| `requirements/` | Bootstrap via `vrooli scenario requirements init {{TARGET}}` (or the business-health wizard for a full contract) | Phase 4 |
| `docs/internal/SEAMS.md` | Create via `knowledge-observatory docs template seams` | Phase 6 |
| `docs/internal/PROBLEMS.md` | Create via `knowledge-observatory docs template problems` | Phase 6 |
| `docs/internal/PROGRESS.md` | Create from template | Phase 6 |
| `docs/manifest.json` | Create with initial entries | Phase 6 |

**Missing infrastructure is work this skill must perform, not an "unresolved discrepancy."** If an artifact doesn't exist, you create it using the CLI tooling listed above — that is core sync work.

#### Phase 2: Capability Extraction

Read implementation code to build a **capability map** — a structured list of what the code actually does:

```
┌─────────────────────────────────────────────────┐
│              CAPABILITY MAP                      │
├─────────────────────────────────────────────────┤
│                                                  │
│  For each component (api, ui, cli):              │
│    1. List every endpoint / route / command       │
│    2. List every feature / behavior              │
│    3. List external dependencies & integrations  │
│    4. List data models & storage schemas         │
│    5. Note configuration options                 │
│    6. Note error handling patterns               │
│                                                  │
└─────────────────────────────────────────────────┘
```

**Decision: What counts as a "capability"?**

| Question | If YES → Capability | If NO → Skip |
|----------|-------------------|--------------|
| Does it handle a user request or action? | Yes | — |
| Does it expose an API endpoint? | Yes | — |
| Does it transform or persist data? | Yes | — |
| Does it integrate with an external service? | Yes | — |
| Is it a utility function used only internally? | — | Skip |
| Is it boilerplate/scaffolding with no business logic? | — | Skip |

#### Phase 3: PRD Sync

Compare the capability map against PRD.md operational targets:

```
            Capability Map              PRD Operational Targets
            ┌──────────┐               ┌──────────────────┐
            │ Feature A │──── matches ──│ OT-P0-001 ✓     │
            │ Feature B │──── missing ──│ (not in PRD) ✗   │
            │           │               │ OT-P0-003 ✗     │──── not in code
            │ Feature C │──── matches ──│ OT-P1-002 ✓     │
            └──────────┘               └──────────────────┘
```

**Actions:**
- Features in code but not in PRD → **Add** operational target
- PRD targets fully implemented → **Mark** checkbox as complete `[x]`
- PRD targets not implemented → **Unmark** checkbox `[ ]` and add note: `(not implemented)`
- PRD targets partially implemented → **Add** inline note describing what is/isn't done
- PRD describes behavior differently than code → **Correct** the PRD description

#### Phase 4: Requirements Sync

**Bootstrap step — if `requirements/` does not exist:**

If Phase 1 detected that `requirements/` is missing, bootstrap it before syncing:

1. **Primary path** — scaffold the registry skeleton from PRD operational targets:
   ```bash
   vrooli scenario requirements init {{TARGET}}
   ```
   This routes to the `prd_missing_requirements` fixer, creating `requirements/index.json`, `requirements/README.md`, and `requirements/01-*/module.json` files from PRD operational targets. For a full guided contract (PRD + requirements together), drive the business-health wizard instead — see the scenario-generation skill.

2. **Fallback** — if the routed `init` cannot resolve, apply the fixer directly:
   ```bash
   business-health fix apply {{TARGET}} --rules prd_missing_requirements
   ```

3. **Validate** the generated structure:
   ```bash
   vrooli scenario requirements validate {{TARGET}} --json
   ```

4. Proceed to the sync step below to verify generated statuses against actual code.

> **Note:** This bootstrap step is NOT what the "Avoid" section prohibits. The "Avoid" caveat applies to creating granular modules for trivial internal helpers — bootstrapping the `requirements/` structure itself is essential infrastructure work.

**Sync step — if `requirements/` already exists (or was just bootstrapped):**

For each module in `requirements/`:

1. **Read the module.json** — check each requirement's status and validation references
2. **Verify against code** — does the referenced code/test actually exist and do what's described?
3. **Update status fields:**

| Code State | Requirement Status |
|------------|-------------------|
| Feature fully working + tests pass | `"implemented"` or `"complete"` |
| Feature exists but no tests | `"implemented"` (note: missing validation) |
| Feature partially working | `"in-progress"` with description of gaps |
| Feature doesn't exist at all | `"pending"` or remove if descoped |
| Test exists and references valid | validation status: `"passing"` |
| Test reference points to deleted file | **Fix** the reference or remove |

4. **Add missing requirements** for capabilities found in code but not in any module
5. **Update validation references** to point to actual test files and function names

#### Phase 5: README Sync

Ensure README.md accurately describes:

| Section | Verify Against |
|---------|---------------|
| Project description / overview | PRD.md + actual behavior |
| Features list | Capability map |
| Setup / installation instructions | Actual dependencies, configs, Makefile |
| Usage examples | Working endpoints, CLI commands |
| API documentation summary | Actual routes and handlers |
| Configuration | Actual env vars, config files |
| Architecture overview | Actual code structure |

**Common README drift patterns:**
- Lists features from PRD that were never built
- Setup instructions reference old dependency versions
- API examples use endpoints that were renamed or removed
- Architecture diagram shows planned structure, not actual

#### Phase 6: Documentation Health

Apply `documentation-health` skill standards:
- Ensure `docs/manifest.json` exists and registers all doc files
- Verify `[CODE: ...]` references in docs point to real files
- Verify `// DOC:` comments in code point to real docs
- Ensure `docs/internal/SEAMS.md`, `docs/internal/PROBLEMS.md`, and `docs/internal/PROGRESS.md` exist (these are "always create" core internal docs per the `documentation-health` skill). If missing, use `knowledge-observatory docs template <type>` to get the template, create the file, then populate with findings from Phases 1-5

---

### **5. Archive Readiness Checklist**

Before declaring sync complete, verify this checklist. A scenario is **archive-ready** when all critical items pass:

```
Archive Readiness Assessment
════════════════════════════

CRITICAL (must pass):
  [ ] PRD.md operational targets match implemented features
  [ ] Every implemented feature has a corresponding requirement
  [ ] Every requirement status reflects actual code state
  [ ] README.md feature list matches reality
  [ ] No spec artifact references code/files that don't exist

IMPORTANT (should pass):
  [ ] Requirements have valid test/validation references
  [ ] README setup instructions are verified working
  [ ] docs/manifest.json is complete
  [ ] Internal docs (SEAMS, PROBLEMS, PROGRESS) are current
  [ ] Architecture documentation matches actual code structure

NICE-TO-HAVE:
  [ ] Bidirectional DOC:/[CODE:] references are complete
  [ ] All docs follow documentation-health standards
  [ ] Configuration documentation covers all options
```

**When a CRITICAL item fails — decision tree:**

```
CRITICAL item fails
    │
    ├─ Infrastructure exists but content is wrong?
    │   └─ Fix the content (normal sync work)
    │
    └─ Infrastructure does not exist at all?
        │
        ├─ requirements/ or modules?
        │   └─ Bootstrap via vrooli scenario requirements init (Phase 4)
        │
        ├─ Core internal doc (SEAMS, PROBLEMS, PROGRESS)?
        │   └─ Create via knowledge-observatory docs template (Phase 6)
        │
        └─ PRD.md?
            └─ STOP — flag to user. PRD.md is a prerequisite input.
```

A CRITICAL item that fails due to missing infrastructure is a signal to **create the infrastructure** using existing CLI tooling, not to log an "unresolved discrepancy."

---

### **6. Anti-Gaming Measures**

Real sync work means:
- **Reading implementation code** to verify what it does, not just trusting existing docs
- **Testing assertions** — if a requirement says "status: implemented", check the code exists
- **Removing inaccuracies** — deleting a wrong requirement is better than leaving it
- **Noting gaps honestly** — if validation references are missing, say so rather than fabricating them

Superficial sync looks like:
- Bulk-updating all requirement statuses to "complete" without checking code
- Copy-pasting code comments into docs without verifying accuracy
- Adding `[CODE: ...]` references to every doc without checking the paths exist
- Marking the archive readiness checklist as passed without doing the checks

---

### **7. Memory Management**

Use the `visited-tracker-tools` skill for tracking visited files, with LOCATION set to `scenarios/{{TARGET}}` and TAG set to `spec-sync`.

#### At Session Start

Read existing findings:
- `scenarios/{{TARGET}}/docs/internal/PROGRESS.md` — what's been completed
- `scenarios/{{TARGET}}/docs/internal/PROBLEMS.md` — known issues
- `scenarios/{{TARGET}}/PRD.md` — current operational targets
- `scenarios/{{TARGET}}/requirements/index.json` — current requirements structure

#### At Session End

Update findings in `scenarios/{{TARGET}}/docs/internal/PROGRESS.md`:
- Record which spec artifacts were synced and what changed
- Note any unresolved discrepancies between code and specs
- List any capabilities found in code that still need requirement coverage

---

### **8. Output Expectations**

You may update:
- `scenarios/{{TARGET}}/PRD.md` — operational targets, descriptions, completion status
- `scenarios/{{TARGET}}/requirements/**` — module.json files, index.json, status fields, validation references
- `scenarios/{{TARGET}}/README.md` — feature descriptions, setup instructions, usage examples
- `scenarios/{{TARGET}}/docs/**` — architecture docs, API docs, internal docs
- `scenarios/{{TARGET}}/docs/manifest.json` — document registration

You **must**:
- Read implementation code before updating any spec artifact
- Verify every requirement status claim against actual code
- Ensure the archive readiness checklist critical items all pass
- Keep the scenario fully functional (no code changes that break anything)
- Follow `documentation-health` skill standards for all doc updates

You must **NOT**:
- Modify implementation code (read-only access to source files)
- Add aspirational requirements for unbuilt features
- Remove requirements without checking if the feature was descoped vs never started
- Fabricate validation references to tests that don't exist

**Avoid:**
- Trusting existing specs without verifying against code
- Bulk status updates without individual verification
- Over-documenting internal utilities — focus on user-facing capabilities
- Creating granular requirements modules for trivial internal helpers (e.g., a separate module for a string-formatting utility). You **should** still bootstrap the `requirements/` structure via `vrooli scenario requirements init` if it does not exist — see Phase 4.
