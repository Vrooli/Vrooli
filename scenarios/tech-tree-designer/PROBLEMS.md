# Tech Tree Designer - Known Issues & Improvement Opportunities

## 📊 Audit Status (2025-10-20)

### Security Improvements ✅
**Before**: 2 vulnerabilities (1 critical, 1 high)
**After**: 0 vulnerabilities

**Fixed Issues:**
- ✅ Hardcoded database password removed (`api/main.go:162`)
- ✅ CORS wildcard replaced with explicit origin validation (`api/main.go:231`)
- ✅ Port defaults removed - now requires `API_PORT` environment variable

### Standards Improvements 🟡
**Before**: 65 violations (4 critical, 17 high, 44 medium)
**After**: 52 violations (0 critical, 8 high, 44 medium)

**Fixed Issues:**
- ✅ Added missing test phases: `test-dependencies.sh`, `test-structure.sh`
- ✅ Added test runner: `test/run-tests.sh`
- ✅ Fixed Makefile: added `start`, `fmt`, `fmt-go`, `fmt-ui`, `lint`, `lint-go`, `lint-ui` targets
- ✅ Fixed service.json binary check path: `api/tech-tree-designer-api`
- ✅ Database password validation added with fail-fast pattern

## 🔴 High Priority Remaining Issues

### 1. Environment Variable Validation (8 high-severity)
**Impact**: Medium
**Effort**: Low
**Files Affected**: `api/main.go`, `cli/tech-tree-designer`, `cli/install.sh`, `ui/vite.config.js`

**Issue**: Many environment variables lack proper validation and fail-fast checks.

**Affected Variables:**
- `VROOLI_LIFECYCLE_MANAGED` (api/main.go:208)
- UI_PORT defaults in CLI and vite.config.js
- Shell color variables in CLI (RED, GREEN, YELLOW, BLUE, CYAN, NC, PURPLE)
- Shell logic variables (API_VERSION, ANALYSIS, RECOMMENDATIONS, SECTORS, etc.)

**Recommendation**:
- Add validation for critical environment variables
- Use fail-fast pattern for required variables
- Shell color variables are standard practice and can be ignored

---

## 🟡 Medium Priority Remaining Issues

### 2. Unstructured Logging (3 medium-severity)
**Impact**: Low
**Effort**: Medium
**Files Affected**: `api/main.go:200, 278, 279`

**Issue**: Using `log.Printf()` instead of structured logging.

**Example:**
```go
log.Printf("🚀 Tech Tree Designer API starting on port %s", port)
```

**Recommendation**:
- Introduce structured logging library (e.g., `zerolog`, `zap`)
- Replace `log.Printf` with structured logger calls
- Include context fields for better observability

### 3. Hardcoded Port Fallbacks (Multiple medium-severity)
**Impact**: Low
**Effort**: Low
**Files Affected**: `cli/tech-tree-designer:9, 486`, `ui/vite.config.js:8`

**Issue**: CLI and UI still have hardcoded port fallbacks.

**Recommendation**:
- Remove fallback values in CLI script
- Require explicit port configuration in all components

### 4. Hardcoded URLs in UI (3 medium-severity)
**Impact**: Very Low
**Effort**: Very Low
**Files Affected**: `ui/index.html:10-12`

**Issue**: Google Fonts CDN URLs hardcoded in HTML.

**Example:**
```html
<link href="https://fonts.googleapis.com/css2?family=Inter..." />
```

**Recommendation**:
- These are standard CDN references and safe to keep
- Alternative: Self-host fonts if offline capability needed
- **Priority**: Very low - industry standard practice

---

## ✅ PRD Validation Needed

### P0 Requirements Status
**None of the P0 requirements are currently checked as complete in the PRD.**

The following need validation before marking as complete:

1. ❓ Import/export tech tree via Graphviz DOT
2. ❓ Semantic zoom + filtering UI
3. ❓ Automated scenario/resource mapping
4. ❓ Maturity tracking per node
5. ❓ Dependency/centrality engine with recommendations API
6. ❓ Decision dashboard with reasoning
7. ❓ REST + CLI parity
8. ❓ Test suite with mocks/fixtures only

**Action Required**:
- Test each P0 requirement systematically
- Document test commands proving functionality
- Update PRD checkboxes based on actual verification

---

## 🔧 Technical Debt

### 1. Test Coverage
**Status**: Partial test infrastructure exists
**Issue**: Tests exist but coverage unknown

**Test Files Present:**
- ✅ `test-unit.sh`
- ✅ `test-integration.sh`
- ✅ `test-business.sh`
- ✅ `test-performance.sh`
- ✅ `test-dependencies.sh` (newly added)
- ✅ `test-structure.sh` (newly added)
- ✅ `run-tests.sh` (newly added)

**Recommendation**:
- Run full test suite to verify all tests pass
- Measure actual test coverage
- Add tests for uncovered critical paths

### 2. CLI Completeness
**Status**: CLI exists with comprehensive command structure
**Issue**: API endpoint coverage unknown

**Recommendation**:
- Verify all CLI commands successfully call corresponding API endpoints
- Test error handling and validation
- Add CLI integration tests

### 3. UI Implementation
**Status**: UI skeleton exists (React/Vite)
**Issue**: Functionality completeness unknown

**Recommendation**:
- Test UI against running API
- Verify all visualization features work
- Add UI automation tests

---

## 📋 Future Enhancements

### 1. Graph Studio Integration
**Priority**: High
**Complexity**: Medium

The PRD specifies Graph Studio as the primary editing surface for tech tree DOT files. Currently this integration may not be complete.

**Tasks**:
- Verify Graph Studio file watching
- Test DOT import/export workflow
- Ensure real-time synchronization

### 2. AI Strategic Analysis
**Priority**: Medium
**Complexity**: High

Ollama integration exists but AI-powered analysis features need validation:
- Path optimization recommendations
- Bottleneck identification
- Impact analysis
- Timeline projections

### 3. What-If Sandbox
**Priority**: Medium (P1 requirement)
**Complexity**: Medium

Hypothetical scenario testing capability for strategic planning.

---

## 🎯 Improvement Roadmap

### Phase 1: Validation & Documentation (Current)
- [x] Fix critical security issues
- [x] Fix critical standards violations
- [x] Add missing test infrastructure
- [ ] Validate P0 requirements
- [ ] Update PRD with accurate checkboxes

### Phase 2: Environment & Configuration
- [ ] Add validation for critical environment variables
- [ ] Remove remaining port fallbacks
- [ ] Document required environment setup

### Phase 3: Observability
- [ ] Implement structured logging
- [ ] Add metrics/telemetry
- [ ] Improve error messages

### Phase 4: Feature Completion
- [ ] Complete P0 requirements
- [ ] Implement P1 requirements
- [ ] Add comprehensive integration tests

---

## 📚 References

- **Baseline Audit**: `/tmp/tech-tree-designer_baseline_audit.json`
- **Post-Improvement Audit**: `/tmp/tech-tree-designer_post_audit.json`
- **PRD**: `PRD.md`
- **Service Contract**: `.vrooli/service.json`

---

**Last Updated**: 2025-10-20
**Next Review**: After P0 validation complete
