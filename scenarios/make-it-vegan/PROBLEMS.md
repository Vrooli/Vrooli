# Problems & Solutions Log

## Issues Discovered

### 1. API-UI Field Mismatch (RESOLVED)
**Issue**: The UI expected different field names than the API provided
- UI expected: `nonVeganIngredients`, `explanations`
- API provided: `nonVeganItems`, `reasons`

**Solution**: Updated API to provide both field sets for compatibility

### 2. Alternative Struct Type Mismatch (RESOLVED)
**Issue**: Compilation error due to missing Notes field and incorrect Rating type
- Alternative struct had `Rating int` but code used it as float64
- Code referenced non-existent `Notes` field

**Solution**: Updated Alternative struct with proper types and added Notes field

### 3. Dynamic Port Detection (RESOLVED)
**Issue**: UI couldn't find API due to dynamic port assignment
- Vrooli assigns random ports in ranges (API: 15000-19999, UI: 35000-39999)
- Static port references would fail

**Solution**: Implemented dynamic port detection in app.js and server.js injection

### 4. Standards Violations (PARTIALLY ADDRESSED)
**Issue**: Scenario auditor found 497 standards violations
- Most violations are formatting/style issues
- No critical security issues found

**Status**: Non-critical, can be addressed incrementally

### 5. n8n Workflow Integration (NOT CRITICAL)
**Issue**: n8n workflows fail to populate during setup
- Workflows don't affect core functionality
- API has full local implementation as fallback

**Status**: Working without n8n, optional enhancement

## Recommendations for Future Improvements

1. **Add Redis caching** for frequently checked ingredients (P1)
2. **Implement PostgreSQL persistence** for user preferences and custom ingredients (P1)
3. **Add nutritional insights** with protein/B12/iron calculations (P1)
4. **Create brand database** for specific product lookups (P1)
5. **Run gofumpt** to address Go formatting violations
6. **Add comprehensive test suite** with bats tests for all endpoints