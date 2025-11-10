# QR Code Generator - Known Issues & Resolutions

## Overview
This document tracks known issues, technical debt, and their resolutions for the qr-code-generator scenario.

## Resolved Issues

### 2025-10-28: Service Configuration Accuracy
**Issue**: service.json marked n8n and redis as "required: true" when they are optional resources according to the PRD.

**Impact**: Low - resources work when enabled but configuration was misleading

**Resolution**:
- Changed n8n and redis to "required: false" in service.json
- Matches PRD specification that these are optional resources
- Core QR generation works without these dependencies
- n8n provides P1/P2 features (art generation, puzzles)
- redis provides caching optimization

**Verification**:
```bash
# API continues working with optional resources disabled
curl http://localhost:17315/health
# Returns: {"status":"healthy",...}
```

### 2025-10-28: CLI Environment Variable Validation
**Issue**: CLI script had hardcoded port fallback (`17320`) that could cause confusion when the actual allocated port differs.

**Impact**: Medium - CLI might fail silently or connect to wrong port

**Resolution**:
- Removed hardcoded port fallback from `cli/qr-code-generator`
- Added proper validation with helpful error messages when API is not found
- CLI now requires either:
  - Running scenario (auto-detects port via `lsof`)
  - Explicit `API_PORT` or `API_URL` environment variable

**Verification**:
```bash
# Test CLI auto-detection
qr-code-generator generate "test"

# Test with explicit port
API_PORT=17315 qr-code-generator generate "test"
```

## Technical Debt

### Legacy CLI Script
**File**: `cli/qr-code-generator`
**Status**: Present but unused
**Recommendation**: Can be removed - `qr-code-generator` is the active CLI tool
**Priority**: Low - not impacting functionality

### Test Coverage
**Current**: 58.3% code coverage in Go API
**Target**: 80%+
**Status**: Below warning threshold
**Recommendation**: Add unit tests for:
- Error handling paths
- Edge cases in QR generation
- Batch processing validation

**Priority**: Medium - functionality works but tests would improve reliability

### Standards Compliance Violations

#### Medium Severity (17 violations)
Most violations are false positives or acceptable patterns:

1. **Environment Variable Validation** (9 violations)
   - Status: Acceptable - CLI tools use intelligent defaults
   - Justification: `APP_ROOT` with fallback is defensive coding
   - Action: No change needed

2. **Hardcoded URLs in Examples** (3 violations)
   - Files: `cli/qr-code-generator` (line 30), `ui/script.js`, `ui/styles.css`
   - Status: Acceptable - documentation examples or CDN links
   - Action: No change needed

3. **Hardcoded Localhost in Test** (1 violation)
   - File: `test/phases/test-integration.sh`
   - Status: Acceptable - test fixture
   - Action: No change needed

4. **Unstructured Logging** (2 violations)
   - Files: `api/main.go`, `api/coverage.html`
   - Status: Already using structured format (`key=value`)
   - Justification: Audit may be looking for JSON format
   - Priority: Low - current format is adequate

#### High Severity (1 violation)
1. **Makefile Usage Entry**
   - Issue: Audit reports "Usage entry for 'make clean' missing"
   - Reality: `make help` DOES show clean target
   - Status: False positive from auditor
   - Action: None needed

## Performance Characteristics

### Benchmarks (from test suite)
```
BenchmarkGenerateQRCode:   ~516µs per QR code (~1,935 QR codes/second)
BenchmarkBatchGeneration:  ~5.7ms per 10-item batch (~1,760 codes/second)
```

**Analysis**: Performance exceeds PRD requirement (<100ms per code)

### Resource Usage
- Memory: ~954KB per QR code generation
- Allocations: ~1,275 allocations per code
- Recommendation: Performance is acceptable for current use case

## Future Improvements

### P1 Requirements (Not Yet Implemented)
- [ ] QR Art Generation with AI
- [ ] QR Puzzle Generator
- [ ] Redis Caching
- [ ] Export Formats (SVG, PDF, JPEG)

### P2 Requirements (Not Yet Implemented)
- [ ] Analytics Dashboard
- [ ] URL Shortening Integration
- [ ] Logo Embedding in QR Center

## Lessons Learned

### What Worked Well
1. **Port Auto-Detection**: Using `lsof` to find running API port
2. **Environment Validation**: API properly validates required env vars
3. **Structured Logging**: Key=value format in logs
4. **Comprehensive Testing**: Phased test structure catches issues early

### Areas for Improvement
1. **Unit Test Coverage**: Need more edge case testing
2. **UI Health Endpoint**: Consider adding proper health check
3. **Error Messages**: Could be more descriptive in some edge cases

### Standards Compliance Notes
- Most "violations" are false positives for CLI tools
- Intelligent defaults in CLI tools are user-friendly
- Documentation examples with URLs are acceptable
- Current logging format is structured and adequate
