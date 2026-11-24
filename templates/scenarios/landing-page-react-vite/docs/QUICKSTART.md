---
title: "Quick Start Guide"
description: "Generate and run a landing-page scenario"
category: "getting-started"
order: 1
audience: ["users", "developers"]
---

# Quick Start Guide

Generate a landing-page scenario, then use its lifecycle commands. The scenario
manifest owns its dependencies, component builds, and ports.

## Prerequisites

- Vrooli CLI installed and on `PATH`.
- Template Manager connected to the local Vrooli instance.

The generated scenario declares its language runtimes and dependencies in
`.vrooli/service.json`. You do not need to start PostgreSQL, build API binaries,
or assign ports by hand.

## 1. Generate a scenario

List available scenario templates, then generate this one:

```bash
template-manager registry list --kind scenario
template-manager generate landing-page-react-vite \
  --id <your-slug> \
  --display-name "<Your Landing Page>" \
  --description "<Brief description>"
```

Template Manager creates the scenario under `scenarios/<your-slug>/`. Complete
any orientation steps declared by that generated scenario.

## 2. Start the scenario

From the generated scenario directory:

```bash
make setup
make start
make status
```

The lifecycle prepares the declared resources, builds the API and UI, assigns
ports, and checks readiness. See [The Declared Scenario](../../../../docs/architecture/the-declared-scenario.md)
for the shared lifecycle contract. Use the scenario's Make targets or
`vrooli scenario` commands for its processes; do not start a component binary
or development server directly.

## 3. Open the landing page

Get the allocated UI and API ports from the Vrooli CLI:

```bash
vrooli scenario port <your-slug> UI_PORT
vrooli scenario port <your-slug> API_PORT
```

Open `http://localhost:<UI_PORT>/` for the public landing page and
`http://localhost:<UI_PORT>/admin` for the admin portal, replacing each
placeholder with the port returned by the command. The API health route is
`http://localhost:<API_PORT>/health`.

For the template's local demo seed, the administrator is `admin@localhost`
with password `admin`. Replace demo credentials through the governed
credential flow before any non-demo deployment; see [Security](SECURITY.md).

## 4. Continue in the owner guides

- [Admin Portal Guide](ADMIN_GUIDE.md) — edit content, manage variants, and
  review analytics.
- [Content Guide](CONTENT_GUIDE.md) — replace seeded copy and product details.
- [Configuration Guide](CONFIGURATION_GUIDE.md) — configure the generated
  scenario's supported settings.
- [Payments API](api/payments.md) — payment routes, test mode, and webhook
  behavior. Never put credentials in source or documentation.

Stripe setup is optional. Use test mode first and follow the configuration and
security guides before enabling payments.

## 5. Validate and stop

```bash
make test
make stop
```

The generated scenario's Test Genie manifest owns its validation phases. Use
`make logs`, `make status`, and
`vrooli scenario status <your-slug>` to inspect lifecycle state.

## Troubleshooting

- **A declared dependency is unavailable:** run `make restart`, then inspect
  `make status` and `make logs`. Lifecycle startup owns required resources.
- **A port does not respond:** resolve it again with `vrooli scenario port`;
  do not assume a fixed port.
- **A component build fails:** run `make setup` and `make test`, then inspect
  the generated scenario's Test Genie findings.
- **Admin login or payments need configuration:** use the
  [Admin Portal Guide](ADMIN_GUIDE.md) and [Configuration Guide](CONFIGURATION_GUIDE.md).
