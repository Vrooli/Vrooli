# Seeding Guidelines

## Core Principle
**Generators run ONCE.** Create the permanent foundation for all future work.

## Time Allocation
- **40% Research** - Prevent duplicates, find patterns
- **40% PRD** - Document complete vision
- **20% Scaffold** - Minimal working structure

## Pre-Seeding Checklist
### Must Pass ALL:
- [ ] **Unique**: <20% overlap with existing
- [ ] **Valuable**: >$10K potential value
- [ ] **Feasible**: Resources available
- [ ] **Integrated**: Benefits ecosystem

If any fail → STOP or PIVOT

## Seeding Requirements

### Research (5 searches minimum)
```bash
vrooli resource qdrant search "[name] implementation"
vrooli resource qdrant search "[category] patterns"
vrooli resource qdrant search "[name] issues problems"
grep -r "[name]" /home/matthalloran8/Vrooli
# Document findings in PRD
```

### PRD Excellence
```markdown
Required sections:
- Executive Summary (problem/solution)
- P0 Requirements (5+ with test commands)
- P1 Requirements (3+ nice-to-haves)
- Technical Specifications
- Success Metrics (measurable)
- Revenue Model (how it makes money)
```

### Minimal Scaffold
```
/[name]/
  ├── PRD.md           # Complete vision
  ├── README.md        # Setup instructions
  ├── manage.sh        # Lifecycle commands
  ├── lib/            # For resources
  │   ├── setup.sh
  │   ├── develop.sh
  │   ├── health.sh
  │   └── test.sh
  ├── api/            # For scenarios
  │   └── main.[ext]  # Health endpoint
  └── tests/
      └── test.sh     # Basic validation
```

### Validation Gates
```bash
# Must pass before marking complete:
./manage.sh develop              # Starts
timeout 5 curl -sf http://localhost:${PORT}/health  # Responds
./test.sh                        # Passes
vrooli resource qdrant search "[name]"  # Indexed
```

## DO's and DON'Ts

### DO ✅
- Research exhaustively
- Document complete vision
- Start minimal
- Follow patterns
- Test basics

### DON'T ❌
- Duplicate existing (research prevents)
- Over-build (improvers enhance)
- Under-document (PRD critical)
- Break patterns (consistency matters)
- Skip validation (must work)

## Example Seed
```bash
# Good seed:
scenarios/invoice-manager/
  ├── PRD.md (2000+ words, complete vision)
  ├── api/main.go (health endpoint only)
  └── tests/test.sh (verifies health)

# Bad seed:
scenarios/another-chat/
  ├── PRD.md (200 words, vague)
  ├── Full implementation (10K lines)
  └── No tests
```

## Remember
- Quality > Quantity
- Foundation > Features
- Vision > Implementation