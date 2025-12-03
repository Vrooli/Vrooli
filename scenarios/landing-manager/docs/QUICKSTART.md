---
title: "Quick Start Guide"
description: "Generate your first landing page in under 5 minutes"
category: "getting-started"
order: 1
audience: ["users", "developers"]
---

# Quick Start Guide

Get your first landing page up and running in under 5 minutes.

## Prerequisites

- Vrooli installed and configured (`vrooli help` works)
- PostgreSQL resource available (`resource-postgres status`)

## Step 1: Start Landing Manager

```bash
# From the Vrooli root directory
cd scenarios/landing-manager
make start
```

Wait for the health check to pass (usually 10-15 seconds). You'll see:
```
✓ API healthy on port XXXXX
✓ UI available on port XXXXX
```

## Step 2: Open the Factory UI

Open your browser to the UI port shown above, or use:

```bash
# Get the UI URL
vrooli scenario port landing-manager UI_PORT
# Then open http://localhost:<port>
```

You'll see the **Landing Page Factory** dashboard:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  Landing Page Factory                                    [⌘K Create]  [?]   │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  AVAILABLE TEMPLATES                                                        │
│  ┌─────────────────────────────────────┐                                    │
│  │  📄 SaaS Landing Page               │                                    │
│  │  ─────────────────────────────────  │                                    │
│  │  Full-featured landing page with    │                                    │
│  │  A/B testing, analytics, and        │                                    │
│  │  Stripe payments built-in.          │                                    │
│  │                                     │                                    │
│  │  Sections: Hero, Features, Pricing, │                                    │
│  │  Testimonials, FAQ, CTA, Footer     │                                    │
│  │                                     │                                    │
│  │  [Select Template]                  │                                    │
│  └─────────────────────────────────────┘                                    │
│                                                                             │
│  ─────────────────────────────────────────────────────────────────────────  │
│                                                                             │
│  GENERATED SCENARIOS (Staging)                                              │
│  ┌────────────────────────────────────────────────────────────────────────┐ │
│  │  No scenarios generated yet. Select a template above to get started.  │ │
│  └────────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Step 3: Generate Your First Landing Page

### Option A: Using the UI (Recommended)

1. Press `⌘K` (Mac) or `Ctrl+K` (Windows/Linux) to open the Create dialog
2. Fill in the form:

```
┌─────────────────────────────────────────────────────────────┐
│  Create New Landing Page                              [X]   │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Template                                                   │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  SaaS Landing Page                            [▼]   │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  Display Name                                               │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  My First Landing                                   │   │
│  └─────────────────────────────────────────────────────┘   │
│  Human-readable name shown in the UI                        │
│                                                             │
│  Slug                                                       │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  my-first-landing                                   │   │
│  └─────────────────────────────────────────────────────┘   │
│  URL-safe identifier (lowercase, hyphens only)              │
│                                                             │
│  [ ] Dry run (preview only, don't create files)             │
│                                                             │
│                               [Cancel]  [Generate]          │
└─────────────────────────────────────────────────────────────┘
```

3. Click **Generate**

### Option B: Using the CLI

```bash
landing-manager generate saas-landing-page \
  --name "My First Landing" \
  --slug "my-first-landing"
```

## Step 4: Start Your Generated Landing Page

After generation completes, you'll see your new scenario in the "Generated Scenarios" section:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  GENERATED SCENARIOS (Staging)                                              │
│  ┌────────────────────────────────────────────────────────────────────────┐ │
│  │  📦 my-first-landing                                                   │ │
│  │  ─────────────────────────────────────────────────────────────────────│ │
│  │  Template: SaaS Landing Page                                          │ │
│  │  Status: ● Stopped                                                    │ │
│  │  Created: Just now                                                    │ │
│  │                                                                       │ │
│  │  [▶ Start]  [🗑 Delete]  [📁 Open Folder]                             │ │
│  └────────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────┘
```

Click **Start** to launch your landing page. The status will change:

```
│  Status: ● Running                                                       │
│  API: http://localhost:15432                                             │
│  UI:  http://localhost:35432                                             │
│                                                                          │
│  [⏹ Stop]  [🔄 Restart]  [📤 Promote]  [📁 Open Folder]                  │
│                                                                          │
│  Quick Links:                                                            │
│  • Public Landing Page                                                   │
│  • Admin Portal                                                          │
```

Or via CLI:
```bash
vrooli scenario start my-first-landing --path ./generated/my-first-landing
```

## Step 5: View Your Landing Page

Click on **Public Landing Page** to see your generated landing page:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                                                          [Get Started]      │
│                                                                             │
│                    Build Landing Pages in Minutes                           │
│                                                                             │
│         The all-in-one platform for creating production-ready               │
│            landing pages with A/B testing built in.                         │
│                                                                             │
│                        [Start Free Trial]                                   │
│                                                                             │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ⚡ Fast Setup        🎯 A/B Testing        💳 Payments                     │
│  Launch in minutes   Test variants         Stripe built-in                  │
│                                                                             │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│                          PRICING                                            │
│  ┌─────────────┐   ┌─────────────┐   ┌─────────────┐                       │
│  │   Starter   │   │    Pro      │   │ Enterprise  │                       │
│  │   $0/mo     │   │  $29/mo     │   │  Custom     │                       │
│  └─────────────┘   └─────────────┘   └─────────────┘                       │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

Access points:

| Surface | URL | Purpose |
|---------|-----|---------|
| **Public Landing** | `http://localhost:<port>/` | What visitors see |
| **Admin Portal** | `http://localhost:<port>/admin` | Analytics & customization |
| **Health Check** | `http://localhost:<port>/health` | API status |

## What You Just Created

Your generated landing page includes:

```
my-first-landing/
├── api/                 # Go API server
│   └── main.go          # Stripe, metrics, variants
├── ui/                  # React + Vite frontend
│   └── src/
│       ├── surfaces/
│       │   ├── public-landing/   # Visitor-facing pages
│       │   └── admin-portal/     # Admin dashboard
├── .vrooli/
│   ├── variants/        # A/B test configurations
│   └── styling.json     # Design tokens
├── initialization/
│   └── postgres/        # Database schema
└── Makefile             # Lifecycle commands
```

## Next Steps

| Goal | Guide |
|------|-------|
| Customize content | See [Admin Guide](ADMIN_GUIDE.md) |
| Understand A/B testing | See [Concepts](CONCEPTS.md#ab-testing) |
| Set up Stripe payments | See [Admin Guide - Stripe Setup](ADMIN_GUIDE.md#stripe-setup) |
| Promote to production | See [Promoting a Scenario](#promoting-to-production) |
| Troubleshoot issues | See [Troubleshooting](TROUBLESHOOTING.md) |

---

## Common Operations

### Preview Without Generating (Dry Run)

See what would be created without writing files:

```bash
landing-manager generate saas-landing-page \
  --name "Test Landing" \
  --slug "test-landing" \
  --dry-run
```

### Stop a Scenario

```bash
# Via Factory UI: Click "Stop" on the scenario card

# Via CLI:
vrooli scenario stop my-first-landing
```

### Promoting to Production

Once you're happy with your landing page in staging:

1. In Factory UI, click **Promote** on the scenario card
2. Confirm the promotion
3. The scenario moves from `generated/` to `scenarios/`

Or via CLI:
```bash
# The API endpoint handles the move
curl -X POST http://localhost:<port>/api/v1/lifecycle/my-first-landing/promote
```

### Delete a Staging Scenario

```bash
# Via Factory UI: Click "Delete" on the scenario card

# Via CLI:
curl -X DELETE http://localhost:<port>/api/v1/lifecycle/my-first-landing
```

---

## Troubleshooting Quick Start

### "API is not reachable"

Landing Manager isn't running. Start it:
```bash
cd scenarios/landing-manager && make start
```

### "Port already in use"

Another service is using the port. Either stop it or let Vrooli auto-allocate:
```bash
make stop
make start
```

### "Database connection failed"

PostgreSQL isn't running:
```bash
resource-postgres start
```

### Generated scenario won't start

Check the logs:
```bash
vrooli scenario logs my-first-landing --tail 50
```

---

**Time to complete**: ~5 minutes
**Next**: [Admin Guide](ADMIN_GUIDE.md) | [Core Concepts](CONCEPTS.md)
