# ✅ Resource Validation Gates

## Purpose
Ensure resources meet v2.0 contract requirements and function correctly before marking tasks complete.

## Five MANDATORY Gates (ALL must pass)

### 1. Functional ⚙️
Validate resource lifecycle and health.
**See: 'resource-testing-reference' section for all commands**

### 2. Integration 🔗
Test resource connections and dependencies.
**See: 'resource-testing-reference' section for integration testing**

### 3. Documentation 📚
Validate documentation completeness:
- [ ] **PRD.md**: Per 'prd-protocol' section requirements
- [ ] **README.md**: Overview, usage examples, configuration
- [ ] **lib/*.sh**: Core scripts properly commented
- [ ] **Test commands**: Each P0 requirement has test validation

### 4. Testing 🧪
Run comprehensive test suite.
**See: 'resource-testing-reference' section for test commands**

### 5. Memory 🧠
Update Qdrant knowledge base with implementation details.

## v2.0 Contract Compliance
**See: 'resource-testing-reference' section for structure validation and contract requirements**

## Generator-Specific Gates

### PRD Gate (50% effort for generators)
See 'prd-protocol' section for comprehensive PRD requirements.

### Uniqueness Gate
**See: 'resource-testing-reference' section for uniqueness validation**

### Scaffold Gate
- [ ] v2.0 directory structure created
- [ ] Health endpoint responds
- [ ] Basic CLI commands functional
- [ ] One P0 requirement demonstrably works

## Improver-Specific Gates

### PRD Accuracy (20% effort for improvers)
Test each claimed requirement and update per 'prd-protocol' section guidelines.

### No-Regression Gate
**See: 'resource-testing-reference' section for regression testing**

### Progress Gate
- [ ] At least 1 PRD requirement advanced (per 'prd-protocol' section)
- [ ] Measurable improvement documented
- [ ] Test commands prove the improvement

## Execution Order
1. **Functional** → Verify lifecycle works
2. **Integration** → Check dependencies
3. **Documentation** → Ensure clarity
4. **Testing** → Validate thoroughly  
5. **Memory** → Update knowledge

**FAIL = STOP** - Fix issues before proceeding

## Quick Reference
- **All test commands**: See 'resource-testing-reference' section
- **Common issues & solutions**: See 'resource-testing-reference' section
- **Performance requirements**: See 'resource-testing-reference' section

## Remember
- Use CLI commands, not direct script execution
- Follow v2.0 contract strictly
- Test everything you claim works
- Document test commands for validation