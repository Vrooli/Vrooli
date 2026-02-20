# Experience Architecture Audit

## Last Updated
2026-02-19

## Personas Identified

| Persona | Primary Job | Current Flow | Ideal Flow |
|---------|-------------|--------------|------------|
| **Server operator** | Run CLI tools (Claude Code, diagnostics) via browser | Open workspace → launch terminal → type/paste commands | Same — currently well-served |
| **Mobile operator** | Quick terminal access from phone/tablet | Open workspace → use floating toolbar for special keys → run commands | Same, but with better gesture support for pane switching |
| **Parent scenario** | Embed terminal capability in larger app | iframe embed → postMessage bridge → session management via API | Same — currently well-served by iframe bridge |

## Friction Analysis

### Mechanical Friction
- **Pane management on mobile**: Single-column layout works, but switching between panes requires scrolling. No swipe gesture support.
- **AI input requires explicit mode switch**: User must click into AiInput component, type prompt, then decide execute vs copy. Could benefit from keyboard shortcut to toggle AI input.

### Cognitive Friction
- **Session vs pane distinction**: Users may not immediately understand that a "pane" is a UI viewport and a "session" is a server-side process. The drawer shows sessions; the workspace shows panes. Terminology is consistent but the mental model requires learning.
- **Policy options**: Expiration policies (never, 1h, 8h, 24h) are clear, but the countdown timer could be confusing if the user doesn't understand it represents remaining session lifetime.

### Discoverability Friction
- **Settings page**: Shortcut profile management and AI provider configuration are on a separate settings page accessed via hash routing. No visual cue on the workspace page that these exist.
- **Keyboard shortcuts**: No keyboard shortcut help/cheatsheet. Mobile toolbar keys are visible but desktop keyboard shortcuts (if any) are not documented in-UI.

## Navigation Integrity
- **Hash routing**: Three routes (`#workspace`, `#sessions`, `#settings`) via [CODE: ui/src/hooks/useHashRoute.ts]. Default is workspace.
- **Label→destination mismatches**: None found — navigation labels match page content.
- **Back/forward coherence**: Hash routing works with browser back/forward buttons correctly.
- **Deep link handling**: Direct URL with hash fragment works correctly (e.g., `#settings` loads settings page).

## Priority Improvements
1. **Add keyboard shortcut for AI input toggle** — reduces mechanical friction for power users
2. **Add subtle navigation indicator** for settings/sessions pages — improves discoverability
3. **Mobile pane swipe gestures** — reduces friction for multi-pane mobile use (P2 scope)
