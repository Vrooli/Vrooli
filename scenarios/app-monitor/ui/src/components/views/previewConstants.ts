/**
 * Timing constants for preview loading and interaction behaviors
 */
export const PREVIEW_TIMEOUTS = {
  /** Timeout for iframe to load before showing error overlay (ms) */
  LOAD: 6000,
  /** Delay before showing "connecting" overlay to avoid flashing (ms) */
  WAITING_DELAY: 400,
  /** iOS Safari autoback guard duration (ms) */
  IOS_AUTOBACK_GUARD: 15000,
  /** Delay before preparing auto-next scenario precomputation (ms) */
  AUTO_NEXT_PREPARE: 4500,
  /** Duration to show status messages before auto-dismissing (ms) */
  STATUS_MESSAGE_DURATION: 1500,
  /** Duration to show copy feedback notifications (ms) */
  COPY_FEEDBACK_DURATION: 1800,
  /** Duration to show inspector success messages (ms) */
  INSPECTOR_MESSAGE_SHORT: 2400,
  /** Duration to show inspector standard messages (ms) */
  INSPECTOR_MESSAGE_STANDARD: 3200,
  /** Duration to show inspector detailed messages (ms) */
  INSPECTOR_MESSAGE_DETAILED: 3600,
} as const;

/**
 * UI layout and interaction constants for preview components
 */
export const PREVIEW_UI = {
  /** Offset for menu popovers from anchor element (px) */
  MENU_OFFSET: 8,
  /** Margin for floating elements from viewport edges (px) */
  FLOATING_MARGIN: 12,
  /** Minimum pointer movement to initiate drag operation (px) */
  DRAG_THRESHOLD: 3,
  /** Default floating position for toolbar in fullscreen mode */
  DEFAULT_FLOATING_POSITION: { x: 16, y: 16 } as const,
} as const;

/**
 * Inspector-specific UI constants
 */
export const INSPECTOR_UI = {
  /** Margin for inspector dialog from viewport edges (px) */
  FLOATING_MARGIN: 12,
  /** Default inspector dialog position */
  DEFAULT_POSITION: { x: 24, y: 12 } as const,
  /** Minimum pointer movement to initiate inspector drag (px) */
  DRAG_THRESHOLD: 3,
} as const;

/**
 * Preview status messages
 */
export const PREVIEW_MESSAGES = {
  CONNECTING: 'Connecting to preview...',
  TIMEOUT: 'Preview did not respond. Ensure the application UI is running and reachable from App Monitor.',
  MIXED_CONTENT: 'Preview blocked: browser refused to load HTTP content inside an HTTPS dashboard. Expose the UI through the tunnel hostname or enable HTTPS for the scenario.',
  NO_UI: 'This application does not expose a UI endpoint to preview.',
} as const;
