/**
 * Spatial navigation bridge — wires `GamepadInputManager` to
 * `SpatialNavManager`, handles mode switching, and relays unhandled
 * actions to the host via the existing shortcut-intent protocol.
 *
 * This is the primary public entry point for the `./spatial` subpath.
 * Scenarios call `initSpatialNav()` and interact with the returned
 * `SpatialNavController`.
 *
 * Zero runtime dependencies. Framework-agnostic.
 */

import {
  GamepadInputManager,
  type GamepadAction,
  type GamepadInputOptions,
} from './gamepadInput.js';
import {
  SpatialNavManager,
  type Direction,
  type FocusGroupMode,
  type FocusGroupOptions,
  type SpatialNavOptions,
} from './spatialNav.js';
import { emitShortcutIntent } from './iframeBridgeChild.js';
// Activation overlay is available as a standalone export for scenarios that
// want an explicit "press any button" prompt.  It is NOT wired into
// initSpatialNav — see `showActivationOverlay` in './activationOverlay.js'.

// ---------------------------------------------------------------------------
// Re-exports (so consumers only need `@vrooli/iframe-bridge/spatial`)
// ---------------------------------------------------------------------------

export { GamepadInputManager, type GamepadAction, type GamepadInputOptions } from './gamepadInput.js';
export {
  SpatialNavManager,
  FOCUSABLE_SELECTOR,
  type Direction,
  type FocusGroupMode,
  type FocusGroupOptions,
  type SpatialNavOptions,
} from './spatialNav.js';
export {
  DEFAULT_SPATIAL_FOCUS_CSS,
  injectSpatialStyles,
  removeSpatialStyles,
} from './spatialNavStyles.js';
export {
  showActivationOverlay,
  type ActivationOverlayOptions,
  type ActivationOverlayHandle,
  type DismissReason,
} from './activationOverlay.js';

// ---------------------------------------------------------------------------
// Public types
// ---------------------------------------------------------------------------

export interface SpatialNavBridgeOptions extends GamepadInputOptions, SpatialNavOptions {
  /**
   * Start gamepad polling immediately after initialisation.
   * Default `true`.
   */
  autoActivate?: boolean;
  /**
   * Relay unhandled gamepad actions to the host (app-monitor) via
   * `emitShortcutIntent`.  Default `true`.
   */
  hostRelay?: boolean;
}

export interface SpatialNavController {
  /** Stop polling, remove listeners, and clean up all state. */
  dispose(): void;
  /**
   * Register a focus group.
   * @returns A dispose function that unregisters the group.
   */
  registerGroup(
    element: HTMLElement,
    mode: FocusGroupMode,
    options?: FocusGroupOptions,
  ): () => void;
  /** Whether spatial mode is currently active. */
  isActive(): boolean;
  /** Manually enter spatial mode (normally auto-detected via D-pad). */
  enterSpatialMode(): void;
  /** Manually exit spatial mode (normally auto-detected via mouse). */
  exitSpatialMode(): void;
  /**
   * Push a modal scope — constrains all spatial navigation to within
   * `element` (e.g., a dialog).  Auto-focuses the first focusable child.
   * Supports nesting (dialog opens dialog).
   */
  pushScope(element: HTMLElement): void;
  /**
   * Pop the current modal scope, restoring the previous one (or root).
   */
  popScope(): void;
}

// ---------------------------------------------------------------------------
// Action → spatial nav wiring
// ---------------------------------------------------------------------------

/** Maps navigational GamepadActions to Directions. */
const ACTION_TO_DIRECTION: Partial<Record<GamepadAction, Direction>> = {
  'navigate-up': 'up',
  'navigate-down': 'down',
  'navigate-left': 'left',
  'navigate-right': 'right',
};

// ---------------------------------------------------------------------------
// initSpatialNav
// ---------------------------------------------------------------------------

/**
 * Initialise gamepad-driven spatial navigation.
 *
 * Call this once in your app's entry point (e.g., `main.tsx`) after
 * `initIframeBridgeChild()`.  It works both inside and outside an iframe.
 *
 * ```ts
 * import { initSpatialNav } from '@vrooli/iframe-bridge/spatial';
 * initSpatialNav();
 * ```
 */
export function initSpatialNav(options?: SpatialNavBridgeOptions): SpatialNavController {
  const hostRelay = options?.hostRelay ?? true;

  const spatialNav = new SpatialNavManager({
    focusableSelector: options?.focusableSelector,
    injectDefaultFocusStyle: options?.injectDefaultFocusStyle,
    rootElement: options?.rootElement,
    getBoundingClientRect: options?.getBoundingClientRect,
    isVisible: options?.isVisible,
  });

  const handleAction = (action: GamepadAction): void => {
    // B/back always navigates back — this is the only guaranteed escape on
    // console browsers where the virtual cursor may not be available.
    if (action === 'back') {
      spatialNav.goBack();
      return;
    }

    // Any other gamepad action activates spatial mode.
    if (!spatialNav.isActive()) {
      spatialNav.enterSpatialMode();
    }

    const direction = ACTION_TO_DIRECTION[action];

    if (direction) {
      const moved = spatialNav.moveFocus(direction);
      if (!moved && hostRelay) {
        emitShortcutIntent({
          action: `gamepad.${action}`,
          outcome: 'unhandled',
          source: 'programmatic',
        });
      }
      return;
    }

    switch (action) {
      case 'select':
        spatialNav.selectFocused();
        break;
      case 'page-next':
        if (!spatialNav.cycleFocusGroup('next') && hostRelay) {
          emitShortcutIntent({
            action: 'gamepad.page-next',
            outcome: 'unhandled',
            source: 'programmatic',
          });
        }
        break;
      case 'page-prev':
        if (!spatialNav.cycleFocusGroup('prev') && hostRelay) {
          emitShortcutIntent({
            action: 'gamepad.page-prev',
            outcome: 'unhandled',
            source: 'programmatic',
          });
        }
        break;
      case 'menu':
        if (hostRelay) {
          emitShortcutIntent({
            action: 'gamepad.menu',
            outcome: 'unhandled',
            source: 'programmatic',
          });
        }
        break;
    }
  };

  const gamepadInput = new GamepadInputManager({
    deadZone: options?.deadZone,
    repeatInitialDelayMs: options?.repeatInitialDelayMs,
    repeatIntervalMs: options?.repeatIntervalMs,
    getGamepads: options?.getGamepads,
    onAction: handleAction,
  });

  if (options?.autoActivate !== false) {
    gamepadInput.start();
  }

  return {
    dispose() {
      gamepadInput.dispose();
      spatialNav.dispose();
    },
    registerGroup(element, mode, groupOptions) {
      return spatialNav.registerGroup(element, mode, groupOptions);
    },
    isActive() {
      return spatialNav.isActive();
    },
    enterSpatialMode() {
      spatialNav.enterSpatialMode();
    },
    exitSpatialMode() {
      spatialNav.exitSpatialMode();
    },
    pushScope(element) {
      spatialNav.pushScope(element);
    },
    popScope() {
      spatialNav.popScope();
    },
  };
}
