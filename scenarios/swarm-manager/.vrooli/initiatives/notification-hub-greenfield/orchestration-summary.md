# Meta-Orchestrator Summary: notification-hub-greenfield

## Source
Planning session covering the greenfield rewrite of notification-hub as an event-driven notification microservice.

## Decisions Made
- Greenfield rewrite, NOT evolution of existing notification-hub (old one uses n8n, Postgres/Redis, lacks proto types)
- Extract useful domain concepts from legacy hub before archiving it
- Multi-tenant from day one (other apps will need this)
- All delivery channels from day one: push (ntfy), email (SMTP), SMS, webhook
- ntfy as push backbone: self-hosted Docker container, iOS app from App Store, APNs relay via upstream-base-url
- Message content stays on local server (only content-free wake-up signal goes through ntfy.sh cloud)
- Event-driven via vrooli-events subscriptions (no modifications to source scenarios needed)
- Users configure notification subscriptions using glob patterns on event IDs
- Storage: SQLite + file-based (no Postgres, no Redis) for portability
- Proto contract in packages/proto/schemas/notification-hub/v1/
- Priority levels: low, normal, high, urgent (mapped to APNs 5/10, FCM normal/high, Web Push RFC 8030 urgency)

## Domain Concepts From Legacy Hub (to preserve)
- Multi-tenant profile system with isolated API keys
- Contact/device management with per-channel preferences and verification status
- Template system with variable substitution and channel-specific content
- Rate limiting: per-profile, per-channel, hourly/daily/monthly caps, cost-based limits
- Notification lifecycle: pending -> queued -> sending -> sent -> delivered/failed
- Unsubscribe scopes: all, per-channel, per-category (marketing vs transactional)
- DND modes and quiet hours (timezone-aware)
- Delivery tracking with per-channel logs and analytics

## Dependency Notes
- Signal extraction and ntfy resource setup can start immediately (no dependency on vrooli-events)
- Greenfield core depends on signal extraction + ntfy resource
- Event subscription wiring depends on BOTH notification-hub core AND vrooli-events core runtime
- Legacy archive can happen as soon as signal extraction completes

## Unresolved Questions Deferred To Workshop
- Exact SQLite schema design for notifications, contacts, templates, delivery logs
- Template format specifics (simple {{var}} substitution vs richer templating)
- SMS provider selection
- Email SMTP provider/configuration
- ntfy topic namespacing strategy
- How notification-hub UI exposes subscription management
- Whether quiet hours should be per-device or per-profile or both
- Webhook delivery retry policy and failure handling
