/**
 * Recording Module - Record Mode for Browser Automation
 *
 * ┌─────────────────────────────────────────────────────────────────────────┐
 * │ FILE GUIDE - What to read based on what you're trying to do:           │
 * ├─────────────────────────────────────────────────────────────────────────┤
 * │                                                                         │
 * │ UNDERSTANDING THE SYSTEM:                                               │
 * │   types.ts         - All type definitions (RecordedAction, etc.)        │
 * │   controller.ts    - Main orchestrator (start/stop recording, replay)   │
 * │                                                                         │
 * │ ADDING A NEW ACTION TYPE (e.g., 'drag'):                                │
 * │   1. action-types.ts   - Add to ACTION_TYPES, aliases, kind mapping     │
 * │   2. action-executor.ts - Register executor with registerActionExecutor │
 * │   3. types.ts          - Add payload interface if needed                │
 * │                                                                         │
 * │ MODIFYING SELECTOR GENERATION:                                          │
 * │   1. selector-config.ts - Configuration (scores, patterns) - EDIT THIS  │
 * │   2. injector.ts        - Runtime code injected into browser - EDIT THIS│
 * │   3. selectors.ts       - Documentation only, not executable            │
 * │                                                                         │
 * │ MODIFYING ACTION BUFFERING:                                             │
 * │   buffer.ts        - In-memory storage with deduplication               │
 * │                                                                         │
 * └─────────────────────────────────────────────────────────────────────────┘
 *
 * ARCHITECTURE OVERVIEW:
 *
 * Browser Page                    Node.js Driver
 * ┌─────────────┐                ┌─────────────────────┐
 * │  injector   │  ─────────────▶│  RecordModeController│
 * │  (JS code)  │  page.expose   │  ├─ normalizeEvent() │
 * │             │  Function()    │  ├─ replayPreview()  │
 * └─────────────┘                │  └─ validateSelector│
 *       │                        └──────────┬──────────┘
 *       │ Captures events                   │
 *       ▼                                   ▼
 * ┌─────────────┐                ┌─────────────────────┐
 * │ DOM Events  │                │  buffer.ts          │
 * │ click, type │                │  (action storage)   │
 * └─────────────┘                └─────────────────────┘
 */

export * from './types';
export * from './action-types';
export * from './selector-config';
// Note: action-executor re-exports ActionType from types, so we selectively export
export {
  registerActionExecutor,
  getActionExecutor,
  hasActionExecutor,
  getRegisteredActionTypes,
  type ActionExecutor,
  type ActionExecutorContext,
} from './action-executor';
export * from './selectors';
export * from './injector';
export * from './controller';
export * from './buffer';
