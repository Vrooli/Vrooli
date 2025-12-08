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
```bash
cd ~/Vrooli
./scripts/manage.sh setup --yes yes
```

---

## Option 1: Using Landing Manager (Recommended)

Landing Manager generates complete landing pages from this template.

```bash
# Generate a new landing page
vrooli scenario start landing-manager

# Follow the prompts to configure your landing page
# This creates a new scenario in scenarios/<your-slug>/
```

Once generated:
```bash
cd scenarios/<your-slug>
make start
```

---

## Option 2: Direct Template Usage

For development or customization of the template itself:

### Step 1: Start PostgreSQL

```bash
resource-postgres start
resource-postgres status  # Verify it's running
```

### Step 2: Build the API

```bash
cd scripts/scenarios/templates/landing-page-react-vite/api
go build -o landing-api .
```

### Step 3: Set Environment Variables

```bash
export API_PORT=8080
export UI_PORT=3000
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/landing_dev"
```

### Step 4: Initialize Database

```bash
psql $DATABASE_URL -f initialization/postgres/schema.sql
psql $DATABASE_URL -f initialization/postgres/seed.sql  # Optional demo data
```

### Step 5: Start the API

```bash
./landing-api
```

### Step 6: Start the UI (new terminal)

```bash
cd scripts/scenarios/templates/landing-page-react-vite/ui
pnpm install
pnpm run dev
```

---

## Accessing Your Landing Page

| Surface | URL | Purpose |
|---------|-----|---------|
| Public Landing | `http://localhost:3000/` | What visitors see |
| Admin Portal | `http://localhost:3000/admin` | Manage content |
| API Health | `http://localhost:8080/health` | Service status |

### Default Admin Credentials

```
Email: admin@localhost
Password: admin123
```

**Change these immediately in production!**

---

## First Steps in Admin Portal

### 1. Check Health

Navigate to `/admin` and log in. The dashboard shows:
- Variant status
- Recent analytics
- Quick actions

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

1. Go to **Customization → Stripe Settings**
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

1. Go to **Customization → Variants**
2. Click **Create New Variant**
3. Name it (e.g., "Holiday Special")
4. Set weight to 50 (splits traffic evenly with Control)

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
vrooli scenario status <slug>

# Get allocated ports
vrooli scenario port <slug> UI_PORT
vrooli scenario port <slug> API_PORT
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
| Manage content effectively | [Admin Guide](ADMIN_GUIDE.md) |
| Write converting copy | [Content Guide](CONTENT_GUIDE.md) |
| Understand the architecture | [Architecture](ARCHITECTURE.md) |
| Deploy to production | [Deployment Guide](DEPLOYMENT.md) |
| Integrate with Stripe | [Payments API](api/payments.md) |

---

## Getting Help

1. Check [FAQ](FAQ.md) for common questions
2. Review [Troubleshooting](TROUBLESHOOTING.md) for specific issues
3. Run `vrooli help` for CLI assistance
