# Add-on: Elder Care

**SKU ID:** `elder-care`
**Status:** `candidate`
**Parent bundles:** `lifestyle`
**Revisit trigger:** *"Revisit when the lifestyle bundle is `active` AND ≥3 distinct prospects explicitly request elder-care tooling."*

## Hypothesis

Households caring for aging parents or grandparents face a set of recurring coordination and monitoring needs that a local-first AI system is well-suited for — precisely because privacy, data control, and reliability matter more here than for most consumer software.

Candidate capabilities:

- **Medication reminders and adherence tracking**
- **Home safety monitoring** — falls, wandering, unusual activity patterns (on-device, not cloud)
- **Family communication** — shared status, delegated tasks, incident logs
- **Medical appointment coordination** — scheduling, transport, follow-up
- **Health trend tracking** — weight, blood pressure, sleep, with alerts on out-of-range patterns

## Why this is a candidate

1. The lifestyle bundle itself is a candidate. Add-ons cannot precede their parent.
2. Elder-care has meaningful regulatory surface (HIPAA in the US if any clinical integration, state-level licensing for adjacent services, consent-and-capacity considerations).
3. The emotional/trust bar is extremely high — software that drops a medication reminder is worse than no software. This add-on should not activate until the underlying Vrooli runtime has proven reliability with lower-stakes scenarios.

## Why it remains valuable to capture

Elder care is a rapidly growing market segment (aging population in developed countries) and a category where local-first, privacy-preserving software has a real structural edge over cloud SaaS. Keeping this candidate documented means when capability and trust mature, we have a dormant target ready rather than scrambling to invent one.

## Things to track while candidate

- Lifestyle bundle activation progress
- Prospects or existing subscribers mentioning elder-care needs
- Regulatory developments (state-by-state caregiver rules)
- Adjacent scenarios that mature and reuse well for elder-care (medication tracking, household safety, family communication)

## If and when promoted

First-pass questions at activation:

1. Which slice first — monitoring, coordination, medication, or communication?
2. Do we sell to the aging individual, their family member, or a professional caregiver? (Buyer and user differ here.)
3. What's the hardware assumption — phone-only, or does this require dedicated household hardware (Tier 4 relevance)?
4. What's the liability posture? This category has more downside exposure than business software.
