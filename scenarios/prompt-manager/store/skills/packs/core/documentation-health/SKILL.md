## Steer focus: Documentation Health

Prioritize **documentation quality, consistency, and bidirectional traceability** between code and documentation across this scenario.

Your goal is to ensure documentation remains accurate, discoverable, and tightly coupled to the code it describes, preventing drift, gaps, and duplication.

Do **not** change core business logic or introduce new features. All changes focus on documentation structure, references, and validation infrastructure.

Required reading:
- `prompt-manager skills read visited-tracker-tools`

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
- `prompt-manager skill read screaming-architecture-audit`

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
│   ├── TEMPORAL-FLOWS.md  # Async patterns, race conditions, workflow maturity (optional)
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
| Architecture decisions | docs/strategy/ or promoted docs/plans/ | ADR format |
| Scratch implementation plans | `vrooli plans add --stdin` | User-scoped plan storage |
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

```bash
knowledge-observatory docs audit {{TARGET}}
```

The audit checks:
- **Infrastructure**: manifest.json, required docs, file counts, misplaced/missing/extra
- **Code coverage**: files with exported symbols but no `// DOC:` references
- **Reference integrity**: broken `[CODE: ...]` references in documentation
- **Manifest registration**: orphaned docs not listed in manifest.json
- **Deduplication**: duplicate heading titles across doc files
- **PRD alignment**: operational targets (OT-*) without corresponding docs

Use `--json` for machine-readable output.

#### Red Flags Checklist

- [ ] Missing `docs/manifest.json` → Create manifest for navigation
- [ ] Files with 500+ lines and no DOC: references → Add documentation links
- [ ] Broken `[CODE: ...]` references → Update paths or remove stale references
- [ ] Orphaned docs not in manifest → Add to manifest or delete if obsolete
- [ ] Duplicate documentation titles → Consolidate or differentiate
- [ ] PRD operational targets without docs → Create implementation docs

---

### **7. Memory Management with Visited Tracker**

Use the `visited-tracker-tools` skill for tracking visited files, with LOCATION set to `scenarios/{{TARGET}}` and TAG set to `documentation-health`.

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

The `path:docs/internal/` directory serves as **persistent agent memory** - documents written by agents to share findings with future agents. These are NOT user-facing documentation.

Fetch templates and their purposes on demand via the knowledge-observatory CLI:

```bash
knowledge-observatory docs templates              # List types with purpose descriptions
knowledge-observatory docs template <type>         # Get template content
```

Available types: seams, problems, progress, invariants,
error-semantics, security-posture, temporal-flows, coherence-notes, experience-audit

#### When to Create vs. Skip Files

| File | Create When | Skip When |
|------|-------------|-----------|
| SEAMS.md | Always - core internal doc | Never skip |
| PROBLEMS.md | Always - core internal doc | Never skip |
| PROGRESS.md | Always - core internal doc | Never skip |
| INVARIANTS.md | System has critical contracts or cross-cutting rules; also tracks unenforced-but-relied-on rules in Gaps section (see `invariant-discovery-and-enforcement`) | Simple CRUD with no invariants |
| ERROR-SEMANTICS.md | User-facing errors matter | Internal tooling only |
| SECURITY-POSTURE.md | Security is a concern | Internal-only, no auth |
| TEMPORAL-FLOWS.md | Async/concurrent operations, lifecycle flows, workflow maturity/spec status | Purely synchronous code |
| COHERENCE-NOTES.md | React UI exists | No React UI |
| EXPERIENCE-AUDIT.md | User-facing scenario | Backend-only service |
