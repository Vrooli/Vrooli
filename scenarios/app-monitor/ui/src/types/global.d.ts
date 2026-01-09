/**
 * Global type augmentations for window object.
 * Provides type safety for debug flags and runtime state.
 */

interface Window {
  /**
   * Debug flag for port computation in AppModal (reserved for future use)
   * Enable via: window.__DEBUG_APP_MODAL_PORTS = true
   */
  __DEBUG_APP_MODAL_PORTS?: boolean;

  /**
   * Debug flag to log UI port resolution in appPreview utils
   * Enable via: window.__DEBUG_UI_PORT = true
   */
  __DEBUG_UI_PORT?: boolean;

  /**
   * Debug flag for scenarios without UI ports
   * Enable via: window.__DEBUG_NO_UI_PORT = true
   */
  __DEBUG_NO_UI_PORT?: boolean;

  /**
   * iOS Safari autoback navigation guard state.
   * Prevents unwanted back navigation on iOS Safari when interacting with iframes.
   */
  __appMonitorPreviewGuard?: {
    /** Whether the guard is currently active */
    active: boolean;
    /** Timestamp when guard was armed (ms) */
    armedAt: number;
    /** Time-to-live for the guard (ms) */
    ttl: number;
    /** Unique key for this guard instance */
    key: string | null;
    /** App ID being guarded */
    appId: string | null;
    /** Path to recover to if back navigation is suppressed */
    recoverPath: string | null;
    /** Whether to ignore the next popstate event */
    ignoreNextPopstate?: boolean;
    /** Timestamp of last suppressed navigation (ms) */
    lastSuppressedAt?: number;
    /** State to restore when recovering */
    recoverState?: unknown;
  };
}
