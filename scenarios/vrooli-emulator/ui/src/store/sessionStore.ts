import { create } from "zustand";
import {
  createSession,
  destroySession,
  executeSessionControl,
  getSession,
  heartbeatSession,
  type ConnectionStatus,
  type ControlResult,
  type Session,
  type SessionConfig,
} from "../lib/api/sessions";
import { postSessionEvent, type SessionEventPayload, type SessionStatus } from "../lib/bridge";

export interface CaptureRef {
  type: "screenshot" | "recording";
  path?: string;
  url?: string;
  takenAt: string;
}

interface SessionState {
  activeSession: Session | null;
  connectionStatus: ConnectionStatus;
  error: string | null;
  lastCapture: CaptureRef | null;
}

interface SessionActions {
  startSession: (config: SessionConfig) => Promise<Session | null>;
  loadSession: (id: string) => Promise<Session | null>;
  stopSession: () => Promise<void>;
  setConnectionStatus: (status: ConnectionStatus) => void;
  setError: (error: string | null) => void;
  executeControl: (action: string, params?: Record<string, unknown>) => Promise<ControlResult>;
  refreshSession: () => Promise<void>;
  reset: () => void;
}

export type SessionStore = SessionState & SessionActions;

let heartbeatInterval: ReturnType<typeof setInterval> | null = null;
let metricsInterval: ReturnType<typeof setInterval> | null = null;

function clearTimers() {
  if (heartbeatInterval) {
    clearInterval(heartbeatInterval);
    heartbeatInterval = null;
  }
  if (metricsInterval) {
    clearInterval(metricsInterval);
    metricsInterval = null;
  }
}

function startHeartbeat(sessionId: string) {
  if (heartbeatInterval) clearInterval(heartbeatInterval);
  heartbeatInterval = setInterval(() => {
    heartbeatSession(sessionId).catch(() => {
      // non-fatal
    });
  }, 30_000);
}

function startMetricsPolling(sessionId: string) {
  if (metricsInterval) clearInterval(metricsInterval);
  metricsInterval = setInterval(() => {
    getSession(sessionId)
      .then((updated) => {
        useSessionStore.setState({ activeSession: updated });
      })
      .catch(() => {
        // non-fatal
      });
  }, 2_000);
}

function toPayload(
  session: Session,
  status: SessionStatus,
  error?: { code?: string; message: string },
): SessionEventPayload {
  const payload: SessionEventPayload = {
    sessionId: session.id,
    status,
    createdAt: session.created_at,
    backend: session.platform ?? "unknown",
    resolution: { width: session.width, height: session.height },
  };
  if (error) {
    payload.error = error;
  }
  return payload;
}

function mapStatus(status: ConnectionStatus): SessionStatus {
  return status;
}

export const useSessionStore = create<SessionStore>((set, get) => ({
  activeSession: null,
  connectionStatus: "disconnected",
  error: null,
  lastCapture: null,

  startSession: async (config) => {
    const { activeSession: old } = get();
    if (old) {
      clearTimers();
      destroySession(old.id).catch(() => {});
    }
    set({
      activeSession: null,
      connectionStatus: "connecting",
      error: null,
      lastCapture: null,
    });
    try {
      const session = await createSession(config);
      if (session.state === "error") {
        const message = session.error ?? "Session failed to start";
        set({ activeSession: null, connectionStatus: "failed", error: message });
        postSessionEvent({
          type: "session.error",
          payload: toPayload(session, "failed", { message }),
        });
        return null;
      }
      set({ activeSession: session, connectionStatus: "connecting" });
      startHeartbeat(session.id);
      startMetricsPolling(session.id);
      postSessionEvent({
        type: "session.created",
        payload: toPayload(session, "connecting"),
      });
      return session;
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to create session";
      set({ connectionStatus: "failed", error: message });
      postSessionEvent({
        type: "session.error",
        payload: {
          sessionId: "",
          status: "failed",
          createdAt: new Date().toISOString(),
          backend: config.platform ?? "unknown",
          resolution: { width: config.width ?? 0, height: config.height ?? 0 },
          error: { message },
        },
      });
      throw err;
    }
  },

  loadSession: async (id) => {
    const { activeSession } = get();
    if (activeSession?.id === id) {
      return activeSession;
    }
    clearTimers();
    set({
      activeSession: null,
      connectionStatus: "connecting",
      error: null,
      lastCapture: null,
    });
    try {
      const session = await getSession(id);
      set({ activeSession: session, connectionStatus: "connecting" });
      startHeartbeat(session.id);
      startMetricsPolling(session.id);
      return session;
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to load session";
      set({ connectionStatus: "failed", error: message });
      return null;
    }
  },

  stopSession: async () => {
    const { activeSession } = get();
    if (!activeSession) return;
    clearTimers();
    try {
      await destroySession(activeSession.id);
    } catch {
      // best effort — still treat session as destroyed client-side
    }
    postSessionEvent({
      type: "session.destroyed",
      payload: toPayload(activeSession, "disconnected"),
    });
    set({
      activeSession: null,
      connectionStatus: "disconnected",
      error: null,
      lastCapture: null,
    });
  },

  setConnectionStatus: (status) => {
    const prev = get().connectionStatus;
    if (prev === status) return;
    set({ connectionStatus: status });
    const { activeSession } = get();
    if (activeSession) {
      postSessionEvent({
        type: "session.state_changed",
        payload: toPayload(activeSession, mapStatus(status)),
      });
    }
  },

  setError: (error) => {
    const nextStatus: ConnectionStatus = error ? "failed" : get().connectionStatus;
    set({ error, connectionStatus: nextStatus });
    const { activeSession } = get();
    if (error && activeSession) {
      postSessionEvent({
        type: "session.error",
        payload: toPayload(activeSession, "failed", { message: error }),
      });
    }
  },

  executeControl: async (action, params) => {
    const { activeSession } = get();
    if (!activeSession) throw new Error("No active session");
    const result = await executeSessionControl(activeSession.id, { action, params });
    try {
      const updated = await getSession(activeSession.id);
      set({ activeSession: updated });
    } catch {
      // best effort
    }
    const captureId = typeof result.data?.["capture_id"] === "string" ? result.data["capture_id"] : undefined;
    if (captureId) {
      const capturePath = typeof result.data?.["path"] === "string" ? result.data["path"] : undefined;
      const captureUrl = typeof result.data?.["url"] === "string" ? result.data["url"] : undefined;
      const capture: CaptureRef = {
        type: action === "start_recording" || action === "stop_recording" ? "recording" : "screenshot",
        takenAt: new Date().toISOString(),
      };
      if (capturePath !== undefined) capture.path = capturePath;
      if (captureUrl !== undefined) capture.url = captureUrl;
      set({ lastCapture: capture });
    }
    return result;
  },

  refreshSession: async () => {
    const { activeSession } = get();
    if (!activeSession) return;
    try {
      const updated = await getSession(activeSession.id);
      set({ activeSession: updated });
    } catch {
      // best effort
    }
  },

  reset: () => {
    clearTimers();
    set({
      activeSession: null,
      connectionStatus: "disconnected",
      error: null,
      lastCapture: null,
    });
  },
}));
