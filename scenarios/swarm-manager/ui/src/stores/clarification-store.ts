/**
 * Zustand store for clarification panel state.
 *
 * Decouples the ClarifyButton (deep in WorkshopItemCard) from the
 * ClarificationPanel (mounted at page level via FloatingPanel portal).
 */

import { create } from "zustand";
import type { BacklogKind, ClarificationThread } from "../types/domain";

interface ClarificationTarget {
  backlogKind: BacklogKind;
  backlogName: string;
  roundNumber: number;
  itemId: string;
  itemTopic: string;
}

interface ClarificationStoreState {
  /** Whether the clarification panel is visible. */
  isOpen: boolean;

  /** Context for the active clarification. Null when closed. */
  target: ClarificationTarget | null;

  /** Active clarification thread (null before first submission). */
  thread: ClarificationThread | null;

  /** True while the initial create-clarification request is in-flight. */
  isCreating: boolean;

  /** Open the panel for a specific decision item. */
  open: (target: ClarificationTarget) => void;

  /** Close the panel and reset all state. */
  close: () => void;

  /** Update the thread (e.g. after polling or continuing). */
  setThread: (thread: ClarificationThread) => void;

  /** Toggle the creating spinner. */
  setCreating: (creating: boolean) => void;
}

export const useClarificationStore = create<ClarificationStoreState>((set) => ({
  isOpen: false,
  target: null,
  thread: null,
  isCreating: false,

  open: (target) =>
    set({ isOpen: true, target, thread: null, isCreating: false }),

  close: () =>
    set({ isOpen: false, target: null, thread: null, isCreating: false }),

  setThread: (thread) => set({ thread }),

  setCreating: (creating) => set({ isCreating: creating }),
}));
