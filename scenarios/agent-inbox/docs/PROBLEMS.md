# Known Problems & Blockers

## Open Issues

### Terminal Session Security
- PTY sessions need proper sandboxing before production use
- Consider using containers or resource limits (cgroups)
- Evaluate attack surface of allowing arbitrary command execution

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
