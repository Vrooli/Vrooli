import { create } from "zustand";
import {
  startDesktopSession,
  stopDesktopSession,
  heartbeatSession,
  type DesktopSession,
  type DesktopSessionConfig,
  type ConnectionStatus,
} from "../lib/api/livedesktop";

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
    const { activeSession } = get();
    if (activeSession) {
      // Stop session in background
      stopDesktopSession(activeSession.id).catch(() => {});
      clearHeartbeat();
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
}));
