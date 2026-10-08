export type SessionEventType =
  | "session.created"
  | "session.state_changed"
  | "session.error"
  | "session.destroyed";

export type SessionStatus =
  | "disconnected"
  | "connecting"
  | "connected"
  | "reconnecting"
  | "failed";

export interface SessionEventPayload {
  sessionId: string;
  status: SessionStatus;
  createdAt: string;
  backend: string;
  resolution: { width: number; height: number };
  error?: { code?: string; message: string };
}

export interface SessionEvent {
  type: SessionEventType;
  payload: SessionEventPayload;
}

interface BridgeMessage {
  v: 1;
  t: "SESSION";
  event: SessionEvent;
}

function resolveParentOrigin(): string {
  const fromEnv: unknown = window.__VROOLI_CONFIG__?.parentOrigin;
  if (typeof fromEnv === "string" && fromEnv.length > 0) {
    return fromEnv;
  }
  if (typeof document !== "undefined" && document.referrer) {
    try {
      return new URL(document.referrer).origin;
    } catch {
      // fallthrough
    }
  }
  return "*";
}

export function postSessionEvent(event: SessionEvent): boolean {
  if (typeof window === "undefined" || window.parent === window) {
    return false;
  }

  const message: BridgeMessage = {
    v: 1,
    t: "SESSION",
    event,
  };

  try {
    window.parent.postMessage(message, resolveParentOrigin());
    return true;
  } catch (error) {
    console.warn("[emulator/bridge] postSessionEvent failed", error);
    return false;
  }
}
