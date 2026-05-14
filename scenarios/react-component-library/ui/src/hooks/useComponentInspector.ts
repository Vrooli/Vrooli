import { useCallback, useEffect, useRef, useState } from "react";

/**
 * Minimal element-inspector state machine that consumes the
 * @vrooli/iframe-bridge INSPECT wire protocol from the preview
 * harness. The host posts `{v:1,t:"INSPECT",cmd:"START"|"STOP"}` to the
 * iframe; the harness replies with INSPECT_STATE / INSPECT_HOVER /
 * INSPECT_RESULT messages whose payload shape matches the bridge's
 * BridgeInspectHoverPayload / BridgeInspectResultPayload types.
 *
 * We re-derive a slim hook here rather than depending on app-monitor's
 * 1k-LOC `useIframeBridge` — RCL only needs the inspect channel.
 */

export interface InspectRect {
  x: number;
  y: number;
  width: number;
  height: number;
}

export interface InspectMeta {
  tag: string;
  id: string;
  classes: string[];
  selector: string;
  label: string;
  ariaLabel: string;
  ariaDescription: string;
  title: string;
  role: string;
  text: string;
}

export interface InspectAncestor {
  depth: number;
  tag: string;
  selector: string;
  id: string;
  classes: string[];
  rect: InspectRect | null;
  documentRect: InspectRect | null;
}

export interface InspectPayload {
  meta: InspectMeta;
  rect: InspectRect | null;
  documentRect: InspectRect | null;
  ancestors: InspectAncestor[];
  selectedAncestorIndex: number;
  pointerType?: string;
  method?: "pointer" | "keyboard";
}

export interface InspectState {
  active: boolean;
  hover: InspectPayload | null;
  result: InspectPayload | null;
  lastReason: "start" | "stop" | "cancel" | "complete" | null;
}

const initial: InspectState = {
  active: false,
  hover: null,
  result: null,
  lastReason: null,
};

const isLastReason = (value: unknown): value is InspectState["lastReason"] =>
  value === "start" || value === "stop" || value === "cancel" || value === "complete" || value === null;

const isInspectPayload = (value: unknown): value is InspectPayload => {
  if (!value || typeof value !== "object") return false;
  const candidate = value as Partial<InspectPayload>;
  return (
    !!candidate.meta &&
    typeof candidate.meta === "object" &&
    (candidate.rect === null || typeof candidate.rect === "object") &&
    Array.isArray(candidate.ancestors) &&
    typeof candidate.selectedAncestorIndex === "number"
  );
};

export interface UseComponentInspectorReturn extends InspectState {
  startInspect: () => boolean;
  stopInspect: () => boolean;
  selected: InspectPayload | null;
}

export function useComponentInspector(
  iframeRef: React.RefObject<HTMLIFrameElement | null>,
): UseComponentInspectorReturn {
  const [state, setState] = useState<InspectState>(initial);
  const stateRef = useRef(state);
  stateRef.current = state;

  useEffect(() => {
    const handler = (ev: MessageEvent) => {
      const d = ev.data as { v?: number; t?: string; [k: string]: unknown } | null;
      if (!d || d.v !== 1 || typeof d.t !== "string") return;
      switch (d.t) {
        case "INSPECT_STATE":
          setState((s) => ({
            ...s,
            active: !!d.active,
            lastReason: isLastReason(d.reason) ? d.reason : s.lastReason,
            hover: d.active ? s.hover : null,
          }));
          return;
        case "INSPECT_HOVER":
          setState((s) => ({
            ...s,
            hover: isInspectPayload(d.payload) ? d.payload : null,
          }));
          return;
        case "INSPECT_RESULT":
          setState((s) => ({
            ...s,
            result: isInspectPayload(d.payload) ? d.payload : null,
            hover: null,
            active: false,
            lastReason: "complete",
          }));
          return;
        case "INSPECT_CANCEL":
          setState((s) => ({ ...s, active: false, hover: null, lastReason: "cancel" }));
          return;
      }
    };
    window.addEventListener("message", handler);
    return () => window.removeEventListener("message", handler);
  }, []);

  const post = useCallback(
    (payload: Record<string, unknown>): boolean => {
      const win = iframeRef.current?.contentWindow;
      if (!win) return false;
      try {
        win.postMessage(payload, "*");
        return true;
      } catch {
        return false;
      }
    },
    [iframeRef],
  );

  const startInspect = useCallback((): boolean => {
    return post({ v: 1, t: "INSPECT", cmd: "START" });
  }, [post]);

  const stopInspect = useCallback((): boolean => {
    if (!stateRef.current.active) return true;
    return post({ v: 1, t: "INSPECT", cmd: "STOP" });
  }, [post]);

  return {
    ...state,
    startInspect,
    stopInspect,
    selected: state.hover ?? state.result,
  };
}
