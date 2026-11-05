import { create } from "zustand";

/**
 * Which context-compaction sheet is open: one session's controls, or the
 * all-sessions "compact idle sessions" list. Nothing here is persisted.
 */
interface ContextCompactionStoreState {
  open: boolean;
  /** Empty means the all-sessions list. */
  sessionId: string;
  show: (sessionId?: string) => void;
  hide: () => void;
}

export const useContextCompactionStore = create<ContextCompactionStoreState>()((set) => ({
  open: false,
  sessionId: "",
  show: (sessionId = "") => { set({ open: true, sessionId }); },
  hide: () => { set({ open: false }); },
}));
