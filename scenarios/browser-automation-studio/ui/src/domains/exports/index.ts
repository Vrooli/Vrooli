/**
 * Exports domain - handles replay generation, export rendering, and media output
 *
 * Structure:
 * - replay/: ReplayPresentation + ReplayPlayer shell (controls below), theming (backgrounds, cursors, chrome)
 * - renderer/: Export rendering components
 * - store.ts: Export state management
 */

// Store
export { useExportStore } from './store';
export type { Export, CreateExportInput, UpdateExportInput, ExportState } from './store';

// Main components
export { default as ReplayPlayer } from "./replay/ReplayPlayer";
export { ExportSuccessPanel } from "./ExportSuccessPanel";
export { ExportsTab } from "./ExportsTab";

// Replay types, constants, and utilities
export * from "./replay";
