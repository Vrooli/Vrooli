---
title: "Frequently Asked Questions"
description: "Common questions about Landing Manager"
category: "operational"
order: 11
audience: ["users", "developers"]
---

# Frequently Asked Questions

## General

### What is Landing Manager?

Landing Manager is a **factory scenario** that generates complete landing page applications. It's not a landing page itself - it creates landing pages from templates.

### What's included in a generated landing page?

Each generated landing page includes:
- React + Vite frontend with public landing and admin portal
- Go (Gin) API server
- PostgreSQL database schema
- A/B testing infrastructure
- Stripe payment integration
- Analytics and metrics tracking

### Can I use this without Vrooli?

The generated landing pages are standalone applications. Once generated, they can theoretically run independently, but they're designed to work within the Vrooli ecosystem for lifecycle management, resource sharing, and deployment.

---

## Getting Started

### How long does it take to create a landing page?

About 60 seconds to generate, plus time for customization. The Quick Start guide walks through the complete process in under 5 minutes.

### Do I need to know coding to use this?

No coding required for basic use. The admin portal provides a UI for:
- Editing content
- Managing A/B variants
- Viewing analytics
- Configuring Stripe

For advanced customization, you may need React/TypeScript knowledge.

### What templates are available?

Currently one template: `saas-landing-page` - a full-featured SaaS landing page with hero, features, pricing, testimonials, FAQ, and CTA sections.

More templates are planned (see [PRD P2 targets](../PRD.md)).

---

## A/B Testing

### How does A/B testing work?

When visitors arrive, they're assigned to a variant based on configured weights. Their assignment is stored in localStorage so they see the same variant on return visits.

See [Concepts - A/B Testing](CONCEPTS.md#ab-testing-system) for the complete flow.

### Can I force a specific variant for testing?

Yes, add `?variant=<slug>` to the URL:
```
http://localhost:3000/?variant=holiday-special
```

### How long should I run an A/B test?

Until you have statistical significance. Rule of thumb:
- Minimum 100 conversions per variant
- At least 2 weeks (to account for weekly patterns)
- Use an [A/B test calculator](https://vwo.com/tools/ab-test-duration-calculator/) for precise duration

### Can I test individual elements instead of whole pages?

Not currently. Landing Manager uses whole-page A/B testing. Element-level testing (multivariate) is a P2 feature.

---

## Payments

### What payment processors are supported?

Stripe only. This covers:
- One-time payments
- Subscriptions (monthly/yearly)
- Credit top-ups
- Customer portal

### Do I need a Stripe account?

Yes. You'll need:
- Stripe account (test or live)
- Publishable key
- Secret key
- Webhook secret

### How do I test payments locally?

1. Use Stripe test mode keys
2. Use test card: `4242 4242 4242 4242`
3. Use [Stripe CLI](https://stripe.com/docs/stripe-cli) to forward webhooks:
   ```bash
   stripe listen --forward-to localhost:3000/api/v1/webhooks/stripe
   ```

### How do bundled apps verify subscriptions?

Via the subscription verification API:
```
GET /api/v1/subscription/verify?user=user@example.com
```

Responses are cached for up to 60 seconds.

---

## Admin Portal

### How do I access the admin portal?

Navigate to `/admin` on your landing page URL. It's not linked from the public page for security.

### What are the default admin credentials?

```
Email: admin@localhost
Password: admin123
```

**Change these immediately in production!**

### Can I have multiple admin users?

Not currently. Single admin user per landing page. Multi-user support is not yet implemented.

### How do I change the admin password?

Currently requires database update:
```sql
-- Generate new bcrypt hash and update
UPDATE admin_users SET password_hash = '<new-hash>' WHERE email = 'admin@localhost';
```

---

## Customization

### How do I change the content?

1. Log into admin portal
2. Go to Customization
3. Select a variant
4. Click on a section to edit
5. Changes save and preview in real-time

### Can I add custom sections?

Yes, but it requires code changes. See [Agent Guide](../api/templates/AGENT.md) for the 3-file process.

### How do I change colors and fonts?

Edit `.vrooli/styling.json` in your generated scenario. Changes require a UI rebuild:
```bash
cd ui && pnpm run build
```

Or use the admin portal for runtime theme changes (primary color only).

### Can AI customize my landing page?

Yes! Use the Agent Customization feature in the admin portal:
1. Write a brief describing what you want
2. Select a persona (optimization approach)
3. The agent creates changes via the constrained API

---

## Deployment

### What's the difference between staging and production?

| Staging | Production |
|---------|------------|
| `generated/<slug>/` | `scenarios/<slug>/` |
| Safe testing area | Live deployment |
| Can be deleted easily | Part of main scenarios |

### How do I promote to production?

In Factory UI, click **Promote** on a staging scenario. This moves it from `generated/` to `scenarios/`.

### Can I deploy outside of Vrooli?

The generated code is standalone. You could:
1. Copy the scenario directory
2. Set up your own PostgreSQL
3. Configure environment variables
4. Run the Go API and Vite UI

However, you'd lose Vrooli's lifecycle management, resource sharing, and deployment infrastructure.

### How do I deploy to the cloud?

See the [Deployment Hub](../../docs/deployment/README.md) for Tier 2+ deployment options (desktop, SaaS, enterprise).

---

## Technical

### What tech stack is used?

**Frontend**: React 18, TypeScript, Vite, TailwindCSS, shadcn/ui, Lucide icons

**Backend**: Go 1.21+, Gin framework, direct PostgreSQL

**Database**: PostgreSQL 14+

### Can I use a different database?

Not without significant code changes. The API uses PostgreSQL-specific features (JSONB, etc.).

### Can I use a different frontend framework?

You could create a new template with a different framework, but existing templates use React.

### How do I add dependencies?

**Frontend**:
```bash
cd ui
pnpm add <package>
```

**Backend**:
```bash
cd api
go get <package>
```

Then rebuild.

### Where are metrics stored?

In PostgreSQL in the `events` table:
```sql
CREATE TABLE events (
    id SERIAL PRIMARY KEY,
    variant_id INTEGER,
    event_type VARCHAR(50),
    event_data JSONB,
    session_id VARCHAR(100),
    visitor_id VARCHAR(100),
    created_at TIMESTAMP DEFAULT NOW()
);
```

---

## Performance

### What Lighthouse scores should I expect?

Target: 90+ for Performance, Accessibility, Best Practices, and SEO.

Actual scores depend on your content (image sizes, video, etc.).

### How many concurrent users can it handle?

Depends on your hardware. The Go API is efficient:
- Single instance: 1000+ req/sec
- With PostgreSQL: 500+ req/sec (database becomes bottleneck)

For high traffic, consider Redis caching and load balancing.

### Is the frontend SSR or SPA?

SPA (Single Page Application). For SSR, you'd need to modify the template significantly.

---

## Troubleshooting

### See Also

- [Troubleshooting Guide](TROUBLESHOOTING.md) - Detailed solutions
- [Problems](PROBLEMS.md) - Known issues
- [Concepts](CONCEPTS.md) - Architecture understanding

---

## Still Have Questions?

1. Check the [documentation index](index.md) for other guides
2. Review [Troubleshooting](TROUBLESHOOTING.md) for specific issues
3. Run `landing-manager help` for CLI assistance
