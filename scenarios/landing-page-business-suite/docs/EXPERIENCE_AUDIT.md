# Experience Architecture Audit – Landing Page React Vite Template

> **Note:** This document captures UX improvements made during development. It serves as a reference for understanding design decisions and identifying future opportunities.

## Purpose Statement

Landing Manager's runtime lets marketers spot conversion signals and react without touching the factory—Analytics explains what's happening, Customization lets them adjust variants/sections, and the public landing should guide prospects directly to a CTA or download.

## Personas & Key Jobs

### Experiment Operator (ops/marketing)
- Check live traffic source
- Find red/yellow variants
- Open analytics for the right time range

### Content Author
- Locate the variant or section that needs edits
- Update copy/weighting
- Preview updates on the public landing

### Prospect/Visitor
- Skim the public page
- Jump to the section they care about (features, pricing, downloads)
- Click the hero CTA

## Flow Insights: Current vs. Ideal

| Issue | Current State | Ideal State |
|-------|---------------|-------------|
| Admin home navigation | Ops users landed on `/admin` with two vague buttons. They needed to guess that "Analytics" tells them what happened and "Customization" fixes it. | Home screen should state the two jobs explicitly and give a preview link to validate the public experience. |
| Variant discovery | Content authors had to scan dozens of variant cards; stale or underperforming variants were only hinted at in Experience Ops. | Let them filter the grid down to the problematic experiments directly from the same signals. |
| Public navigation | Visitors on `/` had no navigation guidance—they had to scroll from hero → features → pricing manually, and admins couldn't see runtime health without dev tools. | Provide a sticky header that surfaces runtime state, anchor links, and a persistent CTA. |
| Analytics context | Once an ops user filtered analytics to a variant/time range, there was no persistent indicator tying that view back to runtime state. | Keep the "what am I looking at?" context in the viewport and links straight to customization/preview. |
| Variant attention signals | Variant cards didn't show *why* a variant needed attention—just that filters existed. | Explain the reason (stale copy, never customized, lowest conversion) inline so the author knows the next edit to ship. |

## Changes Implemented

### Admin Experience Guide (AdminHome)
Added a purpose banner plus three quick flows (audit performance, ship a variant, preview landing) with direct buttons so first-time admins know exactly where to start.

### Variant Filters + Needs-Attention Focus (Customization)
Introduced a search/attention filter bar, applied counts to the active grid, and wired the Needs Attention panel to focus the list, shrinking the "find the broken experiment" loop to one click. Added `highlight variant` and `clear filters` selectors for automation.

### Section-Focused Deep Links
Customization now resolves `focusSectionId` / `focusSectionType` query params so AdminHome, Analytics, and Ops widgets can jump straight into the Section Editor (defaulting to hero) without forcing users to hunt for the right block.

### Public Landing Navigation Rail
Inserted `LandingExperienceHeader` with runtime pills, anchor navigation (desktop + mobile), and a sticky hero CTA button so visitors, operators, and agents can jump to sections instantly while seeing whether fallback copy is active.

### Analytics Focus Rail
Added `AnalyticsFocusBanner` so ops users always know which variant/time range they're analyzing, how it compares to the live runtime, and can reset filters or jump to customization/preview without scrolling.

### Variant Status Storytelling
Added `VariantListSummary` plus inline badges on each variant card that call out last edit time and attention reasons ("Stale · 12d", "Lowest conversion"). This turns the Customization grid into a to-do list instead of an undifferentiated catalog.

### Download CTA Surfacing
The sticky landing header now includes a dedicated download button (when assets exist) so end-users chasing installers don't need to scroll through the marketing narrative to reach entitlements.

### Admin Health Digest (AdminHome)
Snapshot panel now fetches variant and analytics data to surface live runtime state, traffic allocation, and the highest-priority attention variant with direct actions ("Open analytics", "Review in customization", "Adjust weights") so ops personas can triage before diving into a mode.

### Cross-Surface Deep Links
Customization accepts `?focus=<slug>` to auto-filter and scroll to that variant, letting AdminHome (and future surfaces) highlight the broken experiment in one click instead of forcing users to reapply filters manually.

## Opportunities for Future Loops

1. **Customization > Section intent signals** – Surface which section triggered an alert (hero vs. pricing vs. download rail) so deep links can target that block instead of defaulting to hero when context is missing.

2. **Analytics > Saved views** – Persist filter presets (e.g., "Last 30 days · Variant Bravo") and surface them on AdminHome to eliminate the repeated query building for ops personas.

3. **Public landing trust rails** – Add lightweight trust indicators (customer logos, uptime badges) to the sticky header or hero so prospects see proof before scrolling.

4. **Health event log** – Capture recent runtime events (traffic allocation changes, fallback activation, agent jobs) in the AdminHome digest so follow-up actions can be audited without leaving the portal.
