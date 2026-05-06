---
id: vrooli-default
version: 0.1.0
name: Vrooli Operational Console
tokens:
  color:
    background: "#f8fafc"
    shell: "#020617"
    surface: "#ffffff"
    surfaceMuted: "#f1f5f9"
    surfaceRaised: "#ffffff"
    foreground: "#0f172a"
    mutedForeground: "#64748b"
    border: "#cbd5e1"
    primary: "#2563eb"
    primaryForeground: "#ffffff"
    accent: "#0891b2"
    success: "#16a34a"
    danger: "#dc2626"
    warning: "#d97706"
    info: "#0284c7"
    darkBackground: "#020617"
    darkSurface: "#0f172a"
    darkSurfaceMuted: "#1e293b"
    darkForeground: "#f8fafc"
    darkMutedForeground: "#94a3b8"
    darkBorder: "#334155"
  radius:
    control: "0.375rem"
    panel: "0.5rem"
    sheet: "1rem"
    pill: "9999px"
  typography:
    fontFamily: "Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, Segoe UI, sans-serif"
    monoFamily: "JetBrains Mono, Fira Code, SF Mono, Consolas, Liberation Mono, Menlo, monospace"
    baseSize: "16px"
    lineHeight: "1.5"
  spacing:
    unit: "0.25rem"
    touchTarget: "44px"
    desktopSidebar: "20rem"
    desktopSidebarMin: "16.25rem"
    desktopSidebarMax: "30rem"
constraints:
  letterSpacing: "0"
  cardRadiusMaximum: "0.5rem"
  defaultMode: "light"
  supportedModes: ["light", "dark", "system"]
  responsiveBaseline: "mobile-first"
  dominantPalette: "neutral-operational-with-blue-cyan-and-semantic-status"
---

# Vrooli Operational Console Design

`DESIGN.md` is the source of truth for scenario UI decisions. Stack-specific adapters may translate these tokens into CSS, Tailwind, egui, native mobile themes, or future targets, but adapters must not redefine the design language.

## Intent

Vrooli Operational Console is the default design language for scenario applications. It is built for operators, agents, reviewers, maintainers, and builders who need to understand system state quickly and act repeatedly without friction.

The interface should feel calm, technical, dense, legible, and durable. It should borrow the strongest patterns from Swarm Manager and Git Control Tower: dark operational chrome, precise status color, compact information surfaces, resizable desktop panels, mobile-first navigation adaptations, and strong support for long-running workflows. It should not feel like a marketing site, decorative dashboard, consumer social app, or generic purple-gradient AI product.

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

Preferred primitives:

- **Shells:** desktop sidebars, collapsible panes, resizable inspectors, bottom mobile navigation, route-level mobile panels.
- **Dialogs:** compact modal on desktop; bottom sheet or full-screen panel on mobile when content is more than a short confirmation.
- **Lists and tables:** dense rows, sticky headers when useful, clear selected state, empty/loading/error states, and bulk action affordances.
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

Customization should be implemented through tokens and stateful preferences, not one-off component rewrites. Local product themes may extend this design, but they should keep semantic color roles, focus behavior, responsive transformations, and accessibility guarantees intact.

## Workflow Ergonomics

Design from the user's flow, not from component inventory. For each major screen, identify the primary repeated action, the highest-risk action, the most common comparison, and the first thing a new user needs to understand.

Experienced users should be able to move quickly with short pointer travel, predictable keyboard focus, visible shortcuts where appropriate, persisted panel sizes, remembered filters, and stable navigation state. New users should see enough structure, labels, and progressive disclosure to understand what is actionable without reading documentation.

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

### Don't

- Create a marketing-style landing page as the first screen of a scenario application.
- Make mobile a cramped version of desktop.
- Hide important mobile actions behind hover-only affordances.
- Use decorative gradients, orbs, or background effects as the default visual identity.
- Let component libraries or adapter assets become a separate source of design truth.
- Introduce a new product theme without updating the scenario's root `DESIGN.md`.
