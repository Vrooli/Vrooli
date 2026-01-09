# Assumption Mapping & Hardening Report

This document catalogs the key assumptions in the playwright-driver codebase, their risk levels, and how they have been hardened.

## Executive Summary

The playwright-driver is a critical component that bridges the browser automation API with Playwright. During assumption mapping, we identified **11 key assumption categories** across the codebase. Each has been evaluated for risk and hardened where necessary.

## Assumption Categories

### 1. Session Start Request Validation

**Location:** `src/routes/session-start.ts`

**Original Assumption:** Request body contains valid `execution_id`, `workflow_id`, and `viewport` fields.

**Risk:** HIGH - Invalid input causes unclear errors or runtime crashes.

**Hardening Applied:**
- Added explicit validation for required fields (`execution_id`, `workflow_id`, `viewport`)
- Validates viewport has positive `width` and `height` numbers
- Validates `reuse_mode` against allowed values
- Returns clear `InvalidInstructionError` with field-specific hints

**Verification:** TypeScript compile-time + runtime validation

---

### 2. Session Run Instruction Structure

**Location:** `src/routes/session-run.ts`

**Original Assumption:** Instruction object has valid `type`, `index`, `node_id`, and `params`.

**Risk:** HIGH - Malformed instructions cause cryptic handler errors.

**Hardening Applied:**
- Added structural validation before dispatching to handlers
- Returns 400 with `INVALID_INSTRUCTION` code for missing/invalid fields
- Resets session phase to 'ready' on validation failure (prevents stuck state)

**Verification:** Early validation before any handler execution

---

### 3. Request Body Parsing

**Location:** `src/middleware/body-parser.ts`

**Original Assumption:** HTTP stream completes normally and body is valid JSON.

**Risk:** MEDIUM - Client disconnects, timeouts, or malformed JSON cause unhandled rejections.

**Hardening Applied:**
- Added `rejected` flag to prevent double-rejection
- Added `cleanup()` to remove all listeners
- Destroys stream on size limit violation
- Handles `aborted` event for client disconnects
- Handles `close` event for premature disconnection
- Logs size violations with actionable hints

**Verification:** All error paths tested; stream properly cleaned up

---

### 4. Navigation URL Validation

**Location:** `src/handlers/navigation.ts`

**Original Assumption:** URL parameter is valid and safe.

**Risk:** HIGH (Security) - Dangerous protocols (javascript:, file:) could be injected.

**Hardening Applied:**
- Added `validateNavigationUrl()` with protocol whitelist (`http:`, `https:`, `about:`, `data:`)
- Blocks dangerous schemes (`javascript:`, `file:`, `vbscript:`, `data:text/html`)
- Enforces URL length limit (8192 chars)
- Supports relative URLs (Playwright resolves them)

**Verification:** Returns `INVALID_URL` error code for blocked URLs

---

### 5. Concurrent Session Close

**Location:** `src/session/manager.ts`

**Original Assumption:** `closeSession()` is called once per session.

**Risk:** MEDIUM - Idle cleanup and explicit close may race, causing double-close errors.

**Hardening Applied:**
- Added `closingSessionIds` Set to track in-progress closes
- `closeSession()` checks set before proceeding
- Returns early (idempotent) if already closing
- Cleans up set in finally block

**Verification:** Concurrent calls are safely deduplicated

---

### 6. Configuration Parsing

**Location:** `src/config.ts`

**Original Assumption:** Environment variables contain valid numbers.

**Risk:** MEDIUM - Invalid values (e.g., `MAX_SESSIONS=abc`) cause NaN propagation.

**Hardening Applied:**
- Added `parseEnvInt()` helper that returns default on NaN
- Added `parseLogLevel()` and `parseLogFormat()` with validation
- Added min/max constraints to Zod schema for numeric fields
- Logs warnings for invalid config values

**Verification:** Invalid env vars gracefully fall back to defaults

---

### 7. Recording Script Re-injection

**Location:** `src/recording/controller.ts`

**Original Assumption:** Page is ready for script injection after 100ms delay.

**Risk:** MEDIUM - Fast navigations may lose events between pages.

**Hardening Applied:**
- Added exponential backoff retry (100ms, 200ms, 400ms)
- Maximum 3 attempts before giving up
- Ignores errors for closed/detached pages (expected during navigation)
- Checks recording state before each attempt (may have stopped)

**Verification:** Recording continues working across navigations

---

### 8. Screenshot Capture

**Location:** `src/telemetry/screenshot.ts`

**Original Assumption:** `page.viewportSize()` returns valid dimensions.

**Risk:** LOW - Null viewport causes width/height of 0 (non-fatal).

**Status:** Already hardened with `|| 0` fallbacks. No changes needed.

---

### 9. Selector Validation

**Location:** `src/recording/controller.ts` (executeClick, executeType, etc.)

**Original Assumption:** Selectors from recordings are valid.

**Risk:** MEDIUM - Invalid selectors cause action failures.

**Status:** Already validates via `validateSelector()` before use.

---

### 10. Handler Registry

**Location:** `src/handlers/registry.ts`

**Original Assumption:** All instruction types have registered handlers.

**Risk:** LOW - Returns `UNSUPPORTED_TYPE` error for unknown types.

**Status:** Already properly handled with clear error messages.

---

### 11. Zod Schema Validation

**Location:** Various handlers (NavigateParamsSchema, etc.)

**Original Assumption:** Request params match expected schema.

**Risk:** MEDIUM - Schema mismatches cause ZodError.

**Status:** Zod provides clear validation errors. Consider wrapping for consistent error format.

---

## Risk Matrix

| Assumption | Risk Level | Hardened | Test Coverage |
|------------|------------|----------|---------------|
| Session Start Validation | HIGH | YES | Manual |
| Instruction Structure | HIGH | YES | Manual |
| Body Parsing | MEDIUM | YES | Manual |
| URL Validation | HIGH | YES | Manual |
| Concurrent Close | MEDIUM | YES | Manual |
| Config Parsing | MEDIUM | YES | Manual |
| Recording Re-injection | MEDIUM | YES | Manual |
| Screenshot Capture | LOW | Already | Existing |
| Selector Validation | MEDIUM | Already | Existing |
| Handler Registry | LOW | Already | Existing |
| Zod Validation | MEDIUM | Partial | Existing |

## Recommendations for Future Work

1. **Add unit tests** for each hardened assumption to prevent regression
2. **Consider request ID tracking** for better error correlation
3. **Add rate limiting** on session creation to prevent resource exhaustion
4. **Monitor** hardening-related log messages to catch edge cases in production

## Files Modified

- `src/routes/session-start.ts` - Input validation
- `src/routes/session-run.ts` - Instruction validation
- `src/middleware/body-parser.ts` - Stream handling
- `src/handlers/navigation.ts` - URL security
- `src/session/manager.ts` - Concurrent close protection
- `src/config.ts` - Environment parsing
- `src/recording/controller.ts` - Navigation resilience
