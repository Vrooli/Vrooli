/**
 * Gamepad activation overlay — shows a "Press any button" prompt until
 * a gamepad is detected.  Dismissed automatically on gamepadconnected,
 * mousemove, touchstart, keydown, or tap/click on the overlay itself.
 *
 * The overlay starts invisible and fades in after a short delay (~400ms).
 * Desktop and mobile users naturally move the mouse or touch the screen
 * within that window, dismissing it before it's ever visible.  Console/TV
 * users (no mouse, no touch) see it fade in and stay until they press a
 * controller button.
 *
 * Pure DOM, zero dependencies, framework-agnostic.
 */

// ---------------------------------------------------------------------------
// Public types
// ---------------------------------------------------------------------------

export type DismissReason = 'gamepad' | 'mouse' | 'keyboard' | 'touch';

export interface ActivationOverlayOptions {
  /** Custom message. Default: "Press any button on your controller" */
  message?: string;
  /** Custom sub-message. Default: "Tap anywhere to dismiss" */
  subMessage?: string;
  /**
   * Control when the overlay shows:
   * - `'auto'` (default): shows on all platforms but starts invisible;
   *   fades in after ~400ms. Mouse/touch dismisses it instantly, so
   *   desktop/mobile users never see it.  Console users see it appear.
   * - `true`: show immediately (no fade-in delay) regardless of platform.
   * - `false`: never show.
   */
  when?: 'auto' | boolean;
}

// ---------------------------------------------------------------------------
// Styles
// ---------------------------------------------------------------------------

const OVERLAY_ATTR = 'data-spatial-activation';

/** Delay before the overlay becomes visible in 'auto' mode. */
const AUTO_FADE_IN_DELAY_MS = 400;

const OVERLAY_CSS = `
[${OVERLAY_ATTR}] {
  position: fixed;
  inset: 0;
  z-index: 99999;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  background: rgba(2, 6, 23, 0.92);
  color: #e2e8f0;
  font-family: system-ui, -apple-system, sans-serif;
  text-align: center;
  padding: 2rem;
  cursor: pointer;
  -webkit-tap-highlight-color: transparent;
  transition: opacity 0.3s ease-out;
}

[${OVERLAY_ATTR}][data-auto] {
  opacity: 0;
  pointer-events: none;
}

[${OVERLAY_ATTR}][data-visible] {
  opacity: 1;
  pointer-events: auto;
}

[${OVERLAY_ATTR}][data-dismissing] {
  opacity: 0;
  pointer-events: none;
}

[${OVERLAY_ATTR}] .spatial-activation-icon {
  width: 64px;
  height: 64px;
  margin-bottom: 1.5rem;
  opacity: 0.7;
}

[${OVERLAY_ATTR}] .spatial-activation-msg {
  font-size: 1.5rem;
  font-weight: 600;
  margin-bottom: 0.5rem;
}

[${OVERLAY_ATTR}] .spatial-activation-sub {
  font-size: 0.95rem;
  color: #94a3b8;
}

[${OVERLAY_ATTR}] .spatial-activation-pulse {
  animation: spatial-pulse 2s ease-in-out infinite;
}

@keyframes spatial-pulse {
  0%, 100% { opacity: 0.5; }
  50% { opacity: 1; }
}
`.trim();

// ---------------------------------------------------------------------------
// Gamepad SVG icon (simple controller outline)
// ---------------------------------------------------------------------------

const GAMEPAD_SVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="6" width="20" height="12" rx="4"/><circle cx="8" cy="12" r="1.5"/><circle cx="16" cy="12" r="1.5"/><line x1="11" y1="10" x2="13" y2="10"/><line x1="11" y1="14" x2="13" y2="14"/></svg>`;

// ---------------------------------------------------------------------------
// Implementation
// ---------------------------------------------------------------------------

export interface ActivationOverlayHandle {
  /** Remove the overlay immediately. */
  dismiss(): void;
  /** The overlay DOM element, or `null` if the overlay was skipped. */
  element: HTMLElement | null;
  /** Whether the overlay was created (false only when `when: false`). */
  shown: boolean;
}

/**
 * Show the activation overlay.  Returns a handle to dismiss it manually.
 *
 * The overlay auto-dismisses on:
 * - `gamepadconnected` window event
 * - `mousemove`
 * - `touchstart` (anywhere on the page)
 * - `keydown`
 * - tap/click on the overlay itself
 *
 * After dismissal, `onDismiss` is called with the reason.
 */
export function showActivationOverlay(
  options?: ActivationOverlayOptions & {
    onDismiss?: (reason: DismissReason) => void;
  },
): ActivationOverlayHandle {
  const when = options?.when ?? 'auto';
  if (when === false) {
    return { dismiss: () => {}, element: null, shown: false };
  }

  const autoMode = when === 'auto';
  const message = options?.message ?? 'Press any button on your controller';
  const subMessage = options?.subMessage ?? 'Tap anywhere to dismiss';

  // Inject styles
  let styleEl = document.querySelector(`style[${OVERLAY_ATTR}-styles]`);
  if (!styleEl) {
    styleEl = document.createElement('style');
    styleEl.setAttribute(`${OVERLAY_ATTR}-styles`, '');
    styleEl.textContent = OVERLAY_CSS;
    document.head.appendChild(styleEl);
  }

  // Create overlay
  const overlay = document.createElement('div');
  overlay.setAttribute(OVERLAY_ATTR, '');

  // In auto mode, start invisible and fade in after delay.
  // In forced mode (when: true), show immediately.
  if (autoMode) {
    overlay.setAttribute('data-auto', '');
  } else {
    overlay.setAttribute('data-visible', '');
  }

  overlay.innerHTML = `
    <div class="spatial-activation-icon spatial-activation-pulse">${GAMEPAD_SVG}</div>
    <div class="spatial-activation-msg">${message}</div>
    <div class="spatial-activation-sub">${subMessage}</div>
  `;
  document.body.appendChild(overlay);

  let dismissed = false;
  let fadeInTimer: ReturnType<typeof setTimeout> | null = null;

  // In auto mode, fade in after delay. If dismissed before then, user never sees it.
  if (autoMode) {
    fadeInTimer = setTimeout(() => {
      fadeInTimer = null;
      if (!dismissed) {
        overlay.removeAttribute('data-auto');
        overlay.setAttribute('data-visible', '');
      }
    }, AUTO_FADE_IN_DELAY_MS);
  }

  function dismiss(reason: DismissReason): void {
    if (dismissed) return;
    dismissed = true;

    if (fadeInTimer !== null) {
      clearTimeout(fadeInTimer);
      fadeInTimer = null;
    }

    // Animate out (or just remove instantly if never became visible)
    overlay.setAttribute('data-dismissing', '');
    overlay.removeAttribute('data-visible');
    overlay.removeAttribute('data-auto');

    // Clean up after transition
    const cleanup = (): void => {
      overlay.remove();
      styleEl?.remove();
      window.removeEventListener('gamepadconnected', onGamepad);
      window.removeEventListener('mousemove', onMouse);
      window.removeEventListener('keydown', onKeyboard);
      window.removeEventListener('touchstart', onTouch);
      overlay.removeEventListener('click', onClick);
      options?.onDismiss?.(reason);
    };

    // Wait for fade-out transition, with fallback timeout
    overlay.addEventListener('transitionend', cleanup, { once: true });
    setTimeout(cleanup, 400);
  }

  const onGamepad = (): void => dismiss('gamepad');
  const onMouse = (): void => dismiss('mouse');
  const onKeyboard = (): void => dismiss('keyboard');
  const onTouch = (): void => dismiss('touch');
  const onClick = (): void => dismiss('touch');

  window.addEventListener('gamepadconnected', onGamepad);
  window.addEventListener('mousemove', onMouse);
  window.addEventListener('keydown', onKeyboard);
  window.addEventListener('touchstart', onTouch);
  overlay.addEventListener('click', onClick);

  return {
    dismiss: () => dismiss('gamepad'),
    element: overlay,
    shown: true,
  };
}
