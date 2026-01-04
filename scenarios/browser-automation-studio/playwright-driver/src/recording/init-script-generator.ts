/**
 * Init Script Generator for Recording
 *
 * Generates a recording init script that:
 * 1. Runs in MAIN context (via addInitScript)
 * 2. Remains dormant until activated via window message
 * 3. Properly wraps History API in the main context
 * 4. Communicates events via context binding (not page.exposeFunction)
 *
 * ARCHITECTURE:
 * - This replaces the old injector.ts page.evaluate() approach
 * - Uses context.addInitScript() which runs in MAIN world (not isolated)
 * - Message-based activation allows dynamic start/stop without re-injection
 * - Works correctly with rebrowser-playwright's isolated context patches
 *
 * @see context-initializer.ts - Sets up the init script and binding
 * @see selector-config.ts - Configuration source of truth
 */

import * as fs from 'fs';
import * as path from 'path';
import { serializeConfigForBrowser } from './selector-config';

// =============================================================================
// Constants
// =============================================================================

/** Message type for recording control messages */
export const RECORDING_CONTROL_MESSAGE_TYPE = '__VROOLI_RECORDING_CONTROL__';

/** Default binding name for recording events */
export const DEFAULT_RECORDING_BINDING_NAME = '__vrooli_recordAction';

// =============================================================================
// Script Loading
// =============================================================================

/** Cache for loaded recording script template */
let cachedRecordingScriptTemplate: string | null = null;

/**
 * Load the recording script template from disk.
 * Uses caching to avoid repeated file I/O.
 */
function getRecordingScriptTemplate(): string {
  if (cachedRecordingScriptTemplate === null) {
    const scriptPath = path.join(__dirname, 'browser-scripts', 'recording-script.js');
    cachedRecordingScriptTemplate = fs.readFileSync(scriptPath, 'utf-8');
  }
  return cachedRecordingScriptTemplate;
}

/**
 * Clear the script cache.
 * Useful for testing or when scripts need to be reloaded.
 */
export function clearScriptCache(): void {
  cachedRecordingScriptTemplate = null;
}

// =============================================================================
// Init Script Generation
// =============================================================================

/**
 * Generate the recording init script for context.addInitScript().
 *
 * This script:
 * - Runs in MAIN context on every page load
 * - Starts in dormant state (not capturing events)
 * - Listens for activation/deactivation messages
 * - Uses the provided binding name for event communication
 * - Properly wraps History API in main context (the key fix!)
 *
 * @param bindingName - Name of the exposed binding for event communication
 * @returns JavaScript string ready for context.addInitScript()
 */
export function generateRecordingInitScript(bindingName: string = DEFAULT_RECORDING_BINDING_NAME): string {
  const template = getRecordingScriptTemplate();
  const serializedConfig = serializeConfigForBrowser();

  // Replace placeholders in the template
  // Note: BINDING_NAME and MESSAGE_TYPE are string literals in the template (with quotes around them)
  // so we replace the placeholder with just the value (the quotes are already in the template)
  let script = template
    .replace('__INJECTED_CONFIG__', serializedConfig)
    .replace(/__INJECTED_BINDING_NAME__/g, bindingName)
    .replace(/__RECORDING_CONTROL_MESSAGE_TYPE__/g, RECORDING_CONTROL_MESSAGE_TYPE);

  return script;
}

// =============================================================================
// Activation/Deactivation Scripts
// =============================================================================

/** Message type for recording events from MAIN to ISOLATED context */
export const RECORDING_EVENT_MESSAGE_TYPE = '__VROOLI_RECORDING_EVENT__';

/** Storage key for activation state - shared between contexts */
export const RECORDING_ACTIVATION_KEY = '__vrooli_recording_activation__';

/**
 * Generate the activation script to start recording on a page.
 *
 * Uses sessionStorage for cross-context activation since postMessage
 * doesn't work reliably between MAIN and ISOLATED contexts with rebrowser-playwright.
 * Both contexts share sessionStorage, so we can use it for signaling.
 *
 * Called via page.evaluate() when recording starts.
 *
 * @param sessionId - The recording session ID
 * @param bindingName - The name of the exposed binding (unused but kept for API compat)
 * @returns JavaScript string for page.evaluate()
 */
export function generateActivationScript(sessionId: string, bindingName: string = DEFAULT_RECORDING_BINDING_NAME): string {
  return `
(function() {
  // Use sessionStorage for cross-context activation
  // Both MAIN and ISOLATED contexts share sessionStorage
  try {
    sessionStorage.setItem('${RECORDING_ACTIVATION_KEY}', JSON.stringify({
      active: true,
      sessionId: '${sessionId}',
      timestamp: Date.now()
    }));
    console.log('[Recording Activation] Set in sessionStorage for session:', '${sessionId}');
  } catch (e) {
    console.error('[Recording Activation] Failed to set state:', e.message);
  }

  // Also try postMessage as backup
  window.postMessage({
    type: '${RECORDING_CONTROL_MESSAGE_TYPE}',
    action: 'start',
    sessionId: '${sessionId}'
  }, '*');
})();
`;
}

/**
 * Generate the deactivation script to stop recording on a page.
 *
 * This sends a postMessage to the recording init script to deactivate it.
 * Called via page.evaluate() when recording stops.
 *
 * @returns JavaScript string for page.evaluate()
 */
export function generateDeactivationScript(): string {
  return `
(function() {
  console.log('[Recording] Sending deactivation message');
  window.postMessage({
    type: '${RECORDING_CONTROL_MESSAGE_TYPE}',
    action: 'stop'
  }, '*');
})();
`;
}
