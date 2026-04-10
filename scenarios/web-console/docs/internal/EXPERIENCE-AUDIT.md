# Experience Architecture Audit

## Last Updated
2026-03-17

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
- **Session vs pane distinction**: Users may not immediately understand that a "pane" is a UI viewport and a "session" is a server-side process. The unified settings surface now explains the relationship more clearly in the Sessions tab, but the model still requires some onboarding.
- **Policy options**: Expiration policies (never, 1h, 8h, 24h) are clear, but the countdown timer could be confusing if the user doesn't understand it represents remaining session lifetime.

### Discoverability Friction
- **Settings density**: Consolidation improved discoverability by placing sessions, workspace settings, voice, shortcuts, defaults, and integrations behind one entry point, but a few advanced controls still require drilling into dense forms.
- **Keyboard shortcuts**: No keyboard shortcut help/cheatsheet. Mobile toolbar keys are visible but desktop keyboard shortcuts (if any) are not documented in-UI.

## Navigation Integrity
- **Primary settings access**: A single toolbar Settings entry opens the unified settings surface.
- **Responsive behavior**: Desktop uses a wider draggable modal with sidebar tabs. Mobile uses a drawer with a horizontal tab row.
- **Label→destination mismatches**: None found — navigation labels match tab content.

## Priority Improvements
1. **Add keyboard shortcut for AI input toggle** — reduces mechanical friction for power users
2. **Add inline explanatory copy for policies/countdowns** — reduces remaining cognitive friction in the Sessions tab
3. **Mobile pane swipe gestures** — reduces friction for multi-pane mobile use (P2 scope)
