# Event ID Format

## Structure

All events in vrooli-events use a structured, hierarchical ID format:

```
{scenario}.{domain}.{action}.{version}
```

| Segment | Description | Example |
|---------|-------------|---------|
| scenario | Source scenario slug | `swarm-manager` |
| domain | Functional area within the scenario | `backlog`, `release`, `review` |
| action | What happened | `item-completed`, `approved`, `failed` |
| version | Schema version of the payload | `v1` |

## Examples

| Event ID | Meaning |
|----------|---------|
| `swarm-manager.backlog.item-completed.v1` | A backlog item finished processing |
| `swarm-manager.backlog.needs-attention.v1` | A backlog item requires user input |
| `deployment-manager.release.approved.v1` | A deployment release was approved |
| `git-control-tower.review.completed.v1` | A code review finished |
| `agent-manager.run.started.v1` | An agent execution run started |
| `vrooli-events.policy.violation.v1` | A policy rule denied a request |
| `vrooli-events.discovery.resolve.v1` | A scenario resolved another's port (auto-emitted) |

## Glob Pattern Matching

Subscriptions and SSE filters use segment-aware glob patterns:

### Single-segment wildcard: `*`

Matches exactly one segment.

| Pattern | Matches | Does Not Match |
|---------|---------|----------------|
| `swarm-manager.*.created.v1` | `swarm-manager.backlog.created.v1`, `swarm-manager.capture.created.v1` | `swarm-manager.backlog.item.created.v1` (4 segments vs 3 in pattern) |
| `*.backlog.*.v1` | `swarm-manager.backlog.created.v1` | `backlog.created.v1` (too few segments) |

### Multi-segment wildcard: `**`

Matches one or more segments.

| Pattern | Matches | Does Not Match |
|---------|---------|----------------|
| `swarm-manager.**` | `swarm-manager.backlog.created.v1`, `swarm-manager.run.started.v1` | `agent-manager.run.started.v1` (different prefix) |
| `**.completed.v1` | `swarm-manager.backlog.item-completed.v1`, `git-control-tower.review.completed.v1` | `swarm-manager.backlog.created.v1` (different action) |
| `**` | Everything | Nothing (matches all) |

### Empty pattern

An empty or omitted pattern matches all events.

## Conventions

### Action naming

Use past tense for completed actions, present tense for ongoing states:

| Good | Avoid |
|------|-------|
| `item-completed` | `complete-item` |
| `needs-attention` | `attention-needed` |
| `run-started` | `starting-run` |
| `policy-violated` | `violation` |

### Version bumping

Bump the version segment when the payload schema changes in a breaking way:

- `v1` → `v2`: Payload field renamed, removed, or type changed
- Keep `v1`: New optional field added to payload

Subscribers to `swarm-manager.backlog.item-completed.*` receive both `v1` and `v2` events.

### System events

Events emitted by vrooli-events itself use `vrooli-events` as the scenario segment:

- `vrooli-events.policy.violation.v1` — A policy rule denied a request
- `vrooli-events.policy.updated.v1` — A policy rule was created/updated/deleted
- `vrooli-events.discovery.resolve.v1` — A scenario resolved another's port
- `vrooli-events.subscription.delivery-failed.v1` — A webhook delivery failed
