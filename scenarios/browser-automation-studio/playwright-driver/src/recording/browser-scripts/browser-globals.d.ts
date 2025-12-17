/**
 * TypeScript declarations for browser globals used by recording scripts.
 *
 * These globals are created by the recording script (recording-script.js)
 * and used by both the browser-side code and Playwright's exposeFunction.
 *
 * @see ./recording-script.js - Browser-side recording implementation
 * @see ../injector.ts - Node.js module that sets up these globals
 */

import type { RawBrowserEvent } from '../types';

declare global {
  interface Window {
    /**
     * Flag indicating whether recording is currently active.
     * Set to true when recording starts, false when it stops.
     * Used to prevent double-injection and to ignore events after stop.
     */
    __recordingActive?: boolean;

    /**
     * Callback function exposed by Playwright to receive raw browser events.
     * This is set up via page.exposeFunction() in the RecordModeController.
     *
     * @param action - Raw browser event captured by the recording script
     */
    __recordAction?: (action: RawBrowserEvent) => void;

    /**
     * Exposed selector generation function for debugging and testing.
     * Available in browser console when recording is active.
     *
     * @param element - DOM element to generate selectors for
     * @returns Object with primary selector and candidates array
     */
    __generateSelectors?: (element: Element) => {
      primary: string;
      candidates: Array<{
        type: string;
        value: string;
        confidence: number;
        specificity: number;
      }>;
    };
  }
}

export {};
