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
import { serializeConfigForBrowser } from '../validation/selector-config';

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
  const script = template
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

/** Await the MAIN-world recorder through a transferable reply port. */
function generateControlScript(action: 'start' | 'stop', sessionId?: string): string {
  const message = JSON.stringify({ type: RECORDING_CONTROL_MESSAGE_TYPE, action, sessionId });
  // WindowProxy.postMessage addresses child documents across origins without
  // relying on Rebrowser's child-frame evaluation context lookup.
  return `Promise.all((function collect(target) {
    const windows = [target];
    for (let i = 0; i < target.frames.length; i++) windows.push(...collect(target.frames[i]));
    return windows;
  })(window).map(target => new Promise((resolve, reject) => {
    const channel = new MessageChannel();
    const timeout = setTimeout(() => {
      channel.port1.close();
      reject(new Error('Recording control acknowledgement timed out'));
    }, 10000);
    channel.port1.onmessage = (event) => {
      clearTimeout(timeout);
      channel.port1.close();
      if (event.data?.ok === true) resolve(undefined);
      else reject(new Error(event.data?.error || 'Recording control was not acknowledged'));
    };
    target.postMessage(${message}, '*', [channel.port2]);
  })))`;
}

export function generateActivationScript(recordingId: string): string {
  return generateControlScript('start', recordingId);
}

export function generateDeactivationScript(): string {
  return generateControlScript('stop');
}
