/**
 * Execution Viewer components and configuration
 *
 * This module contains the ExecutionViewer and its supporting components.
 * The viewer displays execution progress, replay, screenshots, logs, and history.
 *
 * Architecture Note:
 * ExecutionViewer.tsx is currently a large monolithic component (~4800 lines).
 * Future refactoring should split it into:
 * - TabNavigation: Tab switching UI
 * - ReplayTab: Replay player and theming controls
 * - ScreenshotsTab: Screenshot gallery
 * - LogsTab: Execution logs viewer
 * - ExecutionsTab: Execution history list
 * - ExportDialog: Export modal and configuration
 * - ExecutionHeader: Status bar and action buttons
 */

// Export configuration
export * from "./exportConfig";
