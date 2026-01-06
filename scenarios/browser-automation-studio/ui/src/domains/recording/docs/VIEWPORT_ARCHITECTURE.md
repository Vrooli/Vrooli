# Viewport Architecture

This document describes the viewport flow in the browser-automation-studio recording system.

## Overview

The viewport system manages browser window dimensions across three layers:
1. **UI Layer** (React) - Measures container bounds and displays viewport info
2. **API Layer** (Go) - Forwards viewport dimensions between UI and driver
3. **Driver Layer** (TypeScript/Playwright) - Manages actual browser viewport

## Key Architectural Decisions

### 1. Decoupled Browser vs Display Viewport

The actual Playwright browser viewport is decoupled from the UI display viewport:
- **Browser viewport**: What Playwright uses (sent to CDP screencast)
- **Display viewport**: How content is visually rendered in the UI (may use CSS scaling)

This prevents CDP screencast restarts when toggling replay style settings.

### 2. Fingerprint Override

Browser profile fingerprint settings can override the requested viewport dimensions:
- Location: `playwright-driver/src/session/context-builder.ts`
- Logic: `fingerprint.viewport_width || spec.viewport.width`

This means if a session profile has fingerprint viewport settings, they take precedence.

### 3. Actual Viewport Feedback

The system now returns the **actual** viewport from Playwright after session creation and viewport updates. This allows the UI to show when dimensions differ from requested.

### 4. ViewportProvider Context

The viewport state is now centralized in a React context (`ViewportProvider`) that:
- Wraps the recording session content
- Consolidates all viewport-related state (browser viewport, actual viewport, sync status)
- Uses `ViewportSyncManager` internally for debouncing and backend sync
- Provides `useViewport()` hook for child components to access viewport state
- Supports reactive updates when actual viewport changes from session creation

Benefits:
- Eliminates prop drilling for viewport state
- Single source of truth for all viewport-related data
- Child components (PreviewContainer, RecordPreviewPanel) can access viewport via context
- Props still accepted as fallback for backward compatibility (execution mode)

## Data Flow

```
┌──────────────────────────────────────────────────────────────────────┐
│                           UI LAYER                                    │
│                                                                       │
│  ┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐ │
│  │ PreviewContainer│───▶│ ViewportSyncMgr  │───▶│ BrowserChrome   │ │
│  │                 │    │ (debounce 200ms) │    │ (shows viewport)│ │
│  │ - ResizeObserver│    │                  │    │                 │ │
│  │ - Clamp 320-3840│    │ - Track resize   │    │ - Requested dim │ │
│  └─────────────────┘    │ - Sync to backend│    │ - Actual dim    │ │
│                         └────────┬─────────┘    │ - Mismatch warn │ │
│                                  │              └─────────────────┘ │
└──────────────────────────────────│──────────────────────────────────┘
                                   │ POST /recordings/live/{id}/viewport
                                   ▼
┌──────────────────────────────────────────────────────────────────────┐
│                           API LAYER                                   │
│                                                                       │
│  ┌─────────────────┐    ┌──────────────────┐                        │
│  │ record_mode.go  │───▶│ live-capture/svc │                        │
│  │ UpdateViewport  │    │                  │                        │
│  │                 │    │ - Forward to     │                        │
│  │ - Validate > 0  │    │   session manager│                        │
│  └─────────────────┘    └────────┬─────────┘                        │
│                                  │                                   │
└──────────────────────────────────│──────────────────────────────────┘
                                   │ POST /session/{id}/record/viewport
                                   ▼
┌──────────────────────────────────────────────────────────────────────┐
│                         DRIVER LAYER                                  │
│                                                                       │
│  ┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐ │
│  │ recording-      │───▶│ page.setViewport │───▶│ CDP screencast  │ │
│  │ interaction.ts  │    │ Size()           │    │ restart         │ │
│  │                 │    │                  │    │                 │ │
│  │ - Round dims    │    │                  │    │ - New maxWidth  │ │
│  │ - Return actual │    │                  │    │ - New maxHeight │ │
│  └─────────────────┘    └──────────────────┘    └─────────────────┘ │
│                                                                       │
└──────────────────────────────────────────────────────────────────────┘
```

## Session Creation Viewport Flow

1. UI creates session with `viewport_width` and `viewport_height`
2. API forwards to driver with viewport in spec
3. Driver's `context-builder.ts` applies fingerprint override if present
4. Driver returns `actual_viewport` in response
5. UI stores and displays actual viewport
6. BrowserChrome shows warning if dimensions differ

## Key Files

### UI Layer
- `RecordingSession.tsx` - Main orchestrator, wraps content in ViewportProvider
- `context/ViewportProvider.tsx` - React context for centralized viewport state
- `PreviewContainer.tsx` - Measures bounds, uses context for viewport state
- `RecordPreviewPanel.tsx` - Renders PlaywrightView, uses context for viewport
- `BrowserChrome.tsx` - Displays viewport indicator with actual vs requested
- `utils/ViewportSyncManager.ts` - Debounces and syncs viewport to backend (used by ViewportProvider)
- `hooks/useRecordingSession.ts` - Hook that stores actualViewport from session creation

### API Layer
- `handlers/record_mode.go` - HTTP handlers for viewport endpoints
- `handlers/record_mode_types.go` - Request/response types with ActualViewport
- `services/live-capture/service.go` - Service layer with ViewportDimensions

### Driver Layer
- `session/manager.ts` - Returns actualViewport from session creation
- `session/context-builder.ts` - Applies fingerprint viewport override
- `routes/record-mode/recording-interaction.ts` - Viewport update handler
- `frame-streaming/strategies/cdp-screencast.ts` - Restarts screencast on resize

## Troubleshooting

### Viewport stuck at unexpected dimensions

1. **Check session profile fingerprint settings**: The browser profile may have
   `viewport_width` or `viewport_height` set, which overrides UI-requested dimensions.

2. **Check BrowserChrome indicator**: If it shows "Actual: 1280x900" different from
   "Requested: 1200x800", the fingerprint override is active.

3. **Debug logging**: Enable DEBUG logging in playwright-driver to see viewport
   flow in session creation.

### Viewport not updating

1. **Check ViewportSyncManager debounce**: Updates are debounced by 200ms
2. **Check WebSocket connection**: Viewport updates require active session
3. **Check CDP screencast**: Screencast must restart for new dimensions

## Testing

Key scenarios to test:
1. Session creation returns correct actual viewport
2. Viewport update returns correct actual viewport
3. Fingerprint override is reflected in actual viewport
4. BrowserChrome shows mismatch warning when dimensions differ
5. localStorage persistence of last selected session profile
6. ViewportProvider context provides correct state to children
7. Context mismatch detection works with tolerance

## Usage Example

```tsx
// RecordingSession wraps content in ViewportProvider
<ViewportProvider sessionId={sessionId} actualViewport={sessionActualViewport}>
  <PreviewContainer onBrowserViewportChange={handleBrowserViewportChange}>
    <RecordPreviewPanel sessionId={sessionId} ... />
  </PreviewContainer>
</ViewportProvider>

// Child components can use the hook
function ChildComponent() {
  const { browserViewport, actualViewport, hasMismatch, syncState } = useViewport();

  if (hasMismatch) {
    console.warn('Viewport mismatch:', browserViewport, 'vs', actualViewport);
  }

  return <div>Viewport: {browserViewport?.width}x{browserViewport?.height}</div>;
}
```
