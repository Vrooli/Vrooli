import { create } from "zustand";

// Transient media ownership; never persisted as a claim about a future session.
export const useAgentVoiceStore = create<{
  connected: Record<string, boolean>;
  setConnected: (sessionId: string, connected: boolean) => void;
}>()((set) => ({
  connected: {},
  setConnected: (sessionId, connected) => { set((state) => {
    if (connected) return { connected: { ...state.connected, [sessionId]: true } };
    const { [sessionId]: _released, ...connectedSessions } = state.connected;
    return { connected: connectedSessions };
  }); },
}));
