import { create } from "zustand";
import {
  startDesktopSession,
  stopDesktopSession,
  heartbeatSession,
  getDesktopSession,
  executeDesktopControl,
  type DesktopSession,
  type DesktopSessionConfig,
  type ConnectionStatus,
  type ControlResult,
} from "../lib/api/livedesktop";
import { useCapturesStore } from "./capturesStore";

interface LiveDesktopState {
  activeSession: DesktopSession | null;
  connectionStatus: ConnectionStatus;
  error: string | null;
  isOpen: boolean;
  scenarioName: string | null;
  appPath: string | null;
}

interface LiveDesktopActions {
  open: (scenarioName: string, appPath?: string) => void;
  close: () => void;
  startSession: (config: DesktopSessionConfig) => Promise<void>;
  stopSession: () => Promise<void>;
  setConnectionStatus: (status: ConnectionStatus) => void;
  setError: (error: string | null) => void;
  executeControl: (action: string, params?: Record<string, unknown>) => Promise<ControlResult>;
  refreshSession: () => Promise<void>;
}

export type LiveDesktopStore = LiveDesktopState & LiveDesktopActions;

let heartbeatInterval: ReturnType<typeof setInterval> | null = null;

function clearHeartbeat() {
  if (heartbeatInterval) {
    clearInterval(heartbeatInterval);
    heartbeatInterval = null;
  }
}

function startHeartbeat(sessionId: string) {
  clearHeartbeat();
  heartbeatInterval = setInterval(() => {
    heartbeatSession(sessionId).catch(() => {
      // Heartbeat failure is non-fatal
    });
  }, 30_000);
}

export const useLiveDesktopStore = create<LiveDesktopStore>((set, get) => ({
  activeSession: null,
  connectionStatus: "disconnected",
  error: null,
  isOpen: false,
  scenarioName: null,
  appPath: null,

  open: (scenarioName, appPath) => {
    set({
      isOpen: true,
      scenarioName,
      appPath: appPath ?? null,
      error: null,
      connectionStatus: "disconnected",
    });
  },

  close: () => {
    const { activeSession, scenarioName } = get();
    if (activeSession) {
      // Stop session in background
      stopDesktopSession(activeSession.id).catch(() => {});
      clearHeartbeat();
    }
    // Refresh captures summary so CapturesSection picks up new captures
    if (scenarioName) {
      useCapturesStore.getState().fetchSummary(scenarioName);
    }
    set({
      isOpen: false,
      activeSession: null,
      connectionStatus: "disconnected",
      error: null,
      scenarioName: null,
      appPath: null,
    });
  },

  startSession: async (config) => {
    // Clean up any existing session before starting a new one
    const { activeSession: oldSession } = get();
    if (oldSession) {
      clearHeartbeat();
      stopDesktopSession(oldSession.id).catch(() => {});
    }
    set({ activeSession: null, connectionStatus: "connecting", error: null });
    try {
      const session = await startDesktopSession(config);
      if (session.state === "error") {
        set({
          activeSession: null,
          connectionStatus: "error",
          error: session.error || "Session failed to start",
        });
        return;
      }
      set({ activeSession: session, connectionStatus: "connecting" });
      startHeartbeat(session.id);
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Failed to start desktop session";
      set({ connectionStatus: "error", error: msg });
    }
  },

  stopSession: async () => {
    const { activeSession } = get();
    if (!activeSession) return;
    clearHeartbeat();
    try {
      await stopDesktopSession(activeSession.id);
    } catch {
      // Best effort
    }
    set({
      activeSession: null,
      connectionStatus: "disconnected",
      error: null,
    });
  },

  setConnectionStatus: (status) => set({ connectionStatus: status }),

  setError: (error) => set({ error, connectionStatus: error ? "error" : get().connectionStatus }),

  executeControl: async (action, params) => {
    const { activeSession, scenarioName } = get();
    if (!activeSession) throw new Error("No active session");
    const result = await executeDesktopControl(activeSession.id, { action, params });
    // Refresh session state after control action
    try {
      const updated = await getDesktopSession(activeSession.id);
      set({ activeSession: updated });
    } catch {
      // Best effort refresh
    }
    // Refresh captures summary when a new capture was persisted
    if (result.data?.capture_id && scenarioName) {
      useCapturesStore.getState().fetchSummary(scenarioName);
    }
    return result;
  },

  refreshSession: async () => {
    const { activeSession } = get();
    if (!activeSession) return;
    try {
      const updated = await getDesktopSession(activeSession.id);
      set({ activeSession: updated });
    } catch {
      // Best effort
    }
  },
}));
