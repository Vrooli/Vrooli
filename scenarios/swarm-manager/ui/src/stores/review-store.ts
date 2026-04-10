/**
 * Review Evidence Store
 *
 * Zustand store for managing the "Request More Evidence" panel state.
 * Actual review round data lives in React Query cache — this store only
 * tracks UI-level panel open/close, active thread context, and
 * loading states for the panel.
 */

import { create } from "zustand";
import type { RequestThread } from "../services/review-service";

export interface ReviewRequestTarget {
  round: number;
  evidenceId?: string;
}

interface ReviewStoreState {
  /** Whether the Request More panel is open. */
  requestPanelOpen: boolean;
  /** The round and optional evidence item the request targets. */
  requestTarget: ReviewRequestTarget | null;
  /** Active request thread ID (for continuing a conversation). */
  activeThreadId: string | null;
  /** The loaded thread data. */
  activeThread: RequestThread | null;
  /** True while creating the initial request. */
  isCreating: boolean;
  /** True while sending a follow-up message. */
  isSending: boolean;

  /** Open the Request More panel for a specific round/evidence. */
  openRequestPanel: (round: number, evidenceId?: string) => void;
  /** Close the Request More panel and reset state. */
  closeRequestPanel: () => void;
  /** Set the active thread ID (after creating or selecting a thread). */
  setActiveThread: (thread: RequestThread | null) => void;
  /** Set creating state. */
  setCreating: (creating: boolean) => void;
  /** Set sending state. */
  setSending: (sending: boolean) => void;
}

export const useReviewStore = create<ReviewStoreState>((set) => ({
  requestPanelOpen: false,
  requestTarget: null,
  activeThreadId: null,
  activeThread: null,
  isCreating: false,
  isSending: false,

  openRequestPanel: (round, evidenceId) =>
    set({
      requestPanelOpen: true,
      requestTarget: { round, evidenceId },
      activeThreadId: null,
      activeThread: null,
      isCreating: false,
      isSending: false,
    }),

  closeRequestPanel: () =>
    set({
      requestPanelOpen: false,
      requestTarget: null,
      activeThreadId: null,
      activeThread: null,
      isCreating: false,
      isSending: false,
    }),

  setActiveThread: (thread) =>
    set({
      activeThread: thread,
      activeThreadId: thread?.id ?? null,
    }),

  setCreating: (creating) => set({ isCreating: creating }),
  setSending: (sending) => set({ isSending: sending }),
}));
