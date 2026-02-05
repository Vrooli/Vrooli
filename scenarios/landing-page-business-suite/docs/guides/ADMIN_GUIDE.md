---
title: "Admin Portal Guide"
description: "How to use the admin portal to manage your landing page"
category: "getting-started"
order: 2
audience: ["users", "marketers"]
---

# Admin Portal Guide

This guide covers the admin portal for the Landing Page Business Suite. It is grounded in the UI navigation config, admin page catalog, and route map:
- [CODE: ui/src/surfaces/admin-portal/config/navigation.ts#NAVIGATION_CONFIG]
- [CODE: ui/src/surfaces/admin-portal/config/adminPages.ts]
- [CODE: ui/src/App.tsx]

## Accessing the Admin Portal

- Login URL: `http://localhost:<port>/admin/login`
- Admin home after login: `http://localhost:<port>/admin`
- The admin portal is not linked from the public landing page.

### Default Credentials

```
Email: admin@localhost
Password: changeme123
```

Important: change these credentials immediately in production.

Options for changing credentials:
1. Environment variables (recommended for deployments): set `ADMIN_DEFAULT_EMAIL` and `ADMIN_DEFAULT_PASSWORD` before starting the scenario.
2. Admin portal: change via the Profile page (`/admin/profile`) after logging in.

## Admin Navigation Map

This section mirrors the UI navigation config. If you update `NAVIGATION_CONFIG`, update this map.

### Direct Links

- Home (`/admin`): Admin overview with quick flows, stats, and reset controls.
  - [CODE: ui/src/surfaces/admin-portal/routes/AdminHome.tsx]
- Docs (`/admin/docs`): Built-in documentation viewer backed by `docs/manifest.json`.
  - [CODE: ui/src/surfaces/admin-portal/routes/DocsViewer.tsx]
  - [CODE: docs/manifest.json]

### Landing

- Dashboard (`/admin/landing`): Landing page health, quick flows, and resume actions.
  - [CODE: ui/src/surfaces/admin-portal/routes/LandingDashboard.tsx]
- Customization (`/admin/customization`): Manage landing variants, sections, and weights.
  - [CODE: ui/src/surfaces/admin-portal/routes/Customization.tsx]
- Analytics (`/admin/analytics`): Conversion metrics and variant performance.
  - [CODE: ui/src/surfaces/admin-portal/routes/AdminAnalytics.tsx]
- Branding (`/admin/branding`): Site identity, SEO defaults, and coming soon settings.
  - [CODE: ui/src/surfaces/admin-portal/routes/BrandingSettings.tsx]
- Agent (`/admin/customization/agent`): Trigger AI-powered landing improvements.
  - [CODE: ui/src/surfaces/admin-portal/routes/AgentCustomization.tsx]

### Billing

- Dashboard (`/admin/billing-home`): Stripe readiness, plans status, and quick flows.
  - [CODE: ui/src/surfaces/admin-portal/routes/BillingDashboard.tsx]
- Stripe (`/admin/billing`): Configure Stripe keys and webhook settings.
  - [CODE: ui/src/surfaces/admin-portal/routes/BillingSettings.tsx]
- Plans (`/admin/tiers`): Subscription tier management (marked as "Soon" in UI).
  - [CODE: ui/src/surfaces/admin-portal/routes/TiersManagement.tsx]
- AI Keys (`/admin/api-keys`): Manage AI provider API keys.
  - [CODE: ui/src/surfaces/admin-portal/routes/APIKeysSettings.tsx]

### Apps

- Dashboard (`/admin/apps`): App distribution health and usage quick flows.
  - [CODE: ui/src/surfaces/admin-portal/routes/AppsManagement.tsx]
- Downloads (`/admin/downloads`): App registry, installers, hosting, and artifacts.
  - [CODE: ui/src/surfaces/admin-portal/routes/DownloadSettings.tsx]
- Remote Profiles (`/admin/remote-profiles`): Remote admin sessions for deployed suites.
  - [CODE: ui/src/surfaces/admin-portal/routes/RemoteProfiles.tsx]
- Usage (`/admin/usage`): Monthly AI credit usage reporting.
  - [CODE: ui/src/surfaces/admin-portal/routes/UsageDashboard.tsx]
- Tier Limits (`/admin/tier-limits`): Credit limits per subscription tier.
  - [CODE: ui/src/surfaces/admin-portal/routes/TierLimitsSettings.tsx]
- App Limits (`/admin/app-limits`): Per-app quota configuration.
  - [CODE: ui/src/surfaces/admin-portal/routes/AppLimitsSettings.tsx]

### Users

- Dashboard (`/admin/users`): User activity, feedback, and waitlist stats.
  - [CODE: ui/src/surfaces/admin-portal/routes/UsersDashboard.tsx]
- Accounts (`/admin/accounts`): User accounts, subscriptions, and sessions (UI marks as "Soon").
  - [CODE: ui/src/surfaces/admin-portal/routes/UserAccounts.tsx]
- Feedback (`/admin/feedback`): Triage user feedback and requests.
  - [CODE: ui/src/surfaces/admin-portal/routes/FeedbackManagement.tsx]
- Waitlist (`/admin/waitlist`): Coming soon signups and toggle.
  - [CODE: ui/src/surfaces/admin-portal/routes/WaitlistManagement.tsx]

### Account

- Profile (`/admin/profile`): Update admin email and password.
  - [CODE: ui/src/surfaces/admin-portal/routes/ProfileSettings.tsx]

### Deep Links Used By Editors

- Variant Editor (`/admin/customization/variants/:slug`): Edit variant metadata and sections.
  - [CODE: ui/src/surfaces/admin-portal/routes/VariantEditor.tsx]
- Section Editor (`/admin/customization/variants/:variantSlug/sections/:sectionId`): Edit a single section with live preview.
  - [CODE: ui/src/surfaces/admin-portal/routes/SectionEditor.tsx]
- Analytics Variant Shortcut (`/admin/analytics/:variantSlug`): Opens analytics pre-filtered to a variant.
  - [CODE: ui/src/surfaces/admin-portal/routes/AdminAnalytics.tsx]

## Admin Home

The Admin Home page (`/admin`) is a quick-entry dashboard for the entire suite. It provides:
- Quick links to Landing, Billing, Apps, and Users dashboards.
- Snapshot stats (active variants, traffic allocation, Stripe readiness, live variant).
- A preview shortcut to the public landing page.
- A "Danger Zone" to reset the demo data when needed.

## Landing Management

### Landing Dashboard

Use the Landing dashboard (`/admin/landing`) for at-a-glance landing health and quick flows. It highlights:
- Current live variant and resolution status.
- Variant health summaries and traffic allocation.
- Resume shortcuts for the last variant or analytics view.

### Customization (Variants and Sections)

The Customization page (`/admin/customization`) is the hub for A/B testing:
- Create, edit, archive, and delete variants.
- Adjust traffic weights (relative weights; all-zero means an even split).
- Jump into the Variant Editor or Section Editor.

#### Variant Editor

The Variant Editor (`/admin/customization/variants/:slug`) lets you:
- Update variant metadata (name, slug, axes).
- Add, reorder, and edit sections.
- Edit the entire variant + sections payload as JSON.

#### Section Editor

The Section Editor (`/admin/customization/variants/:variantSlug/sections/:sectionId`) lets you:
- Edit section fields with a live preview.
- Switch between sections in the variant timeline.

### Analytics

The Analytics page (`/admin/analytics`) shows:
- Total visitors, conversions, and conversion rate.
- Variant comparisons with conversion and trend signals.
- Filters for time range and variant.

### Branding

The Branding page (`/admin/branding`) configures:
- Site identity (name, tagline, logos, favicon).
- SEO defaults and social previews.
- Support contact channels.
- Coming soon messaging that pairs with the Waitlist page.

### Agent Customization

The Agent page (`/admin/customization/agent`) triggers AI-assisted landing improvements. Provide:
- A brief describing goals and target audience.
- Optional asset URLs.
- Optional preview mode to review changes before applying.

## Billing and Monetization

### Billing Dashboard

Use the Billing dashboard (`/admin/billing-home`) to:
- Check Stripe readiness.
- Jump to Stripe settings, plans, and AI key management.

### Stripe Settings

The Stripe page (`/admin/billing`) configures:
- Publishable key, secret key, and webhook secret.
- Status indicators for each key.

Related reference docs:
- [DOC: docs/reference/STRIPE_WEBHOOKS.md#stripe-webhooks--signing-secret]
- [DOC: docs/reference/STRIPE_RESTRICTED_KEYS.md#stripe-restricted-keys]

### Plans (Coming Soon)

The Plans page (`/admin/tiers`) is a placeholder. Use:
- Tier Limits for credit allocations.
- Stripe settings for price configuration.

### AI Keys

The AI Keys page (`/admin/api-keys`) lets you:
- Add, test, enable/disable, and remove provider keys.
- Configure the fallback keys used when customers do not provide their own.

## Apps and Usage

### Apps Dashboard

The Apps dashboard (`/admin/apps`) focuses on distribution readiness:
- Downloads health (apps configured, platforms, store links).
- Quick links to Downloads, Usage, Tier Limits, and App Limits.

### Downloads

The Downloads page (`/admin/downloads`) manages:
- App registry entries and platforms.
- Installer hosting, storage settings, and artifacts.
- Previewing the public landing page download section.

Downloads are gated by subscription status in the public experience.

CLI automation (optional):
- Upload + apply a managed artifact: `landing-page-business-suite admin-downloads-upload-managed --file <path> --app-key <app> --platform <platform> --release-version <version>`
- Proxy remote admin calls via stored sessions: `landing-page-business-suite remote-profiles-proxy <id> --method <METHOD> --path /admin/...`

### Desktop Auto-Update Endpoints

LPBS exposes public update endpoints that electron-updater (Generic Provider) polls for new versions.

**URL pattern:** `https://<lpbs-domain>/api/v1/updates/<app-key>/<channel>/<file>`

**Manifest files** (GET, returns YAML):
- `latest.yml` — Windows
- `latest-mac.yml` — macOS
- `latest-linux.yml` — Linux

**Binary downloads** (GET, returns 302 redirect to presigned S3 URL):
- Any other filename is treated as a binary download (e.g., `my-app-1.2.3.exe`).

**Channel mapping:** The `stable` channel maps to the `default` variant_key in download_assets. Other channel names pass through as-is.

**Access control:** By default, update endpoints are public. To restrict access, set the `update_api_key` field on the download app. When set, requests must include `X-Update-Key: <key>` header.

**electron-updater configuration** (in the desktop app's electron-builder config):
```json
{
  "provider": "generic",
  "url": "https://your-lpbs-domain.com/api/v1/updates/your-app-key",
  "channel": "stable"
}
```

If `update_api_key` is set, add `requestHeaders`:
```json
{
  "provider": "generic",
  "url": "https://your-lpbs-domain.com/api/v1/updates/your-app-key",
  "channel": "stable",
  "requestHeaders": { "X-Update-Key": "your-secret-key" }
}
```

Uploads via `admin-downloads-upload-managed` now automatically compute and store SHA512 hashes, which are required for serving update manifests.

### Remote Profiles

Remote Profiles (`/admin/remote-profiles`) let admins store encrypted sessions for deployed
landing-page-business-suite instances so automation can route admin tasks through the local API.
Use this page to create a profile, authenticate to the remote admin portal, verify session status,
and revoke access when needed.

#### Troubleshooting

- **"Remote session expired"**: Click Login to refresh the session cookie.
- **"API base must include /api/v1"**: Use the full remote API base URL (for example, `https://your-domain/api/v1`).
- **Remote test failures**: Confirm the deployment is reachable and the admin credentials are valid.

### Usage

The Usage dashboard (`/admin/usage`) provides:
- Monthly usage totals and active user counts.
- Breakdown by app and top users.
- Period navigation to historical months.

### Tier Limits

The Tier Limits page (`/admin/tier-limits`) sets:
- Monthly AI credit allocations per subscription tier.
- Bulk actions like reset and doubling limits.

### App Limits

The App Limits page (`/admin/app-limits`) sets:
- Per-app quotas beyond global AI credits.
- Limits scoped by subscription tier.

## Users and Feedback

### Users Dashboard

The Users dashboard (`/admin/users`) shows:
- Feedback counts by status.
- Waitlist totals.
- Quick links to feedback and waitlist workflows.

### Accounts (In Progress)

The Accounts page (`/admin/accounts`) lists user accounts and sessions. The UI currently marks this area as "Soon" in navigation; expect workflows to evolve.

### Feedback

The Feedback page (`/admin/feedback`) supports:
- Filtering by type and status.
- Status changes, replies, and deletion.
- Bulk actions for triage.

### Waitlist

The Waitlist page (`/admin/waitlist`) manages:
- Coming soon signups and CSV export.
- The coming soon toggle (pairs with Branding settings).

## Account and Security

### Profile

The Profile page (`/admin/profile`) is the recommended way to:
- Replace the seeded admin email.
- Rotate the admin password.
- Verify whether default credentials are still active.

### Docs Viewer

Use `/admin/docs` to browse this scenario's documentation. The tree is driven by `docs/manifest.json`.

Admin page headers show a documentation icon when a related guide section exists. The link routes into the docs viewer with the correct document path and anchor (driven by `ui/src/surfaces/admin-portal/config/adminPages.ts`), so you land directly on the matching section.

## Best Practices

### A/B Testing

1. Change one variable at a time between variants.
2. Wait for adequate sample size before deciding a winner.
3. Run tests long enough to cover weekly cycles.
4. Record your hypothesis for each variant before reviewing results.

### Content Quality

1. Keep hero headlines short and benefit-driven.
2. Use specific CTAs ("Start Free Trial" beats "Get Started").
3. Validate mobile layout with the live preview.

## Troubleshooting

### "Session expired"

Your login timed out. Log in again at `/admin/login`.

### Changes not saving

1. Check the browser console for errors.
2. Confirm the API is running ("/health" returns "healthy").
3. Refresh the page and retry.

### Analytics missing data

Analytics ingestion can lag by several minutes. If data is stale:
1. Check API health.
2. Verify database connectivity.
3. Review API logs for errors.

### Stripe webhooks failing

1. Verify the webhook secret in Stripe settings.
2. Confirm the webhook endpoint is reachable.
3. Review Stripe's webhook delivery logs for errors.
