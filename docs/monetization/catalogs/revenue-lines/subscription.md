# Revenue Line: Subscription (the product)

> Offer Desk is authoritative for this revenue line's current lifecycle,
> owner, and activation record. This document retains the portfolio role,
> economics, and instrumentation judgment rather than a live line snapshot.

- **Revenue model:** monthly / annual recurring, per-tier, per-bundle
- **Cost structure:** see per-tier COGS in [FINANCIAL_MODEL.md](../FINANCIAL_MODEL.md)
- **Productization target:** this IS the product. No bridging; it's the destination all services lines aim toward.

## Role in the portfolio

Subscription is the destination. Every services line in this folder exists to validate a capability and ultimately convert its clients into subscribers. A services line that never generates subscription conversion is a failure regardless of its own revenue — see [REVENUE_LINES.md](../REVENUE_LINES.md) for the full discipline.

## Why no revisit trigger

Unlike services lines, subscription has no revisit trigger because it is the
intended product destination. Trigger-style discipline applies to activation
of candidate lines; the lifecycle record itself is maintained in Offer Desk.

## Instrumentation

Tracked separately from services revenue. `financial-tracker` reports:

- Subscriber count by tier × bundle
- MRR / ARR
- Churn rate
- Gross margin by tier
- Default-alive position (see [FINANCIAL_MODEL.md](../FINANCIAL_MODEL.md))

Sustained crossover (services > subs for 2+ consecutive months) is flagged to the morning vision walk as a services-trap warning signal.
