# ✅ Validation Gates

## Five MANDATORY Gates (ALL must pass)

### 1. Functional ⚙️
```bash
./manage.sh develop         # Starts
timeout 5 curl -sf http://localhost:${PORT}/health  # Returns 200
./manage.sh stop            # Stops <10s
```

### 2. Integration 🔗
```bash
# Test dependencies connect
# APIs remain compatible
# UI screenshots verified (if applicable)
```

### 3. Documentation 📚
- [ ] README: Overview, Usage, Troubleshooting
- [ ] PRD: Progress tracked
- [ ] Code: Complex logic commented

### 4. Testing 🧪
```bash
./test.sh  # All pass, no regressions
```

### 5. Memory 🧠
```bash
vrooli resource qdrant embeddings refresh --path .
```

## Generator-Specific Gates

### PRD Gate (50% effort)
- [ ] All sections complete
- [ ] 5+ P0 requirements
- [ ] Revenue model defined
- [ ] Technical specs documented

### Uniqueness Gate
- [ ] <20% overlap with existing
- [ ] Adds new value

### Scaffold Gate
- [ ] Structure created
- [ ] Health responds
- [ ] Basic tests pass

## Improver-Specific Gates

### PRD Accuracy (20% effort)
- [ ] Checkboxes verified
- [ ] False ✅ unchecked
- [ ] Progress % updated

### No-Regression Gate
- [ ] Previous features work
- [ ] Performance maintained
- [ ] UI not broken (screenshots)

### Progress Gate
- [ ] 1+ PRD item advanced
- [ ] Measurable improvement
- [ ] Focused change

## Execution Order
Functional → Integration → Documentation → Testing → Memory

**FAIL = STOP** - Fix before proceeding

## Quick Commands
```bash
# Full validation
./manage.sh develop && timeout 5 curl -sf http://localhost:${PORT}/health && ./test.sh

# UI validation (scenarios)
resource-browserless screenshot http://localhost:PORT
Read screenshot.png

# Memory update
vrooli resource qdrant embeddings refresh --path .
```