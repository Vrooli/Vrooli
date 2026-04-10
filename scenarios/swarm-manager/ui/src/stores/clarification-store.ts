/**
 * Zustand store for clarification panel state.
 *
 * Decouples the ClarifyButton (deep in WorkshopItemCard) from the
 * ClarificationPanel (mounted at page level via FloatingPanel portal).
 */

import { create } from "zustand";
import type { BacklogKind, ClarificationThread, WorkshopItem } from "../types/domain";

interface ClarificationTarget {
  backlogKind: BacklogKind;
  backlogName: string;
  roundNumber: number;
  itemId: string;
  itemTopic: string;
  /** Existing thread ID — when set, the panel fetches the thread on open. */
  clarificationId?: string;
  /** Full current item — used for update-decision preview. */
  currentItem?: WorkshopItem;
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

  /** True while fetching an existing thread on panel reopen. */
  isLoading: boolean;

  /** Open the panel for a specific decision item. */
  open: (target: ClarificationTarget) => void;

  /** Close the panel and reset all state. */
  close: () => void;

  /** Update the thread (e.g. after polling or continuing). */
  setThread: (thread: ClarificationThread) => void;

  /** Toggle the creating spinner. */
  setCreating: (creating: boolean) => void;

  /** Toggle the loading state. */
  setLoading: (loading: boolean) => void;
}

export const useClarificationStore = create<ClarificationStoreState>((set) => ({
  isOpen: false,
  target: null,
  thread: null,
  isCreating: false,
  isLoading: false,

  open: (target) =>
    set({ isOpen: true, target, thread: null, isCreating: false, isLoading: !!target.clarificationId }),

  close: () =>
    set({ isOpen: false, target: null, thread: null, isCreating: false, isLoading: false }),

  setThread: (thread) => set({ thread }),

  setCreating: (creating) => set({ isCreating: creating }),

  setLoading: (loading) => set({ isLoading: loading }),
}));
