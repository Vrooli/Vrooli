---
id: vrooli-conversion-landing
version: 0.2.0
name: Vrooli Conversion Landing
description: High-converting landing pages for scenarios, bundles, apps, downloads, demos, and waitlists.
colors:
  primary: "#f97316"
  secondary: "#38bdf8"
  neutral: "#07090f"
  surface: "#0f172a"
  on-surface: "#f3f4f6"
  error: "#ef4444"
  success: "#10b981"
  warning: "#fbbf24"
typography:
  headline-lg:
    fontFamily: Space Grotesk
    fontSize: 48px
    fontWeight: "700"
    lineHeight: 1.05
    letterSpacing: 0em
  body-lg:
    fontFamily: Inter
    fontSize: 17px
    fontWeight: "400"
    lineHeight: 1.65
    letterSpacing: 0em
  body-md:
    fontFamily: Inter
    fontSize: 16px
    fontWeight: "400"
    lineHeight: 1.55
    letterSpacing: 0em
  label-md:
    fontFamily: Inter
    fontSize: 14px
    fontWeight: "700"
    lineHeight: 1.2
    letterSpacing: 0em
rounded:
  md: 1.5rem
  lg: 1.75rem
  full: 9999px
spacing:
  touch: 44px
  container-max: 1200px
  section: 6rem
  gutter: 1.5rem
components:
  button-primary:
    backgroundColor: "{colors.primary}"
    textColor: "#07090f"
    typography: "{typography.label-md}"
    rounded: "{rounded.full}"
    height: "{spacing.touch}"
    padding: 0 1.25rem
  button-primary-loading:
    backgroundColor: "{colors.primary}"
    textColor: "#07090f"
    typography: "{typography.label-md}"
    rounded: "{rounded.full}"
    height: "{spacing.touch}"
    padding: 0 1.25rem
  button-disabled:
    backgroundColor: "#1e293b"
    textColor: "#64748b"
    typography: "{typography.label-md}"
    rounded: "{rounded.full}"
    height: "{spacing.touch}"
    padding: 0 1.25rem
  input-error:
    backgroundColor: "#1f1111"
    textColor: "{colors.error}"
    typography: "{typography.body-md}"
    rounded: "{rounded.md}"
    padding: 1rem
  alert-error:
    backgroundColor: "#1f1111"
    textColor: "{colors.error}"
    typography: "{typography.body-md}"
    rounded: "{rounded.md}"
    padding: 1rem
  toast-success:
    backgroundColor: "#06281f"
    textColor: "{colors.success}"
    typography: "{typography.body-md}"
    rounded: "{rounded.md}"
    padding: 0.875rem
  empty-state:
    backgroundColor: "#1e2433"
    textColor: "#94a3b8"
    typography: "{typography.body-md}"
    rounded: "{rounded.md}"
    padding: 1.5rem
  skeleton:
    backgroundColor: "#1e2433"
    rounded: "{rounded.md}"
    height: 1rem
  inline-progress:
    backgroundColor: "#431407"
    textColor: "{colors.primary}"
    typography: "{typography.body-md}"
    rounded: "{rounded.full}"
    padding: 0.25rem 0.75rem
  retry-action:
    backgroundColor: "transparent"
    textColor: "{colors.secondary}"
    typography: "{typography.label-md}"
    rounded: "{rounded.full}"
    height: "{spacing.touch}"
    padding: 0 1rem
tokens:
  color:
    background: "#07090f"
    surfacePrimary: "#0f172a"
    surfaceMuted: "#1e2433"
    surfaceAlt: "#f6f5f2"
    textPrimary: "#f3f4f6"
    textSecondary: "#94a3b8"
    textMuted: "#64748b"
    accentPrimary: "#f97316"
    accentSecondary: "#38bdf8"
    success: "#10b981"
    warning: "#fbbf24"
    danger: "#ef4444"
    borderSubtle: "rgba(255,255,255,0.08)"
  typography:
    headlineFamily: "Space Grotesk, Inter, ui-sans-serif, system-ui, sans-serif"
    bodyFamily: "Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, Segoe UI, sans-serif"
    bodySize: "17px"
    bodyLineHeight: "1.65"
  radius:
    control: "9999px"
    card: "1.5rem"
    panel: "1.75rem"
  layout:
    containerMaxWidth: "1200px"
    gridColumns: 12
    sectionSpacing: "clamp(5rem, 10vw, 10rem)"
constraints:
  primaryPurpose: "conversion"
  defaultAesthetic: "premium-b2b-case-study"
  primarySurface: "public-landing-page"
  secondarySurface: "admin-portal"
  onePrimaryConversionGoal: true
  variantDriven: true
---

# Vrooli Conversion Landing Design

`DESIGN.md` is the source of truth for landing pages generated to sell, validate, or capture demand for a specific scenario, bundle, app, downloadable product, demo, waitlist, or offer. Stack-specific adapters may translate this contract into CSS, Tailwind, runtime style packs, section presets, or future native targets, but adapters must not redefine the design language.

This is not the general Vrooli marketing-site design. It is a conversion landing-page system for productized offers.

## How To Read This Document

This file mixes two kinds of guidance, and the distinction matters.

- **Binding contract** (must follow): tokens, palette discipline, typography scale, premium-B2B feel target, the "one primary conversion goal per variant" rule, analytics/instrumentation preservation, accessibility/trust floors, mobile-first performance, honest proof rules, and the do/don't list.
- **Illustrative examples** (shape, not checklist): the example component list (hero / sticky header / feature blocks / proof strip / artifact panels / pricing / downloads / FAQ / final CTA), the example section cadence, and any specific copy examples. Use whichever sections and components your offer actually needs — add or omit freely, matched to persona, traffic source, and conversion style.

Rule of thumb: this design tells you *how* a landing page should look, prove, and convert; *which* sections, components, and copy your specific variant uses is governed by the offer and the experiment.

## Intent

Vrooli Conversion Landing exists to move a specific visitor toward a specific action with as little uncertainty and friction as possible. The page should communicate what the offer is, who it is for, why it matters, why it can be trusted, and what the visitor should do next within the first viewport.

The default visual direction is a premium B2B case-study page: structured, artifact-rich, confident, and measurable. It should feel closer to a well-produced product launch or implementation case study than a generic SaaS gradient page. The strongest local precedent is the landing-page business suite's Clause-inspired style: charcoal foundations, orange/cyan accents, Space Grotesk headlines, Inter body copy, layered product panels, proof artifacts, and restrained pill CTAs.

## Conversion Model

Every variant has one primary conversion goal. Examples include checkout, booking a demo, downloading an app, joining a waitlist, requesting an audit, starting a trial, or opening a gated installer. Secondary CTAs may exist for different readiness stages, but they must be visually subordinate and must not create decision paralysis.

The page must support measurable optimization. Design choices should preserve analytics tagging, variant attribution, CTA tracking, scroll depth, download events, pricing events, and funnel-stage reporting. A/B variants may change copy, section order, CTA labels, visual emphasis, proof placement, and offer framing. They should not accidentally remove instrumentation or make results impossible to compare.

Good landing variants are tailored by:

- **Persona:** who the page is speaking to.
- **Job to be done:** what the visitor wants to accomplish.
- **Conversion style:** self-serve checkout, demo-led, founder/operator call, waitlist, resource download, or app download.
- **Traffic source:** search, paid social, email, retargeting, direct link, partner referral, or organic.
- **Awareness level:** unaware, problem-aware, solution-aware, comparison-shopping, or ready to buy.

## Above The Fold

The first viewport must answer three questions quickly:

- What is this?
- Who is it for?
- Why should I care now?

The hero should include a specific outcome-led headline, supporting copy that names the offer and audience, one primary CTA, and a trust or proof cue close to the CTA. The hero must include a real or believable product artifact: screenshot, dashboard panel, app frame, process diagram, metric card, installation surface, case-study preview, or workflow timeline. Do not use abstract gradient art as the primary proof.

Primary CTA placement must work on desktop and mobile. On mobile, stack CTAs cleanly, keep the primary CTA visible early, and avoid pushing the conversion action below ornamental media.

## Section Cadence

The default section arc is:

1. Hero with primary CTA and proof cue.
2. Product artifact or demo preview.
3. Outcome-oriented feature blocks.
4. Social proof, logos, testimonials, or credible usage metrics.
5. Pricing, plan comparison, download entitlement, or offer details.
6. Objection handling through FAQ, security, implementation, compatibility, or support notes.
7. Final CTA that repeats the primary conversion goal.
8. Footer with minimal exits and required legal/support links.

This cadence may change by variant, but every page should still build a clear path from attention to confidence to action. Long pages are acceptable for complex or high-trust offers; short pages are acceptable for low-friction downloads or waitlists. Length should match the amount of objection handling required.

## Visual Language

Use a restrained premium palette. The default is dark case-study chrome with light editorial sections where they clarify pricing, screenshots, or proof. Use one primary accent and one support accent. Orange is the default conversion accent; cyan is the default annotation or technical-support accent; green is reserved for proof, success, entitlement, or availability.

Prefer artifacts over decoration:

- real screenshots
- device frames
- layered dashboards
- metrics cards
- product previews
- process timelines
- pricing cards
- testimonial cards with names/roles/companies
- brand or implementation strips
- downloadable asset previews

Avoid generic mesh gradients, bokeh blobs, decorative orbs, AI-magic illustrations, and stock-like atmosphere. Subtle noise, matte panels, fine borders, and disciplined shadows are allowed when they make artifacts feel tangible.

## Copy And Messaging

Lead with outcomes, not vague capability claims. Avoid "AI-powered" as the main value proposition unless the buyer benefit is concrete. Claims should be specific, defensible, and tied to the offer.

CTA text should be action-specific and value-oriented. Prefer "Book a live review", "Download the macOS app", "Start the $1 intro", "See the ROI walkthrough", or "Join the launch list" over generic "Get Started" when the offer allows it.

Testimonials should be specific. Strong proof includes named people, roles, companies, photos where available, before/after states, time saved, revenue impact, risk reduction, download counts, launch speed, or conversion lift. Do not invent fake proof. If real proof is not available, use honest proxy proof such as implementation screenshots, benchmark notes, security details, or founder/operator context.

## Variants And Configuration

`DESIGN.md` defines the canonical design contract. Runtime configuration such as `.vrooli/styling.json`, `.vrooli/style-packs/*.json`, `.vrooli/variant_space.json`, and `.vrooli/variants/*.json` instantiate that contract for a specific generated landing page.

Agents may use configuration to change section order, copy, CTA labels, persona targeting, visual emphasis, image slots, and style-pack values. They must not treat configuration as permission to abandon the design contract. If a generated scenario intentionally changes its landing design language, update the root `DESIGN.md` first.

Configuration should make landing pages testable:

- Variant axes should describe persona, job to be done, conversion style, and traffic intent.
- Section IDs and CTA IDs should remain stable enough for analytics.
- Style packs should document influences, mood, palette, layout, component kits, imagery slots, and allowed randomization.
- New section types need schema, renderer, analytics coverage, and design guidance.

## Components

Preferred public landing components:

- **Hero:** outcome headline, supporting copy, proof cue, CTA pair, artifact panel.
- **Sticky header:** brand, minimal anchor nav, one CTA. Avoid broad site navigation for paid traffic variants.
- **Feature blocks:** outcome-first, not feature inventory.
- **Proof strip:** logos, user counts, ratings, testimonial quote, metric, or security badge near decision points.
- **Artifact panels:** screenshots, dashboard stacks, device frames, process diagrams, preview posters.
- **Pricing:** clear plan differences, highlighted recommended plan, honest terms, low-friction CTA.
- **Downloads:** platform-specific actions, entitlement notes, install requirements, store links.
- **FAQ:** direct objection handling around price, setup, compatibility, support, security, cancellation, and implementation.
- **Final CTA:** repeats the primary conversion goal with a short confidence-building line.

The admin portal may use the standard operational-console language because it is a working application, not the visitor-facing landing surface. The public page and admin portal can share tokens, but their success criteria differ.

## Mobile And Performance

Design mobile first. Landing traffic is often mobile-heavy, and poor mobile performance damages conversion. The mobile first viewport must preserve the headline, value proposition, proof cue, and primary CTA without requiring pinch zoom or horizontal scroll.

Mobile forms should use the fewest fields possible. Complex conversions should use progressive profiling or multi-step flows rather than a long first form. CTAs need large touch targets, clear spacing, and safe-area awareness.

Performance is part of the design. Optimize images, avoid decorative heavy effects, keep animation restrained, and protect Core Web Vitals. If a visual asset does not increase clarity, trust, or conversion, it is expendable.

## Feedback & State

Conversion surfaces must never leave the visitor unsure whether a click, form submit, checkout step, booking request, download, or gated action worked. Loading, submitting, success, validation-error, request-error, payment-error, unavailable, sold-out, waitlisted, retrying, and confirmation states are part of the landing-page contract.

Primary CTAs should acknowledge interaction immediately. Forms should show field-level validation, preserve entered values on failure, and provide a concise form-level error summary when the submit action fails. Checkout, booking, download, and demo-request flows should explain what happened, what was saved, whether the visitor needs to retry, and what the next step is.

Use inline feedback near the conversion action when the visitor must decide what to do next. Use toasts for lightweight confirmations only. Never rely on silent analytics events, disabled buttons, console errors, or page reloads as user feedback. Error and success states must preserve analytics attribution, CTA IDs, variant IDs, and funnel-stage reporting.

## Request Lifecycle

For every analytics event, lead capture, checkout session, booking request, download entitlement, email signup, variant fetch, pricing fetch, or admin save, design the lifecycle deliberately: idle, pending, success, failure, retrying, and disabled/unavailable. Slow conversions should show stable pending copy and avoid layout shift around the CTA.

If a conversion action fails after data was partially accepted, say so clearly and offer a safe retry or alternate contact path. If instrumentation fails but the user action succeeds, do not block the user; log or mark the instrumentation failure separately. If the user action fails but instrumentation succeeds, show the user-facing failure and preserve the form state.

## Accessibility And Trust

Maintain readable contrast, visible focus, keyboard access, semantic headings, descriptive alt text, and reduced-motion support. Do not hide critical information in images. Pricing, terms, download requirements, and security claims must be readable as text.

Trust is part of UX. Avoid fake logos, vague testimonials, hidden pricing for self-serve offers, unclear cancellation language, misleading scarcity, dark patterns, or visual tricks that create accidental purchases. For enterprise/demo-led offers, it is acceptable to route to sales, but the reason and next step should be clear.

## Do's And Don'ts

### Do

- Start by defining the offer, audience, traffic source, and primary conversion goal.
- Make the above-fold message pass a five-second clarity test.
- Put the primary CTA in the hero and repeat it near the end.
- Place proof near decision points, especially near hero and pricing CTAs.
- Use real product artifacts instead of decorative filler.
- Keep variants measurable and tied to analytics.
- Use configuration to tune copy, section order, and emphasis without breaking the design contract.
- Design pending, success, validation-error, request-error, retry, and confirmation states for every conversion action.
- Preserve form input, CTA attribution, variant IDs, and analytics semantics through loading and failure states.

### Don't

- Describe this as the general Vrooli marketing-site design.
- Give one landing page several competing primary goals.
- Hide the primary CTA below large decorative media on mobile.
- Use generic testimonials, fake logos, or unverifiable claims.
- Let gradients, blobs, or abstract art replace product proof.
- Add broad navigation exits to paid-traffic variants unless the experiment explicitly tests that.
- Remove event tracking or stable section IDs while restyling.
- Leave visitors unsure whether checkout, booking, signup, download, or lead capture succeeded.
- Use disabled CTAs, page reloads, or console errors as substitutes for visible feedback.
