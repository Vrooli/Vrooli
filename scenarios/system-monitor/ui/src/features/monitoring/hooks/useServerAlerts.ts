import { useCallback, useEffect, useRef, useState } from 'react';
import { usePolling } from '../../../shared/hooks/usePolling';
import type { Alert } from '../../../types';
import { fetchServerAlerts, toAlertFeed, type AlertFeed } from '../serverAlerts';

/** The evaluator runs on the threshold-check interval (20 s by default). */
export const SERVER_ALERT_POLL_MS = 15_000;

export interface ServerAlertsState {
  alerts: Alert[];
  feed: AlertFeed;
}

const initialState: ServerAlertsState = { alerts: [], feed: { status: 'loading', ruleIssues: [] } };

/**
 * Polls the server-owned alert state. A failed read keeps the last alerts
 * on screen but marks the feed as errored, so a lost API is never rendered
 * as "no active alerts".
 */
export function useServerAlerts(enabled: boolean): ServerAlertsState {
  const [state, setState] = useState<ServerAlertsState>(initialState);
  const inFlight = useRef<AbortController | null>(null);

  const load = useCallback(async () => {
    inFlight.current?.abort();
    const controller = new AbortController();
    inFlight.current = controller;
    try {
      const response = await fetchServerAlerts(controller.signal);
      if (!controller.signal.aborted) {
        setState(toAlertFeed(response));
      }
    } catch (error) {
      if (controller.signal.aborted) return;
      const message = error instanceof Error ? error.message : 'Alert state is unavailable';
      setState(previous => ({ alerts: previous.alerts, feed: { ...previous.feed, status: 'error', error: message } }));
      throw error;
    }
  }, []);

  useEffect(() => {
    if (!enabled) return undefined;
    void load().catch(() => undefined);
    return () => inFlight.current?.abort();
  }, [enabled, load]);

  usePolling(load, SERVER_ALERT_POLL_MS, enabled, { enabled: true, maxIntervalMs: 120_000 });

  return state;
}
