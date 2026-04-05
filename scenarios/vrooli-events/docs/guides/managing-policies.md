# Managing Policies

Policy rules govern inter-scenario communication: who can call whom, how often, and what happens when things fail.

## Policy Types

### Access Control

Allow or deny calls between scenario pairs.

```bash
# Deny a specific scenario from calling agent-manager
vrooli-events policy create --type access \
  --source "untrusted-scenario" \
  --target "agent-manager" \
  --effect deny \
  --priority 10

# Allow everything from swarm-manager to agent-manager (explicit allow)
vrooli-events policy create --type access \
  --source "swarm-manager" \
  --target "agent-manager" \
  --effect allow \
  --priority 5
```

**Priority**: Higher priority rules win when multiple rules match the same call. Use this for "deny all except..." patterns:

```bash
# Deny all calls to sensitive-service
vrooli-events policy create --type access \
  --source "*" --target "sensitive-service" --effect deny --priority 1

# But allow swarm-manager
vrooli-events policy create --type access \
  --source "swarm-manager" --target "sensitive-service" --effect allow --priority 10
```

### Rate Limiting

Cap request rates between scenario pairs using sliding windows.

```bash
# agent-manager can receive max 100 requests/minute from any single scenario
vrooli-events policy create --type rate_limit \
  --source "*" \
  --target "agent-manager" \
  --max-requests 100 \
  --window 60

# Allow short bursts up to 20 above the limit
vrooli-events policy create --type rate_limit \
  --source "*" \
  --target "agent-manager" \
  --max-requests 100 \
  --window 60 \
  --burst 20
```

When a sender hits the rate limit, the discovery package returns a `RateLimitExceeded` error with a `retry_after` hint.

### Circuit Breaking

Automatically stop calling failing scenarios to prevent cascade failures.

```bash
# If calls to flaky-service fail 5 times in 60 seconds, stop for 30 seconds
vrooli-events policy create --type circuit_breaker \
  --source "*" \
  --target "flaky-service" \
  --failure-threshold 5 \
  --window 60 \
  --cooldown 30
```

**Circuit breaker lifecycle**:
1. **Closed** (normal): Calls proceed. Failures counted.
2. **Open** (tripped): All calls denied immediately. Timer counting down.
3. **Half-Open** (probing): One call allowed through. If it succeeds, return to Closed. If it fails, return to Open.

### Manual Override

Force a circuit breaker into a specific state:

```bash
# Force closed (resume traffic despite failures)
vrooli-events policy override --id <rule-id> --state closed

# Force open (block traffic for maintenance)
vrooli-events policy override --id <rule-id> --state open --ttl 1800
```

Overrides expire after TTL (default: 1 hour) and revert to automatic behavior.

## How Enforcement Works

### Sender Side (EmittingResolver)

Before making a call, the discovery package checks its local policy cache:

1. Is the circuit breaker open? → Return `CircuitOpenError` immediately
2. Is the call denied by access control? → Return `PolicyDeniedError` immediately
3. Is the rate limit exceeded? → Return `RateLimitExceeded` with `retry_after`
4. Proceed with the call

**No network round-trip to vrooli-events** — all checks are against the local cache.

### Receiver Side (PolicyMiddleware)

When a request arrives, the middleware checks:

1. Extract `X-Source-Scenario` header
2. Evaluate receiver-side policy cache
3. If denied → Return 403 with `PolicyDeniedError` JSON body
4. If allowed → Pass to handler

### Cache Freshness

Both caches subscribe to vrooli-events SSE policy push channel. When you create, update, or delete a rule, all connected scenarios receive the update within milliseconds.

If vrooli-events goes down:
- Last-known policy remains in effect
- Configurable per-rule: `fail-open` (allow when cache is stale) or `fail-closed` (deny when cache is stale)
- Global default: `fail-open` (Vrooli scenarios trust each other — policy is governance, not hard security)

## Viewing Violations

```bash
# Recent violations
vrooli-events policy violations --since 1h

# Violations for a specific target
vrooli-events policy violations --target "agent-manager" --since 24h

# JSON output for piping
vrooli-events policy violations --since 1h --json
```

Each violation records: timestamp, source, target, endpoint, rule that triggered, and reason.

## Best Practices

1. **Start with monitoring, not blocking**: Use vrooli-events analytics to understand call patterns before creating deny rules
2. **Use rate limits before circuit breakers**: Rate limits are soft (queue and retry); circuit breakers are hard (immediate denial)
3. **Set reasonable thresholds**: Circuit breaker failure_threshold=5 and cooldown=30s is a good starting point
4. **Use glob patterns for broad rules**: `--source "*" --target "sensitive-service"` applies to all callers
5. **Use priority for exceptions**: Low-priority deny-all + high-priority specific allows is a clean pattern
