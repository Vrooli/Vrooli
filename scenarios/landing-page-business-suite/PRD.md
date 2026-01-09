# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/prd-control-tower/docs/CANONICAL_PRD_TEMPLATE.md`
> **Validation**: Enforced by `prd-control-tower` + `scenario-auditor`
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

**Purpose**: SaaS Landing Page Template is the runtime payload that Landing Manager generates. It ships the **public landing page + admin portal** with A/B testing, analytics, Stripe payments, and subscription verification APIs. Landing Manager copies this template into a new scenario; all runtime behavior lives here, not in the factory.

**Primary users/verticals** (of generated landing pages):
- SaaS founders launching products (e.g., Vrooli Pro bundles)
- Entrepreneurs validating product-market fit with A/B testing
- Marketing teams building conversion-optimized landing pages
- Developers deploying monetized Vrooli scenarios

**Deployment surfaces** (per generated scenario):
- Public landing page: React + Vite SPA served from `/`
- Admin portal: `/admin` with auth, analytics, customization
- API: Go (Gin) REST for variants, metrics, payments, subscription verification
- Agent Integration: On-demand customization via Claude Code/Resource CLI with constrained APIs/files

**Value promise**: Generated landing pages are production-ready with payments, A/B testing, and analytics built-in. Agents can customize safely; marketers can A/B test quickly.

**Runtime spec linkage**: This PRD is the authoritative implementation brief for the landing runtime (landing config API, fallback strategy, subscription/billing endpoints, credit wallet math, download gating, and bundled-app entitlement expectations). The landing-manager factory references this document directly so every generated scenario inherits the bundle-ready experience.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability

#### Admin Portal Core
- [x] OT-P0-005 | Agent integration trigger | Generated landing exposes a customization trigger endpoint consumable by the factory (handoff remains initiated from landing-manager)
- [x] OT-P0-006 | Structured agent input | Agent receives structured brief (text + assets + goals) and writes changes via template-defined APIs/files only (no factory-side mutation)
- [x] OT-P0-007 | Security by obscurity | Admin portal not linked from any public page or sitemap
- [x] OT-P0-008 | Authentication | Admin portal protected by email/password with bcrypt/argon2 password hashing
- [x] OT-P0-009 | Admin home modes | Admin home displays exactly two modes: "Analytics / Metrics" and "Customization"
- [x] OT-P0-010 | Navigation efficiency | Navigate to any customization card in ≤ 3 clicks from admin home
- [x] OT-P0-011 | Breadcrumb navigation | All admin pages show breadcrumb indicating current location (e.g., Admin / Customization / Hero / Variant A)

#### Customization UX
- [x] OT-P0-012 | Split customization layout | Each slide customization page shows form (one column) + live preview (other column), stacked on mobile
- [x] OT-P0-013 | Live preview updates | Form field changes update preview within 300ms (debounced) without page reload

#### A/B Testing Core
- [x] OT-P0-014 | URL variant selection | If `variant_slug` in URL, force that variant regardless of localStorage
- [x] OT-P0-015 | localStorage persistence | If no URL slug and localStorage has variant ID, reuse that variant
- [x] OT-P0-016 | API weight-based selection | If neither URL nor localStorage, API selects variant by weight and frontend stores it in localStorage
- [x] OT-P0-017 | Variant CRUD operations | Admin can create, update weights, disable/archive, and hard delete variants (with confirmation)
- [x] OT-P0-018 | Archived variant handling | Archived variants remain queryable for analytics but ineligible for random selection

#### Metrics Core
- [x] OT-P0-019 | Event variant tagging | All events (page view, scroll, clicks, forms, conversions) include variant_id in payload
- [x] OT-P0-020 | Analytics variant filtering | Admin analytics view filters stats by variant and time range
- [x] OT-P0-021 | Minimum event coverage | System emits events for: page_view, scroll_depth (bands), click (element ID), form_submit, conversion (Stripe success)
- [x] OT-P0-022 | Metrics idempotency | Metrics ingestion is idempotent or deduplicated to avoid double-counting on retries

#### Payments Core
- [x] OT-P0-023 | Analytics summary | Analytics home shows total visitors, conversion rate per variant, top CTAs by CTR
- [x] OT-P0-024 | Variant detail view | Variant detail page shows views, CTA clicks, conversions, conversion rate, basic trend
- [x] OT-P0-025 | Stripe environment config | Each landing page includes Stripe keys from environment variables
- [x] OT-P0-026 | Stripe routes | Each landing page includes routes for creating checkout sessions and handling webhook events
#### Landing Runtime & Fallback
- [x] OT-P0-031 | API-driven landing config + fallback | Public `/` renders ordered sections, pricing, and download CTAs from `GET /api/v1/landing-config?variant=...`, and ships a baked fallback variant (sections + pricing copy) for API timeouts, auth failures, or admin misconfiguration.

#### Subscription, Credits, and Bundled Apps
- [x] OT-P0-032 | Subscription-aware pricing APIs | Backend parses Stripe metadata (products/prices/intro pricing) and exposes `GET /plans`, `POST /billing/create-checkout-session`, `POST /billing/create-credits-checkout-session`, and `GET /billing/portal-url` with correct tier/weight info for both monthly/yearly plans plus $1 intro logic.
- [x] OT-P0-033 | Credits + entitlements | Provide `GET /me/subscription`, `GET /me/credits`, and `GET /entitlements` endpoints that interpret `credits_per_usd`, `display_credits_multiplier`, intro grants, top-ups, donations, and return feature flags for bundled apps (with short-lived caches/offline fallback).
- [x] OT-P0-034 | Download gating for bundled apps | Landing config includes multi-app download sections that surface installers, release notes, and store links curated from the admin Download Settings page; downloads remain disabled until entitlements confirm an active subscription, and emit variant + plan metadata when triggered.

#### Security & Verification
- [x] OT-P0-027 | Webhook signature verification | All webhook endpoints verify Stripe signature before processing
- [x] OT-P0-028 | Subscription verification endpoint | Each landing page exposes GET /api/subscription/verify accepting user identity, returning active/inactive/trial/canceled
- [x] OT-P0-029 | Verification caching | Subscription verification responses cacheable (short TTL) with max 60s lag from webhook changes
- [x] OT-P0-030 | Subscription cancellation endpoint | API exposes POST /api/subscription/cancel validating user and returning status

### 🟠 P1 – Should have post-launch

#### Performance & Accessibility
- [x] OT-P1-001 | Lighthouse performance | Default generated landing page achieves Lighthouse performance score ≥ 90 on desktop
- [x] OT-P1-002 | Time to interactive | Cold load TTI < 2.0s on typical broadband for main landing page
- [x] OT-P1-003 | Lighthouse accessibility | Lighthouse accessibility score ≥ 90 for generated template
- [x] OT-P1-004 | Keyboard accessibility | All interactive elements keyboard-accessible with discernible labels

#### Design & Branding
- [x] OT-P1-005 | Aesthetic guidelines in template | Template specification includes `<frontend_aesthetics>` block in description for design agents
- [x] OT-P1-006 | Custom typography | Default template declares custom font stack (not Inter, Roboto, Arial, or system default)
- [x] OT-P1-007 | CSS theming | Template uses defined CSS variables for colors and spacing
- [x] OT-P1-008 | Non-trivial backgrounds | Template includes at least one layered gradient/shape/texture background (not flat solid)
- [x] OT-P1-009 | Video section support | Template supports at least one video demo section (URL, thumbnail, caption, layout) configurable via admin

### 🟢 P2 – Future / expansion

#### Multi-Template Support
- [ ] OT-P2-001 | Additional templates | Support multiple landing page templates (e.g., SaaS, Product Launch, Lead Magnet, Event Registration)
- [ ] OT-P2-002 | Template marketplace | Community-contributed templates with ratings and reviews

#### Advanced A/B Testing
- [ ] OT-P2-003 | Statistical significance testing | Admin dashboard shows confidence intervals and statistical significance for A/B tests
- [ ] OT-P2-004 | Automatic winner selection | System automatically promotes winning variant after reaching significance threshold
- [ ] OT-P2-005 | Multi-armed bandit | Implement Thompson sampling or UCB for dynamic traffic allocation

#### Enhanced Analytics
- [ ] OT-P2-006 | Heatmap visualization | Click heatmaps and scroll maps per variant
- [ ] OT-P2-007 | Session replay | Record and replay user sessions for UX analysis
- [ ] OT-P2-008 | Funnel analysis | Multi-step conversion funnel tracking (landing → CTA → checkout → success)

#### Agent Capabilities
- [ ] OT-P2-009 | Agent-generated variants | Agent automatically generates and deploys A/B test variants based on conversion data
- [ ] OT-P2-010 | Multivariate testing | Test multiple elements simultaneously (hero copy + CTA button + pricing)
- [ ] OT-P2-011 | Persona-based variants | Generate variants optimized for different user personas

#### Integration & Extensibility
- [ ] OT-P2-012 | CRM integration | Push leads to HubSpot, Salesforce, or custom CRMs
- [ ] OT-P2-013 | Email marketing integration | Sync subscribers to Mailchimp, ConvertKit, etc.
- [ ] OT-P2-014 | Webhook notifications | Custom webhooks for landing page events (form submit, conversion, etc.)

## 🧱 Tech Direction Snapshot

**Preferred stacks**:
- **Landing page template**: React + TypeScript + Vite + TailwindCSS + shadcn + Lucide icons
- **Factory scenario API**: Go (for template management, scenario generation orchestration)
- **Generated landing page APIs**: Go (Gin framework for REST, direct PostgreSQL)

**Data + storage expectations**:
- **postgres**: Required for each generated landing page (content, A/B variants, metrics, subscriptions)
- **redis** (optional): Session caching for high-traffic landing pages
- **qdrant** (optional): Semantic search for template recommendations

**Integration strategy**:
- **Vrooli CLI**: Primary interface for template listing and scenario generation
- **Resource CLI** (claude-code): Agent customization via structured prompts
- **Direct API**: Programmatic landing page creation for advanced users

**Non-goals**:
- Not a general-purpose website builder (focused on landing pages)
- Not a CMS (content is API-driven, not file-based)
- Not a design tool (relies on templates + agent customization)
- Not a hosting platform (users deploy to their infrastructure)

## 🤝 Dependencies & Launch Plan

**Required resources**:
- **postgres**: Essential for all generated landing pages (content, variants, metrics, subscriptions)
- **resource-claude-code**: Agent-based customization of landing pages

**Optional resources**:
- **redis**: Session caching for high-traffic scenarios
- **qdrant**: Template recommendations based on user intent
- **browserless**: Screenshot generation for template previews

**Scenario dependencies**:
- **scenario-authenticator** (future): Multi-tenant admin portal support
- **funnel-builder** (future): Embed conversion funnels in landing pages
- **referral-program-generator** (future): Add referral programs to landing pages

**Operational risks**:
- **Template maintenance**: Templates may become outdated; mitigate with versioning and migration guides
- **Agent quality**: AI-generated customizations may be poor; mitigate with preview + rollback
- **Stripe API changes**: Payment flows may break; mitigate with versioned SDK and monitoring

**Launch sequencing**:
1. **Phase 1 (P0)**: Single template (`saas-landing-page`), basic A/B testing, Stripe integration
2. **Phase 2 (P1)**: Performance optimization, accessibility compliance, video sections
3. **Phase 3 (P2)**: Multi-template support, advanced analytics, agent-generated variants

## 🎨 UX & Branding

**Look & feel**:
- **Public landing pages**: Bold, distinctive design per the `<frontend_aesthetics>` guidelines. Avoid generic "AI slop" aesthetics (Inter font, flat colors, low contrast). Use custom typography, layered gradients, high-contrast color schemes, and asymmetric layouts.
- **Admin portal**: Clean, professional SaaS dashboard aesthetic. Light theme by default, dark theme optional. Focus on clarity and efficiency (metrics-first UI).

**Accessibility**:
- WCAG 2.1 Level AA compliance target (≥ 90 Lighthouse score)
- Keyboard navigation for all interactive elements
- High color contrast ratios (4.5:1 for text, 3:1 for UI components)
- Screen reader-friendly labels and ARIA attributes

**Voice & messaging**:
- **Public landing pages**: Confident, persuasive, user-focused. Tone varies per customization (e.g., professional for B2B, friendly for B2C).
- **Admin portal**: Concise, action-oriented, data-driven. Tone: "Here's what's working, here's what to optimize."

**Branding hooks**:
- **Logo**: Customizable per landing page (agent can replace with user-provided logo)
- **Colors**: CSS variables for primary, secondary, accent colors (agent can override)
- **Typography**: Custom font stack per landing page (not default web fonts)
- **Iconography**: Lucide icons by default (consistent with Vrooli ecosystem)

## 📎 Appendix

### External References

#### Landing Page Best Practices
- [Unbounce Landing Page Guide](https://unbounce.com/landing-page-articles/)
- [CXL Landing Page Optimization](https://cxl.com/blog/landing-page-optimization/)
- [Nielsen Norman Group - Landing Pages](https://www.nngroup.com/articles/landing-pages/)

#### A/B Testing Resources
- [Optimizely AB Testing Guide](https://www.optimizely.com/optimization-glossary/ab-testing/)
- [VWO A/B Test Duration Calculator](https://vwo.com/tools/ab-test-duration-calculator/)
- [Bayesian vs Frequentist Testing](https://cxl.com/blog/bayesian-frequentist-ab-testing/)

#### Design Inspiration (Distinctive Aesthetics)
- [Vercel](https://vercel.com) - Clean, fast, modern
- [Linear](https://linear.app) - Minimalist, distinctive typography
- [Raycast](https://raycast.com) - Bold colors, strong branding
- [Cal.com](https://cal.com) - Professional, trustworthy

#### Stripe Integration
- [Stripe Checkout Documentation](https://stripe.com/docs/checkout/quickstart)
- [Stripe Webhooks Best Practices](https://stripe.com/docs/webhooks/best-practices)
- [Stripe Subscription Lifecycle](https://stripe.com/docs/billing/subscriptions/overview)

### Internal References
- **Resource postgres**: `scripts/resources/resource-postgres/`
- **Resource claude-code**: `scripts/resources/resource-claude-code/`
- **Lifecycle system**: `scripts/lib/lifecycle/`
- **Testing architecture**: `docs/testing/architecture/PHASED_TESTING.md`
- **Template system**: `scripts/scenarios/templates/`

### Design Decisions

**Why whole-page A/B testing instead of element-level?**
- Simpler to implement and reason about
- Avoids combinatorial explosion of variants
- Clearer attribution of conversion impact
- Trade-off: Less granular optimization (acceptable for MVP)

**Why localStorage for variant stickiness?**
- Simple, no backend session management required
- Works without authentication
- Persists across page reloads
- Trade-off: Doesn't persist across browsers/devices (acceptable for MVP)

**Why Go API instead of Node.js?**
- Better performance for metrics ingestion (high throughput)
- Simpler deployment (single binary)
- Strong typing reduces payment/subscription bugs
- Trade-off: Less dynamic than Node.js (acceptable for structured APIs)

**Why Stripe Checkout instead of custom payment form?**
- PCI compliance handled by Stripe
- Faster implementation (hours vs. weeks)
- Better mobile UX (Stripe-optimized)
- Trade-off: Less customization (acceptable for MVP, most users prefer Stripe UX)

### Evolution Path

**v1.0 (Current MVP)**
- Single template (`saas-landing-page`)
- Basic A/B testing (whole-page variants)
- Core metrics (page views, clicks, conversions)
- Stripe checkout + webhook handling
- Admin portal (analytics + customization)

**v2.0 (Planned)**
- Multiple templates (Product Launch, Lead Magnet, Event)
- Statistical significance testing
- Heatmaps and session replay
- Agent-generated A/B variants
- CRM/email marketing integrations

**v3.0 (Future)**
- Visual drag-and-drop builder
- Multivariate testing
- AI-driven conversion optimization
- Cross-landing-page analytics aggregation
- Template marketplace

### Success Metrics (Post-Launch)

**Adoption Metrics**
- Number of landing pages generated per month (target: 10+ by month 3)
- Active users in admin portals (target: 80% of creators)

**Quality Metrics**
- Average Lighthouse performance score (target: ≥ 90)
- Average conversion rate for Vrooli Pro landing page (target: ≥ 5%)
- Admin portal session duration (target: ≥ 10 minutes, indicating deep engagement)

**Business Impact**
- Revenue generated via Stripe integrations (target: $5K+ MRR by month 6)
- Number of A/B tests run (target: 50+ tests across all landing pages)
- Time saved vs. manual landing page creation (target: 95% reduction, weeks → hours)
