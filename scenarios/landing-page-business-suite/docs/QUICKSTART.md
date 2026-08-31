---
title: "Quick Start Guide"
description: "Get your first landing page running in 5 minutes"
category: "getting-started"
order: 1
audience: ["users", "developers"]
---

# Quick Start Guide

Get your landing page running in 5 minutes.

## Prerequisites

Before starting, ensure you have:

- [ ] Vrooli CLI installed (`vrooli --version`)
- [ ] PostgreSQL running (`resource-postgres status`)
- [ ] Go 1.21+ installed (`go version`)
- [ ] Node.js 18+ with pnpm (`pnpm --version`)

If Vrooli isn't set up yet:
initialize the workspace from its repository root using the setup procedure
documented by the installed Vrooli release. Run `vrooli help` to discover the
available lifecycle commands before continuing.

---

## Start Landing Page Business Suite

From the scenario directory, use the Vrooli lifecycle:

```bash
make start
make logs
```

The lifecycle starts the required local resources, assigns the API and UI
ports, and applies the authoritative domain schemas. Do not run the API binary
or a development script directly; that bypasses lifecycle health checks and
port ownership.

To inspect the active endpoints and health status:

```bash
vrooli scenario status landing-page-business-suite
```

To stop the scenario when finished:

```bash
make stop
```

---

## Accessing Your Landing Page

| Surface | URL | Purpose |
|---------|-----|---------|
| Public Landing | `http://localhost:${UI_PORT}/` | What visitors see |
| Admin Portal | `http://localhost:${UI_PORT}/admin` | Manage content |
| API Health | `http://localhost:${API_PORT}/health` | Service status |

Vrooli assigns `API_PORT` and `UI_PORT` at startup. For development access,
use the administrator credentials configured in the scenario's local secret
surface. For production, provision the administrator password through the
credential authority and configure `AUTH_MAGIC_LINK_BASE_URL` through the
supported configuration workflow before starting the scenario. LPBS generates
and witnesses its session and encryption credentials; production requires an
absolute HTTPS magic-link URL.

---

## First Steps in Admin Portal

### 1. Check Health

Navigate to `/admin` and log in. The dashboard shows:
- Quick navigation to Landing, Billing, Apps, and Users dashboards
- Variant and traffic allocation snapshot
- Stripe readiness indicators

### 2. Preview Your Landing Page

Click "Preview Landing" or visit `/` in a new tab.

### 3. Edit Hero Content

1. Go to **Customization**
2. Select the **Control** variant
3. Click **Hero** section
4. Edit the headline and CTA
5. Watch the live preview update
6. Click **Save**

### 4. View Analytics

Go to **Analytics** to see:
- Page views
- CTA clicks
- Conversion rates (once you have payments configured)

---

## Configuring Stripe (Optional)

To enable payments:

### 1. Get Stripe Keys

From [Stripe Dashboard](https://dashboard.stripe.com/apikeys):
- Publishable key: `pk_test_...`
   - Restricted key: `rk_test_...`

### 2. Configure in Admin

1. Go to **Billing → Stripe** (`/admin/billing`)
2. Enter your keys
3. Save

### 3. Set Up Webhooks

In Stripe Dashboard → Developers → Webhooks:
- Endpoint: `http://your-domain/api/v1/webhooks/stripe`
- Events: `checkout.session.completed`, `customer.subscription.*`, `invoice.*`

### 4. Test Payment

Use test card `4242 4242 4242 4242` with any future date and CVC.

---

## Creating Your First A/B Test

### 1. Create a Variant

1. Go to **Customization** (`/admin/customization`)
2. Click **Create New Variant**
3. Name it (e.g., "Holiday Special")
4. Set weight to num[target]:50 (splits traffic evenly with Control)

### 2. Customize the Variant

1. Select your new variant
2. Edit sections (try changing the hero headline)
3. Save changes

### 3. Test It

Add `?variant=holiday-special` to your URL to force that variant.

### 4. Monitor Results

Check **Analytics** after traffic comes in to compare conversion rates.

---

## Common Commands

```bash
# Start scenario (from generated scenario directory)
make start

# Stop scenario
make stop

# View logs
make logs

# Run tests
make test

# Check status
vrooli scenario status "<scenario-name>"

# Get allocated ports
vrooli scenario port "<scenario-name>" UI_PORT
vrooli scenario port "<scenario-name>" API_PORT
```

---

## Troubleshooting Quick Fixes

### "Database connection failed"
```bash
resource-postgres start
```

### "Port already in use"
```bash
make stop
# Wait a few seconds
make start
```

### "Go build failed"
```bash
cd api
go mod tidy
go build -o landing-api .
```

### Can't access admin portal
- Ensure you're at `/admin` (not `/admin/`)
- Clear browser cookies
- Check API is running: `curl http://localhost:8080/health`

---

## Next Steps

| Goal | Document |
|------|----------|
| Manage content effectively | [Admin Guide](guides/ADMIN_GUIDE.md) |
| Write converting copy | [Content Guide](guides/CONTENT_GUIDE.md) |
| Understand the architecture | [Architecture](concepts/ARCHITECTURE.md) |
| Deploy to production | [Deployment Guide](guides/DEPLOYMENT.md) |
| Integrate with Stripe | [Payments API](reference/api/payments.md) |

---

## Getting Help

1. Check [FAQ](guides/faq.md) for common questions
2. Review [Troubleshooting](guides/TROUBLESHOOTING.md) for specific issues
3. Run `vrooli help` for CLI assistance
