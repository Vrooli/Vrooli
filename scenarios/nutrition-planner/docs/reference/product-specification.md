# Daily — complete product and implementation specification

**Version:** 1.0 · **Prepared:** 2026-09-18  
**Audience:** A local coding agent and the developer reviewing its work.  
**Format:** One self-contained implementation handoff. No prototype, conversation, screenshot, second document, or external integration is required to understand the intended product.  
**Working product name:** Daily. The name is provisional; keep branding configurable.  
**Status:** Ready to begin implementation using the recommended defaults below. The user commissioned this specification after reviewing and approving the direction of an interactive UX prototype. Detailed engineering choices in this document are recommendations, not claims that the user separately approved every choice.

> Build an application that makes it easy to eat in a way that fits the user's nutritional goals, dietary restrictions, budget, available effort, and appetite for variety. The ordinary interaction should be choosing whether tonight's suggestion sounds good—not repeatedly researching food, maintaining spreadsheets, or filling in long forms.

## Contents

1. [Read this first: implementation mandate](#1-read-this-first-implementation-mandate)
2. [Product purpose and confirmed needs](#2-product-purpose-and-confirmed-needs)
3. [Release scope, product boundaries, and defaults](#3-release-scope-product-boundaries-and-defaults)
4. [User journeys and information architecture](#4-user-journeys-and-information-architecture)
5. [Visual design and interaction system](#5-visual-design-and-interaction-system)
6. [Onboarding, kitchen configuration, and targets](#6-onboarding-kitchen-configuration-and-targets)
7. [Today, weekly plans, priorities, and feedback](#7-today-weekly-plans-priorities-and-feedback)
8. [Meal collection, flexible capture, and editing](#8-meal-collection-flexible-capture-and-editing)
9. [Recipe map, reading view, cooking mode, and recipe printing](#9-recipe-map-reading-view-cooking-mode-and-recipe-printing)
10. [Grocery list, price book, and pantry UX](#10-grocery-list-price-book-and-pantry-ux)
11. [Import, export, backups, and printing](#11-import-export-backups-and-printing)
12. [Domain model, contracts, and invariants](#12-domain-model-contracts-and-invariants)
13. [Nutrition, supplements, and data quality](#13-nutrition-supplements-and-data-quality)
14. [Planning engine and explainable optimization](#14-planning-engine-and-explainable-optimization)
15. [Cost, inventory, leftovers, and shopping calculations](#15-cost-inventory-leftovers-and-shopping-calculations)
16. [AI assistance, data acquisition, and recurring jobs](#16-ai-assistance-data-acquisition-and-recurring-jobs)
17. [Architecture and application contracts](#17-architecture-and-application-contracts)
18. [Persistence, concurrency, privacy, and failure recovery](#18-persistence-concurrency-privacy-and-failure-recovery)
19. [Performance, operations, and commercial readiness](#19-performance-operations-and-commercial-readiness)
20. [Worked examples and embedded implementation fixtures](#20-worked-examples-and-embedded-implementation-fixtures)
21. [Acceptance criteria and verification strategy](#21-acceptance-criteria-and-verification-strategy)
22. [Delivery sequence and definition of done](#22-delivery-sequence-and-definition-of-done)
23. [Open decisions, assumptions, and deferred possibilities](#23-open-decisions-assumptions-and-deferred-possibilities)
24. [Grounding and external references](#24-grounding-and-external-references)
25. [Maintaining this as the single handoff document](#25-maintaining-this-as-the-single-handoff-document)

## 1. Read this first: implementation mandate

### 1.1 What you are being asked to deliver

Implement a real, persistent application with a polished interface and a deterministic planning core. Reconstruct the UX described here, then complete the domain behavior needed to support actual daily use. The previous prototype established the interaction direction; it did not implement a complete nutrition optimizer.

The foundational loop is:

1. Learn the user's food rules, kitchen, routine, and preferences with little effort from them.
2. Capture foods, recipes, supplements, prices, and available stock, allowing incomplete information.
3. Produce a practical plan and explain the meaningful tradeoffs.
4. Make the immediate action obvious: prepare, assemble, swap, or choose a fallback.
5. Record lightweight corrections and actual consumption when the user chooses to provide them.
6. Improve subsequent suggestions without turning the user's life into a bookkeeping task.

A dinner planner with sample numbers is an intermediate milestone. The first personally useful release must also represent the rest of the day's food and supplements, configurable nutrition targets, and honest data completeness. The original goal must survive the implementation of the attractive interface.

### 1.2 Instructions to the implementing agent

- Read this document fully once, then use the section links and requirement IDs while implementing individual slices.
- Inspect the actual repository, its instructions, application conventions, authentication, persistence, tests, and available integrations before choosing framework details.
- Preserve the product's low-effort experience. Put advanced information behind an intentional disclosure, without withholding information necessary to make a decision.
- Implement complete vertical slices: UI action, domain operation, persistence, recalculation, error behavior, and meaningful verification.
- Use the release boundaries in Section 22. Do not declare the product complete after reproducing demo screens.
- Keep source-backed data, user assertions, calculated values, and AI suggestions distinguishable.
- Treat external data, uploaded files, and model output as input to validate. They do not authorize changes to the user's rules or account.
- Never invent personal nutrition targets, allergies, spending baselines, prices, or external account access.
- Use explicit development fixtures for demos. A fresh real workspace must not silently inherit sample pantry stock or the fictional user's targets.
- If a dependency is unavailable, implement and expose its supported fallback. Do not leave a button claiming a capability that does nothing.
- Record implementation decisions and completion evidence in the maintenance area of this same specification when it becomes the repository's canonical copy.
- Ask about a genuine product fork only when it materially changes the deliverable and no recommended default below resolves it. Continue independent work while awaiting that decision.

### 1.3 Authority and requirement language

| Label | Meaning | How to act |
| --- | --- | --- |
| Confirmed need | Explicitly requested by the user | Preserve throughout implementation. |
| UX direction | Demonstrated in the prototype and positively received | Reproduce the described interaction and visual intent; reasonable accessibility and usability improvements are allowed. |
| Recommended default | Concrete design selected here to make implementation possible | Implement unless repository evidence or a material contradiction requires a documented adjustment. |
| Open decision | A choice that remains unconfirmed | Use the stated interim behavior; do not silently invent personal facts or business commitments. |
| Deferred | Retained product idea outside the specified early releases | Preserve extension points only where inexpensive; do not build speculative infrastructure. |

Within the selected release, **must** describes an acceptance requirement. **Should** is the preferred implementation with a documented reason needed for deviation. **May** is optional. A statement about a later release does not move that feature into the current milestone.

If requirements conflict, prioritize explicit user needs, domain correctness, trustworthy data handling, and then the default UX. Identify the conflict in the decision log. Do not weaken an exclusion or rewrite historical records to make a test pass.

### 1.4 Scope of this handoff

This document replaces the need to send the agent the living brief, prior brainstorming, the hosted prototype, or source code from that prototype. Relevant decisions, ideas, examples, and remaining questions are consolidated here. External references explain selected standards or data providers; this specification states the intended behavior independently of those references.

The destination repository is not inspected by this document's author. References to Vrooli are integration guidance to verify locally, not an assertion that a particular service, endpoint, database, or dependency already exists there.

## 2. Product purpose and confirmed needs

### 2.1 The user's situation

The originating user follows a vegan diet and tries to cover macro- and micronutrients while obtaining substantial protein. They favor meals with little preparation or cooking. A carefully planned but repetitive routine becomes boring, leading to more expensive alternatives or eating whatever is immediately available. Their original plan did not explicitly optimize cost.

Relevant purchases include protein powder, nutritional yeast, omega-3 products, and other supplements. The system must be able to represent these alongside ordinary food. Their exact products, doses, current routines, nutrition targets, and prices have not been supplied or confirmed for this project.

The user will not stick with a solution that requires continuous research or repeatedly figuring out how to make meals. This is a primary design constraint, not a secondary convenience feature.

Although the originating user would select vegan, they explicitly want a product that supports other diets and could eventually be monetized. There is no confirmed subscription price, billing provider, commercial launch date, or household-sharing requirement.

### 2.2 Confirmed requirement register

| ID | Requirement | Product consequence |
| --- | --- | --- |
| C01 | Balance nutrition, protein, effort, variety, and cost | Planning must expose configurable priorities and enforce required constraints. |
| C02 | Minimize recurring research, preparation, and decision work | Capture once, reuse information, offer immediately usable suggestions, and make feedback optional. |
| C03 | Personally vegan; support other diets | Diet is workspace configuration rather than a global code assumption. |
| C04 | Handle restrictions and allergies | Exclusions are explicit, independently editable rules with visible evidence limits. |
| C05 | Playful appliance selection | A visual multi-select kitchen setup influences which preparation methods fit. |
| C06 | Questions establish initial priorities | Onboarding translates ordinary-language answers into editable planning settings. |
| C07 | Add/edit a meal with as little or much information as available | A name-only draft is valid; enrichment can happen later. |
| C08 | Import/export and attractive printable PDFs | Portable data and print output are first-class features. |
| C09 | More intuitive recipes, including recipe infographics | The same recipe supports an ingredient/action map, reading view, and focused cooking mode. |
| C10 | Preserve the whole idea across handoff | This file contains product intent, implementation behavior, rationale, and deferred ideas. |
| C11 | Allow a local agent to implement without the hosted app | Layout, interactions, contracts, examples, and tests must be reconstructable from text. |

### 2.3 Outcomes and measurements

The product hypothesis is that affordable variety and practical fallbacks improve follow-through, which may reduce actual food spending and improve coverage of configured nutritional goals. Do not present that hypothesis as a proven savings result.

Measure, when the user permits and data exists:

- Time and interaction count needed to get a first usable plan.
- Repeated information requests and manual edits required per week.
- Suggestion acceptance and voluntary swap reasons, with an explicit unknown category for missing feedback.
- Planned versus recorded food spending, including optional purchases outside the plan.
- Active preparation, elapsed time, cleanup burden, and batch sessions separately.
- Repetition and variety by meal, flavor, texture, and meal family.
- Coverage and missing-data status for each configured nutrient target.
- Ingredient waste or corrections when voluntarily recorded.
- AI/provider cost and maintenance burden relative to the value of the automation.

Recommended usability goals: a returning user can see the next meal immediately; begin guided preparation in one action; see swap alternatives without writing a reason; and save a name-only meal without entering nutrition. These are acceptance goals, not claims about measured performance of the prototype.

## 3. Release scope, product boundaries, and defaults

### 3.1 Release definitions

| Release | Purpose | Required capabilities |
| --- | --- | --- |
| R0 — durable UX foundation | Reconstruct and improve the demonstrated experience with real persistence | Setup; all primary screens; manual meal capture/editing; recipe views; dinner planning; prices and stock entered manually; saving; JSON/CSV/PDF portability; core invariants; local demo fixtures. |
| R1 — personally useful planning | Support the originating user's actual full routine | Baseline routine capture; configurable meal slots; foods and supplements; nutrient identities/bases; full-day evaluation; real price observations; component/batch/leftover modeling; household serving count for a single profile; temporary preference modes; explainable deterministic planning. |
| R2 — low-maintenance assistance | Reduce the work of obtaining and maintaining information | AI-assisted text/link/label ingestion; provider-backed food lookup; bounded price refresh; preference inference; optional scheduled drafts and notifications. |
| R3 — commercial expansion | Make deployment and commercial operations suitable for other customers | Entitlements and usage budgets; billing if approved; commercial onboarding polish; tested subscription lifecycle; optional sharing/households if separately selected. |

R0 is a reviewable foundation. R1 is the default target for calling the first usable product complete. R2 makes the original low-maintenance ambition substantially stronger and should follow in deliberate slices. Account isolation and data ownership must exist from R0; monetization does not justify postponing them.

### 3.2 Non-goals for the initial releases

Automated checkout, autonomous supplement dosing, diagnosis or treatment, real-time global grocery-price coverage, social feeds, an unrestricted recipe marketplace, calibrated adherence prediction without training data, and a generic optimization platform are outside R0/R1. Do not implement payment-taking or place orders merely because an export or shopping list exists.

Meal recommendations are planning assistance. They must not claim allergy safety, clinical adequacy, or verified nutritional completeness beyond the data and configured policy actually evaluated.

### 3.3 Recommended default decisions

| Decision | Default | Rationale / change rule |
| --- | --- | --- |
| Initial experience | Dinner-first dashboard with an expandable full-day view | Retains the easy entry point while supporting the whole diet. |
| Planning horizon | Seven consecutive local calendar days, editable | Removes the prototype's fixed dates without demanding a calendar setup. |
| Actual diet defaults | Unselected on first visit; vegan selected in the explicit personal demo | A commercial product must not assign the founder's diet to everyone. |
| Target values | Blank until entered or explicitly accepted from a vetted reference policy | Personal goals are not known. |
| Currency/units | Editable locale settings; USD and metric for supplied demo fixtures | Store currency explicitly and preserve entered units. |
| Account model | One private workspace per authenticated user initially | Supports later commercialization without building sharing prematurely. |
| Person count | One profile with adjustable recipe and planned serving counts | Mixed-diet multi-person optimization is a separate feature. |
| Review of changes | Preview substantial replans, imports, and source replacements | Users retain control without approving every minor edit. |
| Diet/allergy rules | Required eligibility conditions | Preference weights never override them. |
| Protein/calorie targets | Configurable as required or preferred; no silent enforcement before configured | Preserve user choice and incomplete-data honesty. |
| Supplement schedules | User-defined fixed schedules; no automatic dose adjustment | Prevent the optimizer treating pills as freely adjustable nutrient variables. |
| AI | Optional adapter; deterministic paths remain useful | Avoid making every interaction depend on a model call. |
| Purchasing | Shopping plan and exports only | Financial actions require a separately designed flow. |
| Name/visual brand | Daily, cobalt/white/lime | Preserve the demonstrated direction, keep replaceable. |

### 3.4 Prototype shortcuts to replace explicitly

| Prototype behavior | Production behavior |
| --- | --- |
| A fixed September 14–20 sample week | Calendar dates in the user's timezone with week navigation. |
| Twelve illustrative meals and example nutrient totals | Clearly labeled demo fixtures plus user records and source-linked data. |
| Sample pantry automatically subtracted | Only actual workspace inventory or explicitly enabled demo inventory. |
| Dinner-only totals | Explicit dinner scope in R0; full-day planned/recorded views in R1. |
| Simple cost/effort/repetition ranking | Versioned scoring and constraint evaluation with known-data checks and plan-level effects. |
| Broad reviewed/unreviewed flag | Separate authoring, evidence, and rule-evaluation states. |
| Text parsing based mainly on headings | Conservative manual organizer first, validated assisted extraction in R2. |
| Recipe changes update a single live record | Immutable revisions and historical snapshots. |
| One saved workspace snapshot | Domain records and transactional changes, with export bundles as snapshots. |
| A generic generated bowl photo | Licensed/original assets with honest labels, or attractive ingredient typography. |
| Fixed dietary enums | Data-driven presets compiled into explicit rules, with composable custom restrictions. |

## 4. User journeys and information architecture

### 4.1 UX-IA-01 — Application shell

Top-level destinations are **Today**, **Your week**, **Groceries**, and **Meals**. **Your kitchen** opens setup/preferences. **Import / export** is globally accessible and also available contextually in Meals, Your week, Groceries, and recipes.

For R1, nutrition analysis, inventory/price details, and supplements are secondary destinations reached from relevant summaries or Your kitchen. Keep the four primary destinations stable; do not add a top-level tab for every domain entity.

Desktop: brand at left, navigation near center, kitchen and transfer actions at right. Mobile: compact header and a persistent four-item navigation area; use a bottom bar when the repository's established app pattern supports it. Give the active destination a clear label and visual state. Route state must be bookmarkable where useful; dialogs may use route/query state if consistent with the app shell.

Suggested route meanings (adapt path prefixes to the host app): `/today`, `/week?start=YYYY-MM-DD`, `/groceries?plan=id`, `/meals`, `/meals/:id`, `/kitchen`, `/nutrition`, `/data`. A refresh must preserve the meaningful destination, selected week, and recipe, not reopen an arbitrary default day.

### 4.2 Journey A — A first useful plan

1. New user enters a brief four-step setup or chooses Explore first.
2. They choose a diet approach, exclusions, and appliances.
3. Three questions establish initial cost, effort, and variety settings.
4. The final step summarizes the rules and tells them how many available recipes can currently be evaluated as fitting them.
5. With sufficient recipes, a draft week is generated. Without them, the user can add a favorite or explore clearly labeled sample content.
6. The user sees one recommended meal and simple reasons. They can start, swap, edit setup, or inspect the week.
7. R1 offers a short follow-up to capture the rest of the day and targets; it does not demand personal measurements just to explore recipes.

A first plan with incomplete data is useful if it clearly describes what has and has not been evaluated. It must not display a reassuring nutrition-coverage badge merely because seven slots are filled.

### 4.3 Journey B — Capture an existing routine

The user pastes a rough routine or adds a sequence of meal names. The application creates a review queue with detected meals, recurring slots, foods, and supplements. It asks only material questions: quantity/basis, exact product where fortification matters, and whether a repeated item is a separate consumption or the same item described twice.

An example input might mention a daily smoothie, a tofu meal, crunchy snacks, and recurring supplements. The system preserves the source text. It must not invent a smoothie recipe, product brand, serving weight, dose, or daily target. Reusable recurring slots become baseline templates after review. Later weeks can vary the flexible slots while keeping accepted anchors.

### 4.4 Journey C — A tired evening

Open Today → see the next meal → choose “Not a cooking kind of night?” → see eligible alternatives ranked for minimal active effort → compare time/cost and material target effects → select one → grocery requirements and future dependent batch uses are recalculated → begin or mark consumed. Giving a reason is optional. This can be a one-meal override without changing permanent priorities.

### 4.5 Journey D — Shopping and changes

Review a week → inspect derived groceries → see quantity needed, stock available, estimated package count, price source/date, and unknowns → check off items as shopping progresses → explicitly confirm purchases if inventory should be updated. A shopping checkmark alone is not a purchase receipt, a stock adjustment, or consumption.

If a meal changes, preserve checked states for materially unchanged line requirements. Mark increased or changed requirements for review. Show a compact summary such as “Rice increased by 150 g; chickpeas removed.” Do not clear the entire checklist on every swap.

### 4.6 Journey E — Cooking and logging

Open a recipe → choose map, read, or focused mode → adjust servings if needed → follow steps or use a timer → finish cooking. “Cooked this batch” and “I ate this” are distinct actions in R1. Confirm a default consumed portion with one action, optionally adjust it, or leave consumption unrecorded. Undo creates a correction and reverses linked inventory effects transactionally.

### 4.7 Journey F — A new week

Carry forward accepted routine anchors, current preferences, credible stock, and actual feedback. Generate a draft for the next date range, preserve locked commitments, explain meaningful changes, and allow apply/undo. A scheduled job in R2 prepares a draft without silently replacing a plan the user already arranged.

### 4.8 Journey G — Moving data

Export a complete backup or recipe collection. On import, inspect schema version, counts, unsupported fields, unresolved references, and duplicate decisions before any live write. Add meals without replacing setup, or explicitly restore a full workspace with a recoverable checkpoint. Print useful recipe and grocery PDFs independently of backup functionality.

## 5. Visual design and interaction system

### 5.1 UX-VIS-01 — Visual direction

Daily should feel calm, capable, and slightly playful. The interface gives a clear recommendation, with supporting detail available when wanted. Preserve generous whitespace, a strong cobalt primary action, warm lime highlights, white surfaces, dark blue text, and restrained metadata. Avoid turning the landing view into a wall of nutrient gauges.

Recommended design tokens, derived from the prototype:

| Token | Value / behavior |
| --- | --- |
| Canvas | `#F7F8FA` |
| Surface | `#FFFFFF` |
| Primary cobalt | `#234BD8`; hover approximately `#173FC2` |
| Primary text | `#172444` or `#1C2539` |
| Supporting text | Start near `#657086`; verify contrast for actual sizes/backgrounds |
| Border | `#E3E7ED` |
| Pale blue panel | `#EDF2FF` |
| Lime accent | `#DDF78B` with dark green text, never white text |
| Error | Dark red text plus icon and explanation; color is not the only signal |
| Radius | 8–10 px controls; 12–16 px cards; 16–20 px large dialogs |
| Spacing | 4 px base; most grouping uses 8, 12, 16, 24, 32 px |
| Main content width | Approximately 1,216 px inside a 1,312 px outer region |
| Desktop header | Approximately 88–96 px tall |
| Main page title | 40–44 px desktop, 28–32 px mobile; compact letter spacing |
| Card title | 24–30 px hero; 18–22 px collection card |
| Body | 14–16 px; line-height 1.5–1.7 |
| Secondary metadata | Normally 12–13 px; essential information must remain comfortably readable |
| Font | Repository's existing accessible sans-serif; system sans-serif is an acceptable fallback |

These values establish direction. Adjust individual values to meet contrast and accessibility requirements. Do not copy low-contrast incidental text from a screenshot as a requirement.

### 5.2 UX-VIS-02 — Layout and responsive behavior

Desktop Today uses a roughly 1.8:1 split: primary meal card on the left and compact context cards on the right. The hero contains an optional food image or an ingredient composition panel, meal identity, four key metrics, a short reason, and prominent actions. Supporting cards summarize the selected scope, kitchen rules, and the wider week.

At narrower widths, stack the main card and supporting content. Keep the main action visible without traversing a large analytics panel. Meal collections use three cards per row on wide screens, two at medium widths, and one on small screens. Week rows become compact cards with day, meal, and key metrics; do not squeeze all desktop columns into 360 px.

Dialogs have a maximum height around 90–94 dynamic viewport units and a single clear scroll area. Primary actions remain reachable with an on-screen keyboard. On mobile, a full-screen editor or sheet is preferable to a tiny centered dialog. Never scroll the whole page sideways because a recipe map is wide; confine map scrolling to its region and provide the reading alternative.

### 5.3 UX-VIS-03 — Controls and accessibility

Use the target repository's established component system. Preserve semantic buttons, labels, radio groups, checkboxes, tabs, dialogs, tables, and keyboard behavior. Accessible names must identify icon-only actions. Close dialogs with Escape, restore focus to their trigger, and trap focus only while the dialog is active.

Set a product target of WCAG 2.2 AA. Test keyboard paths, zoom/reflow, visible focus, meaningful labels, non-color status cues, and reduced motion. This is an implementation target, not a certification claim. [WCAG 2.2](https://www.w3.org/TR/WCAG22/)

Use at least a comfortable 44 px target for frequent touch actions as a product default. Motion should be short and purposeful: appliance hover/lift, selection transitions, and page appearance. Honor reduced-motion preferences. Announce save errors and import outcomes through appropriate live regions without reading every slider movement aloud.

### 5.4 UX-VIS-04 — Copy and progressive disclosure

Use reassuring, concrete copy: “Dinner, decided.”, “Save something good.”, “Your kitchen”, “What sounds better?”, “Amount not set”, and “Based on the values available.” Prefer “Fits your current rules” over “Safe for your allergy.” Avoid shame, streak pressure, or implying that eating something different is failure.

Show only material caveats near an action. Put provenance, individual nutrient lines, planner diagnostics, and source comparisons in details panels. Do not expose database, provider routing, optimization weights, or job internals in ordinary eating flows unless they help the user's decision.

When an advanced feature is not implemented in the selected release, omit it or clearly describe its available fallback. Do not display a fake progress animation, fake live price, or successful import toast without a corresponding result.

## 6. Onboarding, kitchen configuration, and targets

### 6.1 UX-ONB-01 — Four-step setup shell

Desktop: a cobalt rail approximately 240 px wide occupies the left of a 950–1,000 px dialog. It contains the Daily wordmark, “Good food. Your rules.”, a four-step progress list, and “Everything can be changed later.” The main white pane contains the active form, description, and Back/Continue actions. Mobile omits the rail and shows a compact step indicator.

Steps are **Your food**, **Your kitchen**, **Your rhythm**, and **Ready**. Setup edits are a draft until the final apply. Navigating back preserves all entered values. On first visit, Explore first allows browsing; on later edits the secondary action is Cancel. Closing with meaningful unsaved changes offers Keep editing / Discard, while an unchanged close is immediate.

Persist a resumable setup draft separately from active rules if the user leaves. Do not treat an unfinished draft allergy change as active until applied; the UI must clearly label the difference. On completion, save profile and plan revision atomically or stage the plan separately with an explicit apply state.

### 6.2 UX-ONB-02 — Food approach

Use radio cards for Vegan, Vegetarian, Pescatarian, Everything, Plant-forward, and My own way. A short description explains each. Show the actual excluded groups in an editable details area; a diet name alone is not an exhaustive policy.

Recommended presets:

| Preset | Initial rules |
| --- | --- |
| Vegan | Exclude meat/poultry, fish/seafood, dairy, eggs, honey, and identified animal-derived ingredients; display review-needed for ambiguous derivatives. |
| Vegetarian | Exclude meat/poultry, fish/seafood, and identified slaughter-derived ingredients; dairy/eggs allowed unless separately excluded. |
| Pescatarian | Exclude meat/poultry; seafood allowed unless separately excluded. |
| Everything | No diet-derived exclusions; individual restrictions still apply. |
| Plant-forward | A preference for plant-based meals; define an optional user-configured animal-meal frequency cap. No hidden prohibition. |
| My own way | Named combination of explicit exclusions and preferences. |

The list is an initial UX, not the complete set of possible diets. Gluten exclusion, low-sodium goals, cuisine preferences, ethical preferences, and macro ranges are composable settings. Do not claim certification of religious or clinical diets solely from coarse ingredient tags. Store preset versions and the expanded active rules so a future preset update cannot silently change the user's contract.

Allergies and ingredient exclusions appear below the diet cards as selectable chips plus an “Anything else off the menu?” input. Initial chip options can include peanuts, tree nuts, soy, wheat, gluten, milk, eggs, fish, shellfish, and sesame. These are product options, not a claim that all are one legally equivalent category. Distinguish wheat exclusion from gluten exclusion, and milk allergy from lactose preference/intolerance.

Represent user-defined exclusions through canonical ingredient concepts where possible, with preserved original text. Resolve ambiguous exclusions during review. “Nuts” should prompt a concise clarification rather than silently equating it with one taxonomy node. Free text must not be the only matching mechanism for imported branded ingredients.

Allergen evidence has states such as declared present, declared absent within a specified evidence scope, precautionary statement, unknown, and conflicting. A missing tag is unknown rather than proof of absence. Restricted allergens with unresolved evidence make a recipe ineligible for automatic recommendation under that profile until reviewed. User review records an assertion; it does not create a verified cross-contact guarantee.

### 6.3 UX-ONB-03 — Kitchen crew

Eight initial appliance tiles: Microwave, Stove, Oven, Air fryer, Blender, Kettle, Rice cooker, Pressure cooker. Each tile includes a recognizable line icon, label, playful subtitle, and visible selected checkbox. Selected tiles use a pale green background and dark green icon. The entire tile is clickable and keyboard operable.

Example subtitles: Microwave “The two-minute hero”; Stove “One-pan possibilities”; Oven “Let the oven do it”; Blender “Blend, sip, repeat.” Store appliances as stable capabilities, not the display text.

Allow zero appliances. Then show “We’ll start with meals you can assemble.” Do not assume access to a microwave because a recipe was previously selected. R1 adds optional refrigeration/freezer access, storage constraints, and basic-tool assumptions behind “More about my kitchen.” Ordinary hand assembly may assume utensils only if disclosed; oven and stove are not interchangeable unless a recipe has a reviewed alternative method.

Choosing an appliance should animate subtly. A playful interface must still show a clear state, accessible name, and adequate target size. Countertop-device models, wattage, and pressure settings are optional advanced information, not required onboarding fields.

### 6.4 UX-ONB-04 — Initial priorities

Ask three ordinary-language questions. Defaults below are policy values, not nutritional percentages or probability estimates:

| Question | Answer | Initial weight |
| --- | --- | --- |
| How should we think about your budget? | A little flexibility / Keep it sensible / Every dollar counts | Cost 30 / 55 / 100 |
| It's a weekday. How much cooking sounds good? | As little as possible / A few easy steps / I enjoy cooking | Effort 100 / 70 / 25 |
| How do you feel about repeat meals? | Favorites on repeat / Some familiar, some new / Keep surprising me | Variety 20 / 35 / 100 |

A first setup defaults to the middle answers if the user has not selected anything; visibly mark them. When reopening a profile with slider values between these choices, show “Custom balance” rather than pretending one answer exactly represents it. Choosing an answer explicitly changes that dimension; simply navigating through the step preserves custom values.

Ask R1 follow-up questions only as relevant: weekday active-time cap, willingness to batch cook, cleanup aversion, and an optional spending target. A high effort preference is a ranking preference; an explicit maximum active time is a constraint. Explain that distinction in normal language.

### 6.5 UX-ONB-05 — Review and apply

Show diet, exclusions, selected appliances, and priority bars. The matching-meal count must be calculated from the same eligibility service used by the planner, with counts for “fits”, “needs review”, and “excluded” available in details. If no eligible meals exist, say so and offer Add a meal or Adjust setup. Never relax a restriction to produce a nonzero count.

Applying modified rules should show a change preview when an existing plan is affected. Uneaten incompatible meals need replacement or an explicit unresolved slot; locked future meals require conflict handling. Already-recorded consumption remains historical and can be flagged in context but is never erased or retroactively prevented.

### 6.6 UX-ONB-06 — R1 nutrition and routine setup

After the simple setup, offer “Bring in my usual meals” and “Set nutrition goals.” Both are revisitable. Support daily meal slots, recurring baseline meals, optional supplements, planning days, serving counts, and target entry. The originating user's exact quantities and goals are unknown; no historic memory values should be silently populated as current facts.

Each nutrition target has a nutrient identity, unit, lower/upper bound as applicable, evaluation period, enforcement level, source, and effective date. Show user-entered values separately from reference suggestions. Explain incomplete evaluations without forcing every micronutrient to be configured before meals can be saved.

No automatic calorie prescription, medical questionnaire, or supplement recommendation is required to begin. A reference-target feature is optional until its population selection, sources, units, limits, and review policy are explicitly implemented. The NIH describes several distinct reference-intake categories; they must not be treated as interchangeable targets. [NIH nutrient recommendations](https://ods.od.nih.gov/HealthInformation/nutrientrecommendations.aspx)

### 6.7 UX-ONB-07 — Routine and target editors

The R1 routine editor shows rows for breakfast, lunch, dinner, snacks, and optional user-named slots. Each row can be flexible, an accepted recurring meal/product combination, or intentionally open. Select weekdays and an optional usual local time. Allow multiple items in a slot, such as a smoothie plus toast, without requiring the user to invent one giant recipe. Copy a day's structure across selected days, then edit exceptions.

A recurring template is separate from dated occurrences. Generate occurrences for the requested horizon with pinned recipe/product revisions and quantities. Editing a template affects future generation; offer a preview to update existing future occurrences. It never creates historical consumption or changes an already recorded event. Avoid materializing an infinite recurrence into the database.

Nutrition goals open as a compact editable table with nutrient, unit, lower bound, upper bound, period, and required/preferred/informational setting. Advanced details contain source, effective dates, contribution scope, and reference-policy provenance. Require at least one bound for an active target and reject a lower bound above its upper bound. Setting a daily target does not create a weekly target automatically. Deactivating a target retains its history and removes it from future enforcement after its effective end date.

The supplement editor selects a product and its dose unit, displays the entered label basis, and asks for the user's schedule and quantity. If the dose-to-nutrient mapping is unresolved, save the schedule as incomplete and keep contributions unknown. Offer Pause and Resume without deleting the product or earlier intake. Settings explain that the planner retains the user's specified quantity.

## 7. Today, weekly plans, priorities, and feedback

### 7.1 UX-TOD-01 — Today dashboard

**Purpose:** Answer “What should I eat next, and what do I do?” with minimal navigation.

**Initial hierarchy:** local date/meal context; large page title; compact priority controls; the next meal; contextual daily/meal information; a weekly summary. The selected meal is usually the next uncompleted scheduled meal. R0 may default to today's dinner; R1 must use actual configured slots. An earlier selected day should clearly read “Tuesday's dinner,” not “Tonight.”

The hero card includes:

- Meal name, one-line description, and an optional image or ingredient composition.
- Estimated active preparation time; distinguish elapsed cooking time in details.
- Energy, protein, portion cost, and effort/time, each with its own unknown state.
- A concise, calculation-backed reason: for example, “Uses your opened rice; 6 minutes of active prep.”
- A primary **Let's make it** action, a secondary **Swap meal**, and a quieter **See the recipe map** action.
- A menu or explicit link for edit, move, lock, skip, or remove as appropriate to the current release.

A missing quantity/price does not become `$0.00`. A known zero is valid for a zero-cost ingredient assertion or zero active minutes, but must be distinguishable from a missing value. A total with unknown components reads “$3.10 known + unpriced items” or equivalent.

Below the hero, use an easy-night card with a lightning icon and “Not a cooking kind of night?” This opens an effort-focused swap scoped to the selected meal. If the selected meal has already been consumed, offer the next slot rather than suggesting a replacement for history.

R0 sidebar: selected dinner's protein/energy, kitchen summary, and weekly portion-cost/variety summary. R1 sidebar: selected scope control (**Meal / Day**), planned and recorded totals clearly separated, the rest of today's slots, and material gaps. Detail opens nutrition analysis. Avoid a universal “all nutrients covered” indicator.

### 7.2 UX-TOD-02 — Empty, incomplete, and unavailable states

| State | What the user sees | Available recovery |
| --- | --- | --- |
| Loading persisted workspace | Stable shell and bounded skeletons | Do not render sample food as if it were saved user data. |
| No configured plan | A short invitation to generate or choose a meal | Generate draft / Choose meal / Add favorite. |
| No eligible meals | Rule conflict summary with counts | Add a matching meal / Review restrictions; no automatic relaxation. |
| Plan exists but data incomplete | Meal remains usable with unknown metrics | Fill relevant detail / Continue without a completeness claim. |
| Meal was archived after scheduling | Pinned revision remains available | Replace future occurrence if desired; preserve history. |
| Account load failed | Clear load error and retry | Do not allow edits against a fake empty account that could overwrite real data. |
| Save failed after local edit | Edited state remains visible with pending marker | Retry / Export pending changes / Resolve conflict. |
| Provider unavailable | Existing local records still work | Manual entry, cached results labeled by age, or retry. |
| No eligible low-effort alternative | Honest no-match result | Broaden the effort preference only with user action; retain required rules. |

### 7.3 UX-SWP-01 — Swap interaction

Open a dialog titled **What sounds better?** Show up to four useful alternatives immediately. Reason chips are **Something different**, **Less effort**, **Lower cost**, and **Missing ingredients**. Choosing a reason reranks or filters alternatives; no reason is mandatory.

Each alternative shows name, active time, protein if known, portion cost if known, and a short reason. Highlight “Best match” only relative to the current candidate pool and scoring policy. Do not imply global optimality.

Missing ingredients exposes ingredients from the current occurrence. Selecting one excludes alternatives that require the missing canonical ingredient/product unless a reviewed available substitute is part of that alternative. Match by identity and equivalence rules, not only exact display strings. The temporary exclusion belongs to this swap; it does not become a permanent dislike or allergy.

Every candidate must pass active hard constraints and equipment requirements. A low cost or variety score cannot compensate for a failed exclusion. Show blocked alternatives only in a separate inspectable area with reasons; they must not be selectable as compliant recommendations.

**Selection scope:** default to the selected future occurrence and its planned servings. If the recipe is repeated elsewhere, leave other occurrences unchanged. Offer “Replace matching future occurrences” separately with a count and preview. Never change consumed occurrences.

**Consequences:** calculate the updated day/week nutrients, portion and checkout costs, required groceries, batch dependencies, and unresolved targets. R0 may apply a valid single-slot swap immediately with Undo when there are no material dependencies. R1 should show a compact confirmation if it creates a required-target conflict, abandons a planned batch, changes a locked dependent occurrence, or substantially changes checkout requirements. A normal compatible swap remains quick.

**Undo:** restore the prior occurrence and derived plan state only if the affected revisions still match. If newer edits conflict, offer a new preview instead of overwriting them. Persist the operation, not just a toast timer.

### 7.4 UX-WEK-01 — Weekly plan

Show the actual date range, dinner/full-day scope, and previous/next week navigation. Provide a compact summary of known portion cost, estimated checkout when available, active prep, and variety. Label the denominator and missing-data status of averages.

R0 uses one dinner row per day. R1 supports configured breakfast/lunch/dinner/snack slots, routine anchors, supplements, and open/social slots. Prefer a readable day-grouped list on mobile. A desktop matrix is optional if it remains understandable; a week should not require drag-and-drop to operate.

Each occurrence shows date/slot, meal, servings, relevant metrics, status, and lock indicator. Selecting it opens its detail or makes it the Today focus with an explicit date. Actions include inspect recipe, swap, move, lock/unlock, duplicate to selected future days, skip, and mark consumed. All drag actions require a keyboard/menu equivalent.

A user can keep favorite anchors, leave one slot open, and request regeneration of only unlocked future slots. Past uneaten slots do not automatically become consumed; mark them unrecorded, with optional skip/log actions. An open slot contributes unknown future intake, not zero or assumed compliance.

### 7.5 UX-WEK-02 — Replan preview

Changing priorities from the week view generates a preview with changed/unchanged/locked rows and a before/after summary. The default apply scope is the selected week's unlocked future occurrences. The summary identifies any unknown cost, nutrient, or time contributions and all infeasible required constraints.

Applying the preview uses the input revision captured at generation. If profile, prices, recipes, stock, or the current plan changed materially, invalidate/revalidate and show an updated preview. Preserve the user's locks. Record the solver/scoring version and input references used.

Provide **Apply to my week**, **Keep current plan**, and, where appropriate, **Try another draft**. Regenerate may vary a deterministic seed or diversify candidate selection, but it must preserve constraints and explain which inputs changed. Do not charge repeated AI calls for a purely deterministic rerank.

### 7.6 UX-PRI-01 — Presets and custom priorities

Use four visible presets: **Balanced**, **Save more**, **Easy mode**, **More variety**. The prototype used approximate weights shown below; these are recommended initial configuration, not a mathematical claim of calibrated utility:

| Preset | Cost | Effort | Variety |
| --- | ---: | ---: | ---: |
| Balanced | 55 | 70 | 35 |
| Save more | 100 | 40 | 20 |
| Easy mode | 25 | 100 | 40 |
| More variety | 35 | 45 | 100 |

A custom priorities sheet has three labeled sliders, plain-language endpoints, an impact preview, and a link to fixed dietary/kitchen rules. Show values as relative importance (for example `55 / 100`), not “55% of the diet” or a probability. The values need not sum to 100; normalize only within the scoring service.

Display a preset as selected only when its configured vector exactly matches. A persisted custom vector remains Custom after reload. Prevent an all-zero vector from producing undefined normalization: use an explicitly documented equal-weight fallback and show that state, or require at least one nonzero weight before apply. Recommended default: retain the prior nonzero vector and explain the invalid edit.

R1 adds an optional scope control: this meal, through a selected date, or default for future plans. A temporary mode overlays preferences and expires in local time without mutating the baseline. It never changes allergies, ethical exclusions, or configured required nutrition rules.

### 7.7 UX-FDB-01 — Consumption and lightweight feedback

Support **I ate this**, optional portion adjustment, optional actual time, and Undo/correct. Logging does not require rating, weighing food, or explaining a deviation. A scheduled meal is not evidence that it was eaten.

Offer one-tap reasons after a voluntary swap or deviation: Bored, Too much work, Missing ingredients, Craving something else, Ate out socially, Still hungry, and Other. Also support “Keep this in rotation”, “Less often”, and “Don't suggest this meal”. A meal-level exclusion differs from excluding every ingredient in it.

Use feedback conservatively. A rejection due to missing ingredients is not evidence of disliking the meal. A single social meal is not a failure. Explicit dislikes outrank weak inferred preferences. Missing feedback remains unknown. Provide a way to inspect and reset learned preferences.

R1 can implement simple transparent rules such as a configurable cooldown after “Less often”. R2 can add learned ranking when supported by enough data, but must not display invented adherence percentages. Product success cannot be inferred only from click acceptance.

## 8. Meal collection, flexible capture, and editing

### 8.1 UX-MEA-01 — Meal collection

The Meals page has the title **Your kind of good.**, an Add meal action, search, filters, and Import/export. Show the total collection count and count fitting the current setup. Filters include Fits my setup, Needs review, Drafts, Favorites, and Archived; advanced filters may include meal type, appliance, active time, flavor, and source.

Cards show a readable name, optional description, available metrics, readiness/fit status, Edit, Recipe, and Use in selected slot. A collection can contain a meal outside the active diet; storage is permitted even when recommendation is blocked. The status should explain the difference rather than hide the record.

Add a quiet end-card: “Make room for a favorite. Start with a name. The rest can wait.” Search spans meal names, aliases, ingredient names, and approved tags. Show a clear no-results state with a reset action. Search should not make an external AI call per keystroke.

Support duplicate, archive, and recover. Archiving removes a recipe from future candidate discovery while preserving pinned revisions in plans/history. Permanent deletion, if implemented, must specify how referenced historical data is retained or removed; it is not the ordinary remove action.

### 8.2 UX-MEA-02 — Minimal entry and quick capture

A new-meal editor opens on **Quick capture**, with **Meal details** as the adjacent tab. Editing an existing meal opens its details. The quick capture field accepts a name, rough notes, or pasted recipe text. A URL can be stored as a source without requiring a successful fetch. In R2, link/photo/file modes can share the capture surface when their adapters are configured.

A nonempty trimmed name is sufficient to save a draft. Optional quantities, nutrients, price, preparation, and method remain null/absent. No field is silently set to zero because an HTML input is empty. Do not require an image, ingredient list, nutritional completeness, or a star rating to save an idea.

Conservative R0 text organization:

- Preserve the full original text and its source type.
- Recognize headings such as Ingredients, Steps, Method, Instructions, or Directions.
- Extract explicit amounts and units only when unambiguous; handle decimals and common fractions or retain the text unchanged if unsupported.
- Keep “to taste”, “one can”, “a scoop”, and “a handful” as unresolved quantities unless a selected product has a reviewed mapping.
- Preserve numbering without turning `1.5 cups` into `5 cups`.
- Never infer nutrition, diet suitability, a brand, ingredient/action links, or appliance availability merely from the meal's name.
- Present the result as a draft to review. Reorganizing text while editing must keep the existing recipe identity and create a proposed revision, not an accidental duplicate.

If processing fails, keep the text and offer Save draft or Edit manually. If the user switches between tabs, do not discard typed content. If an automated extraction finishes after the user edited a field, show a comparison; do not overwrite the newer edit.

### 8.3 UX-MEA-03 — Detailed editor

Recommended groups, in this order:

| Group | Fields and behavior |
| --- | --- |
| Identity | Name, short description, optional image, meal type, flavor/texture/cuisine tags, favorites. |
| Yield and effort | Recipe yield, serving definition, active minutes, elapsed minutes, cleanup burden, batch suitability, portion bounds. |
| Ingredients | Ingredient/product reference or unresolved text, amount, unit, preparation state, optionality, substitutions, source. |
| Nutrition | Derived profile where ingredients are mapped; optional whole-recipe or per-serving assertion with explicit basis; provenance and missing fields. |
| Cost | Derived ingredient cost or explicit estimate with currency/date/basis; never add both. |
| Method | Ordered steps, ingredient allocations, intermediate components, dependencies, equipment/method alternatives, optional duration. |
| Rules/evidence | Diet groups, allergens, precautionary statements, equipment, reviewed assertions and unresolved conflicts. |
| Source and notes | Original source URL/text, user notes, attribution/license metadata where applicable. |

A details editor may be long; keep the first screen focused on name and the fields the user already has. Use collapsible sections for advanced data and an always-understandable Save action. Save should be allowed for a valid draft with incomplete information.

Rows can be added, reordered, and removed with keyboard-accessible controls. Show units alongside quantities. Ingredient amounts refer to the whole recipe yield unless a conspicuous basis selector says otherwise. Removing an ingredient must identify associated step allocations, remove or resolve those links, and invalidate affected assessments. Removing a step with downstream dependents requires a repair preview; do not silently create a broken recipe graph.

Validation messages belong next to the actual fields. Reject negative amounts, NaN/infinity, invalid yield, duplicate local IDs, broken references, and impossible dependency cycles. Do not claim every unsupported format is a negative-number error. Preserve all valid edits when one field fails.

### 8.4 UX-MEA-04 — Readiness and evidence workflow

Store authoring status separately from current-profile eligibility:

- **Draft:** usable as a saved idea; not automatically recommended.
- **Ready:** authoring review sufficient for the explicitly supported operations.
- **Archived:** no longer offered to new plans; historical references remain usable.

“Ready” does not mean nutritionally complete or verified safe for every diet. For the active profile, the evaluator independently reports Fits current rules, Excluded, or Needs information. An unknown calorie total may permit a general meal suggestion while preventing a claim that a hard calorie ceiling is satisfied.

Review only the missing or conflicting information material to matching. Avoid making the user reconfirm every allergen after changing a description. Ingredient/product/method changes invalidate relevant review results; spelling, notes, and visual tags usually do not. Evidence freshness is tied to semantic revisions, not a universal checkbox forever.

R0 may implement a simpler explicit review action as long as missing evidence is still visible and required constraints are not bypassed. R1 must have the separate states described here.

### 8.5 UX-MEA-05 — Versioning and scheduled uses

Save an edit as a new immutable recipe revision, with the recipe retaining its stable logical ID. Offer to update eligible future uses with an impact preview. The default is to use the new revision for new planning; existing occurrences remain pinned until an explicit update operation selects them.

Consumption records always refer to the revision and quantity actually recorded. Later edits to a product's nutrients, recipe ingredients, or serving yield cannot silently change past totals. A user can intentionally correct a historical entry, which records a correction with its own time and reason.

Concurrent edits return a conflict with the saved revision and the local draft. Offer compare/reapply or save as a separate copy. Never resolve a conflict by losing the local form silently.

### 8.6 UX-MEA-06 — R1 components and meal families

Support reusable components such as cooked grains, a sauce, roasted vegetables, a smoothie base, or a protein filling. A component can have its own recipe, yield, nutrients, effort, storage notes, and preparation graph. Component relationships must be acyclic.

A meal family is a reviewed template such as protein + grain + vegetables + sauce + crunch. It has typed slots, compatible options, portion ranges, preparation constraints, and flavor/texture tags. Instantiating a family creates a concrete, inspectable meal revision with actual quantities and choices. It must not remain an opaque prompt whose nutrient totals cannot be reproduced.

Start with a few supported families and user meals. Do not require a general recipe-composition language before R1 can be useful. The extension point is the domain model and pure calculation service, not a separate generic optimization platform.

## 9. Recipe map, reading view, cooking mode, and recipe printing

### 9.1 UX-REC-01 — Shared recipe viewer

The recipe detail shows title, description, source/readiness context, serving selector, time estimate, Edit, and Recipe PDF. Below it are three tabs: **Recipe map**, **Read recipe**, and **Cook step by step**. The same versioned recipe data drives all views.

The prototype's map was inspired by Michael Chu's tabular recipe notation, which brings ingredient quantities and preparation actions into one compact representation. The interview also identifies difficulty representing discarded and reintroduced ingredients. Daily's explicit quantity allocations and intermediate outputs below are a proposed extension, not a claim of exact compatibility with that notation. [Cooking for Engineers interview](https://coolinfographics.com/blog/2010/4/26/cooking-for-engineersrecipe-infographics-and-interview.html)

### 9.2 UX-REC-02 — Ingredient/action map

Rows represent ingredient uses or prepared components. Columns represent ordered actions or stages. The left column shows ingredient name, quantity for selected yield, and relevant preparation qualifier. Headers show step number, action title, and dependency cue: Start anytime, After step 1, or After steps 1 + 2. A marked cell means this action uses the stated ingredient allocation; a blank cell means no direct allocation is recorded.

Clicking a header or marked cell opens that action in focused mode. Tooltips or expandable details identify quantities, outputs, and instructions. Inactive cells must not look like unfilled required form inputs. Unmapped ingredients appear in a clearly labeled “Not linked to a step” state; absence of links does not erase them from nutrition or groceries.

For wide graphs, group actions into stages, allow horizontal scrolling inside the map, and keep the ingredient column readable. A table whose columns imply order is not an elapsed-time Gantt chart. Do not position steps on a time axis unless durations and resource constraints support that claim.

R0 may show a table with ingredient links and dependency labels. R1 must model intermediate outputs, split uses, and joins so the map communicates how prepared components become the meal. A separate graph view is optional; the stored graph is required for consistent semantics.

### 9.3 UX-REC-03 — Branches, joins, and ingredient allocation

Every step has stable ID, title, instruction, direct ingredient allocations, input component references, output component definitions, dependency IDs, optional duration, and selected preparation method. Dependencies form a DAG. A step can start when its prerequisites are met; an ingredient's appearance in a later step does not imply it is purchased twice.

For an ingredient used in two steps, store allocations against one recipe ingredient use. Example: 30 g sauce is mixed in and 10 g reserved for finishing from a total of 40 g. Do not duplicate two 40 g ingredient rows. Discarded water, drained liquid, or unused marinade needs an explicit disposition and an appropriate nutrition treatment; do not subtract arbitrary retention percentages.

Prepared outputs are named: warm grain base, chopped vegetables, dressing, cooked batch. Joining outputs does not re-add their input nutrients. An optional step must define whether omitting it changes ingredient inclusion, output yield, or subsequent dependencies.

A simple parallel flow:

```mermaid
flowchart TD
    A["Heat rice and edamame"] --> C["Combine components"]
    B["Cube tofu and prepare slaw"] --> C
    C --> D["Add sauce and serve"]
```

This diagram describes dependencies only. The application should let the user perform independent tasks in parallel without implying that every appliance can run multiple jobs simultaneously.

### 9.4 UX-REC-04 — Conventional reading view

Desktop uses an ingredient column and a wider method column; mobile stacks them. Ingredients show scaled quantities and unit/basis qualifiers. Method steps show titles, clear instructions, optional time, and dependency hints when needed. Include notes and source links in a secondary expandable area. Text must remain selectable and accessible.

A name-only meal displays “No ingredients added yet” and “No method added yet,” with an Edit action. A ready-to-eat item may deliberately have no method; that is different from a missing method for a complex dish. Do not force unnecessary steps onto an assembly-free food.

### 9.5 UX-REC-05 — Focused cooking mode

Display one action prominently: step number/title, full instruction, needed ingredients/components, quantity, appliance setting if actually specified, and next/back controls. Provide direct access to the step list without losing progress. Independent actions can be shown as “You can do this while the base heats.”

The user may preview any step even if its dependencies are unfinished. Reading a step is not completion. Explicit completed states and timer state must be distinct from the currently viewed step. A Next button advances the view; if it also marks completion, label and document that behavior consistently. Recommended default: mark complete with a separate checkbox, Next only navigates.

Optional timers use absolute timestamps, persist across reload, and do not imply that food is cooked safely when the timer ends. Use only supplied recipe time/temperature instructions; do not invent safety-critical settings. If a timing integration is unavailable, show the textual duration.

At the end: R0 supports “I ate this” with quantity and Undo. R1 offers “Finished cooking” to create a prepared batch and a separate “I ate a serving.” A user can cook without consuming, consume leftovers without cooking again, or close the viewer without logging anything.

### 9.6 UX-REC-06 — Scaling and preparation methods

Scaling the viewer changes ingredient quantities and calculated totals for the displayed yield. It does not change the recipe's stored canonical yield or the planned occurrence until explicitly applied. Display per-serving and selected-yield totals with unambiguous labels.

Use a numeric serving field plus stepper or a small selector. Support fractional servings where the recipe allows them. Do not round eggs, capsules, packages, or indivisible components to fractions silently; represent discrete constraints and show an explicit adjustment suggestion. Serving scale does not automatically scale time, temperature, appliance capacity, or batch feasibility linearly.

A microwave and stovetop variant are distinct reviewed preparation methods sharing ingredients where appropriate. Eligibility requires at least one complete viable method. Do not require every appliance mentioned in all alternate methods at once. Switching methods updates steps and effort; it also triggers nutrition re-evaluation if quantities or preparation state change.

### 9.7 UX-REC-07 — Recipe PDF

Produce a real text/vector PDF with embedded fonts, not a screenshot of a scroll container. Include recipe name, selected yield, clearly scoped nutrition/cost, ingredients with quantities, a compact map, numbered method, source/notes, material allergen declarations, and page numbers.

Defaults: support both A4 and US Letter; choose locale preference, offer the other. Use 16–18 mm margins, readable 9–11 pt body text, 18–24 pt title, restrained cobalt headers, pale alternating rows, and dark text that remains legible in grayscale. Avoid printing navigation, interactive buttons, or large decorative food imagery by default.

Wide maps should be split into labeled stage groups or use an optional landscape page. Repeat ingredient names and headers as needed. Never shrink a 20-step map until text becomes unreadable. Keep section titles with following content; wrap long ingredient names and URLs; repeat table headings across pages. Use consistent footer and page counts. Include generated date and recipe revision in small print.

The export must reflect the current selected serving scale and revision. Generate and inspect PDFs with long names, Unicode, fractions, missing values, many ingredients, and multi-page methods. A success toast must follow successful generation/download initiation; provide a visible retry/download link when browsers block automatic saving, and test the actual file path in the target environment.

## 10. Grocery list, price book, and pantry UX

### 10.1 UX-GRO-01 — Derived shopping list

Groceries is derived from an explicitly selected plan/range and the current inventory policy. Its header states the scope: for example, “For your meals, September 21–27.” Show checked/remaining counts without implying that checked rows were bought.

Group lines by practical category or store, with user-selectable sorting. Each line shows ingredient/product, total planned need, stock considered, amount missing, selected package plan if known, and estimated checkout. Unknown amount, unit conversion, package size, or price is visible separately. A row can have a known needed mass but an unknown package count.

Actions: check/uncheck, adjust intended purchase quantity, choose product/package, mark Already have, add a manual item, inspect contributing meals, export, and confirm purchased quantities. An “Already have” action creates a stock assertion with its own basis; it is stronger than a shopping checkmark and needs an editable quantity or explicit qualitative status.

Manual grocery items remain distinct from plan-derived requirements. Replanning cannot delete them. If a user overrides a derived purchase quantity, retain the override with a mismatch badge when plan need later changes.

### 10.2 UX-GRO-02 — Checkout summary

Show known package subtotal, count of unpriced items, and excluded costs such as tax, delivery, or membership conditions when they are not modeled. Do not title an incomplete subtotal “Total cost.” Portion cost and checkout cost should both be available, clearly labeled.

Example: “$18.00 in priced packages + 2 unpriced items.” A stocked ingredient contributes zero additional checkout for this trip, but it still has a consumed portion cost when a cost basis is available.

Do not label a speculative best price as the user's actual accessible price. Price observations have currency, retailer, product/package, observed time, availability, and conditions such as membership or minimum quantity. An extra store may add travel/effort and delivery fees; include those as separate tradeoffs when known.

### 10.3 UX-GRO-03 — Pantry confidence and corrections

Start with low-maintenance states: Confirmed amount, Estimated amount, Have some, Out, and Unknown. Attach last-updated time and origin. Avoid fabricated confidence percentages. The user can correct an estimate with a quick Yes/No or amount response when it matters.

The planner reserves ingredients for future cooking without subtracting them from on-hand stock as if already consumed. Cooking a batch consumes raw ingredients once and creates prepared stock. Eating a batch portion reduces prepared stock, not the raw ingredients a second time.

In R0, conservative exact quantities and manual adjustments are sufficient. R1 supports batches and leftovers. R2 may infer likely stock from events, clearly distinguish inference from confirmed stock, and ask a short question when uncertainty changes a shopping recommendation materially.

### 10.4 UX-GRO-04 — Purchase and inventory updates

Checking a grocery box only updates checklist state. An explicit **Add purchases to pantry** action previews quantities and prices, supports partial fulfillment, and creates idempotent purchase/stock events. The user can dismiss this and keep approximate stock.

If a receipt importer is introduced, it stages purchases for review, matches packages and units, and detects possible duplicates against manually recorded purchases. Receipt parsing must not silently create consumption events. Confirmed actual spend is separate from planned spend and price observations.

### 10.5 UX-GRO-05 — Price book and product comparisons

The R1 price book is reachable from Groceries and an ingredient's details. Show product, package amount/unit, retailer, observed price/currency, date, conditions, and a usable-data status. Search/filter by product or ingredient, and add a price in a compact form. Editing an observation creates a correction/new observation; it does not rewrite a past purchase.

A product comparison groups genuinely substitutable options under a reviewed ingredient role. Show package price, comparable usable-unit price, expected packages for the current plan, additional checkout, and leftovers. Optionally show cost per configured protein amount when protein data and product basis are known and positive. This is an economic comparison of modeled quantities, not a claim that all protein sources or diets are nutritionally equivalent.

For example, a fictional $4 package containing 48 g modeled protein has a cost of $4 × 25/48, or about $2.08 per 25 g protein. Retain unrounded values during comparison. Unknown protein yields an unavailable comparison, not division by zero. Mixed currencies or incompatible prepared-weight bases cannot be ranked as though directly comparable.

Selecting an alternate product previews the affected recipe quantities, nutrition, allergen/diet evidence, package cost, and shopping lines. A cheaper formulation cannot bypass current restrictions. Give the user an easy way to keep their current brand even if another option scores better. Membership, coupons, extra trips, and delivery conditions stay visible.

## 11. Import, export, backups, and printing

### 11.1 UX-DAT-01 — Transfer center

A dialog/page titled **Your food, portable.** contains **Export & print** and **Import** tabs. Export cards have distinct names and explanations:

1. Week & grocery PDF: a printable menu and shopping checklist.
2. Complete backup: all supported workspace records, excluding secrets.
3. Meal collection: selected recipes and required dependent food/component data.
4. Grocery spreadsheet: human-readable CSV for the selected shopping plan.

A recipe has its own Recipe PDF action. Exports must not require a paid AI provider, additional nutrient completeness, or a subscription upgrade merely to retrieve the user's own data. Commercial policies can limit optional compute, not confiscate data.

### 11.2 UX-DAT-02 — Import review and transaction

Accept a file or pasted JSON for structured imports. For plain text, direct users to Quick capture. Detect the format/version before parsing into the live model. Enforce configurable byte and entity limits before expensive work. R0 defaults: 2 MB pasted/file JSON, 200 recipes, 80 ingredient uses per recipe, 30 steps per recipe, 10,000 characters of notes. These are configurable abuse/UX limits, not permanent domain limitations; later releases should support bounded larger backups through jobs.

Review shows record counts, source format/version, duplicates, ID conflicts, missing dependent records, unsupported fields, validation errors with paths, and the proposed action. The default is **Add meals only**. Full backup imports also offer **Restore everything** with a precise impact summary and a recoverable pre-restore checkpoint.

Do not mutate live data during parse or preview. Apply all selected valid changes in a transaction. For a multi-record import with invalid entries, default to rejecting the full import and exposing errors; an optional explicit “Import valid records only” must show exactly what is omitted and preserve rejected content for correction.

Recipes that are identical after canonical normalization may be skipped. A stable-ID collision with different content must offer Keep existing, Copy incoming, or Replace with a new revision where supported. Default to Copy incoming, remapping all internal references consistently. Never identify duplicate recipes solely by name.

### 11.3 UX-DAT-03 — Full restore behavior

A restore replaces the active workspace content covered by the manifest, not identity, authentication, subscription, provider credentials, or system policy. Export/restore may carry profile preferences but cannot import someone else's tenant authority or ownership grants.

Before applying: validate all references; show counts for removals/replacements; create a recoverable checkpoint; compare expected workspace revision; and require explicit Restore confirmation. A stale preview must be rebuilt. On failure, leave the previous workspace intact. Restoration jobs must be resumable/idempotent with a terminal outcome visible to the user.

Imported plans are evaluated under their imported active rules. Conflicting future occurrences become visibly unresolved or flagged; do not silently replace imported meal choices during restore. Offer **Repair this plan** as a subsequent preview. Historical consumption stays preserved as imported facts/assertions with provenance.

### 11.4 UX-DAT-04 — Weekly PDF and CSV

Weekly PDF: local date range, selected meal scope, daily menu, serving counts, preparation notes, batch tasks, clearly labeled known costs, unresolved slots, and a shopping checklist grouped meaningfully. Include supplemental schedules only when selected; do not insert private profile details or all allergy history on every page by default.

Support monochrome printing and sensible page breaks. A short seven-dinner plan should normally fit one menu page plus one grocery page, but content correctness and readable text take precedence over an arbitrary page count. Long lists continue with headers. Keep a group heading with its first item.

CSV columns: item, planned need, unit, stock considered, amount to buy, selected package size/count, estimated price, currency, price date, checked state, and notes. Keep numeric columns numeric when values are known, with explicit completeness/status columns where useful. CSV is a spreadsheet view, not the canonical lossless backup.

Escape fields correctly and neutralize formula interpretation for user-controlled text. Test target spreadsheet applications; do not promise universal protection from one prefix, especially after a user re-saves the file. JSON preserves the unmodified original strings for lossless transfer. [OWASP CSV injection guidance](https://community.owasp.org/attacks/CSV_Injection)

### 11.5 UX-DAT-05 — Portable format policy

Use a namespaced format identifier and an independent schema version. Keep application build version, export-format version, entity revision, and source-provider version distinct. Section 20 contains a concrete envelope example and legacy compatibility fixture.

A complete export includes profile/rules, foods/products and nutrient assertions, recipes/revisions and components, plans/occurrences, targets, supplement schedules, prices, inventory/batches/events, consumption/corrections, preference feedback, and referenced source metadata according to the release's implemented features. Secrets, auth tokens, raw model credentials, private infrastructure URLs, and billing instruments are excluded.

Source attachments may be embedded in a bounded archive in a later format, or explicitly omitted with a manifest report. A JSON-only export that omits original photos must say so. Never advertise “complete backup” if required operational data is silently missing.

Round-trip acceptance compares semantic content after normalization, not JSON property order. Derived totals and solver caches may be exported for explanation but must be marked derived and recomputed/validated on import.

## 12. Domain model, contracts, and invariants

### 12.1 DOM-01 — Domain boundaries and vocabulary

Separate the following concepts even if the initial database co-locates them:

| Concept | Meaning | Must not be confused with |
| --- | --- | --- |
| Food concept | Generic edible item/state, such as cooked lentils | A particular package or store offer. |
| Product | Specific branded/packaged formulation | Every product with a similar display name. |
| Food/product revision | Snapshot of composition, serving definitions, and evidence | Mutable current catalog metadata. |
| Ingredient use | Quantity of a food/product/component in one recipe | An additional food purchase each time a step references it. |
| Recipe | Stable meal/component identity | One immutable version of its quantities and method. |
| Recipe revision | Immutable composition, yield, method, and assertions | Current recipe pointer. |
| Meal component | A reusable prepared or ready-to-use part | A free substitute with identical nutrition. |
| Meal family | Reviewed template of compatible slots/options | A fully quantified meal. |
| Planned occurrence | Date/slot, pinned recipe, selected servings, status | Evidence of consumption. |
| Prepared batch | Actual output quantity from a cooking event | A scheduled future cooking task. |
| Consumption event | What the user recorded eating/taking | Automatic plan acceptance. |
| Price observation | Dated package offer or user assertion | A universally current market price. |
| Purchase event | Recorded acquisition/spend | A checkbox on a shopping list. |
| Inventory assertion/event | Evidence about stock or a stock change | A precise measurement when only inferred. |
| Target | A configured bound/preference with scope | A universally appropriate dietary recommendation. |
| Source artifact | Original text/label/provider record | Approved structured truth. |
| Assessment | Evaluation against a versioned policy and input revisions | A permanent property of a meal for all users. |

### 12.2 DOM-02 — Identity, ownership, and revisions

Every user-owned aggregate belongs to a server-authorized workspace. User-provided tenant IDs never establish access. Stable logical IDs and immutable content revisions are separate. References from plans and history identify the exact revision they use.

Use opaque IDs generated by the chosen stack. A sample prefix such as `recipe_` is illustrative, not a requirement to implement a custom ID scheme. Record creation/update timestamps in UTC. Store local dates and IANA timezone separately for calendar semantics. A revision number is an integer used for concurrency; it is not a timestamp.

Deletions should normally archive or tombstone referenced records. A source record update creates a new revision; it does not mutate a record's historical snapshot. Keep revision lineage to explain how an imported/extracted value became an accepted one.

### 12.3 DOM-03 — Decimal quantities, units, and money

Use decimal or rational arithmetic for domain calculations. Serialize decimals as strings in canonical APIs/exports to avoid unintended floating-point drift. UI formatting may use ordinary numeric display helpers, but persistence and aggregation should preserve precision and units.

Quantities require unit, dimension, preparation state where relevant, and basis. `100 g dry rice` and `100 g cooked rice` are different nutrient bases. `1 scoop` is not a mass until mapped for a specific product/version. Do not apply a universal grams-per-cup conversion. A density, edible-yield factor, or piece weight must be explicit, source-linked, and applicable to the exact item/state.

Use a unit registry with aliases and validated conversions. R0 needs mass, volume, count, servings, and packages. R1 adds product-specific piece/capsule/scoop conversions. Support `g`, `kg`, `mg`, `ug`, `mL`, `L`, count, serving, and package as canonical units where applicable. Preserve entered `oz`, `lb`, cups, tbsp, tsp, and fractions as display metadata after resolving the correct locale/basis. Distinguish fluid ounces from mass ounces.

Use ISO currency codes. Store transaction/package money as integer minor units with a currency-specific exponent, and maintain higher precision internally for allocated portion cost. Do not assume every currency has two decimal places or that prices in different currencies can be summed. Currency conversion is deferred unless a dated exchange-rate source and policy are implemented.

Round quantities only for display or explicitly discrete actions. Round package counts upward where appropriate. Round money at documented transaction/report boundaries, not every ingredient multiplication. Unknown and explicit zero are distinct in all serializers and forms.

### 12.4 DOM-04 — Representative type contracts

These TypeScript contracts specify semantics. Adapt naming and transport details to repository conventions while preserving the distinctions. They are not a complete generated ORM schema.

```ts
/** Canonical decimal text, finite and validated; arithmetic uses a decimal library. */
type Decimal = string;
type Id = string;
type LocalDate = string; // Strict YYYY-MM-DD; validated as an actual date.
type Instant = string;   // UTC ISO-8601 instant.
type CurrencyCode = string;

type RevisionRef = { id: Id; revision: number };
type UnitId = string;
type NutrientId = string;

type Quantity = {
  amount: Decimal;
  unitId: UnitId;
  originalText?: string;
};

type EvidenceRef = {
  sourceId: Id;
  locator?: string; // Label panel, provider field path, page, or source-text span.
};

type Provenance = {
  origin: 'user' | 'label' | 'provider' | 'calculated' | 'model_proposal';
  evidence: EvidenceRef[];
  review: 'unreviewed' | 'user_confirmed' | 'source_checked' | 'conflicting';
  observedAt?: Instant;
  reviewedAt?: Instant;
  reviewerId?: Id;
  calculationVersion?: string;
};

/** Unknown is represented explicitly. Zero is a known decimal value. */
type NutrientValue =
  | { state: 'unknown'; reason: string }
  | {
      state: 'known';
      amount: Decimal;
      unitId: UnitId;
      qualifier: 'declared' | 'estimated' | 'calculated';
      provenance: Provenance;
    };

type NutritionProfile = {
  basisQuantity: Quantity;
  preparationState: string; // Examples: dry, cooked, drained, as_packaged.
  nutrients: Record<NutrientId, NutrientValue>;
};

type IngredientSource =
  | { kind: 'food'; ref: RevisionRef }
  | { kind: 'product'; ref: RevisionRef }
  | { kind: 'component'; ref: RevisionRef }
  | { kind: 'unresolved'; text: string };

type IngredientUse = {
  id: Id; // Local to this recipe revision, unique within it.
  source: IngredientSource;
  quantity: Quantity | null;
  quantityText?: string;
  preparationNote?: string;
  optional: boolean;
  includedByDefault: boolean;
};

type StepAllocation = {
  ingredientUseId: Id;
  quantity: Quantity | null; // Null means linked but allocation not quantified.
  disposition: 'incorporated' | 'processing_aid' | 'reserved' | 'discarded';
};

type RecipeStep = {
  id: Id;
  title: string;
  instruction: string;
  after: Id[];
  allocations: StepAllocation[];
  inputOutputIds: Id[]; // References named outputs of earlier dependency steps.
  outputs: { id: Id; name: string; quantity: Quantity | null }[];
  activeMinutes: Decimal | null;
  elapsedMinutes: Decimal | null;
  equipmentCapabilities: string[];
  optional: boolean;
};

type PreparationMethod = {
  id: Id;
  label: string;
  steps: RecipeStep[];
  requiredCapabilities: string[];
  activeMinutes: Decimal | null;
  elapsedMinutes: Decimal | null;
  maxBatchYield?: Quantity;
};

type RecipeRevision = {
  recipeId: Id;
  revision: number;
  name: string;
  description: string;
  yield: Quantity | null; // Null for incomplete drafts; positive when used for portion calculations.
  servingDefinition?: string;
  ingredients: IngredientUse[];
  methods: PreparationMethod[];
  assertedNutrition?: NutritionProfile;
  nutritionAuthority: 'ingredients' | 'assertion';
  assertedCost?: { amount: Decimal; currency: CurrencyCode; basis: 'recipe' | 'serving'; provenance: Provenance };
  tags: string[];
  sourceIds: Id[];
  createdAt: Instant;
};
```

Runtime schemas must validate these contracts. TypeScript alone does not validate a file, API body, database migration, or model result. Unknown nutrient keys require registry resolution rather than silent dropping or an assumption that their units match.

### 12.5 DOM-05 — Target and assessment contracts

```ts
type Target = {
  id: Id;
  nutrientId: NutrientId;
  unitId: UnitId;
  lower: Decimal | null;
  upper: Decimal | null;
  enforcement: 'required' | 'preferred' | 'informational';
  period:
    | { kind: 'local_day' }
    | { kind: 'rolling_days'; days: number }
    | { kind: 'calendar_week'; weekStartsOn: number };
  contributionScope: 'all_intake' | 'food_only' | 'supplement_only' | 'qualified_sources';
  qualifiedSourceRuleId?: Id;
  provenance: Provenance;
  effectiveFrom: LocalDate;
  effectiveThrough?: LocalDate;
};

type Evaluation = {
  targetId: Id;
  status: 'pass' | 'fail' | 'unknown' | 'not_applicable';
  knownAmount: Decimal;
  missingContributors: Id[];
  completeness: 'complete_for_scope' | 'partial';
  reasonCodes: string[];
  evaluatedScope: { start: LocalDate; end: LocalDate; timezone: string };
};

type Eligibility = {
  status: 'eligible' | 'ineligible' | 'needs_information';
  reasons: { code: string; ruleId: Id; affectedRefs: RevisionRef[] }[];
  viableMethodIds: Id[];
  profileRevision: number;
  recipeRef: RevisionRef;
  evaluatorVersion: string;
};
```

An assessment identifies its input revisions and policy version so cached results can be invalidated. A profile change, product composition update, or altered preparation method can change eligibility without changing a recipe's name.

### 12.6 DOM-06 — Persistence entity inventory

| Entity | Minimum persisted information |
| --- | --- |
| Workspace / profile | Owner, locale/timezone/currency, settings revision, setup progress, active rules, appliances, preferences. |
| DietRule / preset | Canonical rule definition, required/preferred status, source/preset version, effective dates. |
| NutrientDefinition | Stable identity, canonical unit, form semantics, allowed conversions, source mappings. |
| Food / Product + revision | Identity, preparation state, serving mappings, nutrient profile, ingredient/allergen evidence, source provenance. |
| SourceArtifact | Type, source URL/provider ID/text or attachment reference, fetched/entered date, hash, parser version, privacy scope. |
| Recipe + revision | Stable identity/status and immutable yield, ingredients, methods, source references. |
| MealFamily | Typed slots, compatible options, portion bounds, reviewed generation rules. |
| NutritionTarget | Bounds, period, contribution scope, enforcement, provenance, effective dates. |
| SupplementSchedule | Product revision, dose quantity, local recurrence, start/end, user confirmation, pause state. |
| Plan + revision | Date range/timezone, scope, profile reference, solver inputs/version/seed, status. |
| PlannedOccurrence | Local date, slot, recipe revision or explicit open item, servings, method, lock, batch/source relation. |
| PreparationTask | Scheduled recipe/batch output, resources, active/elapsed effort, dependent occurrences. |
| PreparedBatch | Actual recipe revision/yield, remaining amount, preparation time, storage assertions. |
| ConsumptionEvent | Actual local/UTC time, pinned content/quantity, source occurrence/batch, snapshot totals, provenance, correction linkage. |
| PriceObservation | Product/package, amount/currency, retailer/conditions, observed date, validity/freshness, source. |
| InventoryEvent / assertion | Item/batch, amount/unit, event type, evidence/source, effective time, superseded/correction reference. |
| ShoppingPlan / line | Plan/input revisions, derived quantities, selected packages, overrides, check state, purchase links. |
| PurchaseEvent | Bought quantity/packages, actual spend/currency, retailer/date, receipt source, inventory application key. |
| Feedback / preference | Explicit reason/action, scope, timestamp, source occurrence, inferred/explicit distinction. |
| Import / research / export job | Input reference, status, dedup key, budget, progress, errors, staged results, applied operation ID. |
| Change / operation record | Actor, request/idempotency key, expected/applied revision, affected IDs, undo/correction linkage. |

Use normal relational constraints and indexes for actual access paths. Full event sourcing is not required. An append-only correction history for important facts plus current projections is sufficient. Do not keep one endlessly growing JSON blob as the only independently editable/account-queryable store in a commercial multi-user app.

### 12.7 DOM-07 — Non-negotiable invariants

1. Missing values are not zero; an absent allergen tag is not evidence of absence.
2. Every derived total names its basis, scope, and completeness.
3. Required exclusions cannot be traded against cost, effort, or variety.
4. Planned, cooked, purchased, and consumed are distinct states/events.
5. Historical intake uses immutable revisions/snapshots unless explicitly corrected.
6. A recipe's ingredient nutrients are counted once regardless of how many steps reference them.
7. A batch's raw ingredients are consumed once when cooking is recorded; later portions reduce batch stock.
8. Future reservations do not consume on-hand stock.
9. Changing a display unit does not change the underlying quantity.
10. Identical backed-up content is not duplicated because JSON property order differs.
11. Restore/import cannot grant account permissions or install credentials.
12. Cancellation and retry cannot apply the same purchase, consumption, import, or replan twice.
13. Missing feedback is not dislike or noncompliance.
14. Model text cannot directly write prices, targets, allergy assertions, or inventory without validated domain operations.
15. A best found plan is not described as globally optimal unless that property was actually established.

## 13. Nutrition, supplements, and data quality

### 13.1 NUT-01 — Nutrient registry and source mapping

The registry must support energy; protein; carbohydrate with definition retained; total fat; fiber; relevant sugars/saturated fat; and extensible micronutrients. Initial R1 support should include calcium, iron, zinc, iodine, selenium, sodium, potassium, magnesium, B12, vitamin D, folate, and other imported supported vitamins. This is a feature inventory, not a statement that every user needs the same targets.

Keep nutrient forms distinct: folate versus folate equivalents, vitamin A forms/equivalents, niacin equivalents, and omega-3 ALA/EPA/DHA must not be silently combined under incompatible units or interpreted as interchangeable. A product named “omega-3” needs its actual labeled composition before those individual nutrient contributions are known.

Use stable internal IDs and provider mappings, with data type, unit, method/basis, and source version retained. Do not identify a nutrient only by a localized label. Mass conversions within one identity are allowed; form-dependent conversions require an explicit reviewed rule. Do not apply one global IU-to-mass conversion across vitamins.

A generic food profile and a branded product profile may differ materially. Select the correct preparation state and edible/drained basis. Do not add generic nutrient estimates to a label profile as if both were independent intake. If filling missing label nutrients from a generic match, store each estimate separately with its own provenance and the user's chosen data-quality policy.

FoodData Central is a reasonable initial provider adapter: its documentation distinguishes analytically based and branded-label data types. Preserve that distinction in source metadata; a provider ID alone does not establish complete nutrients or a user's exact product match. [USDA data documentation](https://fdc.nal.usda.gov/data-documentation/)

### 13.2 NUT-02 — Arithmetic and authority selection

For a nutrient concentration `c` per basis quantity `b`, consumed compatible quantity `q` contributes:

```text
contribution = c × q / b
recipe_total = sum(included ingredient contributions)
per_serving = recipe_total / recipe_yield_in_servings
occurrence_total = per_serving × planned_or_consumed_servings
```

All values must be on compatible units/states. Resolve nested components recursively using their pinned revisions, with cycle/depth checks. An omitted optional ingredient contributes nothing; an included ingredient with unknown quantity contributes an unknown amount. “To taste” should remain unresolved unless the user intentionally treats the ingredient as excluded from evaluation with a visible reason.

A recipe can use ingredient-derived nutrition or an asserted recipe/serving profile as its authoritative path. Never add both. Display alternate data as a comparison if needed. If a user edits ingredients while using an asserted total, flag the assertion as potentially stale and require a choice before treating it as an updated complete profile.

Do not apply cooking loss/retention factors without specific applicable data. A prepared food profile may already include cooking effects. Applying retention again would double-adjust. Energy can be declared independently of macro-derived estimates; do not force equality through arbitrary nutrient modifications.

### 13.3 NUT-03 — Unknowns, zero, estimates, and coverage

For each nutrient, aggregate known contributions and track unresolved contributors. Display “at least the known subtotal” only when the underlying contributions are nonnegative and the meaning is clear; do not present that subtotal as the exact total. A missing quantity means multiple nutrients can be unresolved even if the food profile itself is complete.

Separate three ideas:

- **Data completeness:** which contributing amounts/bases/nutrient values are available.
- **Target evaluation:** whether the configured bounds can be evaluated against those values.
- **Evidence quality:** whether values are label declarations, user assertions, estimates, or source-checked data.

A lower-only target can be met by the known subtotal even when additional nonnegative contributions are unknown, but the total remains incomplete. An upper limit cannot pass when unbounded unknown contributions remain; it can fail if the known subtotal already exceeds it. A range requires both bounds to be assessed. These are statements about the entered model values, not biological certainty.

Recommended default evaluation:

| Condition | Result |
| --- | --- |
| Complete modeled total within applicable bounds | Pass, with evidence quality shown separately. |
| Known subtotal exceeds an upper bound | Fail, even if other contributors are unknown. |
| Known subtotal meets a lower-only bound | Pass for that bound; keep partial completeness visible. |
| Known subtotal below a minimum with unresolved contributions | Unknown, not definite deficiency. |
| Known subtotal within upper bound but additional amount unbounded | Unknown, not a pass. |
| Complete modeled total below a minimum | Fail against the configured target. |
| Target not configured / period outside evaluation scope | Not applicable. |

If the chosen policy demands source quality stronger than available assertions, an arithmetic pass may still be **not eligible for a validated claim**. Store that reason. Avoid an aggregate green badge that hides an unknown required target.

### 13.4 NUT-04 — Evaluation periods and scopes

Evaluate each target over its configured period. Do not average every micronutrient across a week by default. A weekly summary may report average daily intake descriptively without replacing daily constraints. An upper limit attached to a day cannot be satisfied by averaging an over-limit day with an under-limit day.

Scope selectors distinguish Selected meal, Planned day, Recorded so far, Expected day (recorded past slots plus planned remaining slots), and Selected week. An occurrence with a consumption record must not contribute both planned and actual amounts to Expected day. An unrecorded past slot remains unknown, not automatically replaced with the plan.

For partial consumption, show the recorded quantity and let the user explicitly keep a remaining planned quantity or finish the occurrence at the actual amount. Expected day can include that explicit remainder, but never the original full planned quantity alongside the consumed portion. Multiple intake events against one occurrence need a clear summed actual quantity and correction history.

A dinner-only plan cannot claim full-day compliance. In R1, use the user's baseline meals/recurring items as anchors where accepted, and mark open/unrecorded items explicitly. Supplement schedules contribute to planned/expected views according to their actual dates; recorded intake requires an event.

### 13.5 NUT-05 — Supplement representation

A supplement product has serving definition, dose unit, constituent nutrient amounts, other relevant constituents if tracked, source label/version, and dietary/allergen evidence. Creatine, caffeine, protein powder, and nutritional yeast may be represented as products/foods with their appropriate identities rather than assumed to be equivalent nutrient categories.

R1 schedules support daily or selected weekdays, one or more local times, start/end dates, pause, and an explicit dose quantity. Keep the dose fixed during meal optimization. The planner can show a coverage gap, but cannot invent or increase a dose to satisfy it. A dose edit requires a user action and creates a new schedule revision.

Distinguish amount of product from amount of nutrient: 1 g of an oil product is not 1 g of each fatty acid, and one tablet's mass is not its listed nutrient dose. Record actual product serving mappings. A product used inside a recipe and separately scheduled can represent two real intakes; detect likely duplicate capture for review but do not delete one based on matching names alone.

Do not include medical recommendations or population reference amounts in seed data as personal defaults. Reference policies need a documented source, population applicability, nutrient form, contribution scope, and effective version. Missing an upper-limit entry does not mean unlimited intake is known safe. This specification does not prescribe doses.

### 13.6 NUT-06 — Analysis UX and explanations

Offer a compact summary by configured targets, with expandable contributors. For each row show nutrient identity/unit, amount or known subtotal, target bounds, period, status, missing contributors, and source details. Sort material unresolved/failed required targets first without alarming language.

A suggested fix should be concrete: “The soy milk has no B12 value recorded. Add its label or leave B12 coverage unknown.” If a substitution is proposed, show how it affects taste/effort/cost and the relevant modeled nutrient values. Do not ask users to inspect every nutrient matrix before they can start dinner.

## 14. Planning engine and explainable optimization

### 14.1 PLN-01 — Planning inputs and outputs

Inputs: profile/rules and revision; date range/timezone; configured slots and anchors; targets; eligible recipe/component/product revisions; viable methods; permitted portion ranges; prices; stock/reservations; prepared batches; recent recorded feedback/history; locked occurrences; temporary mode; solver configuration/version; and an optional deterministic seed.

Output is a **draft** with concrete occurrences/preparation tasks, target evaluations, costs/completeness, effort schedule, shopping requirements, reasons, unresolved slots/conflicts, input references, and run metadata. Apply is a separate transactional command against an expected plan/workspace revision.

No output field may be populated with an LLM's free-text assertion in place of a domain calculation. An explanation can be rendered by a model, but the facts it describes must come from the evaluated result.

### 14.2 PLN-02 — Required constraints versus preferences

Required constraints include active food/allergen exclusions; viable equipment/method; explicit time/capacity constraints; user-locked occurrences; permitted portions; batch availability; and targets marked required under the configured data policy. Preferred targets and user priorities contribute penalties/rewards without masquerading as guarantees.

A budget may be a preference or a required cap, selected explicitly. If it is a required checkout cap, unknown prices prevent validating that cap. Use the same principle for maximum preparation time and incomplete duration data. Unknown cost must not make a candidate artificially look cheaper than priced alternatives.

Locked occurrences are hard user commitments for normal replanning. If a new allergy conflicts with a future lock, return a conflict and ask the user to replace/unlock that occurrence; do not keep displaying it as compliant. Historical consumption remains a historical record regardless of current eligibility.

### 14.3 PLN-03 — Initial algorithm

Use a bounded, deterministic search before considering a more sophisticated solver. A recommended R0 algorithm filters candidates by rules/methods, ranks them by known cost/effort/repetition with uncertainty penalties, and selects dinners sequentially while preserving locks. Make its limitations visible.

R1 needs plan-level evaluation because nutrients, package costs, leftovers, and repetition interact. A practical default:

1. Expand accepted routine anchors and fixed supplement occurrences.
2. Resolve candidate meal revisions, preparation methods, and permitted portion choices for each flexible slot.
3. Reject ineligible candidates and retain structured reasons for exclusions.
4. Precompute per-candidate nutrient vectors, effort, ingredient demands, and known-data masks.
5. Construct bounded candidate plans using beam search, greedy insertion plus repair, or a repository-appropriate constraint solver.
6. Evaluate full-period required constraints, batch flows, checkout packages, resource/time caps, and completeness.
7. Rank feasible evaluated plans by a transparent objective; retain a best partial diagnostic plan when no fully evaluable plan is found.
8. Apply a local improvement pass for substitutions/portion adjustments if within budget.
9. Return the draft and evidence; apply only after revision validation and user-selected scope.

Recommended configurable search defaults: up to 20 useful candidates per slot, a smaller branching subset such as 8 during search, beam width around 24, and a foreground time budget around 2 seconds on the reference development environment. Benchmark rather than promise those numbers. Large or expensive searches run as cancellable jobs and keep the current plan usable.

A heuristic timeout means “No feasible plan found in this search,” not proof that none exists. A directly contradictory rule set or an exhaustive small finite search may establish infeasibility; describe the actual evidence.

### 14.4 PLN-04 — Objective design

Within valid plans, minimize a documented normalized score. A starting structure is:

```text
score(plan) =
  w_cost    × normalized_cost(plan)
+ w_effort  × normalized_effort(plan)
+ w_variety × repetition_penalty(plan)
+ configured_preferred_target_penalties(plan)
+ uncertainty_penalties(plan)
+ optional_change_penalty(plan, current_plan)
```

Weights come from preferences, not medical authority. Normalize the three raw preference weights by their nonzero sum for the displayed base score; other penalties have separately versioned scales. Fix metric normalization scales in versioned policy rather than renormalizing every candidate pool unpredictably. State which cost objective is active: portion cost, incremental checkout, or a user-visible combination. Do not compare dollars directly to minutes without normalization or an explicit user-approved valuation.

Effort should include active work, dishes/cleanup, novel tasks, shopping stops, and batch-session overhead where modeled. Keep elapsed time visible separately. Variety considers exact recipe repetition plus flavors/textures/families, preventing cosmetic recipe duplicates from gaming “different dinners.” Use actual history where available and distinguish planned history from recorded history.

Preferred nutrient targets have bounded penalties. Additional protein or vitamins above a satisfied preferred goal do not produce an unlimited reward. Required upper bounds remain separate checks. The optimizer must not keep increasing portion sizes merely to gain a score advantage.

Change penalty protects a mostly accepted plan from unnecessary churn. Reranking should retain existing acceptable meals when differences are negligible and the user has not asked for a fresh week.

An optional user-facing effort tier provides a shortcut alongside measured time: **0 Grab and eat**, **1 Heat or mix**, **2 Simple assembly**, **3 Cook a meal**, **4 Batch or involved preparation**. These are editable qualitative descriptions, not universal minute thresholds. A batch session can be tier 4 while its later leftover portions are tier 1. Store tier separately from active/elapsed minutes and cleanup burden; changing the tier does not fabricate durations.

### 14.5 PLN-05 — Portion, batch, and resource feasibility

Each recipe has default serving size and allowed range/discrete steps. Avoid impractical fractional packaging, capsule counts, and extreme portion sizes. Portion choices are candidates, not unrestricted continuous variables unless supported.

Prepared batch inventory is allocated across dated occurrences. A planned cooking task can produce multiple servings; its shopping demand and active prep occur once. Leftover occurrences consume batch portions and add only reheating/assembly effort. The user controls storage/use-by assumptions, with source notes where available; do not invent shelf-life safety.

The preparation DAG gives dependency order. Resource constraints determine actual scheduling: one oven, one microwave, pan capacity, and time windows when configured. Parallel independent steps cannot automatically share an exclusive appliance simultaneously. R0 can report entered total time; R1 must avoid claiming a computed critical path when capacity data is absent.

### 14.6 PLN-06 — Infeasibility, uncertainty, and repair

Return actionable reason codes and affected rules/slots. Examples: no reviewed soy-free candidate; every allowed dinner requires a missing appliance; a locked meal exceeds a configured upper bound; seven dinners cannot establish full-day intake; missing prices prevent a budget check; a leftover is scheduled before its batch is prepared.

Offer a ranked set of possible user actions: add a candidate, fill a missing value, choose a viable alternate method, unlock a meal, broaden a preference, change the planning scope, or adjust a configured target deliberately. Required restrictions are never changed automatically.

Do not claim a minimal conflict set unless the algorithm computed one. It is acceptable to show “These two conditions are blocking the current candidates” with bounded evidence. Include the search-space/budget limit in advanced diagnostics, not in the ordinary dinner headline.

### 14.7 PLN-07 — Learning, spontaneity, and emergency foods

Allow open/social slots and optional wildcard allowances without a hardcoded number per week. An open slot reserves an explicit known budget if the user supplies one, otherwise remains unknown. It does not secretly contribute a nutritionally ideal meal.

Support reviewed emergency foods and low-effort meals that the user actually likes. A higher portion price can still be useful compared with a recorded or user-supplied alternative, but do not invent a prevented-takeout saving or probability. Show “More expensive than this planned meal; less work” when that is the available evidence.

Longer-term adherence prediction, probabilistic inventory, and personalized expected-waste modeling are deferred experiments. Preserve their proposed inputs and evaluation criteria, but begin with transparent rules and voluntary feedback.

## 15. Cost, inventory, leftovers, and shopping calculations

### 15.1 CST-01 — Portion cost and checkout cost

For a package price `P`, usable compatible package quantity `Q`, and consumed quantity `q`:

```text
allocated_portion_cost = P × q / Q
missing_quantity = max(0, planned_requirement - usable_available_stock)
packages_to_buy = ceil(missing_quantity / package_quantity)
estimated_checkout = packages_to_buy × package_price
```

Use edible/drained quantity where that is the selected cost basis, with the conversion recorded. If a can has only net weight and the recipe uses drained weight, do not silently treat them as equal. With multiple package options, choose a purchase combination under the selected policy and retain the selected offers in the result.

Inventory on hand can reduce additional checkout without making consumed food economically free. A full ingredient's purchase cost should not be charged again to every leftover portion. Recipe-level asserted cost overrides a derived estimate only with explicit authority selection; retain an explanation of its coverage.

### 15.2 CST-02 — Price observation selection

Store product/package, retailer, amount/currency, date observed, valid-through if known, availability, membership/coupon/minimum-buy conditions, and source. Prefer the user's configured accessible retailer/offer set and evidence quality. Generic ingredient names do not authorize substituting a different branded formulation when nutrients or allergens differ.

Recommended freshness policy: configurable by source and category; show age and permit manual prices to remain usable with an “older estimate” label. Do not invent a universally correct number of days. A price refresh produces a new observation, not an overwrite of actual recorded purchases or historical consumed-cost snapshots.

A missing price remains unpriced. For ranking, use a documented uncertainty penalty or conservative configurable estimate marked internal-to-ranking. Do not display that fallback as an observed price or use it to declare a required budget satisfied.

### 15.3 INV-01 — Stock ledger and projections

Use events or equivalent transactional records for purchase, manual assertion, correction, preparation consumption, batch creation, portion consumption, waste, transfer, and reversal. The current projection can be recomputed and tested. Every operation has an idempotency key.

Recommended default stock allocation uses earliest relevant use-by assertion first, where compatible with product/quality rules. Unknown storage status is not treated as guaranteed usable. “Have some” may support a prompt but cannot subtract an invented exact quantity from a mandatory shopping requirement.

Future reservations are separate from on-hand. Available-for-new-plan may be computed as on-hand minus active reservations, but the planner must exclude its own replaced reservations when rebuilding that same plan. Otherwise every replan appears to run out of stock.

### 15.4 INV-02 — Batch accounting

At cooking confirmation, consume raw ingredient quantities once, create an actual batch with yield and its recipe revision, and release the fulfilled preparation reservation. If actual yield differs, recalculate per-portion allocation under the recorded batch basis and show the change. Do not change the original recipe yield silently.

A leftover occurrence references a batch or planned batch output. Consuming it reduces that batch's available quantity and records nutrients/cost from the pinned batch snapshot. Canceling a future leftover meal releases a reservation but does not discard the food. Waste requires an explicit event or clearly labeled estimate.

Undo cooking is valid only if no downstream batch portions have been consumed or otherwise allocated in a way that cannot be reversed atomically. Otherwise use a correction workflow that explains affected records. Undo consumption restores the relevant batch amount exactly once.

### 15.5 INV-03 — Approximate inventory without excessive prompts

Treat uncertain inventory as data quality, not a demand for precise daily weighing. Maintain last confirmed amount and inferred changes. Ask for confirmation only when it materially changes a recommendation, for example before deciding whether another package is needed.

A “Do you still have tofu?” Yes response means available/qualitative unless a specific amount was shown in the question. No sets an out-of-stock assertion at that time. Both can supersede an old estimate while preserving event history.

### 15.6 CST-03 — True savings and tradeoffs

Present planned portion cost, expected checkout, and recorded actual spend separately. Savings claims require a named baseline, comparable quantities/date range, included fees, and known-data status. If the baseline is absent, show the proposed costs and tradeoffs without a savings percentage.

Optional effective-cost analysis can add expected waste, extra delivery, travel, and decision burden, but must display assumptions rather than hide them in a single dollar number. The user can reject a nominal saving that creates an extra shopping trip. Automation itself has a cost; a research job for a tiny possible saving needs a bounded value/cost policy.

## 16. AI assistance, data acquisition, and recurring jobs

### 16.1 AI-01 — What AI does and what ordinary code does

Use AI where interpreting messy input, suggesting a plausible variation, or explaining a tradeoff reduces work. Use deterministic code for arithmetic, authorization, validation, state transitions, required constraints, unit conversion, revision checks, and applying changes. The application must remain useful when no model provider is configured.

| Task | Appropriate AI contribution | Required deterministic control |
| --- | --- | --- |
| Pasted meal notes | Propose structured ingredients, steps, and a title. | Preserve source; validate schema; retain unresolved quantities. |
| Product label photo | Locate and transcribe visible fields with evidence spans/crops. | Check units, basis, field conflicts, and required user review. |
| Recipe URL | Organize permitted source content into a draft. | Controlled fetch, source attribution, graph/reference validation. |
| Meal variation | Propose ingredient or method alternatives. | Recalculate and recheck all affected required constraints. |
| Weekly plan | Propose candidates or explain a deterministic result. | Run the planner; never accept prose as proof of feasibility. |
| Price research | Find candidate offers from configured sources. | Verify package, conditions, date, source, currency, and identity. |
| Nutrition gap explanation | Explain supplied assessment results. | Supply scoped totals and missing data; reject unsupported claims. |
| Preference inference | Propose a reversible preference update. | Keep inferred versus explicit preferences separate; allow correction. |

An AI-generated “vegan” label is a proposal, not a verified diet assertion. A model's confidence score is not a calibrated probability of allergy safety, nutritional accuracy, or product availability.

### 16.2 AI-02 — Extraction and review pipeline

The pipeline is **capture → extract → validate → compare → review → apply**. Persist a draft/job boundary between stages so retrying a model request does not create another meal.

1. Capture the original source, permitted attachment reference, source type, and an integrity hash.
2. Build a narrowly scoped request containing only the information needed for that task.
3. Request a versioned structured response. An adapter converts provider-specific output to a domain proposal.
4. Validate syntax, quantities, references, supported units, and semantic consistency. Reject a malformed field without losing the original source.
5. Attach evidence to each material proposed fact. Distinguish a transcription from an inference.
6. Compare against the current draft revision. Mark fields edited since extraction began as conflicts.
7. Show a compact review of material ambiguities and consequential changes. Do not require review of every formatting choice.
8. Apply accepted changes through the same domain operation used by the manual editor, with an expected revision and idempotency key.

A proposal should contain its schema version, source IDs, base draft revision, proposed field changes, evidence references, unresolved questions, and validation warnings. Do not include executable tool instructions in imported recipe data. The model cannot decide which workspace to write into or grant itself permission to publish, purchase, or contact a third party.

If a label displays energy and protein but omits iodine, preserve iodine as unknown. If a photo is unreadable, request a clearer crop or let the user enter the field manually. A best guess may be stored as an explicitly estimated proposal where useful, but must not silently replace a legible label value.

### 16.3 AI-03 — URL and attachment ingestion

R2 link ingestion should handle direct recipe pages and readable text before attempting complex authenticated sites. Store the URL immediately even if fetching is unavailable. Show fetch/extraction status separately from whether the meal draft was saved.

Fetch through a controlled server adapter. Restrict supported schemes; reject local/private/link-local destinations; validate resolved addresses and each redirect; bound response size, redirects, and time; isolate outbound credentials. Avoid arbitrary server-side requests to user-supplied targets. These controls address the class of issues described by the [OWASP SSRF Prevention Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html).

Treat all fetched content and document text as untrusted data. A page that says “ignore previous instructions” has no authority over the application. Scripts, hidden instructions, embedded forms, and external links do not become domain actions. Render imported rich text with a safe allowed subset or plain text. Enforce attachment MIME/size limits and private access; do not execute document macros.

Keep source attribution and applicable content-use metadata. Do not promise arbitrary-site scraping or unrestricted redistribution of third-party recipe text/photos. A personal source reference and an imported user draft do not automatically authorize inclusion in a public commercial catalog. Provider access and content rights are deployment decisions, not invented capabilities.

### 16.4 DATA-01 — Nutrition provider adapter

Begin with manual entries and a reproducible test fixture. Add a provider interface with search, detail retrieval, source identity, source version/observed date, nutrient mapping, caching, and error classification. A suggested implementation is USDA FoodData Central, whose official API documents search/details endpoints, key-based access, and request limits. Keep credentials server-side and respect the provider's current limits rather than hardcoding a promise of unlimited requests. See the [FoodData Central API guide](https://fdc.nal.usda.gov/api-guide/).

Search results must show enough information to select the correct item: generic/branded identity, preparation state, brand where relevant, serving/basis, and source type. “Rice” is insufficient to equate dry rice, cooked rice, and a microwave product. A selected result creates a pinned local source revision. Later refreshes propose new revisions rather than silently changing past intake.

Match products by strong identifiers when available, supplemented by package/formulation information. Name similarity can suggest candidates but cannot certify identity. Expose “None of these; add manually” prominently. Unknown mappings are queued for resolution, not silently assigned to a vaguely similar food.

### 16.5 DATA-02 — Price and receipt adapters

A price adapter returns structured observations, not a single universal price for an ingredient. It must preserve the offer conditions described in CST-02. A store with no configured integration still supports manual package prices and links.

Start with user-entered recurring purchases because those may capture most relevant costs with little setup. Later research can prioritize a small set of expensive, frequently consumed products. The app may suggest that protein powder or a staple deserves a price comparison, but it cannot infer the user's exact current Amazon purchases from this specification.

Receipt, order-history, or email access is opt-in and source-specific. Request only the needed scope and explain what will be imported. A receipt parser stages purchase proposals, highlights uncertain quantities/product matches, and deduplicates by source transaction identity plus normalized line content. Do not automatically turn every historical receipt into current pantry stock; old purchases may already have been consumed.

Store retailer/API adapters behind interfaces. Amazon, Costco, Walmart, Target, grocery delivery services, and other retailers are potential sources, not promised supported integrations. Evaluate actual access, terms, geographic coverage, and maintenance cost before implementing an adapter. No integration in R0–R2 performs checkout or places an order.

### 16.6 JOB-01 — Background job contract

Persist jobs with states `queued`, `running`, `waiting_for_review`, `succeeded`, `failed`, and `canceled`. A job can succeed at extraction while its proposal remains unapplied. Track that application status separately. A proposal that becomes stale requires another comparison against current revisions.

Every job records its workspace, type, deduplication key, input revisions, attempt count, timestamps, provider, budget, result reference, and safe error code. Progress should describe meaningful stages; never display invented precise completion percentages.

Retry transient failures with bounded backoff and provider-aware handling. Do not retry schema/permission failures indefinitely. A canceled job may still receive an external response; discard or retain it as canceled evidence without applying it. Job completion cannot overwrite newer user edits.

Set explicit limits for tokens/spend, runtime, retries, items, and concurrent jobs. Estimate and record actual usage where providers expose it. Keep user-facing budgets understandable, such as a configurable monthly research allowance. A failed or unavailable model provider leaves manual editing and deterministic planning operational.

### 16.7 JOB-02 — Low-maintenance recurring behavior

R2 may generate a proposed upcoming week at a user-selected local time, optionally suggest a short preparation session, and identify a few meaningful price/data questions. Scheduling is opt-in. Store the recurrence in the profile's IANA timezone and define behavior when the user changes zones.

The default scheduled action creates a draft. It does not replace accepted plans, alter supplement schedules, mark food eaten, or buy groceries. A quiet in-app badge is the default notification surface; email/push messages require an enabled channel and user preference.

A useful digest might say: “Next week's draft reuses two staples, adds two new dinners, and needs one pantry check.” Its numbers must come from that draft. Suppress repetitive questions already answered or dismissed. Deduplicate a scheduled run by workspace, recurrence, and intended local period so retries and daylight-saving changes do not create multiple plans.

## 17. Architecture and application contracts

### 17.1 ARCH-01 — Integrate with the actual repository

Before changing code, inspect repository instructions, existing application structure, auth/session handling, persistence, migration tooling, UI components, jobs, test commands, and configuration conventions. The user's broader environment may be Vrooli, but this document does not assert that a particular Vrooli service, endpoint, resource, or code generator exists.

Prefer repository-native solutions. If it is a TypeScript web repository, a typed component UI, runtime-validated API contracts, and pure domain modules are a reasonable default. If its backend is another language, preserve the contracts and share generated schemas or fixtures where useful. Do not replace an established platform merely to match the prototype's framework.

Reuse existing authentication, provider routing, model roles, storage, telemetry, scheduling, and notification facilities when they meet the requirements. Do not create a general-purpose optimization platform as a prerequisite to building the food app. Identify genuinely reusable domain utilities only after the product has a working vertical slice.

### 17.2 ARCH-02 — Module boundaries

Recommended boundaries, independent of folder names:

| Module | Owns | Must not own |
| --- | --- | --- |
| Profile and rules | Diet, restrictions, kitchen capabilities, targets, preferences. | Billing truth or recipe parsing. |
| Food catalog | Food/product identities, revisions, nutrient evidence, unit mappings. | Unreviewed model text as canonical truth. |
| Recipe library | Drafts, immutable revisions, method graphs, components/families. | Current inventory balance. |
| Nutrition engine | Unit-aware totals, coverage, target evaluation. | Network fetches or UI state. |
| Planning engine | Candidate eligibility, search, plan assessment, explanations. | Direct persistence side effects during search. |
| Inventory and costs | Stock events, packages, observations, purchase/batch projections. | Assuming planned actions actually happened. |
| Intake and feedback | Consumption, corrections, explicit feedback. | Mutating historical recipe revisions. |
| Transfer and documents | Versioned import/export, migration, printable layout. | Bypassing domain validation on import. |
| Provider/job adapters | External requests, structured proposals, bounded execution. | Deciding permissions from model output. |
| Application services | Authorization, transactions, revisions, operation orchestration. | Duplicated nutrient formulas in handlers. |
| UI | Interaction state, presentation, accessible editing. | A second incompatible planner or cost engine. |

Keep deterministic calculations callable without a browser, database, network, or model. Pass snapshots and policies into them; return values, provenance, and structured reasons. This makes the critical behavior testable and makes migrations away from one framework less expensive.

### 17.3 ARCH-03 — State ownership and invalidation

Separate server facts, derived assessments, editor drafts, and transient interface state. Opening a recipe tab does not mutate a recipe. Dragging a slider can update a local preview, while applying a profile change is an explicit persisted operation.

The server is authoritative for domain operations. The UI may optimistically display simple reversible changes, but must reconcile the returned revision or show failure. Expensive assessments should be keyed by relevant input revisions and evaluator version. A price update invalidates cost results; it need not erase a completed cooking session.

A form tracks dirty fields and its base revision. Blank numeric input remains blank/null. Avoid converting empty strings with generic number coercion. On a conflict, keep the local draft and offer reload/compare; never refresh away the user's work automatically.

Use an explicit dependency map for derived data. At minimum, ingredient composition affects nutrition and eligibility; method changes affect equipment/effort; price changes affect cost; stock changes affect shopping; target changes affect assessments; plan/portion changes affect all plan summaries. Do not rely on arbitrary page reloads for correctness.

### 17.4 API-01 — Operations and transport semantics

The following operation names are normative; exact REST paths or RPC names may follow the repository. All mutations are scoped to the authenticated workspace, validated on the server, and return an operation identity plus resulting revision where applicable.

| Operation | Input | Result / important behavior |
| --- | --- | --- |
| Read workspace summary | Date range, selected profile. | Active settings, plans, data-quality summary, revisions. |
| Save onboarding draft | Draft fields, expected draft revision. | Resumable draft; active settings unchanged. |
| Apply profile | Reviewed draft/base profile revision. | New active revision and affected-plan conflicts. |
| Create/update recipe | Draft or proposed revision, expected recipe revision. | Stable recipe ID, immutable content revision, readiness/assessment. |
| Archive recipe | Recipe ID, expected revision. | Removed from future candidates; history intact. |
| Generate plan | Scope, seed, policies, pinned input revisions. | Draft, assessments, candidate/search diagnostics. |
| Apply plan | Draft ID/hash, expected plan and relevant input revisions. | Atomic accepted plan or stale/conflict error. |
| Quote/apply swap | Occurrence, candidate revision/method/quantity. | Scoped impact preview; apply checks the quote inputs again. |
| Record/correct intake | Actual quantity/time, source reference, correction link. | Idempotent event and refreshed actual totals. |
| Confirm preparation | Recipe/method, actual yield, ingredient usage. | Batch plus stock effects applied together. |
| Edit/check shopping line | Line ID, desired state/override, expected revision. | Updated state; checking alone does not purchase. |
| Record purchase | Confirmed lines, actual packages/prices, source. | Purchase and inventory effects exactly once. |
| Stage import | File/manifest, requested import mode. | Validation report and reviewable proposed changes. |
| Apply import | Staged proposal ID/hash, policy choices, expected revision. | Transaction result with created/updated/skipped/conflicted counts. |
| Export backup / recipe / PDF / CSV | Scope, format/version, layout options. | Download or job with immutable input snapshot. |
| Cancel/retry job | Job ID, requested action. | Explicit state transition; no duplicate application. |

Idempotency keys identify a logical mutation. Repeating the same key and same payload returns the original result; the same key with a materially different payload returns a conflict. Scope keys to workspace and operation type. Keep the retention policy explicit and long enough for normal client/job retries; durable external transaction identities prevent old receipt imports from double-applying even after request-cache expiry.

### 17.5 API-02 — Example request and error

This is an illustrative transport body for generating a seven-day draft. Dates and IDs are examples, not initial application data. Input revisions must resolve to immutable records, and the server validates ownership.

```json
{
  "operation": "generate_plan",
  "workspaceId": "ws_example",
  "requestId": "request_example_001",
  "scope": {
    "start": "2026-09-21",
    "end": "2026-09-27",
    "timezone": "America/New_York",
    "slots": ["breakfast", "lunch", "dinner", "snack"]
  },
  "basePlanRevision": 4,
  "profileRef": {"id": "profile_example", "revision": 3},
  "seed": "example-week-1",
  "costMode": "additional_checkout",
  "inputSnapshotId": "inputs_example_001",
  "preserveLockedOccurrences": true
}
```

`workspaceId` is a requested scope, never proof of authorization. If the profile changes before applying this proposal, the application must revalidate or reject it as stale. Example error:

```json
{
  "error": {
    "code": "STALE_INPUTS",
    "message": "Your food settings changed after this draft was created.",
    "retryable": false,
    "operationId": "operation_example_002",
    "details": {
      "conflicts": [
        {
          "entityType": "profile",
          "entityId": "profile_example",
          "expectedRevision": 3,
          "currentRevision": 4
        }
      ],
      "suggestedAction": "revalidate_draft"
    }
  }
}
```

Use stable error codes such as `VALIDATION_FAILED`, `UNAUTHORIZED`, `NOT_FOUND`, `REVISION_CONFLICT`, `STALE_INPUTS`, `REQUIRED_CONSTRAINT_FAILED`, `MISSING_REQUIRED_EVIDENCE`, `UNSUPPORTED_IMPORT_VERSION`, `IMPORT_LIMIT_EXCEEDED`, `PROVIDER_UNAVAILABLE`, `BUDGET_EXCEEDED`, and `OPERATION_CANCELED`. Return field paths for validation errors. Do not expose another workspace's existence through detailed authorization errors.

Separate solver outcomes from transport errors. A valid request can return a partial draft and “no feasible result found within this search” as domain data. That should not become a generic HTTP 500 or an empty success toast.

### 17.6 API-03 — CLI and automation integration

If the repository has a standard scenario/application CLI, expose the same services through it. Useful commands include status, validate-data, list-recipes, generate-plan with dry-run output, export-backup, and inspect-job. Make machine-readable output available and keep diagnostic prose out of JSON stdout.

Do not implement a parallel planner in CLI code. Mutating CLI commands need the same authorization, expected revision, and idempotency behavior as the UI. Printing or exporting may use a local output path, but paths must not allow access outside the authorized execution context. Never print provider tokens or full private label images in diagnostics.

For a standalone repository without an existing CLI convention, a full end-user CLI is optional. Reliable development commands, migration instructions, and fixture/test runners are required.

## 18. Persistence, concurrency, privacy, and failure recovery

### 18.1 SYS-01 — Durable state and honest save feedback

Persist user settings, drafts, recipes/revisions, plans, feedback, shopping state, and relevant R1 records in durable storage. A browser refresh, new session, or another authorized device should not reset the account to demo seed data.

Show `Saving…`, `Saved`, or a clear failure state for edits whose save status is otherwise ambiguous. A successful local React update is not a persisted save. If autosave is used, serialize/coalesce pending edits per entity and preserve the last acknowledged revision. Do not send competing full-workspace snapshots that overwrite each other out of order.

Offer retry without losing the local draft. A network failure after the server commits is resolved by the idempotency key or re-reading the operation result, not by blindly creating a duplicate. A conflict is distinct from a network error and must be presented accordingly.

The prototype's durable account behavior is part of the expected UX. Development fixtures and anonymous exploration must be deliberately isolated from real accounts. Do not claim production durability while using only browser storage.

### 18.2 SYS-02 — Transactions and revision checks

The following operations must have all-or-nothing domain effects: profile activation, plan application, import/restore application, purchase plus stock update, preparation plus batch creation, and intake correction plus batch adjustment.

Use database transactions where possible. If the platform spans stores or queues, record a durable operation and use a tested outbox/compensation strategy for external side effects; do not call several unrelated writes “atomic.” Email or provider calls should not hold a database transaction open.

Compare expected revisions inside the transaction. Keep revision checks scoped to affected entities where practical so unrelated recipe edits do not block a grocery checkbox. A full restore legitimately needs a broader workspace barrier. Return the new revision and affected projections to the client.

Undo is a new operation with a known precondition, not a client-side rewind of all state. If later dependent actions prevent exact reversal, offer a correction and explain the affected facts. Preserve the original event and correction link for auditability.

### 18.3 SYS-03 — Identity and access isolation

Derive the actor and accessible workspace scope from the authenticated server session. Validate ownership of every referenced recipe, file, plan, batch, source, and job, including nested references inside imports and AI proposals. Never trust a client-supplied owner ID.

Use the platform's normal secure session and request protections. Administrative/provider credentials remain server-side. Client bundles, export files, logs, URLs, and generated PDFs must not contain secrets. Authorization applies equally to downloads and background job result URLs.

Public/community recipe sharing is deferred. Until implemented, saved recipes and source attachments are private to their workspace. A future sharing feature must deliberately handle attribution, permission, redaction, and copied versus linked revisions.

### 18.4 SYS-04 — Personal data and observability boundaries

Treat food logs, restrictions, targets, supplement schedules, receipt details, and labels as private personal data. Send a model only the task-relevant subset. Do not include account identifiers in a model prompt when a local opaque reference is enough. Disclose configured provider behavior in appropriate product settings.

Logs should contain operation IDs, timings, counts, error codes, and safe revision references. Avoid raw recipe text, receipt contents, medical notes, access tokens, and attachment URLs by default. Debug capture, if supported, requires deliberate bounded activation and a retention policy.

Support export of owned domain data without a paid upgrade. For a commercial deployment, implement account deletion and define retention, backups, and deletion timing using the actual hosting/provider policies. Do not make unverified regulatory certification claims. This specification establishes product requirements; deployment-specific legal wording still needs appropriate review when commercial release is undertaken.

### 18.5 SYS-05 — Migrations, backup, and restore

Version database migrations independently from portable export schemas and calculation algorithms. Test upgrading a populated prior database, not only creating an empty one. Back up before destructive migrations under the repository's deployment process.

A native export identifies its format, schema version, creation time, included record kinds, and omissions. Include enough historical revisions to resolve exported plans and intake. A source attachment omitted from a text-only export must be described as omitted, with its metadata/reference preserved; do not label that export a complete attachment backup.

Full restore stages and validates first. Create a recoverable checkpoint before replacing domain state. Restore into the currently authorized workspace using an explicit identity remapping policy; never import original account membership, credentials, entitlements, or access tokens. If restoration fails, the old usable state must remain available.

Recommended compatibility policy: write the current schema; support the previous supported native version through an explicit tested migrator; reject future versions with actionable copy. Keep small legacy prototype recipe files supported through the separate adapter in section 20.8. A permissive JSON parser is not a migration strategy.

### 18.6 SYS-06 — Offline behavior, time, and background changes

R0 must clearly distinguish online persisted state from unsaved local edits. Optional cached read access is useful, but full offline multi-device sync is not required. Do not show an offline purchase or consumption as server-confirmed without a durable queued-operation design.

If offline mutation queues are added later, use client-generated operation IDs, preserved expected revisions, and explicit conflict handling. Retrying an offline batch action must not consume ingredients twice. Queue status and blocked operations must be inspectable.

Store UTC instants for events and IANA timezones plus local dates for planning. A meal intended for Monday remains Monday when displayed in that plan's timezone. Daylight-saving transitions must not add or remove planned slots. For ambiguous scheduled local times, choose and document one execution policy and deduplicate by intended local period. Editing a profile timezone should preview future scheduling changes without rewriting historical event instants.

## 19. Performance, operations, and commercial readiness

### 19.1 OPS-01 — Performance and responsiveness

Use a declared reference dataset and development environment when measuring. A useful R1 benchmark contains 1,000 saved recipe revisions, 90 days of plans/intake, 200 products, and a seven-day full-day planning request. These are test sizes, not product limits.

Recommended interaction goals: immediate local control feedback; a useful initial screen without waiting for AI; foreground plan search near the bounded budget in PLN-03; long exports/imports shifted to progress-visible jobs. Measure representative devices before making public speed promises.

Paginate/search large collections and cache versioned assessments. Batch provider calls where permitted. Avoid recalculating the entire history for a single checkbox. Use virtualization only where the measured list size warrants it; preserve keyboard access and stable focus.

A slow solver can return its best validated draft plus search status. A slow provider cannot freeze manual editing. Expensive work should have cancellation and a visible status, with partial results clearly identified.

### 19.2 OPS-02 — Diagnostics and supportability

Provide health checks for database access, migration version, job execution, and configured provider availability. A provider not configured is a supported state, not necessarily a failed application health check.

Capture operation duration, job retries/failures, validation categories, stale-write conflicts, calculation versions, and import counts. A support diagnostic export should omit personal content by default and say what it contains. Correlate errors by operation ID so the user need not paste their whole diet into a bug report.

Expose user-facing data health where actionable: unresolved ingredients, old price observations, unknown targets, incomplete nutrient coverage, and failed scheduled jobs. Avoid a frightening global score whose meaning is unclear. Prioritize issues that change the current week's decision.

### 19.3 BIZ-01 — Commercial foundations without premature billing

The user wants a product that can eventually serve different diets and be monetized. Build configurable policies, private workspace ownership, provider adapters, and entitlement boundaries accordingly. Do not hardcode vegan-only fields into the data model or UI copy.

R0–R2 do not require a checkout/paywall. In development and personal use, capabilities can be enabled by configuration. Possible future paid value includes larger planning/research allowances, additional integrations, advanced household features, and richer automation. These are hypotheses; no pricing or willingness to pay has been validated.

Enforce expensive-job limits on the server per workspace/account, with clear remaining-budget messages. A model outage or exhausted AI allowance must not block viewing recipes, editing meals, accessing existing plans, or exporting owned data. Nutrition safety checks and required restrictions are core behavior, never an upsell.

### 19.4 BIZ-02 — Future subscriptions and sharing

If R3 billing is commissioned, use the selected provider's server-verified entitlement state. Handle webhook retries idempotently, out-of-order updates, cancellation/end dates, and provider outages. Keep billing customer identifiers and secrets out of portable meal backups. Test entitlements separately from local UI visibility.

Household collaboration requires deliberate roles and profile-to-person mapping. Multiplying a recipe's serving count is supported earlier; treating multiple people as one nutritional profile is not. Child, pregnancy, clinical, and other specialized target workflows require separate product decisions and reference handling, not a silent extension of an adult default.

Defer public catalogs, social feeds, affiliate ranking, and advertising unless explicitly prioritized. If affiliate relationships are ever added, preserve transparent price comparisons and disclose ranking incentives rather than presenting them as neutral optimization.

## 20. Worked examples and embedded implementation fixtures

All numbers and product identities in this section are **synthetic test data**. They are not a recommended diet, verified commercial prices, or the user's personal targets. Build executable fixtures from these examples. They specify expected behavior more precisely than attractive sample dashboards alone.

### 20.1 FIX-01 — Saving almost no information

Input: the user opens Add meal, types `My usual breakfast`, and presses Save draft without supplying anything else.

Expected result:

- One stable recipe identity and its first content revision are persisted.
- Authoring status is Draft. Yield, nutrients, cost, and effort are unknown; ingredient and method collections are empty.
- The collection shows the title and an understandable “Add details” action.
- The card does not show `0 kcal`, `0 g protein`, `$0.00`, or `0 min`.
- It is available to edit after refresh, but is excluded from automatic planning until the relevant readiness requirements are met.
- A failed save retains the entered title locally and offers retry with the same logical operation key.

A null yield is valid for a draft. Portion-based calculations and automatic scheduling require a positive resolved yield or an explicit user-confirmed per-portion assertion with a usable serving basis. The system must not pretend that a default input placeholder established the meal's actual yield.

### 20.2 FIX-02 — Complete synthetic recipe arithmetic

Create “Example tofu crunch bowl”, recipe revision 1, yielding exactly two servings. Every quantity below refers to the complete two-serving recipe. All products use the same ready-to-eat/as-used gram basis for this fixture, so no raw-to-cooked conversion is required. Energy and protein are fictional declared values; all other nutrients are unknown unless separately provided.

| Ingredient use ID | Item | Recipe grams | Protein per 100 g | Energy per 100 g | Package grams | Package price USD |
| --- | --- | ---: | ---: | ---: | ---: | ---: |
| `i_tofu` | Example ready-to-eat tofu | 400 | 12 g | 100 kcal | 400 | 4.00 |
| `i_rice` | Example ready-to-eat rice | 300 | 3 g | 130 kcal | 500 | 2.00 |
| `i_edamame` | Example prepared edamame | 200 | 11 g | 120 kcal | 500 | 3.00 |
| `i_sauce` | Example sauce | 40 | 5 g | 200 kcal | 200 | 2.00 |
| `i_slaw` | Example ready-to-eat slaw | 200 | 1 g | 25 kcal | 300 | 1.50 |

Exact calculations:

```text
Recipe protein = 400/100×12 + 300/100×3 + 200/100×11 + 40/100×5 + 200/100×1
               = 48 + 9 + 22 + 2 + 2 = 83 g
Per-serving protein = 83 / 2 = 41.5 g

Recipe energy = 400/100×100 + 300/100×130 + 200/100×120 + 40/100×200 + 200/100×25
              = 400 + 390 + 240 + 80 + 50 = 1,160 kcal
Per-serving energy = 1,160 / 2 = 580 kcal

Recipe allocated cost = 400/400×4 + 300/500×2 + 200/500×3 + 40/200×2 + 200/300×1.5
                      = 4 + 1.2 + 1.2 + 0.4 + 1 = 7.8 USD
Per-serving allocated cost = 7.8 / 2 = 3.9 USD
```

At 1.5 servings, the occurrence contributes **62.25 g protein, 870 kcal, and $5.85 allocated cost**. Ingredient quantities multiply by 0.75 relative to the two-serving recipe. The recipe itself remains unchanged. Its cost is complete for these listed ingredients under these observations; it excludes taxes, energy, delivery, and other unmodeled costs.

### 20.3 FIX-03 — Packages, pantry, and leftovers

Plan four servings of FIX-02 across two occurrences. The complete raw/as-used requirement is twice the recipe. Confirm exact starting pantry amounts as below. No other plans reserve this stock, all offers are available, and only the listed package size exists for each item.

| Item | Required g | Confirmed stock g | Missing g | Packages to buy | Checkout USD | Remaining after all four servings g |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Tofu | 800 | 200 | 600 | 2 × 400 g | 8.00 | 200 |
| Rice | 600 | 100 | 500 | 1 × 500 g | 2.00 | 0 |
| Edamame | 400 | 0 | 400 | 1 × 500 g | 3.00 | 100 |
| Sauce | 80 | 50 | 30 | 1 × 200 g | 2.00 | 170 |
| Slaw | 400 | 100 | 300 | 1 × 300 g | 1.50 | 0 |
| **Total** | | | | | **16.50** | |

Four servings have **$15.60 allocated food cost**, while the additional checkout is **$16.50**. Both numbers are correct and answer different questions. The difference reflects starting stock and leftover purchased quantities. Neither number is automatically actual spend.

Before purchase confirmation, the shopping proposal does not increase on-hand inventory. After confirming exactly these purchases, inventory increases once. Confirming preparation of the four-serving batch consumes the listed ingredient requirements once and creates four batch servings. Eating two servings now leaves two batch servings. Eating those two tomorrow reduces the batch to zero without consuming the raw ingredients again.

The remaining ingredient stock in the table still exists separately from the prepared batch. Canceling tomorrow's planned occurrence releases its reservation; it does not erase the two prepared servings. A subsequent explicit waste action can do that.

### 20.4 FIX-04 — Unknown nutrients and target scope

Remove the sauce's protein and energy values from FIX-02 while preserving its known quantity. The result is **40.5 g known protein per serving plus an unknown sauce contribution**, and **540 kcal known energy per serving plus an unknown sauce contribution**. Do not substitute the complete fixture's old values into the new source revision.

For synthetic nonnegative nutrient quantities:

| Example target | Known information | Correct assessment |
| --- | --- | --- |
| Protein lower bound 40 g for this portion | Known subtotal 40.5 g, sauce unknown. | Lower bound is met; completeness remains partial. |
| Energy upper bound 600 kcal for this portion | Known subtotal 540 kcal, sauce unknown. | Unknown; the missing contribution could exceed the bound. |
| Energy range 500–600 kcal | Lower bound established, upper bound unresolved. | Overall range assessment unknown. |
| Energy upper bound 500 kcal | Known subtotal already 540 kcal. | Fail, even with missing information. |
| Vitamin B12 target with no ingredient values | No known contribution. | Unknown, not “0 consumed.” |

The lower-bound result is permissible because nonnegative missing contributions cannot undo a lower bound already met. This does not authorize an “all nutrition complete” badge. If a domain quantity can be signed or has another mathematical interpretation, do not reuse this reasoning without checking that interpretation.

For a separate fully known synthetic day, use breakfast 30 g protein/500 kcal, lunch 35 g/600 kcal, the complete FIX-02 dinner 41.5 g/580 kcal, and snack 10 g/200 kcal. The day totals **116.5 g protein and 1,880 kcal**. A configured test target of protein at least 100 g and energy at most 2,000 kcal passes for that complete scope. Those test targets must never become application defaults.

If only dinner is planned, show 41.5 g and 580 kcal for dinner with “rest of day not covered.” Do not report that the day's lower protein target failed as if no other food will be eaten, and do not report that a daily calorie ceiling is guaranteed satisfied. An actual-versus-target view can show the known intake so far while retaining incomplete-day context.

### 20.5 FIX-05 — Recipe graph with parallel preparation

Use the FIX-02 ingredient IDs. This JSON represents one `PreparationMethod`, not an entire recipe or import envelope. It demonstrates branches, named outputs, and a join. Instructions intentionally defer product-specific heating details to the selected package; this fixture does not invent temperatures or safety claims.

```json
{
  "id": "method_microwave",
  "label": "Microwave and assemble",
  "requiredCapabilities": ["microwave"],
  "activeMinutes": null,
  "elapsedMinutes": null,
  "steps": [
    {
      "id": "s_warm",
      "title": "Warm the base",
      "instruction": "Warm the ready-to-eat rice and prepared edamame according to their selected package instructions.",
      "after": [],
      "allocations": [
        {"ingredientUseId": "i_rice", "quantity": {"amount": "300", "unitId": "g"}, "disposition": "incorporated"},
        {"ingredientUseId": "i_edamame", "quantity": {"amount": "200", "unitId": "g"}, "disposition": "incorporated"}
      ],
      "inputOutputIds": [],
      "outputs": [{"id": "o_base", "name": "Warm base", "quantity": null}],
      "activeMinutes": null,
      "elapsedMinutes": null,
      "equipmentCapabilities": ["microwave"],
      "optional": false
    },
    {
      "id": "s_prep",
      "title": "Prepare the crunch",
      "instruction": "Portion the ready-to-eat tofu and slaw into a mixing bowl.",
      "after": [],
      "allocations": [
        {"ingredientUseId": "i_tofu", "quantity": {"amount": "400", "unitId": "g"}, "disposition": "incorporated"},
        {"ingredientUseId": "i_slaw", "quantity": {"amount": "200", "unitId": "g"}, "disposition": "incorporated"}
      ],
      "inputOutputIds": [],
      "outputs": [{"id": "o_crunch", "name": "Tofu and slaw", "quantity": null}],
      "activeMinutes": null,
      "elapsedMinutes": null,
      "equipmentCapabilities": [],
      "optional": false
    },
    {
      "id": "s_combine",
      "title": "Dress and combine",
      "instruction": "Combine both prepared parts with the sauce.",
      "after": ["s_warm", "s_prep"],
      "allocations": [
        {"ingredientUseId": "i_sauce", "quantity": {"amount": "40", "unitId": "g"}, "disposition": "incorporated"}
      ],
      "inputOutputIds": ["o_base", "o_crunch"],
      "outputs": [{"id": "o_bowl", "name": "Finished bowl", "quantity": {"amount": "2", "unitId": "serving"}}],
      "activeMinutes": null,
      "elapsedMinutes": null,
      "equipmentCapabilities": [],
      "optional": false
    },
    {
      "id": "s_serve",
      "title": "Serve",
      "instruction": "Divide the finished bowl into the two declared servings.",
      "after": ["s_combine"],
      "allocations": [],
      "inputOutputIds": ["o_bowl"],
      "outputs": [],
      "activeMinutes": null,
      "elapsedMinutes": null,
      "equipmentCapabilities": [],
      "optional": false
    }
  ]
}
```

Expected map: Warm the base and Prepare the crunch are both initially available; Dress and combine depends on both; Serve depends on the combination. Each ingredient enters once. Referencing `o_bowl` during serving must not add all its nutrients a second time.

Negative variants: a step depending on itself; `s_warm` depending on `s_serve`; a missing output ID; a quantified sauce allocation of 60 g when only 40 g exists; and duplicate output IDs. All are rejected with specific, editable error paths. A null allocation can remain an unresolved visual link, but it cannot prove a quantified split is valid.

### 20.6 FIX-06 — Required rules beat preferences

Synthetic active profile: vegan, excludes soy, owns no heating appliances, permits manual assembly, and has no numeric nutrition target yet. For this fixture, “reviewed compatible” means explicit evidence for every active rule; a blank tag list alone does not qualify.

| Candidate | Evidence / method | Result |
| --- | --- | --- |
| Cheap tofu bowl | Contains soy. | Ineligible regardless of low price. |
| Chicken salad | Contains meat. | Ineligible under vegan rules. |
| Lentil salad A | Plant-based composition reviewed, soy status unresolved. | Needs information; not an automatic recommendation. |
| Lentil salad B | Reviewed compatible with active rules, ready-to-eat assembly. | Eligible. |
| Lentil soup | Reviewed compatible ingredients, only method requires stove. | Ineligible due to unavailable capability. |
| Bean bowl with alternatives | Compatible ingredients; one microwave method and one reviewed ready-to-eat assembly method. | Eligible using the assembly method. |

Setting cost weight to 100 and other preference weights to 0 changes ranking among eligible candidates only. It cannot make the first two candidates eligible. If no candidate remains, show the specific blocker and offer to add/review a meal or intentionally edit settings; do not silently loosen the rules.

### 20.7 FIX-07 — Revisions, retries, and actual intake

1. Schedule one serving of FIX-02 revision 1 and record it eaten. Snapshot values are 41.5 g protein and 580 kcal with their source/calculation versions.
2. Edit the recipe's yield from two to four servings, leaving ingredient totals unchanged, and save revision 2. Revision 2 is 20.75 g protein and 290 kcal per declared serving.
3. New uses can select revision 2. The historical consumption remains pinned to revision 1 and its original values.
4. The user may explicitly correct the historical record if its original serving interpretation was wrong. That creates a correction linked to the original event, with an impact preview; it is not a side effect of editing the recipe.
5. Send the original record-intake request again with the same idempotency key and payload. There is still one logical consumed portion.
6. Send a different quantity with that same key. Return a key/payload conflict instead of silently modifying the first event.
7. Attempt a plan update with expected revision 4 after another session saved revision 5. Keep revision 5 and return a conflict; preserve the stale client's proposed changes for comparison.

### 20.8 FIX-08 — Legacy prototype recipe migration

The prototype accepted a single recipe object, an array of such objects, an object containing `recipes`, and a workspace backup with `schemaVersion: 1`. Support these through a specifically named legacy adapter, not by guessing that every version-1 JSON file belongs to Daily.

This is a complete minimal legacy recipe object:

```json
{
  "id": "meal_legacy_example",
  "name": "My usual breakfast",
  "subtitle": "",
  "calories": null,
  "protein": null,
  "cost": null,
  "minutes": null,
  "flavor": "Your recipe",
  "servings": 1,
  "items": [],
  "steps": [],
  "groups": [],
  "allergens": [],
  "appliances": [],
  "reviewed": false,
  "source": "",
  "notes": "Saved without details in the earlier prototype.",
  "sample": false
}
```

A legacy workspace contains `schemaVersion`, `profile`, `recipes`, `plan`, `completed`, and `checked`. The profile fields are `diet`, `customName`, `excludedGroups`, `allergens`, `avoid`, `appliances`, `priorities` (`cost`, `effort`, `variety`), and `setupComplete`. Its `plan` is exactly seven recipe IDs or nulls. `completed` holds day indexes 0–6; `checked` holds grocery item keys. It has no reliable actual dates, actual consumption quantities, or purchased-stock ledger.

Legacy ingredient rows contain `id`, `name`, nullable numeric `quantity`, string `unit`, and optional `stockKey`. Step rows contain `id`, `title`, `detail`, nullable `minutes`, ingredient-ID `inputs`, and earlier-step-ID `after`. Preserve source text and links; convert finite numeric values to canonical decimal text without inventing additional precision.

Migration rules:

- Import legacy content as a draft/proposed reviewed record with provenance `legacy_import`; implement that label as source metadata, not an undeclared `Provenance.origin` enum value.
- Preserve `reviewed` as a legacy assertion, not a blanket modern allergy clearance. Empty `allergens` and `groups` do not prove absence.
- Preserve supplied yield but flag the prototype's common default of one for review when serving basis is ambiguous. Do not silently discard an explicit quantity the user entered.
- The legacy interface treated calories, protein, and cost as per-serving card values. Import non-null values as per-serving assertions with that legacy basis and unknown original evidence/currency where not established. Ask for currency if needed before combining costs.
- Import `minutes` as a legacy effort estimate whose active-versus-elapsed meaning is unresolved. Do not populate both fields as if independently measured.
- Ingredient `inputs` become unquantified step allocations. They link the recipe map but do not multiply ingredient totals.
- Convert `after` edges to validated dependencies. Missing/cyclic references remain import errors or explicitly repaired draft warnings; never silently omit them from a supposedly intact recipe.
- The legacy `sample` flag, when set to true, stays illustrative. Importing does not turn demo numbers into verified real data.
- For a legacy workspace, ask the user to assign a start date or import the seven choices as a reusable week template. Preserve completion marks as legacy UI history, not authenticated consumption events. Preserve checked shopping keys as legacy checks, not purchases.
- Map diet/appliance identifiers through an explicit lookup. Supported legacy diets are vegan, vegetarian, pescatarian, omnivore, flexitarian, and custom; food groups are Meat, Fish, Milk, Eggs, Honey. Unknown future identifiers produce reviewable warnings rather than broad matches.

This adapter prevents the new architecture from stranding the user's old data while acknowledging the old format's limitations.

### 20.9 FIX-09 — Native portable envelope

Recommended new native formats are `daily.recipes` and `daily.workspace`, starting at `schemaVersion: 2` to distinguish them from the prototype's unnamespaced version 1. The format string is authoritative; a version number alone is not sufficient identification.

Each record has `kind`, stable `id`, positive integer `revision`, and `data`. Its portable key is `(kind, id, revision)`. Immutable content references must resolve to the appropriate kind and revision within the export or to an explicitly declared unresolved external reference. For a recipe collection, include the transitive food/product/component/source closure needed to interpret it, or mark an intentionally unresolved ingredient/source as such.

The following is a complete minimal recipe-collection envelope for FIX-01. Because it has no source attachments, an empty attachment list omits nothing. The `recipe` identity record's revision tracks its mutable metadata; its `currentRevision` points to the separate immutable `recipe_revision` content number.

```json
{
  "format": "daily.recipes",
  "schemaVersion": 2,
  "exportId": "export_example_001",
  "createdAt": "2026-09-18T12:00:00Z",
  "scope": {
    "kind": "recipe_collection",
    "recipeIds": ["recipe_draft_example"]
  },
  "manifest": {
    "recordKinds": ["recipe", "recipe_revision"],
    "recordCount": 2,
    "attachmentsIncluded": true,
    "omissions": []
  },
  "records": [
    {
      "kind": "recipe",
      "id": "recipe_draft_example",
      "revision": 1,
      "data": {
        "currentRevision": 1,
        "status": "draft",
        "favorite": false
      }
    },
    {
      "kind": "recipe_revision",
      "id": "recipe_draft_example",
      "revision": 1,
      "data": {
        "recipeId": "recipe_draft_example",
        "revision": 1,
        "name": "My usual breakfast",
        "description": "",
        "yield": null,
        "ingredients": [],
        "methods": [],
        "nutritionAuthority": "ingredients",
        "tags": [],
        "sourceIds": [],
        "createdAt": "2026-09-18T11:59:00Z"
      }
    }
  ],
  "attachments": []
}
```

For `daily.workspace`, `scope.kind` is `workspace_backup`; include domain profile/configuration, historical revisions, plans/occurrences, targets, supplement schedules, batches, intake/corrections, stock, prices, purchases, shopping overrides, feedback, and resumable domain drafts supported by the release. Do not export operational provider secrets, authentication state, billing entitlements, or transient job credentials. Include relevant proposal/source content needed to continue a saved draft, with private references handled explicitly.

Define the complete record-kind union and runtime schemas in code using DOM-06. An R0 implementation need not manufacture R1 records, but its manifest must accurately declare supported and included kinds. An unknown required record kind/version blocks complete restore; it cannot be ignored with a “success” label. Optional future fields need an explicit compatibility policy.

Attachment entries identify stable attachment ID, media type, byte length, content hash, and either included content/bundle location or explicit omission reason. For small JSON-only exports, inline base64 may be supported within limits; larger complete backups can use a bundle with this manifest and attachments. Label the format clearly. Do not create an export that silently depends on expiring signed URLs.

Round-trip test: export, import into an empty authorized workspace, re-export, canonicalize domain content, and compare semantic equality after permitted ownership/ID remapping. Ignore export timestamps, export IDs, object-key ordering, and regenerated operation metadata; preserve all user facts, quantities, unknown states, relationships, and historical revisions.

### 20.10 FIX-10 — Objective calculation and explanations

For a simple isolated slot, suppose two eligible candidates have normalized losses:

| Candidate | Cost loss | Effort loss | Repetition loss |
| --- | ---: | ---: | ---: |
| Familiar bowl A | 0.2 | 0.5 | 0.9 |
| New salad B | 0.5 | 0.2 | 0.1 |

With Balanced weights 55, 70, 35 and no other penalties in this fixture:

```text
A = (55×0.2 + 70×0.5 + 35×0.9) / 160 = 0.484375
B = (55×0.5 + 70×0.2 + 35×0.1) / 160 = 0.28125
```

Lower loss wins, so B ranks first. An explanation can say it offers less repetition and lower effort despite higher cost. It must not invent a dollar saving. With Save more weights 100, 40, 20, A scores 0.3625 and B scores 0.375, so A ranks first. Keep normalizers fixed while comparing these candidates; rescaling each candidate separately destroys comparability.

This fixture is a score test, not proof that these arbitrary weights model human preferences. In the full planner, batch effects, shared packages, hard constraints, unknowns, and change penalties can alter the plan-level outcome.

### 20.11 FIX-11 — Starter content and empty-state coverage

Provide a deliberately small, useful starter collection with clear illustrative labels. It should demonstrate the interaction range, not pretend to be a verified comprehensive nutrition database. Recommended examples include:

| Example | UX purpose |
| --- | --- |
| Tofu crunch bowl | Parallel preparation, ingredient joins, high-level protein display. |
| Bean-and-rice bowl | Pantry overlap and optional flavor variation. |
| Lentil pasta | Stove method and component reuse. |
| Curry bowl | Batch preparation and future leftover occurrence. |
| Noodle bowl | Similar ingredients with a different flavor/format. |
| Chickpea wrap | Ready-to-eat assembly and portable meal. |
| Dumpling bowl | Packaged ingredient identity and package instructions. |
| Lentil salad | No heating appliance method. |
| White bean salad | Minimal effort and a short method. |
| Egg toast | Demonstrates non-vegan diet filtering. |
| Tuna and bean bowl | Demonstrates fish-related rules and another diet. |
| Chicken and rice | Demonstrates general product breadth and meat exclusion. |

Recipes beyond the fully specified synthetic fixture require actual authored/reviewed quantities, instructions, sources, and data quality labels before they become reliable recommendations. Names alone do not establish nutrition or suitability. Seed import should be optional for real accounts, repeat-safe, and clearly separate from the user's own content. The starter collection must not disappear merely because the user chooses a diet; incompatible meals can remain visible in the collection with a reason while being excluded from planning.

## 21. Acceptance criteria and verification strategy

### 21.1 TEST-01 — What constitutes evidence

Verify the risky behavior at the lowest useful layer, then exercise complete user journeys through the real UI and persistence boundary. Prefer exact tests for quantities, constraints, revisions, and migrations; integration tests for transactions/auth; and a focused browser suite for the main flows. Do not write dozens of tests that merely assert that a label equals the implementation's own constant.

Use synthetic fixtures with explicit sources and unknowns. Do not use live provider calls as the only way to test math or planning. Contract-test provider adapters against recorded permitted responses and separately smoke-test configured integrations. Test that the app works with providers disabled.

Acceptance is release-specific. R0 can pass its gates while R1 work remains, but must be described as the UX foundation rather than the complete personally useful nutrition-planning product. R1 is the default target for a finished initial implementation under this handoff. R2 and R3 are later capabilities unless explicitly commissioned.

### 21.2 TEST-02 — Core acceptance matrix

| Test ID | Release | Requirement area | Concrete pass condition |
| --- | --- | --- | --- |
| ACT-001 | R0 | UX-ONB-01 | Fresh real account starts with intentional setup/empty state, not a silently adopted demo profile. |
| ACT-002 | R0 | UX-ONB-01 | Back/Next, refresh, and resume preserve an onboarding draft; cancel does not mutate active settings. |
| ACT-003 | R0 | UX-ONB-02 | Switching between diet presets produces the documented rules and preserves explicitly separate restrictions. |
| ACT-004 | R0 | UX-ONB-02 | Missing allergen evidence yields Needs information under an active exclusion; empty tags do not pass. |
| ACT-005 | R0 | UX-ONB-03 | Zero selected appliances is valid; compatible ready-to-eat assembly meals can be offered. |
| ACT-006 | R0 | UX-ONB-04, UX-PRI-01 | Initial answers set documented weights; reopening does not quantize saved custom slider values. |
| ACT-007 | R0 | UX-ONB-05 | Applying a new restriction flags affected future locks; historical intake is unchanged. |
| ACT-008 | R0 | UX-MEA-02 | FIX-01 saves and reloads a name-only draft with unknown values preserved. |
| ACT-009 | R0 | UX-MEA-02 | Pasted decimals/fractions remain correct or unresolved; original text survives parse failure. |
| ACT-010 | R0 | UX-MEA-03 | Invalid field errors preserve other edits; blank numeric fields never become zero. |
| ACT-011 | R0 | UX-MEA-04 | Draft/readiness and eligibility are separate; a ready meal can still be excluded. |
| ACT-012 | R0 | UX-MEA-05 | Editing creates a new revision; archived recipes remain resolvable in past plans. |
| ACT-013 | R0 | UX-REC-02/03 | FIX-05 renders both branches and their join with ingredient quantities linked correctly. |
| ACT-014 | R0 | UX-REC-03 | Cycle, dangling reference, duplicate output, and excessive split allocation fail with specific errors. |
| ACT-015 | R0 | UX-REC-04/05 | Map, read, and cook views use the same revision; navigating Next does not mark food eaten. |
| ACT-016 | R0 | UX-REC-05 | A started timer survives rerender/resume using an absolute deadline; no invented step duration appears. |
| ACT-017 | R0 | UX-REC-06 | Scaling changes displayed quantities, preserves recipe content, and does not multiply cooking time blindly. |
| ACT-018 | R0 | UX-SWP-01 | Swap preview includes actual scoped changes; applying affects only selected future occurrences. |
| ACT-019 | R0 | UX-WEK-02 | Replan preserves valid locks, previews changes, and rejects stale input revisions. |
| ACT-020 | R0 | PLN-02 | FIX-06 exclusions cannot be overcome by any preference weight. |
| ACT-021 | R0 | PLN-06 | No-candidate outcome explains blockers and offers repair without silently relaxing required rules. |
| ACT-022 | R0 | UX-TOD-02 | Empty, loading, partial, and error states do not display sample numbers as real user facts. |
| ACT-023 | R0 | UX-GRO-01 | Equivalent units combine only through valid mappings; unresolved quantities remain visibly unresolved. |
| ACT-024 | R0 | UX-GRO-02 | FIX-03 package count and checkout totals are exact under the stated stock assumptions. |
| ACT-025 | R0 | UX-GRO-04 | Checking a shopping line does not alter purchased inventory or recorded spend. |
| ACT-026 | R0 | UX-DAT-02 | Import staging/cancel changes nothing; applying is atomic and retry-safe. |
| ACT-027 | R0 | UX-DAT-02 | Exact canonical duplicates skip; same-ID different-content imports use the chosen conflict policy with references remapped. |
| ACT-028 | R0 | UX-DAT-03 | Restore preserves account authority, creates a recoverable checkpoint, and rejects malformed references before replacement. |
| ACT-029 | R0 | UX-DAT-05 | FIX-09 round-trip preserves semantic domain content, including null/unknown values. |
| ACT-030 | R0 | FIX-08 | Legacy fixture imports with explicit evidence limitations; seven index-based choices do not become invented dated intake. |
| ACT-031 | R0 | UX-REC-07 | Recipe PDF downloads and opens; long text/Unicode/map branches remain legible without clipped content. |
| ACT-032 | R0 | UX-DAT-04 | Weekly PDF includes its selected scope and open slots; ingredient checklist is printable on A4 and Letter. |
| ACT-033 | R0 | UX-DAT-04 | CSV handles commas, quotes, line breaks, non-ASCII text, and formula-like cells under its documented policy. |
| ACT-034 | R0 | SYS-01 | A saved edit persists after refresh and a new session; failed saves visibly retain the draft. |
| ACT-035 | R0 | SYS-02 | Two concurrent writes cannot silently overwrite each other; stale clients receive a conflict. |
| ACT-036 | R0 | SYS-03 | Cross-workspace reads/writes/downloads, including nested foreign references, are denied. |
| ACT-037 | R0 | UX-VIS-02/03 | Main flows work at narrow mobile and desktop sizes with keyboard navigation and visible focus. |
| ACT-038 | R0 | UX-VIS-03 | Dialog focus is contained/restored; labeled controls and non-color status cues are present. |
| ACT-039 | R1 | NUT-01/02 | FIX-02 computes exact totals; changing display units leaves underlying quantities unchanged. |
| ACT-040 | R1 | NUT-03 | FIX-04 distinguishes unknown, zero, partial lower-bound pass, and unresolved upper bounds. |
| ACT-041 | R1 | NUT-04 | Full-day versus dinner-only assessment is explicit; fixed past plus future totals do not double-count eaten occurrences. |
| ACT-042 | R1 | NUT-05 | Paused/selected-day supplement schedules contribute only on applicable local dates at confirmed quantities. |
| ACT-043 | R1 | NUT-05 | Editing a supplement product/schedule does not silently rewrite prior intake or suggest a new dose. |
| ACT-044 | R1 | UX-MEA-06 | Component expansion is acyclic, counted once, and produces a concrete pinned meal composition. |
| ACT-045 | R1 | PLN-03/04 | Same input snapshot, seed, and algorithm version produce a reproducible result; FIX-10 scores match. |
| ACT-046 | R1 | PLN-05 | Planned portions respect bounds, batch yield, viable method, and applicable resource constraints. |
| ACT-047 | R1 | PLN-06 | Search timeout is distinguished from proven infeasibility; a partial result does not claim all targets met. |
| ACT-048 | R1 | INV-01 | Replanning excludes its own replaced reservations and does not progressively subtract stock twice. |
| ACT-049 | R1 | INV-02 | FIX-03 preparation consumes ingredients once; later portions consume the batch only. |
| ACT-050 | R1 | INV-02 | Intake undo/correction restores stock once; dependent batch changes use a valid correction workflow. |
| ACT-051 | R1 | INV-03 | “Have some” remains qualitative and cannot create a fabricated precise checkout reduction. |
| ACT-052 | R1 | CST-01/03 | Allocated cost, checkout estimate, and actual spend remain distinct; savings need a comparable baseline. |
| ACT-053 | R1 | FIX-07 | Historical snapshots survive recipe edits; duplicate requests and changed-payload key reuse behave as specified. |
| ACT-054 | R1 | SYS-05 | Populated database migration and backup restore preserve historical references and unknowns. |
| ACT-055 | R1 | SYS-06 | Timezone/DST fixtures preserve intended local dates, one scheduled period, and historical instants. |
| ACT-056 | R2 | AI-02 | Malformed or unsupported model fields stay proposals/errors; newer user edits are not overwritten. |
| ACT-057 | R2 | AI-03 | Malicious page instructions cannot mutate settings or trigger actions; prohibited fetch destinations are blocked. |
| ACT-058 | R2 | DATA-01 | Raw/cooked and branded/generic mismatches require resolution rather than an automatic false match. |
| ACT-059 | R2 | DATA-02 | Duplicate receipt ingestion cannot duplicate purchases; old receipts do not imply current stock. |
| ACT-060 | R2 | JOB-01 | Retry/cancel/provider outage/budget limits leave manual functionality usable and prevent duplicate application. |
| ACT-061 | R2 | JOB-02 | Scheduled runs create one reviewable draft per intended period and never silently replace accepted plans. |
| ACT-062 | R3 | BIZ-01/02 | Entitlements are server-enforced; own-data access/export remains available under the stated product policy. |
| ACT-063 | R3 | BIZ-02 | Duplicate/out-of-order billing events cannot create inconsistent entitlements or alter food-domain backups. |

### 21.3 TEST-03 — Property and invariant tests

Use property-based tests where they catch classes of numerical/reference mistakes rather than mirroring code. Useful properties include:

- Scaling a complete linear recipe by a positive factor scales its ingredient contributions by that factor, subject to explicitly modeled discrete constraints.
- Splitting and recombining a quantified ingredient across valid steps does not change recipe nutrient totals.
- Equivalent supported unit conversions round-trip within the declared precision policy.
- Adding an unknown contributor cannot produce a complete total, except where an explicit authoritative whole-meal assertion is selected instead of ingredient calculation.
- Increasing confirmed stock cannot increase required package count when offers and all other assumptions remain fixed.
- Reordering independent graph steps or JSON object keys does not change semantic calculations or duplicate detection.
- A required-exclusion failure remains a failure under every soft-weight setting.
- Retrying the same committed operation does not change event counts or stock projections.

Define precision and rounding expectations explicitly. Test money for currencies with different minor-unit conventions using synthetic configured currencies; do not assume all amounts display with exactly two decimals. Preserve full decimal precision internally and round at documented display/payment boundaries.

### 21.4 TEST-04 — End-to-end review scripts

**Script A: cold start to a real week.** Create a fresh account, choose vegan, select only microwave, answer priority questions, review the summary, and complete setup. Save a minimal breakfast draft. Add/review one usable meal, generate an available scope, apply it, open its recipe, change a future slot, and reload. Verify that every visible total describes its actual scope and incomplete data.

**Script B: incompatible settings.** Start with a week containing a soy-based locked dinner, then add a soy exclusion. The profile applies, the conflict is visible, the lock does not make the dinner eligible, and a repair preview replaces only explicitly selected future occurrences. Past intake stays unchanged.

**Script C: tired-evening swap.** Activate a temporary low-effort preference without altering diet restrictions. Inspect a compatible low-prep candidate, apply it, verify shopping differences, then undo before dependent actions occur. Return to the base preference after the configured expiry.

**Script D: batch and actual use.** Run FIX-03 through actual purchase, preparation, two consumption events, and an intake correction. Inspect ingredient stock, batch quantity, plan state, and actual nutrient history. None should depend solely on a checkbox's visual appearance.

**Script E: data portability.** Create custom content with Unicode, an unknown quantity, a historical revision, and a source note. Export, stage it in another authorized workspace, inspect conflicts, cancel, then apply. Re-export and compare canonical domain content. Test a corrupt reference and confirm no partial application.

**Script F: document quality.** Print a short recipe and a deliberately long branching recipe, each with long ingredient names and Unicode text. Print a week containing open slots and unknown prices. Open actual generated PDF bytes and inspect A4/Letter pages at normal reading size. Confirm fonts, clipping, page breaks, header repetition, legend, and checkbox usability; an HTTP 200 alone is insufficient.

**Script G: access and failure.** Simulate a save response lost after commit, a stale second tab, a provider timeout, and a foreign-workspace nested reference. Each has a distinct recoverable outcome. No retry creates a duplicate and no error exposes another user's data.

**Script H: low-maintenance assistance, when R2 is included.** Capture a messy recipe, accept only reviewed fields, let a scheduled draft run, and dismiss an unnecessary pantry question. Later runs respect that feedback; none infer consumption, purchase, or a new supplement dose.

### 21.5 TEST-05 — Traceability to confirmed user needs

| Confirmed need | Principal implementation sections | Evidence |
| --- | --- | --- |
| C01: balance nutrition, effort, variety, and cost | 6.6, 7, 12–15, 20.2/20.4/20.10 | Exact totals, scope/unknown handling, targets, and explainable tradeoffs. |
| C02: little preparation or decision work | 6.3/6.4, 7, 9, 14, 16 | Useful defaults, viable methods, focused swaps, bounded automation. |
| C03: vegan personal use and support for other diets | 6.2, 12–14, 19.3 | Configurable presets and rules, mixed starter examples, FIX-06. |
| C04: dietary restrictions and allergies | 6.2, 12–14 | Independent required exclusions, evidence states, FIX-06. |
| C05: appliance picker | 6.3, 9.6, 14.5 | Accessible playful selection and method-level feasibility. |
| C06: initial priority questions | 6.4, 7.6 | Answer-to-weight mapping and custom persistence. |
| C07: flexible meal add/edit | 8, 16.2, 20.1 | Name-only draft through reviewed complete recipe. |
| C08: import/export and printable PDFs | 9.7, 11, 20.8/20.9 | Actual downloads, visual review, semantic round-trip. |
| C09: intuitive recipe representations | 9, 20.5 | Shared map/read/cook content and dependency validation. |
| C10: preserve the whole idea through handoff | 2–3, 23, 25 | Confirmed needs, rationale, deferred ideas, and maintained decisions. |
| C11: one self-contained handoff | This entire document | Agent can begin without the prototype or earlier conversations. |

## 22. Delivery sequence and definition of done

### 22.1 BUILD-01 — First session for the implementing agent

Read the complete document once, then inspect the repository. Record the selected initial release target, actual stack, auth/storage conventions, provider availability, and any conflict with this specification. Recommended target is R1, delivered through R0 milestones first.

Create a requirement-to-implementation checklist using the IDs here. This can live in normal task tracking or the implementation-status section of this document; the user does not need to supply another specification. Distinguish pending work from already implemented prototype behavior. This document does not mean that the local repository has already passed any acceptance test.

Start with one vertical slice: authenticated workspace → name-only meal draft → durable save → collection display → reopen/edit → native export. This proves ownership, persistence, null handling, and domain boundaries before a large screen build hides foundational problems.

Do not begin by asking the user to re-enter every meal, research every nutrient, or choose a model provider. The app can be built and tested with clearly labeled fixtures. Ask only when a missing decision blocks a consequential choice that has no safe reversible default.

### 22.2 BUILD-02 — Milestones and exit gates

| Milestone | Deliverable | Exit gate |
| --- | --- | --- |
| M0: repository fit | Architecture mapping, runtime schemas, auth scope, migrations, fixture harness. | Existing conventions respected; basic ownership and schema checks work. |
| M1: durable collection | Shell, minimal capture, editor, recipe identity/revisions, collection, local source notes. | FIX-01 persists; errors/conflicts retain edits; no fabricated numeric defaults. |
| M2: setup and recipe UX | Four-step onboarding, appliance picker, presets, rules, map/read/cook, scaling. | FIX-05/06 pass; required exclusions and graph constraints enforced. |
| M3: usable R0 planning | Today/Week/Priorities, manual/suggested slots, swaps, locks, shopping preview, feedback. | Complete R0 journeys with explicit limited scope; no hidden resets or silent replans. |
| M4: R0 portability/polish | Legacy/native import, export, printable recipe/week PDFs, CSV, responsive/accessibility review. | R0 matrix passes, actual documents inspected, backup round-trip verified. |
| M5: R1 nutrition | Food/product revisions, nutrient registry, units, targets, full-day routine, supplement schedules. | FIX-02/04/07 pass; complete versus partial coverage is accurate. |
| M6: R1 practical planning | Components/families, batch inventory, price book, package costing, bounded planner, feedback integration. | FIX-03/10 and R1 matrix pass; initial app is useful without AI. |
| M7: R2 assistance | Configured extraction/providers, research jobs, recurring drafts, bounded budgets. | R2 matrix passes with provider failure and adversarial-input cases. |
| M8: R3 commercial rollout | Chosen entitlements/billing, operational policies, any commissioned sharing. | Deployment-specific commercial requirements and R3 gates met. |

Some UI work can overlap within a milestone, but protect the dependency order: robust quantities before nutrition claims, source/rule evidence before automatic eligibility, inventory semantics before leftovers, and transaction/idempotency behavior before automation.

### 22.3 BUILD-03 — Definition of done for R1

R1 is complete when a fresh user can configure their diet/kitchen/priorities, capture existing meals with little information, refine them as needed, create a full-day plan within the limits of available data, make easy swaps, understand recipe preparation, obtain an honest grocery estimate, record actual use, and move/print their data. Relevant records survive refresh/new sessions and no required constraint is silently traded away.

All R0/R1 acceptance conditions must be implemented and verified or explicitly listed as a scoped deviation accepted by the product owner. “The button exists” is not completion. A mock provider, hardcoded total, in-memory save, empty import handler, or PDF button that only opens a blank print page must be labeled unfinished.

Document the actual test commands/results, known limitations, configured versus unavailable integrations, migration/backup process, and run instructions using repository conventions. Normal source code, tests, and generated schema files are expected implementation outputs; the one-document constraint concerns the handoff input, not a ban on maintainable code organization.

Make the result reviewable with representative seeded and empty-account states. Keep illustrative facts visibly illustrative. Do not report clinical adequacy, measured cost savings, or adherence improvements without appropriate real evidence.

### 22.4 BUILD-04 — Scope changes and blockers

Use the recommended defaults for routine reversible choices and record material deviations. If repository constraints require a different transport, framework, or database shape, preserve the specified semantics and explain the choice. Escalate only choices that materially change user behavior, cost, access, or irreversible data handling.

If credentials are unavailable, implement the adapter boundary and manual fallback; do not block unrelated authorized work. If an external feature cannot be completed, leave a clear capability state and a specific next requirement. Do not create fake successful data to conceal a missing integration.

Purchasing, sending external notifications, publishing, billing activation, or broad third-party account access requires authorization appropriate to that action. This implementation handoff authorizes building the app, not inventing those future authorizations.

## 23. Open decisions, assumptions, and deferred possibilities

### 23.1 DEC-01 — Decisions the agent can use now

| Decision | Current recommended answer | Why it is safe to proceed |
| --- | --- | --- |
| Working name | Daily; keep configurable. | Preserves prototype continuity without asserting final branding. |
| Initial completion target | R1 through staged R0 delivery. | Covers the user's actual daily nutrition/planning problem. |
| Personal onboarding defaults | Ask diet; blank targets; explicit locale/currency; no imported private purchases. | Avoids fabricating personal facts. |
| First AI dependency | None required for core workflows. | Manual capture plus deterministic planning already provides value. |
| Planning scope | Actual dated seven-day plan with full-day slots in R1. | Fits weekly shopping while retaining individual swaps. |
| Required rules | Exclusions and explicitly required constraints cannot be optimized away. | Makes behavior understandable and consistent. |
| Data quality | Unknown remains unknown; provenance is inspectable. | Prevents false precision and makes improvement incremental. |
| Historical facts | Immutable content references plus explicit corrections. | Recipe edits should not rewrite the past. |
| Portability | Native JSON/bundle, recipe/week PDF, shopping CSV. | Separates editable data from convenient printed output. |
| Monetization | Architectural readiness now, actual billing later. | Preserves product options without adding setup friction. |

### 23.2 DEC-02 — Personalization questions to collect in the product

These questions improve actual recommendations but do not block implementation:

- What are the user's real recurring meals, products, serving quantities, and supplement schedules?
- Which appliances and storage options are available? Which are actually pleasant to use?
- What are the user's self-selected nutrient targets and their sources? Which are required versus informational?
- Which meals should remain stable, and how many new meals per week feel useful rather than tiring?
- Which textures, flavors, ingredients, and preparation tasks are liked or avoided?
- What does “low effort” mean: short active time, no chopping, few dishes, no stove, or minimal decisions?
- Which stores, currencies, memberships, delivery preferences, and package sizes apply?
- What weekly cost goal and comparison baseline should be used, if any?
- How comfortable is the user with leftovers, batch preparation, pantry estimates, and periodic stock questions?

Collect these progressively. The product should become more useful after a few real actions instead of requiring a large questionnaire before the first plan.

### 23.3 DEC-03 — Decisions needing repository or deployment evidence

Actual framework/database/auth stack, available Vrooli services, model/provider routing, deployment target, nutrient source access, retailer integration rights, attachment storage, retention policy, and billing provider must be established from the real environment. This document intentionally does not assert API keys, subscriptions, commercial licenses, or services the user has not supplied.

Nutrition reference-target sourcing is a separate decision from calculator correctness. If future setup offers automatically suggested targets, define the applicable population inputs, source/version, excluded contexts, explanation, and user confirmation before enabling it. Do not ship an undocumented one-size-fits-all target calculator.

### 23.4 DEC-04 — Deferred idea inventory

Preserve the following ideas for future evaluation without making them dependencies of R1:

| Idea | Potential value | Evidence or capability needed before building |
| --- | --- | --- |
| Adherence/temptation model | Suggest realistic variety and convenient backup foods. | Voluntary longitudinal feedback, calibrated evaluation, no judgmental assumptions. |
| Probabilistic inventory | Reduce exact stock entry. | Reliable event history and understandable uncertainty; no fabricated confidence percentages. |
| Research agents for recurring staples | Find meaningful price/product improvements. | Authorized sources, package identity, freshness, budget/value limits. |
| Receipt/order/email ingestion | Reduce manual price and purchase capture. | Opt-in source access, deduplication, old-stock handling, privacy controls. |
| Meal-photo capture | Faster meal/label entry. | Evidence-based extraction, user correction, honest portion uncertainty. |
| Expected-waste/effective-cost optimization | Avoid false savings from bulk purchases or extra trips. | User-specific usage and clearly separated assumptions. |
| Automated grocery ordering | Reduce shopping effort. | Explicit purchase authorization, reviewable cart, retailer support, substitutions/payment/error handling. |
| Household collaboration | Shared shopping with individual needs. | Roles, profiles, ownership, conflict handling, person-specific targets. |
| Public recipe marketplace/catalog | Broader variety and distribution. | Content rights, moderation, provenance, commercial validation. |
| Generic reusable optimizer | Reuse proven constraint logic elsewhere. | Evidence that extraction simplifies working products instead of delaying them. |
| Longer-horizon adaptive planning | Learn how much repetition and novelty works. | Measurable outcomes, reversible adaptation, stable preference controls. |

### 23.5 DEC-05 — Main product risks and response

**Too much setup:** name-only drafts, progressive questions, existing-routine capture, and a useful manual path. Measure time to a first usable plan rather than the number of filled fields.

**False nutrition precision:** visible scope, source/basis, unknown propagation, target provenance, and no automatic dose changes. Favor a useful incomplete answer over an invented complete one.

**Variety that increases work:** vary flavor or assembly around reusable staples/components; retain familiar meals; show the preparation impact of novelty. Do not maximize a count of unique recipe names as a proxy for enjoyment.

**Cheap food that creates waste or friction:** show package checkout and portion cost separately; consider storage and leftovers; keep extra-store effort visible. A theoretical saving is not automatically worthwhile.

**An attractive app with fragile data:** prove vertical slices, revisions, idempotency, and recovery before broad automation. Keep the UI's save and uncertainty states honest.

**Overbuilding a platform:** complete the meal-planning loop before public catalogs, subscriptions, generic agent systems, or forecasting. Maintain extension points without creating empty frameworks for hypothetical features.

## 24. Grounding and external references

### 24.1 What came from the user and prototype

The core goals, vegan personal use, support for other diets, low preparation burden, variety/cost tradeoffs, onboarding/appliance/priorities requests, flexible meal editing, import/export/print requirements, recipe-view inspiration, and single-document handoff requirement come from the user's conversation and current product brief.

The visual direction, four primary navigation areas, diet/appliance choices, preset weights, minimal draft workflow, recipe map/read/cook concept, legacy schema behavior, and durable-account expectation were consolidated from the current prototype and brief. This specification explicitly replaces prototype shortcuts identified in section 3.4. It does not claim the prototype already implements the R1 planning architecture.

The detailed domain contracts, conservative evidence policies, release gates, fixture arithmetic, migration requirements, and recommended architecture are engineering/product recommendations in this document. They should not be misrepresented as individually approved user decisions. They are concrete defaults the local agent can implement and the user can revise.

### 24.2 Reference list

The following pages were consulted on September 18, 2026. They provide context for specific design choices; the agent does not need to read them to understand the requirements here. Recheck current provider documentation when implementing an integration.

| Reference | Role in this specification |
| --- | --- |
| [Cooking for Engineers recipe infographics interview — Cool Infographics](https://coolinfographics.com/blog/2010/4/26/cooking-for-engineersrecipe-infographics-and-interview.html) | User-requested inspiration for showing ingredient/action relationships compactly. The interactive graph and data contract here are this product's design. |
| [USDA FoodData Central API guide](https://fdc.nal.usda.gov/api-guide/) | Primary reference for the suggested nutrition-data adapter. |
| [USDA FoodData Central data documentation](https://fdc.nal.usda.gov/data-documentation/) | Source-type distinctions and provenance context. |
| [NIH Office of Dietary Supplements: Nutrient Recommendations and Databases](https://ods.od.nih.gov/HealthInformation/nutrientrecommendations.aspx) | Distinct reference-intake concepts and target provenance; not a source of personal targets in this document. |
| [W3C WCAG 2.2](https://www.w3.org/TR/WCAG22/) | Accessibility target for the interface. |
| [OWASP CSV Injection](https://community.owasp.org/attacks/CSV_Injection) | Export threat context and limitations of spreadsheet interpretation. |
| [OWASP SSRF Prevention Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html) | Threat context for fetching user-supplied URLs. |

No source above verifies this document's illustrative food values, sample prices, user-specific goals, or business hypotheses. Those remain explicitly synthetic, unknown, or recommended as labeled.

## 25. Maintaining this as the single handoff document

### 25.1 DOC-01 — Canonical specification and continuity

When this file is placed in the implementation repository, make that copy the canonical product/implementation specification for this effort. Keep it under version control. Earlier brainstorms and prototypes remain historical context; they should not independently redefine current requirements.

Update this document when an important product decision changes. Record the date, decision, reason, affected requirement IDs, and implementation impact. Preserve rejected/deferred ideas in a short decision log when they explain a later choice. Avoid silently deleting a requirement because it was inconvenient to implement.

For future AI sessions, provide this file plus the actual repository state. A new agent should read the current specification, current implementation status, and test results before continuing. Do not depend on another conversation's memory to know whether a feature is complete or an assumption was accepted.

Keep source code and executable schemas authoritative for exact current transport details after implementation, while this document remains authoritative for intended product behavior. When they diverge, fix or explicitly resolve the discrepancy; do not let two incompatible definitions persist unnoticed.

### 25.2 DOC-02 — Initial decision and implementation log

| Date | ID | Status | Decision / note |
| --- | --- | --- | --- |
| 2026-09-18 | D-001 | Confirmed user request | Produce one extremely detailed self-contained document for a local implementation agent. |
| 2026-09-18 | D-002 | Confirmed product direction | Preserve low-effort nutrition planning while supporting variety, cost awareness, multiple diets, flexible entry, portability, and richer recipe views. |
| 2026-09-18 | D-003 | Recommended default | Deliver R1 through R0 milestones; defer provider automation and billing unless commissioned. |
| 2026-09-18 | D-004 | Recommended default | Use deterministic domain calculations and validated AI proposals, with explicit unknowns and immutable historical references. |
| 2026-09-18 | D-005 | Open pending repository inspection | Select actual implementation stack and integration details using the target repository's conventions. |
| 2026-09-18 | D-006 | Open pending in-app personalization | Collect real meals, products, targets, supplements, prices, and preferences progressively; do not treat examples as personal data. |

**Implementation status at handoff:** this is the implementation specification, not a claim that the new local implementation or acceptance suite exists. The earlier prototype informed the UX. M0–M8 and applicable ACT tests must be assessed against the actual target repository.

### 25.3 DOC-03 — The instruction to begin

> Build Daily according to this specification, using the target repository's established conventions. Read the full document, inspect the repository, select the R1 target through staged R0 milestones, and begin with the durable minimal-meal vertical slice. Preserve required dietary rules, explicit unknowns, immutable history, honest cost/nutrition scope, and low-friction UX. Use the embedded fixtures and acceptance criteria to verify behavior. Make routine reversible implementation choices using the recommended defaults; record material deviations and surface only genuine blockers. Do not require access to the original prototype or previous conversations to proceed.
