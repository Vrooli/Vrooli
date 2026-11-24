---
title: "Quick Start Guide"
description: "Start and validate Landing Page Business Suite"
category: "getting-started"
order: 1
audience: ["users", "developers"]
---

# Quick Start Guide

Use the scenario lifecycle to prepare, start, inspect, and stop Landing Page
Business Suite (LPBS). Its manifest owns required resources, component builds,
and dynamically assigned ports.

## Prerequisites

- Vrooli CLI installed and connected to the local Vrooli runtime.
- A local checkout of the Vrooli workspace.

The declared scenario contract documents lifecycle ownership and port discovery
([The Declared Scenario](../../../docs/architecture/the-declared-scenario.md)).
Use the toolchain and workspace setup instructions for your installed release
([Getting Started](../../../docs/scenarios/getting-started.md)). The LPBS
manifest declares PostgreSQL and AI Gateway as required dependencies; do not
start resources or component servers separately.

## Start LPBS

From `scenarios/landing-page-business-suite`:

```bash
make setup
make start
make status
```

The lifecycle prepares declared resources and schemas, builds the API and UI,
assigns ports, and checks readiness. To inspect processes and logs, use
`vrooli scenario status landing-page-business-suite` and `make logs`. Stop it
with `make stop`.

## Open the landing page

Resolve the allocated endpoints; LPBS does not use fixed ports:

```bash
vrooli scenario port landing-page-business-suite UI_PORT
vrooli scenario port landing-page-business-suite API_PORT
```

Open `http://localhost:<UI_PORT>/` for the public page and
`http://localhost:<UI_PORT>/admin` for the admin portal. The API health route
is `http://localhost:<API_PORT>/health`. Replace each placeholder with the
port returned by the CLI.

For local login, use the administrator configured through LPBS's local secret
surface. For production, provision the administrator credential through the
credential authority and set the canonical `PUBLIC_BASE_URL` through supported
configuration before startup. See [Security](reference/SECURITY.md) and
[Configuration](guides/CONFIGURATION_GUIDE.md). LPBS generates and witnesses
its session and encryption credentials; production requires an absolute HTTPS
public URL. Existing installations can keep `AUTH_MAGIC_LINK_BASE_URL` as a
compatibility fallback.

## Review and publish content

Open `/admin/presentation/control` or select an existing variant in
Presentation. Edit the configured product-hero block, save the draft, and
inspect its private root or app-detail preview. Resolve validation errors and
missing capability or asset qualification before choosing **Publish draft**;
saving alone does not change the public page. Review-fixture artwork is not a
released asset, and upcoming remote/device features must remain marked coming
soon. Follow the [Admin Guide](guides/ADMIN_GUIDE.md) for content and variant
workflows.

## Configure payments (optional)

Start in Stripe test mode. Configure payment credentials through the governed
Billing → Stripe workflow in `/admin/billing`; never put secret values in source
or docs. Follow [Configuration](guides/CONFIGURATION_GUIDE.md), [restricted
Stripe keys](reference/STRIPE_RESTRICTED_KEYS.md), [webhook setup](reference/STRIPE_WEBHOOKS.md),
and the [Payments API guide](reference/api/payments.md). After changing
configuration, run `make restart` and the deployment readiness check. It
reports missing fields and route readiness without returning secret values.

## Validate and troubleshoot

```bash
make test
make status
make logs
```

- If a declared dependency is unavailable, run `make restart`, then inspect
  status and logs. Lifecycle startup owns required resources.
- If a port does not respond, resolve it again with `vrooli scenario port`;
  do not assume a fixed port.
- For build failures, run `make setup` and `make test`, then inspect the
  scenario's Test Genie findings.
- For admin, content, or deployment details, use the owner guides below.

## Owner guides

- [Admin Guide](guides/ADMIN_GUIDE.md) — administration, content, variants,
  and analytics.
- [Content Guide](guides/CONTENT_GUIDE.md) — product content and copy.
- [Configuration Guide](guides/CONFIGURATION_GUIDE.md) — supported settings.
- [Architecture](concepts/ARCHITECTURE.md) and [Deployment](guides/DEPLOYMENT.md).
- [FAQ](guides/faq.md) and [Troubleshooting](guides/TROUBLESHOOTING.md).
