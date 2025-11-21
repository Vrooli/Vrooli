# Research: Landing Manager

> **Generated**: 2025-11-21
> **Status**: Initial Research Complete

## 1. Uniqueness Check

### Repository Scan Results
Searched the Vrooli repository for scenarios with overlapping functionality:

```bash
rg -i 'landing|page.*factory|template.*manager' /home/matthalloran8/Vrooli/scenarios/
```

**Finding**: No existing scenario provides landing page generation and management capabilities.

### Related Scenarios (Complementary, Not Overlapping)

#### 1. funnel-builder (`/scenarios/funnel-builder`)
- **Purpose**: Multi-step conversion funnels (forms, quizzes, lead capture)
- **Relationship**: Could be *embedded within* landing pages created by landing-manager
- **Distinction**: Funnel-builder focuses on sequential flows; landing-manager creates full landing pages

#### 2. referral-program-generator (`/scenarios/referral-program-generator`)
- **Purpose**: Generates referral/affiliate programs for scenarios
- **Relationship**: Could *monetize* landing pages created by landing-manager
- **Distinction**: Referral programs are a monetization layer; landing-manager creates the landing pages themselves

#### 3. campaign-content-studio (`/scenarios/campaign-content-studio`)
- **Purpose**: Marketing content creation and management
- **Relationship**: Could provide content *for* landing pages
- **Distinction**: Content studio is about content creation; landing-manager is about page structure and deployment

#### 4. brand-manager (`/scenarios/brand-manager`)
- **Purpose**: Brand asset and identity management
- **Relationship**: Could provide branding guidelines *to* landing-manager
- **Distinction**: Brand management vs. landing page implementation

### Uniqueness Conclusion
**Landing-manager is unique** in the Vrooli ecosystem. It is a *meta-scenario* that:
- Generates new landing-page scenarios from templates
- Manages the lifecycle of landing-page scenarios
- Provides admin portals with A/B testing and analytics
- Integrates payments and subscription verification
- Orchestrates agent-based customization

No other scenario combines these capabilities.

---

## 2. Domain Research

### Landing Page Builders (External Reference)

#### Commercial Solutions
1. **Unbounce** (https://unbounce.com)
   - Drag-and-drop builder
   - A/B testing built-in
   - Conversion intelligence
   - Price: $90-$225/month

2. **Instapage** (https://instapage.com)
   - AI-driven optimization
   - Heatmaps and analytics
   - AdMap integration
   - Price: $199-$299/month

3. **Leadpages** (https://leadpages.com)
   - Template library
   - Lead capture focus
   - Basic A/B testing
   - Price: $37-$239/month

4. **ConvertFlow** (https://convertflow.com)
   - On-site conversion funnels
   - Personalization engine
   - Multi-step forms
   - Price: $99-$999/month

#### Open Source Solutions
1. **GrapesJS** (https://grapesjs.com)
   - Web builder framework
   - Plugin-based architecture
   - Free and open source

2. **Mobirise** (https://mobirise.com)
   - Offline page builder
   - Bootstrap-based
   - Free tier available

### Key Features Common to Leaders
- **Visual Editing**: Drag-and-drop or live preview editing
- **A/B Testing**: Built-in variant testing with statistical analysis
- **Analytics**: Conversion tracking, heatmaps, session replays
- **Integrations**: CRM, email marketing, payment processors
- **Templates**: Pre-built templates for common use cases
- **Mobile Optimization**: Responsive design by default
- **Speed Optimization**: Built-in performance optimizations

### Market Gaps Landing-Manager Fills
1. **Local-First**: No SaaS fees, runs on user's infrastructure
2. **Agent-Native**: AI agents can customize landing pages programmatically
3. **Scenario Generation**: Creates full deployable scenarios, not just pages
4. **Ecosystem Integration**: Native integration with Vrooli resources and scenarios
5. **Subscription Verification**: Built-in verification API for app access control
6. **Meta-Scenario Pattern**: Factory pattern enables infinite landing page scenarios

---

## 3. Technical Research

### Template Architecture Patterns

#### 1. React + Vite + TailwindCSS
- **Rationale**: Fast builds, modern DX, excellent performance
- **Trade-offs**: Requires Node.js ecosystem knowledge
- **Landing-Manager Fit**: ✅ Perfect for generated landing pages

#### 2. Go API Backend
- **Rationale**: High performance, simple deployment, strong typing
- **Trade-offs**: Less dynamic than Node.js for rapid prototyping
- **Landing-Manager Fit**: ✅ Excellent for API + metrics + payments

#### 3. PostgreSQL Storage
- **Rationale**: ACID compliance, JSONB for flexible schema
- **Trade-offs**: More overhead than SQLite, less than MongoDB
- **Landing-Manager Fit**: ✅ Required for multi-variant data and analytics

### A/B Testing Approaches

#### Variant Selection Strategies
1. **URL Parameter**: `?variant=control` (simple, testable)
2. **localStorage**: Sticky variant per browser (user consistency)
3. **Server-Side**: API assigns variant by user_id (requires auth)
4. **Cookie-Based**: Similar to localStorage but server-readable

**Landing-Manager Approach**: Hybrid (URL override → localStorage → API weight-based)

#### Metrics Collection
- **Client-Side Tracking**: JavaScript events (page views, clicks, scrolls)
- **Server-Side Tracking**: API logs (form submissions, conversions)
- **Session Management**: Track user journey across visits

### Payment Integration Patterns

#### Stripe Checkout Models
1. **Hosted Checkout**: Redirect to Stripe-hosted page (easiest)
2. **Embedded Checkout**: Stripe-hosted form embedded in page (balanced)
3. **Custom Payment Form**: Full control with Stripe Elements (complex)

**Landing-Manager Approach**: Hosted checkout for simplicity, with webhook-based subscription tracking

#### Subscription Verification API Models
1. **Token-Based**: App passes JWT, landing page verifies subscription
2. **User ID Lookup**: App passes user_id, landing page queries DB
3. **Webhook Push**: Landing page pushes subscription events to app

**Landing-Manager Approach**: Model 2 (User ID lookup) + webhook for updates

---

## 4. Aesthetic Guidelines Research

### Frontend Aesthetics (`<frontend_aesthetics>` compliance)

#### Anti-Patterns to Avoid (Generic "AI Slop")
- ❌ Inter/Roboto/Arial default fonts
- ❌ Flat solid color backgrounds
- ❌ Generic grid layouts
- ❌ Low-contrast pastel colors
- ❌ Default shadcn theme without customization

#### Best Practices for Distinctive Design
- ✅ Custom font stacks (e.g., Sora, Space Grotesk, Work Sans)
- ✅ Layered gradients or textured backgrounds
- ✅ Bold, high-contrast color schemes
- ✅ Asymmetric layouts with visual hierarchy
- ✅ CSS variable-based theming for brand consistency

#### Design Inspiration References
1. **Vercel** (vercel.com) - Clean, modern, performance-focused
2. **Linear** (linear.app) - Minimalist, fast, distinctive typography
3. **Raycast** (raycast.com) - Bold colors, strong branding
4. **Supabase** (supabase.com) - Developer-friendly, vibrant
5. **Cal.com** (cal.com) - Professional, trustworthy, accessible

---

## 5. External References

### Payment Processing
- [Stripe Checkout Documentation](https://stripe.com/docs/checkout)
- [Stripe Webhooks Guide](https://stripe.com/docs/webhooks)
- [SCA Compliance](https://stripe.com/docs/strong-customer-authentication)

### A/B Testing
- [Optimizely AB Testing Guide](https://www.optimizely.com/optimization-glossary/ab-testing/)
- [Statistical Significance Calculator](https://abtestguide.com/calc/)
- [Bayesian vs. Frequentist Testing](https://cxl.com/blog/bayesian-frequentist-ab-testing/)

### Analytics & Metrics
- [Google Analytics 4 Event Tracking](https://developers.google.com/analytics/devguides/collection/ga4/events)
- [Mixpanel Funnel Analysis](https://mixpanel.com/blog/funnel-analysis/)
- [Amplitude Product Analytics](https://amplitude.com/blog/product-analytics)

### Performance & Accessibility
- [Lighthouse Performance Scoring](https://web.dev/performance-scoring/)
- [WCAG 2.1 Guidelines](https://www.w3.org/WAI/WCAG21/quickref/)
- [Core Web Vitals](https://web.dev/vitals/)

### Legal & Compliance
- [GDPR Cookie Consent](https://gdpr.eu/cookies/)
- [CCPA Compliance](https://oag.ca.gov/privacy/ccpa)
- [Stripe PCI Compliance](https://stripe.com/docs/security/guide)

---

## 6. Risk Assessment

### Technical Risks

#### 1. Template Maintenance Burden
- **Risk**: Templates may become outdated as web standards evolve
- **Mitigation**: Version templates, provide migration guides
- **Severity**: Medium

#### 2. Agent Customization Quality
- **Risk**: AI agents may generate poor-quality or broken customizations
- **Mitigation**: Preview mode, rollback capabilities, validation checks
- **Severity**: High

#### 3. Port Allocation Conflicts
- **Risk**: Generated landing page scenarios may conflict with existing services
- **Mitigation**: Use Vrooli lifecycle system for dynamic port allocation
- **Severity**: Low (Vrooli handles this)

### Business Risks

#### 1. Market Saturation
- **Risk**: Many landing page builders exist
- **Mitigation**: Focus on unique value props (local-first, agent-native, ecosystem integration)
- **Severity**: Medium

#### 2. Stripe Dependency
- **Risk**: Stripe API changes could break payment flows
- **Mitigation**: Version Stripe SDK, monitor deprecation notices
- **Severity**: Low

### Operational Risks

#### 1. Database Schema Evolution
- **Risk**: Landing page scenarios may need schema migrations over time
- **Mitigation**: Include migration system in generated scenarios
- **Severity**: Medium

#### 2. A/B Test Statistical Validity
- **Risk**: Users may draw conclusions from insufficient sample sizes
- **Mitigation**: Display confidence intervals, require minimum sample size
- **Severity**: Medium

---

## 7. Open Questions

### For Future Iterations
1. Should landing-manager provide a visual drag-and-drop builder, or rely on agent customization?
   - **Current Decision**: Agent customization for MVP, consider visual builder in P2

2. How should video content be optimized for performance?
   - **Current Decision**: Support external embeds (YouTube, Vimeo) for MVP, defer self-hosted to P2

3. Should analytics be real-time or batch-processed?
   - **Current Decision**: Near-real-time (5-minute lag acceptable) for MVP

4. How to handle GDPR/CCPA compliance for analytics tracking?
   - **Current Decision**: Document compliance requirements, provide cookie banner component

5. Should landing-manager support multiple templates in MVP?
   - **Current Decision**: One template (`saas-landing-page`) for MVP, more in P1

---

## 8. Next Steps for Improvers

### Immediate Priorities (P0)
1. Implement template generation CLI (`vrooli scenario template list`)
2. Build admin portal authentication system
3. Create A/B variant selection logic (URL → localStorage → API)
4. Integrate Stripe checkout and webhook handling

### Short-Term (P1)
1. Build analytics dashboard with variant breakdown
2. Implement agent customization system
3. Create video demo section component
4. Add Lighthouse performance monitoring

### Long-Term (P2)
1. Visual drag-and-drop builder
2. Multi-template support
3. Advanced A/B statistical analysis
4. Cross-landing-page analytics aggregation

---

**Research Status**: ✅ Complete
**Uniqueness Verified**: ✅ Yes
**Market Fit Validated**: ✅ Yes
**Technical Feasibility**: ✅ Confirmed
**Risk Assessment**: ✅ Documented
