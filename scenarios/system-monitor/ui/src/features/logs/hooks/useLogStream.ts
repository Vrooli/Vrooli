import { useState } from 'react';
import { usePolling } from '../../../shared/hooks/usePolling';

/**
 * useLogStream — drives 10s background refreshes for the logs page,
 * automatically pausing when the user scrolls away from the top of the
 * table (i.e. they're inspecting historical results, not chasing tail).
 *
 * Returns:
 *   - paused / setPaused   — for an explicit pause toggle in RefreshControl
 *   - atTop / setAtTop     — wired by LogTable via its scroll handler
 */
export interface UseLogStreamOptions {
  intervalMs?: number;
  enabled?: boolean;
  onTick: () => void | Promise<void>;
  initialPaused?: boolean;
  initialAtTop?: boolean;
}

export interface UseLogStreamResult {
  paused: boolean;
  setPaused: (v: boolean) => void;
  atTop: boolean;
  setAtTop: (v: boolean) => void;
}

export function useLogStream({
  intervalMs = 10_000,
  enabled = true,
  onTick,
  initialPaused = false,
  initialAtTop = true,
}: UseLogStreamOptions): UseLogStreamResult {
  const [paused, setPaused] = useState(initialPaused);
  const [atTop, setAtTop] = useState(initialAtTop);

  usePolling(onTick, intervalMs, enabled && !paused && atTop);

  return { paused, setPaused, atTop, setAtTop };
}
