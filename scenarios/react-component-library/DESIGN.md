---
id: react-component-library
version: 0.1.0
name: React Component Library Console
description: Design intent for the library workbench; runtime values are owned by BaseStyles.
---

# Vrooli Operational Console Design

This document records **design intent**, not current behavior or another authored token palette. Runtime token values come from BaseStyles and its generated consumers; app-local geometry lives in `ui/src/app-tokens.css`. See the [token contract](docs/reference/token-contract.md) and [checked behavior register](docs/internal/TESTING.md#behavior-claim-register). Recommendations below require focused checks and visual review before being claimed as delivered.

## Registry Projection

Component previews and catalogue surfaces must reflect the component registry's typed projection, not reinterpret source files independently. `component.json` owns stable component facts such as identity, display metadata, slot, tags, versions, and design-style affinities. Version source headers are hints for source-local metadata; structural hints such as `@version`, `@status`, `@deps`, and latest-version `@category` are validated by the indexer and either promoted into typed registry fields or reported as disagreement findings.

Preview specimens must come from versioned data, not ad hoc UI code. A component version's `story.json` is the source of truth for named preview states; the editor renders those stories as a gallery, the browser gate sweeps every story frame, and experience-component specs bind reusable behavior claims to those same names.

The UI should therefore render registry facts from API read models. It should not prefer a header value over a manifest- or projection-owned value when they disagree.

## Layout

Use full-width application surfaces. Prefer navigation, toolbars, tables, forms, split panes, file/code viewers, graph canvases, command bars, and compact panels over oversized hero sections or decorative cards.

Desktop layouts should maximize screen space. Sidebars and inspector panels sit beside the main content and reduce the main content width when opened. Resizable sidebars and split panes are preferred for work surfaces where users compare lists, details, diffs, logs, previews, or execution output. Preserve enough adjacent content width that opening a panel never makes the workspace unusable.

Mobile layouts are not shrunken desktop layouts. Sidebars become full-width drawers or route-level panels. Main route navigation may become bottom navigation when it replaces desktop sidebar or header navigation. Complex dialogs become full-screen panels or bottom sheets, especially when they contain forms, review steps, filters, file actions, or multi-step decisions. Mobile primary actions should live near the bottom edge, respect safe-area insets, and support optional left-handed or right-handed placement when the action is frequent.

Use cards only for repeated records, focused tools, modals, and intentionally framed objects. Do not wrap whole page sections in floating cards. Keep page-level structure unframed unless the content is a true object or tool.

## Color

Default presentation is light mode with a neutral work surface and optional dark shell chrome. Dark mode is first-class and should preserve the deep slate operational feel of existing Vrooli tools. System mode should follow the platform preference.

Use neutral slate surfaces as the dominant base. Use blue for primary commands and selected navigation, cyan for technical emphasis, green for success or completed work, amber for warnings or pending attention, and red for destructive, failed, or blocked states. Do not rely on color alone for status; pair it with labels, icons, position, or shape.

Avoid one-note palettes dominated by a single hue family. Avoid decorative gradient blobs, bokeh, and atmospheric backgrounds. Gradients may be used sparingly for product-specific hero metrics or specialized visualizations, but not as the default application background.

## Typography

Use Inter or the platform sans stack for application UI. Use a monospace stack for code, diffs, hashes, paths, identifiers, logs, tabular metrics, and command output.

Base body text is 16px for mobile input safety and accessibility. Dense desktop panels may use 14px body text when the interaction benefits from scanning, but controls must remain legible and targets must remain usable. Support user font-size scaling. Letter spacing is zero by default except for rare compact labels where local implementation has a clear reason.

Reserve hero-scale type for true landing or product-identity screens. Scenario applications should usually use compact page titles, section headers, labels, and status text sized to their container.

## Components

Controls should be predictable, stable, and optimized for repeated work. Use icon buttons for familiar tool actions, segmented controls for modes, toggles or checkboxes for binary settings, sliders or inputs for numeric values, menus for option sets, tabs for sibling views, and command bars for high-frequency workflow actions.

Example primitives (illustrative, not a feature checklist — include whatever your scenario actually needs, styled to match these patterns):

- **Shells:** desktop sidebars, collapsible panes, resizable inspectors, bottom mobile navigation, route-level mobile panels.
- **Dialogs:** compact modal on desktop; bottom sheet or full-screen panel on mobile when content is more than a short confirmation.
- **Lists and tables:** dense rows, sticky headers when useful, clear selected state, empty/loading/error states, and bulk action affordances.

For card-oriented collections, use the four-layer `CollectionList` contract
documented in [COLLECTIONS.md](docs/concepts/COLLECTIONS.md). Use `DataTable`
and `ResourceCollection` when the user needs columns, headers, sorting, or
tabular density; do not force card content into a table-shaped API.
- **Status:** semantic chips, health indicators, badges, progress bars, and validation summaries with both text and color.
- **Code and files:** monospace text, stable line height, visible focus, preserved whitespace, horizontal scroll affordances, and clear added/modified/deleted states.
- **Actions:** primary action near the active work surface; destructive actions separated, confirmed, and visually distinct.

Text must not overflow or overlap at mobile or desktop sizes. Fixed-format controls such as boards, tiles, counters, toolbars, nav items, and badges need stable dimensions or responsive constraints so dynamic content cannot shift the surrounding layout.

## Responsiveness

Design mobile first, then expand to tablet and desktop. Mobile should provide complete capability, not a read-only fallback. If a workflow is important on desktop, define the mobile equivalent deliberately.

Breakpoints should be treated as behavior changes, not only width changes. Common transformations:

- Desktop sidebar -> mobile full-width drawer.
- Desktop modal -> mobile bottom sheet or full-screen panel.
- Desktop header/sidebar route navigation -> mobile bottom navigation where appropriate.
- Desktop split panes -> mobile stepwise panels or tabs with preserved context.
- Desktop hover affordance -> mobile visible affordance or long-press-safe alternative.

Use safe-area padding for mobile bottom and top chrome. Primary touch targets should be at least 44px. Avoid hiding essential actions behind hover-only UI.

## Customization

Every generated scenario should be able to support light, dark, and system mode unless the product explicitly documents a different reason. Font size scaling, reduced motion, and RTL layout should be planned from the start. For mobile-heavy workflows with frequent bottom actions, support optional left-handed and right-handed action placement when feasible.

Build a full settings/preferences surface covering everything your scenario actually needs — theme, font scale, locale, accessibility preferences, account or workspace controls, notification toggles, and scenario-specific options. Do not treat any specific subset of preferences mentioned elsewhere in this document as the complete set; those are illustrative. Style and behavior of the settings surface are governed by this design; *which* settings exist is governed by your scenario's users.

Customization should be implemented through tokens and stateful preferences, not one-off component rewrites. Local product themes may extend this design, but they should keep semantic color roles, focus behavior, responsive transformations, and accessibility guarantees intact.

## Workflow Ergonomics

Design from the user's flow, not from component inventory. For each major screen, identify the primary repeated action, the highest-risk action, the most common comparison, and the first thing a new user needs to understand.

Experienced users should be able to move quickly with short pointer travel, predictable keyboard focus, visible shortcuts where appropriate, persisted panel sizes, remembered filters, and stable navigation state. New users should see enough structure, labels, and progressive disclosure to understand what is actionable without reading documentation.

## Feedback & State

Every user-triggered operation needs visible state. Loading, submitting, saving, syncing, refreshing, empty, partial, stale, success, validation-error, request-error, permission-denied, offline, and retry states are part of the design contract, not implementation polish.

Buttons that start asynchronous work should acknowledge the click immediately, show a busy state, prevent duplicate submission when duplicate work would be harmful, and restore a usable state when the operation finishes. Forms should preserve user input on failure, place field-level validation near the affected control, and show a form-level summary when the submit action fails. Lists, tables, panels, and dashboards should have purposeful loading, empty, partial, and error states instead of blank space.

Use inline feedback near the action when the user needs to continue working in context. Use toasts only for transient confirmation or background results. Use alert panels for failures that need reading, retry, or escalation. Error messages should explain what happened, what is still safe, and the next available action without exposing stack traces, secrets, raw tokens, or irrelevant internals.

## Request Lifecycle

For every network call, long-running local task, file operation, generation step, or resource mutation, design the lifecycle deliberately: idle, pending, success, failure, retrying, and disabled/unavailable. Slow operations should show progress, skeletons, spinners, streaming output, or queued status appropriate to the surface. If exact progress is unknown, show an indeterminate but visible pending state with stable layout.

Optimistic updates are allowed only when rollback is clear. If an optimistic change fails, restore the previous state or mark the item as unsynced with a retry action. Background sync should expose freshness, last-updated time, stale data, and reconnection status when the result affects decisions.

## Accessibility

Interactive controls need visible focus states, disabled states, hover/active states where supported, and readable contrast in both light and dark modes. Do not rely on color alone to communicate status. Keep target sizes usable for mouse, keyboard, touch, and remote/TV-like pointer input when relevant.

Respect reduced-motion preferences. Animations should clarify spatial change, such as drawers and sheets entering from their origin, and should never block task completion. Scrollbars and overflow regions must be discoverable on pointer-based devices.

## Do's and Don'ts

### Do

- Start UI work by reading this file and mapping the main user flows.
- Prefer compact operational surfaces over decorative composition.
- Support light, dark, and system mode from the beginning.
- Use responsive behavior changes for sidebars, dialogs, navigation, and split panes.
- Use semantic status colors consistently and pair them with text or icons.
- Preserve user preferences for theme, font scale, panel sizing, filters, and active views when useful.
- Design loading, empty, partial, success, validation-error, request-error, and retry states for every asynchronous workflow.
- Preserve user input and provide a clear next step when a form submission or mutation fails.

### Don't

- Create a marketing-style landing page as the first screen of a scenario application.
- Make mobile a cramped version of desktop.
- Hide important mobile actions behind hover-only affordances.
- Use decorative gradients, orbs, or background effects as the default visual identity.
- Duplicate BaseStyles values in application-owned token files.
- Claim an intended theme is implemented without regenerating and checking its runtime tokens.
- Leave users without visible feedback after they submit, save, generate, refresh, or delete something.
- Use silent failure, blank panels, disabled controls without explanation, or toasts as the only record of a blocking error.
