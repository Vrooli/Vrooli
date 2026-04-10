# Experience Architecture Audit

**Last Updated:** 2026-01-26

## Purpose
This scenario helps admins manage a SaaS landing page and its pricing/billing configuration so they can publish a live, Stripe-backed subscription experience quickly and safely.

## Core Personas & Jobs
- **First-time admin**: understand where pricing lives, connect Stripe, and publish a clean pricing table.
- **Billing/ops admin**: reconcile Stripe pricing with local plan configuration, resolve conflicts safely.
- **Marketing/editor**: adjust plan copy/visibility without breaking billing integrity.

## Key Flows (Current vs Ideal)
- **Job: Import Stripe pricing**
  - **Current (pre-fix):** Plan Management → Import from Stripe → per-row action dropdowns (import/overwrite/skip) → import.
  - **Ideal:** Plan Management → Import from Stripe → select prices with checkboxes → explicit conflict warnings → import.

## Friction Points
- **Cognitive:** Action dropdowns required users to interpret three modes per price and understand overwrite behavior.
- **Mechanical:** Repetitive dropdown interactions for large catalogs.
- **Discoverability:** Overwrite consequences were easy to miss.

## Improvements Implemented (2026-01-26)
- Replaced action dropdowns with checkbox selection (skip = unselected).
- Added explicit conflict warnings and overwrite messaging.
- Added product-level selection and bulk actions (select new/conflicts, clear selection).
- Added filter tabs and search to navigate large Stripe catalogs.
- Added overwrite confirmation checkbox when conflicts are selected.
- Clarified copy about Stripe being the source of truth for overwrites.

## Additional Opportunities (Not Implemented)
 - Provide inline preview of mapped local plan fields before overwriting.
