# Improver Assessment Phase: Audit Current State

## Purpose
**IMPROVERS FIX & ENHANCE EXISTING** - Assessment reveals truth vs documentation.

## Assessment Allocation: 20% of Total Effort

### Primary Mission: Find the Lies
1. **Audit current state** - What REALLY works?
2. **Identify breakage** - What's broken despite claims?
3. **Plan fixes** - What needs immediate attention?

### MANDATORY Testing (Trust Nothing)
```bash
# Test what PRD claims works
vrooli [resource/scenario] [name] develop  # Start it
curl localhost:$PORT/health                # Health check
./test.sh                                  # Run tests

# For EACH ✅ in PRD.md:
# Test it. If broken, uncheck it.
# Document: "BROKEN: [specific error]"
```

### Search for Known Issues
```bash
# Find existing problems documented
vrooli resource qdrant search "[target name] issue" docs
vrooli resource qdrant search-all "[target name] broken"
rg "TODO|FIXME|HACK|BUG" . --type-add 'code:*.{js,go,py}'

# Find past fix attempts
vrooli resource qdrant search "fixed [target name]" code
grep -r "workaround\|temporary\|hack" .
```

### Code Quality Audit
```bash
# Find technical debt
grep -r "hardcoded\|localhost\|3000" .  # Config issues
find . -size 0                           # Empty files
ls -la lib/ | grep -v "\.sh"            # Missing scripts
```

## Improver Assessment Output Template
```markdown
## Assessment Results
**Target**: [target name]
**Current PRD Completion**: X% (was Y%, adjusted for lies)

### False Checkmarks Found (Uncheck These)
- [ ] Feature X - BROKEN: [specific error]
- [ ] Feature Y - PARTIAL: [only Z works]
- [ ] Feature Z - MISSING: [not implemented]

### Actual Working Features
- [✓] Feature A - Tested and confirmed
- [✓] Feature B - Works as documented

### Priority Fixes Needed
1. [CRITICAL]: [What's completely broken]
2. [HIGH]: [What partially works]
3. [MEDIUM]: [What needs enhancement]
```

## Assessment Quality Gates

### Gate 1: Truth Detection (MUST PASS)
- [ ] Every ✅ in PRD tested
- [ ] False completions unchecked
- [ ] Real status documented
- [ ] No assumptions made

### Gate 2: Issue Identification (MUST PASS)
- [ ] Breaking issues found
- [ ] Root causes identified
- [ ] Fix complexity estimated
- [ ] Dependencies checked

### Gate 3: Fix Planning (MUST PASS)
- [ ] Fixes prioritized by impact
- [ ] Quick wins identified
- [ ] Complex fixes noted
- [ ] Time estimates realistic

## What Improvers DON'T Assess
❌ **DON'T assess market need** - It exists, we're improving it
❌ **DON'T assess uniqueness** - It's already built
❌ **DON'T assess templates** - We're modifying, not copying
❌ **DON'T assess revenue potential** - Focus on fixing

## Improver Assessment Mindset
Think: **"What claims are false?"**
NOT: "What could we build?"

Think: **"What's the smallest fix with biggest impact?"**
NOT: "How do we rebuild this?"

Think: **"How do we advance PRD checkboxes?"**
NOT: "How do we create new features?"

## Time Boxing
- Start & test: 5 minutes MAX
- PRD validation: 10 minutes MAX
- Issue search: 3 minutes MAX
- Documentation: 2 minutes MAX
- **TOTAL: 20% of task time**

Remember: **Improvers make things ACTUALLY WORK.** Assessment reveals the truth.