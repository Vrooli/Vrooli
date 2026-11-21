import type { CDPSession } from 'rebrowser-playwright';

/** A viewport is phone-class when its shorter edge is a phone width. */
export function isMobileViewport(viewport?: { width: number; height: number }): boolean {
  return !!viewport && Math.min(viewport.width, viewport.height) <= 480;
}

/**
 * Turn touch emulation on for the page a CDP session is attached to, which is
 * what makes `pointer: coarse`, `hover: none` and a nonzero maxTouchPoints
 * true. Chromium drops it when another session is created on the page or the
 * document commits or finishes loading, so every owner of a session on a
 * phone-class page asserts it again: a capture that reports a mouse measures
 * every phone tap target as too small. The command is idempotent.
 */
export async function assertPhoneTouch(cdp: CDPSession): Promise<void> {
  await (cdp as unknown as { send: (method: string, params: unknown) => Promise<unknown> }).send(
    'Emulation.setTouchEmulationEnabled',
    { enabled: true, maxTouchPoints: 1 }
  );
}
