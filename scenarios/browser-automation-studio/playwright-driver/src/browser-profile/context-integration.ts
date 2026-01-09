/**
 * Browser Context Integration
 *
 * Integration point between browser profiles and Playwright browser contexts.
 * Provides the key for storing/retrieving behavior settings on BrowserContext.
 *
 * This module exists to provide a stable import path for both:
 * - session/context-builder.ts (writes settings)
 * - handlers/behavior-utils.ts (reads settings)
 *
 * Using a Symbol ensures no collision with other properties on the context.
 */

import type { BehaviorSettings } from '../types/browser-profile';

/**
 * Symbol key for storing BehaviorSettings on BrowserContext.
 * Use this to attach/retrieve human-like behavior configuration.
 */
export const BEHAVIOR_SETTINGS_KEY = Symbol('behaviorSettings');

// Extend BrowserContext type to include behavior settings
declare module 'rebrowser-playwright' {
  interface BrowserContext {
    [BEHAVIOR_SETTINGS_KEY]?: BehaviorSettings;
  }
}
