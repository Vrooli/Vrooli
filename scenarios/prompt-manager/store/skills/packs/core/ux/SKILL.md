---
name: "ux"
description: "User experience quality across all interfaces"
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["steer","ux"]
  tags: ["skill","mobile","responsive"]
  icon: "users"
  status: "active"
  targetDimensions: ["accessibility","visual","ui"]
  defaultScope: "refactor-scope"
  revision: 50
  createdAt: "2025-01-15T00:00:00Z"
  updatedAt: "2026-02-17T17:17:26Z"
  requires:
    scenarios: ["prompt-manager"]
    commands: ["prompt-manager skill", "prompt-manager skill read"]
  origin:
    kind: "authored"
---
## Steer focus: UX Improvement

> **Ladder position:** R3 (features hardened — ui/visual/accessibility). See `prompt-manager skill read scenario-maturity-ladder` for rung context and `prompt-manager skill read improvement-do-and-dont` for what counts as a real improvement.

Prioritize **user experience quality** across all interfaces in this scenario.
Do **not** break functionality or regress tests; all changes must maintain or improve completeness.

Required reading:
- `prompt-manager skill read visited-tracker-tools knowledge-observatory-tools`

Focus on producing a **professional, polished, friction-free user experience**, guided by the following principles:

### **1. Clarity & Understanding**

* Read the `problems` doc for `{{TARGET}}` using `knowledge-observatory-tools` to understand existing UX issues.

* Ensure all UI elements have **clear affordances and signifiers** indicating how they should be used.
* Add **help buttons, tooltips, hint text, or short explainer components** wherever concepts, parameters, or interactions may be confusing.
* Favor **concise text**, intuitive labels, and consistent terminology across the UI.
* Ensure **testability is part of UX clarity**:
  * Add stable `data-testid` selectors for key UI elements so BAS workflows can validate the experience end-to-end
  * Use the scenario selector registry (`ui/src/constants/selectors.ts`) - never hardcode selectors
  * Reference selectors in components: `data-testid={selectors.dashboard.newProjectButton}`
  * See **browser-automation-studio** skill for validating selectors with BAS CLI
  * See **e2e-testing** skill for selector registry standards and workflow authoring patterns

Optional reading:
- `prompt-manager skill read browser-automation-studio e2e-testing`

### **2. Layout & Information Hierarchy**

* Organize content so the user’s attention naturally follows an **F-shaped visual scanning pattern**:

  * primary actions located where attention begins
  * secondary elements placed where they’re discoverable but unobtrusive
* Reduce unnecessary cognitive load by grouping related controls and trimming redundant steps.

### **3. Reduce Friction**

* Minimize clicks, scrolling, typing, and context switching needed to complete **common user journeys**.
* Identify and smooth out high-friction areas:

  * excessive dialogs or confirmations
  * unclear navigation paths
  * repetitive manual inputs
* Make the most frequent workflows **as direct, obvious, and streamlined as possible**.

### **4. Professional Interaction Design**

* Prefer **icons (Lucide)** over emojis for UI communication.
* Use consistent spacing, alignment, padding, and component sizing.
* Avoid clutter; prioritize readability and clean composition.
* Improve empty states, loading states, and error states so the experience remains smooth and emotionally neutral.
* Ensure first-time users immediately understand what the scenario does, how to use it, and what the next step is. Enhance empty states with clarity, guidance, and calls to action.
* Where appropriate, add small but meaningful micro-interactions (hover/focus states, loading indicators, optimistic UI cues) that enhance feedback without clutter

### **5. Responsiveness & Device Adaptation**

Mobile is not a secondary concern — most scenarios will be accessed on constrained viewports.
Every view must degrade gracefully from desktop to mobile without requiring a separate "mobile version."

#### 5.1 The Mobile Viewport Test

Before considering any view or component done, mentally render it in a constrained mobile viewport
with a virtual keyboard open. If it is not usable, readable, and non-overflowing, it needs adaptation.

#### 5.2 Vertical Space Budget

The core mobile constraint is vertical space. Evaluate what earns screen real estate:

```
Is this element needed for the PRIMARY action on this screen?
    ├─ YES → Keep visible, but make it compact
    └─ NO → Can the user access it on demand (menu, expand, long-press)?
         ├─ YES → Hide it; surface via progressive disclosure
         └─ NO → Is it critical for orientation (status, context)?
              ├─ YES → Condense to minimal indicator (icon, badge, chip)
              └─ NO → Remove from mobile view entirely
```

* Headers are the most common offender — they accumulate elements that each seem small but together consume disproportionate space.
* Secondary actions belong behind overflow/ellipsis menus on mobile.
* Metadata rows (tags, labels, selectors) should condense to summary representations (counts, chips) with expandable access to full content.
* When a desktop view has both an app-level header and a page-level header, the mobile version should consolidate into one, carrying essential actions from both.

#### 5.3 Overflow & Containment

Every element that accepts variable-length content must have an explicit containment strategy.
The two failure modes are **horizontal overflow** (content pushes beyond viewport width) and
**vertical bloat** (content expands without bound, pushing other elements off-screen).

| Content type | Strategy |
|---|---|
| Prose / user-generated text | Word-wrap within container; container width should decrease on smaller viewports |
| Code / preformatted text | Horizontal scroll within a bounded container |
| URLs, file paths, hashes | Break at any character to prevent overflow |
| Labels, tags, metadata | Truncate with access to full text (tooltip, expand) |
| Lists of items (tags, chips) | Show limited count with "+N more" pattern |

#### 5.4 Layout Adaptation

Layouts that distribute content horizontally on desktop should redistribute vertically or into
layered surfaces on mobile.

```
Does this layout place 2+ content regions side by side?
    ├─ YES → On mobile, should the user see both simultaneously?
    │    ├─ YES (rare) → Stack vertically, each taking full width
    │    └─ NO → Use navigation between views (back button, tabs, drawer)
    └─ NO (single column already) → Verify spacing and padding reduce appropriately
```

Common patterns to reason about:
* **Multi-column grids** → single column on mobile, with appropriate item representation (e.g., card grids often become simpler list items with dividers, not just a single-column card stack)
* **Sidebars** → drawers, sheets, or overlay panels
* **Tab bars with many tabs** → scrollable or collapsed into a selector
* **Split panes** → single pane with navigation between views

#### 5.5 Input & Interaction Areas

On mobile, the virtual keyboard consumes roughly half the viewport. Everything surrounding an
input field competes for the remaining space.

* Count the UI elements surrounding the primary input. If there are more than 2-3, some must be collapsed, deferred, or hidden when the input is active.
* Toolbars and action bars around inputs should be consolidated — prefer a single compact row with overflow over multiple stacked rows.
* Auxiliary UI (suggestions, previews, options panels) should overlay or replace content rather than pushing the input area around.
* The input itself should feel native (e.g. if updating a message input box, study how professional messaging/chat apps like iMessage and ChatGPT handle input, and match that behavior rather than inventing novel interactions).

#### 5.6 Spacing Adaptation

Spacing appropriate for desktop is almost always too generous for mobile. Every container visible
at multiple viewport sizes should use responsive spacing.

* Reduce padding and margins at smaller breakpoints — the space that creates "breathing room" on desktop becomes wasted space on mobile.
* Gaps between items should tighten proportionally.
* Modals and dialogs should approach edge-to-edge on mobile rather than floating with large margins.

**Anti-pattern:** any fixed spacing value on a container that appears at every viewport size without responsive variants.

#### 5.7 Touch & Interaction Targets

* Interactive elements need a minimum comfortable touch target (~44px).
* Adjacent touch targets need enough spacing that mis-taps are rare.
* Expand hit areas via padding rather than making icons larger.
* Swipe gestures and long-press should supplement but never replace visible controls.

#### 5.8 Safe-Area Awareness

Elements fixed to viewport edges (input bars, floating buttons, bottom navigation) must account
for device safe areas (notches, home indicators, rounded corners). Use environment-aware insets
rather than hardcoded offsets.

#### 5.9 Mobile Readiness Checklist

* [ ] Responsive spacing on containers (not fixed padding at all sizes)
* [ ] Variable-length content has containment (wrap, truncate, or scroll)
* [ ] Headers don't accumulate uncondensed elements
* [ ] Layout redistributes for narrow viewports
* [ ] Input areas remain usable with virtual keyboard
* [ ] No horizontal overflow at narrow viewport widths
* [ ] Touch targets are comfortably sized
* [ ] Fixed-position elements account for safe areas

### **6. Memory Management with Visited Tracker**

Use the `visited-tracker-tools` skill for tracking visited files, with LOCATION set to `scenarios/{{TARGET}}/ui` and TAG set to `ux`.

### **8. Output Expectations**

You may update:

* components
* styles
* labels
* UI flows
* help/tooltip systems
* interaction logic
* responsive styles
* empty and loading states

You **must**:

* keep the scenario fully functional
* avoid regressions
* increase user clarity and ease-of-use
* meaningfully improve UX quality without gaming metrics

Focus the loop on delivering **practical, targeted UX improvements** that make the scenario genuinely easier and more enjoyable to use.

**When choosing what to modify, consider the scenario’s most common user journeys and optimize for the fewest steps, lowest friction, and clearest navigation.**

**Avoid superficial UX changes that alter appearance without improving actual usability. Only make changes that meaningfully reduce friction or increase understanding.**

---

### **9. Documentation**

Use `knowledge-observatory-tools` to read the current `problems` doc for `{{TARGET}}`, then update the **UX Issues** section with your findings (high-friction areas, clarity gaps, mobile responsiveness issues, empty/loading/error state improvements, remaining UX debt).
