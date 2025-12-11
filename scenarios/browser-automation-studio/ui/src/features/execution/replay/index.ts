/**
 * Replay feature exports
 *
 * Contains replay-related components and configuration:
 * - Theme options (backgrounds, cursors, chrome styles)
 * - ReplayPlayer component (re-exported from parent)
 * - WatermarkNotice component for entitlement notices
 */

// Theme configuration options
export * from "./replayThemeOptions";

// Entitlement notice components
export { WatermarkNotice } from "./WatermarkNotice";
