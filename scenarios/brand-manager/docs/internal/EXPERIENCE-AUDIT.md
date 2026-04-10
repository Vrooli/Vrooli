# Experience Architecture Audit - 2026-03-26

## Scenario Purpose
Brand Manager helps users create, manage, and apply brand identities across Vrooli scenarios so they can maintain consistent visual and voice branding without manual per-scenario configuration.

## Core Personas & Primary Jobs

### 1. Brand Creator (first-time user)
- **Job 1**: Understand what Brand Manager does
- **Job 2**: Create their first brand identity
- **Job 3**: See how it will look applied

### 2. Brand Manager (returning user)
- **Job 1**: Edit or update an existing brand
- **Job 2**: Apply a brand to a scenario
- **Job 3**: Scan scenarios for branding compliance

### 3. Brand Auditor (ops/quality user)
- **Job 1**: Scan scenarios for brand markers (CSS/JSON)
- **Job 2**: Run audit evaluations against standards
- **Job 3**: Review and understand brand standards/rules

## Current vs Ideal Flows

### Brand Creator: "Create my first brand"
- **Current**: Land on Brand Library (empty state) -> Click "Create Brand" -> Fill form -> Save -> Redirected to detail page
- **Ideal**: Same flow works well. The empty state CTA is clear. Minor improvement: could show a brief "what is a brand?" hint for first-timers.
- **Status**: Good. No major friction.

### Brand Manager: "Edit my brand"
- **Current**: Brand Library -> Click brand card -> Brand Detail -> Click Edit -> Form (pre-populated) -> Save
- **Ideal**: Same flow. Edit button is prominent in the detail header. Back button correctly returns to brand detail.
- **Status**: Good.

### Brand Auditor: "Scan a scenario for compliance"
- **Current**: Nav bar -> Scanner -> Type scenario name -> Click Scan -> View results + audit
- **Ideal**: Same flow. Scanner input supports Enter key. Results are well-structured with summary stats.
- **Status**: Good.

### Brand Auditor: "Understand what standards exist"
- **Current**: Nav bar -> Standards -> View rule list
- **Ideal**: Same. Simple reference page with severity badges.
- **Status**: Good.

## Friction Points

### Mechanical (low)
- Scanner requires typing scenario name manually; could benefit from autocomplete (future enhancement)

### Cognitive (low)
- Brand form has many fields which could feel overwhelming; sections help but no progressive disclosure

### Discoverability (addressed in this phase)
- **Fixed**: Nav links now show active state (white text + font-medium) so users always know where they are
- **Fixed**: Scanner and Standards pages registered in selector registry for automation workflows

## Navigation

### Intended Navigation Graph
```
Brand Library (home) <-> Brand Detail <-> Brand Edit
Brand Library -> Brand Create
Scanner (top nav)
Standards (top nav)
```

### Navigation Audit Results
- **Active state**: Added to nav links (Scanner, Standards, Brand Manager title)
- **Back buttons**: Brand Detail -> Library, Brand Form -> Detail (edit) or Library (create), Scanner/Standards -> Library
- **aria-current**: Added to active nav buttons for accessibility
- **Deep links**: Hash-based routing supports direct linking to any page
- **Refresh**: Route state preserved via hash (no server-side routing needed)

## Improvements Implemented (Phase 18)
1. Active navigation state highlighting (text-white + font-medium for current section)
2. aria-current="page" on active nav buttons for accessibility
3. Selector registry completeness (scanner + standards pages fully registered)
4. CSS design tokens wired into Tailwind config for coherence

### Navigation Integrity Audit (Phase 18-iter3)

**Label → Destination truthfulness**: All navigation labels accurately describe their destination.
- "Back to Library" on Scanner/Standards correctly goes to `/brands`
- "Back to Library" on BrandDetail correctly goes to `/brands`
- "Back" on BrandForm goes to `/brands/{id}` (edit) or `/brands` (create) — correct for both modes
- Edit button label matches edit mode destination

**Back/Forward coherence**: All back buttons use explicit `onNavigate("/path")` rather than browser history, which is consistent for a hash-based SPA embedded in an iframe. The top-level pages (Scanner, Standards) always return to Brand Library, which is correct as the canonical parent.

**Deep link support**: Hash-based routing fully supports direct links to any page. Browser refresh preserves route state. However, scanner results are not deep-linkable (scan target is in component state, not URL).

**Edge cases verified**:
- Direct link to `/brands/{id}` works (fetches brand on mount)
- Direct link to `/brands/{id}/edit` works (fetches brand for form pre-population)
- Unknown routes default to brand library (safe fallback)
- Health indicator persists across all pages

## Recommended Future Improvements
1. Scanner scenario name autocomplete (list known scenarios from API)
2. Recent brands section on the library page for quick access
3. Brand comparison view (side-by-side two brands)
4. Progressive disclosure on brand form (collapse optional sections)
5. Breadcrumb trail for multi-level navigation within brand section
6. Deep-linkable scanner results (persist scan target in URL hash)
7. Search query persistence in URL for shareable filtered views
8. Loading skeletons instead of generic "Loading..." text
