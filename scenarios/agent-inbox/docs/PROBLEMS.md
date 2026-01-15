# Known Problems & Blockers

## Open Issues

### React Error #310 - Intermittent Crash on Message Send (Investigated 2026-01-15)
**Severity:** Low (caught by ErrorBoundary, recoverable via retry)
**Symptoms:**
- React error #310 ("Too many re-renders") thrown when sending a message
- Error caught by `ChatContent` ErrorBoundary
- UI shows error fallback with retry button, does not white-screen crash

**Investigation Findings:**
- Deep audit of 20+ React files found no static code issues that would cause this
- All hooks follow proper discipline (no conditional hooks, no setState during render)
- useCompletion uses request ID tracking to prevent stale updates
- AbortController properly cancels in-flight requests
- Stable empty array constants used throughout

**Root Cause Assessment:**
- Likely an intermittent race condition during rapid state updates when:
  1. Message is sent
  2. Chat list refreshes
  3. Streaming response starts
  4. Multiple components update simultaneously
- Cannot reproduce reliably; error is transient

**Fixes Applied:**
- `TemplateSelector.tsx:255-258` - fixed inconsistent optional chaining
- `ChatView.tsx` - added granular ErrorBoundary wrappers around ChatHeader, AsyncOperationsPanel, MessageList, and MessageInput for better crash isolation (2026-01-15)

**Mitigation:**
- ErrorBoundary catches error and provides retry button
- User can retry message send, usually succeeds on second attempt
- Granular error boundaries now isolate crashes to specific sections (header, message list, or input) rather than the entire chat view

**Future Consideration:**
- If error becomes frequent, add React DevTools profiler analysis in dev mode
- Consider adding error telemetry to track frequency and stack traces

### OpenRouter Rate Limits
- Need graceful handling when rate limits are hit
- Consider request queuing or user notification

## Deferred Ideas

### Multi-user Support
- Current design is single-user
- Would need auth system and per-user data isolation
- Consider for P1/P2 iteration

### Local Model Fallback
- If OpenRouter is unavailable, could fallback to Ollama for basic chat
- Not in current scope, but worth considering

### Mobile Responsive Design
- Current design is desktop-first
- Mobile layout would require significant UI restructuring
- Explicitly out of scope per PRD

## Resolved

(None yet - scenario just initialized)
