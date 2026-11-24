export type BridgeShortcutOutcome = 'handled' | 'unhandled' | 'noop';

export interface BridgeShortcutIntent {
  action: string;
  outcome?: BridgeShortcutOutcome;
  chord?: string;
  source?: 'keyboard' | 'programmatic';
  requestId?: string;
  ts?: number;
  detail?: Record<string, unknown> | null;
}

export const HOST_SHORTCUT_ACTION_OPEN_GLOBAL_SWITCHER = 'host.open-global-switcher';

/** Relay a keyboard or programmatic shortcut to the host without loading the full child bridge. */
export function emitShortcutIntent(
  intent: BridgeShortcutIntent,
  options?: { parentOrigin?: string },
): boolean {
  if (typeof window === 'undefined' || window.parent === window) {
    return false;
  }
  const action = typeof intent.action === 'string' ? intent.action.trim() : '';
  if (!action) {
    return false;
  }

  const payload = {
    v: 1 as const,
    t: 'SHORTCUT' as const,
    intent: {
      action,
      outcome: intent.outcome ?? 'unhandled',
      chord: intent.chord ?? undefined,
      source: intent.source ?? 'keyboard',
      requestId: intent.requestId,
      ts: typeof intent.ts === 'number' ? intent.ts : Date.now(),
      detail: intent.detail ?? null,
    },
  };

  try {
    window.parent.postMessage(payload, options?.parentOrigin ?? '*');
    return true;
  } catch (error) {
    console.warn('[BridgeChild] Shortcut intent postMessage failed', error);
    return false;
  }
}
