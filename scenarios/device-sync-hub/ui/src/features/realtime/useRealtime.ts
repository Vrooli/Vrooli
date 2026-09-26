import { useCallback, useEffect, useReducer, useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { buildApiUrl } from "@vrooli/api-base";

import { API_BASE } from "../../api/client";
import { ITEMS_QUERY_KEY } from "../transfer/queries";
import {
  decodeEvent,
  EventType,
  initialRealtimeState,
  realtimeReducer,
  type RealtimeState,
} from "./events";

/** Connection lifecycle surfaced to the top-bar indicator. */
export type RealtimeStatus = "connecting" | "open" | "closed";

export interface UseRealtimeResult extends RealtimeState {
  status: RealtimeStatus;
  /** Dismiss the current pairing banner without approving/rejecting. */
  dismissPairing: () => void;
}

/**
 * Subscribe to the hub's realtime SSE stream for the paired device.
 *
 * `EventSource` cannot set headers, so the device token rides the `?token=`
 * query param (the server's deviceauth middleware accepts it for this stream).
 * Item events invalidate the react-query items cache; presence and pairing
 * requests flow through the pure `realtimeReducer`. The connection is torn down
 * on unmount or when the device token changes, and the browser's built-in
 * EventSource auto-reconnect drives recovery (we mirror its state).
 */
export function useRealtime(deviceToken: string | null): UseRealtimeResult {
  const queryClient = useQueryClient();
  const [state, dispatch] = useReducer(realtimeReducer, initialRealtimeState);
  const [status, setStatus] = useState<RealtimeStatus>("connecting");
  const sourceRef = useRef<EventSource | null>(null);

  const dismissPairing = useCallback(() => dispatch({ type: "dismiss-pairing" }), []);

  useEffect(() => {
    if (!deviceToken) {
      setStatus("closed");
      return;
    }

    const url = buildApiUrl(`/realtime/events?token=${encodeURIComponent(deviceToken)}`, {
      baseUrl: API_BASE,
    });
    setStatus("connecting");
    const source = new EventSource(url);
    sourceRef.current = source;

    source.onopen = () => setStatus("open");
    source.onerror = () => {
      // EventSource auto-reconnects unless it was explicitly closed; surface the
      // gap so the indicator can show "reconnecting".
      setStatus(source.readyState === source.CLOSED ? "closed" : "connecting");
    };
    source.onmessage = (message) => {
      const event = decodeEvent(message.data as string);
      if (!event) return;
      if (event.type === EventType.ITEM_ARRIVED || event.type === EventType.ITEM_DELETED) {
        void queryClient.invalidateQueries({ queryKey: ITEMS_QUERY_KEY });
        return;
      }
      dispatch({ type: "event", event });
    };

    return () => {
      source.close();
      sourceRef.current = null;
    };
  }, [deviceToken, queryClient]);

  return { ...state, status, dismissPairing };
}
