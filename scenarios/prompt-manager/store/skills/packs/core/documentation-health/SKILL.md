## Steer focus: Documentation Health

Prioritize **documentation quality, consistency, and bidirectional traceability** between code and documentation across this scenario.

Your goal is to ensure documentation remains accurate, discoverable, and tightly coupled to the code it describes, preventing drift, gaps, and duplication.

Do **not** change core business logic or introduce new features. All changes focus on documentation structure, references, and validation infrastructure.

---

### **1. Why This Skill Exists**

Documentation suffers from predictable failure modes:
- **Drift**: Code changes without corresponding doc updates
- **Gaps**: Features added without documentation
- **Duplication**: Agents can't find existing docs, so they create new ones
- **Orphaning**: Docs become disconnected from the code they describe

The root cause: **No systematic approach to bidirectional code↔docs traceability.**

This skill provides concrete patterns that ensure agents across multiple sessions maintain documentation the same way, preventing entropy over time.

---

### **2. Documentation Hierarchy**

Documentation should align with the mental model hierarchy used in screaming-architecture-audit:

Required reading:
- `prompt-manager skills read screaming-architecture-audit`

```
                      PRD.md
                   (Why does this exist?)
                         │
                         ▼
              Operational Targets
           (What must it accomplish?)
                         │
                         ▼
            Technical Requirements
         (How do we measure success?)
                         │
                         ▼
               Implementation Docs
        (How does the code achieve this?)
                         │
                         ▼
                    Code Files
               (The actual solution)
```

**Decision Tree: Where Does This Doc Belong?**

```
                    What type of content?
                           │
         ┌─────────────────┼─────────────────┐
         ▼                 ▼                 ▼
    Business/Why      Technical How     Reference/API
         │                 │                 │
         ▼                 ▼                 ▼
    PRD.md or         docs/guides/      docs/reference/
    docs/concepts/    docs/architecture/ or inline JSDoc
```

---

### **3. Standard docs/ Directory Layout**

```
docs/
├── manifest.json          # Navigation & metadata (REQUIRED for UI display)
├── QUICKSTART.md          # First-touch experience
├── concepts/              # Mental model documentation
│   ├── ARCHITECTURE.md    # High-level system design
│   └── GLOSSARY.md        # Domain vocabulary
├── guides/                # Task-oriented walkthroughs
│   ├── getting-started.md
│   └── troubleshooting.md
├── reference/             # API, CLI, config specs
│   ├── api-endpoints.md
│   ├── cli-commands.md
│   └── configuration.md
├── internal/              # Developer-only docs (agent memory)
│   ├── SEAMS.md           # Integration boundaries, responsibility zones, testability
│   ├── PROBLEMS.md        # Known issues, tech debt, deferred work
│   ├── PROGRESS.md        # Development history, what's been completed
│   ├── INVARIANTS.md      # System contracts that must never be violated (optional)
│   ├── ASSUMPTIONS.md     # Implicit beliefs not yet validated (optional)
│   ├── ERROR-SEMANTICS.md # Error categories, recovery paths (optional)
│   ├── SECURITY-POSTURE.md # Security hardening status (optional)
│   ├── TEMPORAL-FLOWS.md  # Async patterns, race conditions (optional)
│   ├── COHERENCE-NOTES.md # React coherence audit (React UIs only)
│   └── EXPERIENCE-AUDIT.md # UX friction analysis (user-facing only)
└── plans/                 # Architecture decisions, proposals
```

#### Document Placement Decision Table

| Content Type | Primary Location | Secondary Reference |
|--------------|------------------|---------------------|
| Why this exists | PRD.md | docs/concepts/ |
| How to use it | docs/QUICKSTART.md | docs/guides/ |
| API contract | docs/reference/api-endpoints.md | Inline JSDoc |
| CLI usage | docs/reference/cli-commands.md | --help output |
| Config options | docs/reference/configuration.md | Schema files |
| Known issues | docs/internal/PROBLEMS.md | GitHub Issues |
| Architecture decisions | docs/plans/ | ADR format |
| Code behavior | Inline comments | docs/reference/ |

---

### **4. Bidirectional Reference Format**

Establish consistent, searchable formats for linking code and documentation.

#### Code-to-Doc References (DOC: comments)

Use `// DOC:` comments to link code to its documentation:

**TypeScript/JavaScript:**
```typescript
// DOC: docs/reference/api-endpoints.md#user-authentication
// DOC: PRD.md#OT-P0-003
export function authenticateUser(token: string): Promise<User> {
```

**Go:**
```go
// DOC: docs/reference/api-endpoints.md#health-check
// HealthHandler returns the service health status.
func (s *Server) HealthHandler(w http.ResponseWriter, r *http.Request) {
```

**Multiple References:**
```typescript
// DOC: docs/guides/getting-started.md#configuration
// DOC: docs/reference/configuration.md#environment-variables
// DOC: PRD.md#OT-P0-003
const config = loadConfiguration();
```

#### Doc-to-Code References ([CODE: ...] syntax)

Use bracketed syntax in markdown that can be validated:

```markdown
## Authentication Flow

The authentication process is implemented in:
- [CODE: src/auth/authenticator.ts#authenticateUser] - Main entry point
- [CODE: src/auth/token.ts#validateToken] - Token validation
- [CODE: api/handlers/auth.go:AuthHandler] - HTTP handler

See also: [REQ: OT-P0-005] for the requirement this implements.
```

#### Reference Format Specification

| Reference Type | Format | Example |
|----------------|--------|---------|
| Code file | `[CODE: path/to/file.ext]` | `[CODE: src/auth/token.ts]` |
| Code function | `[CODE: path/to/file.ext#functionName]` | `[CODE: src/auth/token.ts#validateToken]` |
| Code line | `[CODE: path/to/file.ext:lineNumber]` | `[CODE: api/main.go:142]` |
| Requirement | `[REQ: ID]` | `[REQ: OT-P0-005]` |
| Doc section | `[DOC: path/to/doc.md#section]` | `[DOC: docs/QUICKSTART.md#setup]` |

#### Protective Comments for Critical Documentation

When documentation is critical for understanding, add protective comments:

```typescript
// ╔════════════════════════════════════════════════════════════════╗
// ║  IMPORTANT: This implements the core authentication flow.     ║
// ║  DOC: docs/concepts/ARCHITECTURE.md#authentication            ║
// ║  Before modifying, read the documentation and update it       ║
// ║  if your changes affect the described behavior.               ║
// ╚════════════════════════════════════════════════════════════════╝
```

---

### **5. manifest.json Documentation Pattern**

Every scenario with UI documentation should include a `docs/manifest.json`:

```json
{
  "version": "1.0.0",
  "title": "Scenario Documentation",
  "description": "Brief description of documentation scope",
  "defaultDocument": "QUICKSTART.md",
  "sections": [
    {
      "id": "getting-started",
      "title": "Getting Started",
      "icon": "rocket",
      "documents": [
        {
          "path": "QUICKSTART.md",
          "title": "Quick Start",
          "description": "Get running in 5 minutes",
          "audience": ["users", "developers"]
        }
      ]
    },
    {
      "id": "reference",
      "title": "Reference",
      "icon": "book",
      "documents": [
        {
          "path": "reference/api-endpoints.md",
          "title": "API Reference",
          "description": "REST API documentation"
        }
      ]
    },
    {
      "id": "internal",
      "title": "Internal",
      "icon": "lock",
      "visibility": "developers-only",
      "documents": [
        {
          "path": "internal/PROBLEMS.md",
          "title": "Known Issues",
          "internal": true
        }
      ]
    }
  ],
  "navigation": {
    "primary": ["getting-started", "reference"],
    "secondary": ["internal"]
  }
}
```

**Manifest Rules:**
- All docs in `docs/` should be registered in the manifest
- Use `visibility: "developers-only"` for internal docs
- Group related docs into sections
- Provide descriptions for discoverability

---

### **6. Documentation Health Audit**

Run this audit when joining a project, before major work, or periodically for maintenance.

#### Step 1: Documentation Infrastructure Check

```bash
# Check for manifest.json
ls -la scenarios/{{TARGET}}/docs/manifest.json 2>/dev/null || echo "MISSING: docs/manifest.json"

# Check for required docs
for doc in PRD.md docs/QUICKSTART.md docs/concepts/ARCHITECTURE.md; do
  [ -f "scenarios/{{TARGET}}/$doc" ] || echo "MISSING: $doc"
done

# Count total documentation files
rg --files -g "*.md" scenarios/{{TARGET}}/docs | wc -l
```

#### Step 2: Find Code Without Documentation References

```bash
# TypeScript/JavaScript files without DOC: comments
rg -l "export (function|class|const|interface)" scenarios/{{TARGET}}/ui/src --type ts -g "!*.test.*" -g "!*.spec.*" | while read f; do
  if ! rg -q "DOC:" "$f"; then
    echo "NO_DOC_REF: $f"
  fi
done

# Go files without DOC: comments
rg -l "^func " scenarios/{{TARGET}}/api --type go -g "!*_test.go" | while read f; do
  if ! rg -q "DOC:" "$f"; then
    echo "NO_DOC_REF: $f"
  fi
done
```

#### Step 3: Find Documentation With Broken Code References

```bash
# Extract CODE references and validate they exist
rg "\[CODE:\s*([^\]#:]+)" scenarios/{{TARGET}}/docs -o -r '$1' | while read ref; do
  file=$(echo "$ref" | sed 's/[#:].*$//')
  if [ ! -f "scenarios/{{TARGET}}/$file" ]; then
    echo "BROKEN_CODE_REF: $ref"
  fi
done
```

#### Step 4: Find Orphaned Documentation

```bash
# Docs not referenced in manifest.json
if [ -f "scenarios/{{TARGET}}/docs/manifest.json" ]; then
  rg --files -g "*.md" scenarios/{{TARGET}}/docs | while read doc; do
    relpath=$(echo "$doc" | sed "s|scenarios/{{TARGET}}/docs/||")
    if ! rg -q "\"path\":\s*\"$relpath\"" scenarios/{{TARGET}}/docs/manifest.json; then
      echo "ORPHANED_DOC: $doc"
    fi
  done
fi
```

#### Step 5: Find Duplicate Documentation

```bash
# Find docs with similar titles (potential duplication)
rg "^#\s+(.+)$" scenarios/{{TARGET}}/docs -o -r '$1' --no-filename | sort | uniq -d | while read title; do
  echo "DUPLICATE_TITLE: $title"
  rg -l "^#\s+$title$" scenarios/{{TARGET}}/docs
done
```

#### Step 6: Validate PRD Alignment

```bash
# Check if PRD operational targets are documented
if [ -f "scenarios/{{TARGET}}/PRD.md" ]; then
  rg "OT-P[0-9]+-[0-9]+" scenarios/{{TARGET}}/PRD.md -o | sort -u | while read ot; do
    if ! rg -q "$ot" scenarios/{{TARGET}}/docs; then
      echo "UNDOCUMENTED_OT: $ot"
    fi
  done
fi
```

#### Red Flags Checklist

- [ ] Missing `docs/manifest.json` → Create manifest for navigation
- [ ] Files with 500+ lines and no DOC: references → Add documentation links
- [ ] Broken `[CODE: ...]` references → Update paths or remove stale references
- [ ] Orphaned docs not in manifest → Add to manifest or delete if obsolete
- [ ] Duplicate documentation titles → Consolidate or differentiate
- [ ] PRD operational targets without docs → Create implementation docs

---

### **7. Memory Management with Visited Tracker**

To ensure **systematic coverage without repetition**, use `visited-tracker`:

**At the start of each iteration:**
```bash
visited-tracker least-visited \
  --location scenarios/{{TARGET}} \
  --pattern "**/*.md" \
  --tag documentation-health \
  --name "{{TARGET}} - Documentation Health" \
  --limit 5
```

**After analyzing each file:**
```bash
visited-tracker visit <file-path> \
  --location scenarios/{{TARGET}} \
  --tag documentation-health \
  --note "<summary: what was validated, what references were added>"
```

**When a file meets all documentation standards:**
```bash
visited-tracker exclude <file-path> \
  --location scenarios/{{TARGET}} \
  --tag documentation-health \
  --reason "Documentation complete and references valid"
```

---

### **8. Relationship to Other Skills**

| Skill | Focus | When to Use Together |
|-------|-------|---------------------|
| screaming-architecture-audit | Mental model alignment | Documentation-health provides the docs that screaming-architecture reads first |
| react-coherence | Code organization | Coherence patterns should be documented; docs should reference coherence decisions |
| refactor | Code cleanup | After refactoring, update DOC: references and [CODE: ...] links |
| code-cleanup | Dead code removal | Remove documentation for deleted code |

**Recommended sequence:**
1. **documentation-health** (audit) → understand documentation state
2. **screaming-architecture-audit** → align architecture with documented mental model
3. **refactor/code changes** → implement improvements
4. **documentation-health** (update) → sync docs with code changes

---

### **9. Scenario Constraints**

* Do **not** change the scenario's core workflows, APIs, or business logic
* Do **not** introduce new features unrelated to documentation
* Do **not** over-document trivial code (getters, setters, obvious utilities)

---

### **10. Output Expectations**

You may update:
* `docs/manifest.json` to register all documentation files
* Code files to add `// DOC:` comments linking to relevant documentation
* Documentation files to add `[CODE: ...]` references linking to implementation
* Documentation structure to follow the standard layout

You **must**:
* Keep the scenario fully functional and non-regressed
* Ensure all `[CODE: ...]` references point to valid files
* Ensure all `// DOC:` comments point to valid documentation
* Register new docs in `manifest.json`
* Document PRD operational targets in implementation docs

**Avoid:**
* Documentation that restates the code without adding context
* Over-documenting trivial functions
* Creating documentation that will immediately become stale
* Duplicating information that belongs in a single source of truth

Focus on **documentation that helps agents quickly understand the scenario** and maintain accurate mental models across sessions.

---

### **11. Internal Document Templates**

The `docs/internal/` directory serves as **persistent agent memory** - documents written by agents to share findings with future agents. These are NOT user-facing documentation.

#### Purpose of Internal Docs

| File | Purpose | Contributing Skills |
|------|---------|---------------------|
| SEAMS.md | Integration boundaries, responsibility zones, decision points, testability | seam-discovery, boundary-of-responsibility, decision-boundary-extraction, cognitive-load-reduction, signal-and-feedback-surface-design, change-axis-and-evolution-resilience-audit, explore |
| PROBLEMS.md | Known issues, tech debt, test gaps, UX issues, cleanup history | code-cleanup, refactor, react-stability, test, ux |
| PROGRESS.md | Development history, completed milestones | (general tracking) |
| INVARIANTS.md | System contracts that must never be violated | invariant-discovery-and-enforcement, idempotency-replay-safety-hardening |
| ASSUMPTIONS.md | Implicit beliefs not yet validated | assumption-mapping-and-hardening |
| ERROR-SEMANTICS.md | Error categories, recovery paths, user messaging | error-semantics-recovery-path-design, failure-topography-and-graceful-degradation |
| SECURITY-POSTURE.md | Security hardening status by category | security |
| TEMPORAL-FLOWS.md | Async patterns, race conditions, checkpoint flows | temporal-flow-audit, progress-continuity-interruption-resilience |
| COHERENCE-NOTES.md | React state patterns, duplication, styling audit | react-coherence |
| EXPERIENCE-AUDIT.md | Persona mapping, friction analysis, navigation | experience-architecture-audit, navigation-integrity-audit |

#### SEAMS.md Template

```markdown
# Seams & Architecture Boundaries

## Last Updated
[Date]

## Integration Seams
[Discovered seams where behavior can be substituted for testing/variation]
- **Seam name**: location, purpose, testability status

## Responsibility Zones
[Layer boundaries and what each layer owns]
- Entry/presentation:
- Coordination/orchestration:
- Domain rules:
- Integrations/infrastructure:
- Cross-cutting concerns:

## Decision Points
[Extracted decision logic and where it lives]
- **Decision**: criteria, location, test coverage

## Change Axes
[How changes land in the codebase]
- Primary change axis: [what changes most often]
- Current cost of change: [localized vs. scattered]

## Observability Surface
[Logging, metrics, feedback mechanisms]
- Signal gaps identified:
- Feedback surfaces added:

## Architecture Clarity Notes
[Simplifications made, complexity hot spots remaining]

## Exploration Log
[Architectural variations attempted and outcomes]
```

#### PROBLEMS.md Template

```markdown
# Known Issues & Technical Debt

## Last Updated
[Date]

## Code Quality Debt
[From refactor, code-cleanup]
- **Area**: issue description, severity, recommended fix

## Test Gaps
[From test skill]
- Coverage gaps:
- Flaky tests:
- Weak assertions:

## Stability Issues
[From react-stability]
- Crash-prone areas:
- Unsafe data access:
- Missing error boundaries:

## UX Issues
[From ux skill]
- Friction points:
- Clarity gaps:
- Accessibility concerns:

## Cleanup History
[Major code removal patterns and outcomes]
- [Date]: Removed X, pattern was Y, outcome was Z
```

#### INVARIANTS.md Template

```markdown
# System Invariants

## Last Updated
[Date]

## Critical Invariants
[Must never be violated - type-enforced or test-validated]

| Invariant | Domain Concept | Enforcement | Test Coverage |
|-----------|----------------|-------------|---------------|
| [description] | [what concept] | [types/validation/tests] | [test file] |

## Important Invariants
[Should be preserved but violation is recoverable]

## Replay/Idempotency Invariants
[Operations that must be safe to retry]
- **Operation**: idempotency key, safe retry pattern

## Enforcement Mechanisms
[How each invariant is protected]
```

#### ASSUMPTIONS.md Template

```markdown
# Documented Assumptions

## Last Updated
[Date]

## Data Shape Assumptions
[Expected structure of inputs/outputs]
- **Assumption**: where it's made, what breaks if wrong

## Behavioral Assumptions
[User behavior, external system behavior]

## Timing Assumptions
[Ordering, delays, timeouts]

## Environment Assumptions
[What the code expects from its runtime]

## Hardening Status
| Assumption | Status | Moved to INVARIANTS |
|------------|--------|---------------------|
| [description] | implicit/validated/hardened | [yes/no] |
```

#### ERROR-SEMANTICS.md Template

```markdown
# Error Semantics & Recovery Paths

## Last Updated
[Date]

## Error Categories

### Configuration Errors
- Recovery: [how to recover]
- User message pattern: [template]
- Machine hint: `{ category: 'CONFIG', ... }`

### Validation Errors
- Recovery: [how to recover]
- User message pattern: [template]
- Machine hint: `{ category: 'VALIDATION', field: '...', ... }`

### Connectivity Errors
- Recovery: [how to recover]
- User message pattern: [template]
- Retry strategy: [exponential backoff, etc.]

### Permission Errors
- Recovery: [how to recover]
- User message pattern: [template]

### Internal Logic Errors
- Recovery: [how to recover]
- Logging pattern: [what to log]

## Failure Modes
[From failure-topography-and-graceful-degradation]

| Flow | Failure Mode | Current Behavior | Desired Behavior |
|------|--------------|------------------|------------------|
| [flow] | [what fails] | [what happens now] | [graceful degradation] |
```

#### SECURITY-POSTURE.md Template

```markdown
# Security Posture

## Last Updated
[Date]

## Hardening Status by Category

### Secrets Management
- [ ] No hardcoded secrets
- [ ] Environment variable validation
- [ ] Secret rotation support
Status: [hardened/partial/needs-work]

### Authentication & Authorization
- [ ] Auth flows reviewed
- [ ] Permission checks consistent
- [ ] Session handling secure
Status: [hardened/partial/needs-work]

### Input Validation
- [ ] All user input validated
- [ ] SQL injection prevented
- [ ] XSS prevented
Status: [hardened/partial/needs-work]

### Error Handling
- [ ] No sensitive data in errors
- [ ] Consistent error responses
Status: [hardened/partial/needs-work]

## Known Vulnerabilities
[Issues identified but not yet fixed]

## Priority Hardening Areas
1. [Most critical]
2. [Second priority]
```

#### TEMPORAL-FLOWS.md Template

```markdown
# Temporal Flows & Async Patterns

## Last Updated
[Date]

## Async Flows Identified

| Flow | Entry Point | Async Operations | Completion Signal |
|------|-------------|------------------|-------------------|
| [name] | [where it starts] | [what's async] | [how we know it's done] |

## Race Conditions
[Identified race conditions and their status]
- **Location**: description, mitigation status

## Timing Assumptions
[Implicit ordering or delay assumptions]

## Checkpoint Flows
[From progress-continuity-interruption-resilience]
- **Flow**: checkpoints, resume entrypoints, interruption handling

## Concurrency Concerns
[Shared state, locking, coordination patterns]
```

#### EXPERIENCE-AUDIT.md Template

```markdown
# Experience Architecture Audit

## Last Updated
[Date]

## Personas Identified

| Persona | Primary Job | Current Flow | Ideal Flow |
|---------|-------------|--------------|------------|
| [who] | [what they're trying to do] | [how it works now] | [how it should work] |

## Friction Analysis

### Mechanical Friction
[Too many clicks, slow operations]

### Cognitive Friction
[Confusing UI, unclear state]

### Discoverability Friction
[Features hard to find]

## Navigation Integrity
[From navigation-integrity-audit]
- Label→destination mismatches:
- Back/forward coherence issues:
- Deep link handling:

## Priority Improvements
1. [Highest impact]
2. [Second priority]
```

#### When to Create vs. Skip Files

| File | Create When | Skip When |
|------|-------------|-----------|
| SEAMS.md | Always - core internal doc | Never skip |
| PROBLEMS.md | Always - core internal doc | Never skip |
| PROGRESS.md | Always - core internal doc | Never skip |
| INVARIANTS.md | System has critical contracts | Simple CRUD with no invariants |
| ASSUMPTIONS.md | Code makes implicit assumptions | Well-typed, explicit code |
| ERROR-SEMANTICS.md | User-facing errors matter | Internal tooling only |
| SECURITY-POSTURE.md | Security is a concern | Internal-only, no auth |
| TEMPORAL-FLOWS.md | Async/concurrent operations | Purely synchronous code |
| COHERENCE-NOTES.md | React UI exists | No React UI |
| EXPERIENCE-AUDIT.md | User-facing scenario | Backend-only service |
