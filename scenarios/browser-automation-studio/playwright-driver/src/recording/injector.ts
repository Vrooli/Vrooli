/**
 * Recording Injector
 *
 * JavaScript code that gets injected into browser pages to capture user actions.
 * This module loads the browser script from a separate file and injects it with
 * configuration from selector-config.ts.
 *
 * ARCHITECTURE:
 * - recording-script.js: Browser-side JavaScript (runs in page context)
 * - cleanup-script.js: Script to deactivate recording
 * - browser-globals.d.ts: TypeScript declarations for browser globals
 * - selector-config.ts: Configuration source of truth (Node.js)
 *
 * The separation allows:
 * - Proper IDE support for browser JavaScript (syntax highlighting, etc.)
 * - Independent unit testing of selector logic
 * - Clear boundary between Node.js and browser execution contexts
 *
 * @see ./browser-scripts/recording-script.js - Browser-side implementation
 * @see ./browser-scripts/browser-globals.d.ts - TypeScript declarations
 */

import * as fs from 'fs';
import * as path from 'path';
import type { RawBrowserEvent } from './types';
import { serializeConfigForBrowser } from './selector-config';

// =============================================================================
// Script Loading
// =============================================================================

/**
 * Cache for loaded scripts to avoid repeated file I/O.
 */
let cachedRecordingScript: string | null = null;
let cachedCleanupScript: string | null = null;

/**
 * Load a script file from the browser-scripts directory.
 */
function loadBrowserScript(filename: string): string {
  const scriptPath = path.join(__dirname, 'browser-scripts', filename);
  return fs.readFileSync(scriptPath, 'utf-8');
}

/**
 * Get the raw recording script template (without config injection).
 * Uses caching to avoid repeated file reads.
 */
function getRecordingScriptTemplate(): string {
  if (cachedRecordingScript === null) {
    cachedRecordingScript = loadBrowserScript('recording-script.js');
  }
  return cachedRecordingScript;
}

/**
 * Get the cleanup script content.
 * Uses caching to avoid repeated file reads.
 */
function getCleanupScriptTemplate(): string {
  if (cachedCleanupScript === null) {
    cachedCleanupScript = loadBrowserScript('cleanup-script.js');
  }
  return cachedCleanupScript;
}

// =============================================================================
// Public API
// =============================================================================

/**
 * Generate the recording script that will be injected into pages.
 *
 * This function:
 * 1. Loads the browser script template
 * 2. Injects the serialized configuration from selector-config.ts
 * 3. Returns a self-contained JavaScript string ready for page.evaluate()
 *
 * The configuration is injected by replacing the `__INJECTED_CONFIG__` placeholder
 * in the script template with the actual serialized configuration object.
 *
 * @returns JavaScript string for browser execution
 */
export function getRecordingScript(): string {
  const template = getRecordingScriptTemplate();
  const serializedConfig = serializeConfigForBrowser();

  // Replace the config placeholder with actual configuration
  return template.replace('__INJECTED_CONFIG__', serializedConfig);
}

/**
 * Get the cleanup script to remove recording event listeners.
 *
 * @returns JavaScript string for browser execution
 */
export function getCleanupScript(): string {
  return getCleanupScriptTemplate();
}

/**
 * Clear the script cache.
 * Useful for testing or when scripts need to be reloaded.
 */
export function clearScriptCache(): void {
  cachedRecordingScript = null;
  cachedCleanupScript = null;
}

/**
 * Type definition for the raw record action function exposed to browser context.
 * This receives raw browser events before normalization.
 * @internal
 */
export type RawRecordActionCallback = (action: RawBrowserEvent) => void;
