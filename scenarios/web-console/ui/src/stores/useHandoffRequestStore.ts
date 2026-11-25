import { create } from "zustand";

export interface HandoffRequest {
  id: number;
  sourceSessionId: string;
  payload: string;
}

interface HandoffRequestState {
  request: HandoffRequest | null;
  nextId: number;
  open: (sourceSessionId: string, payload?: string) => void;
  consume: (id: number) => void;
}

export const useHandoffRequestStore = create<HandoffRequestState>()((set) => ({
  request: null,
  nextId: 1,
  open: (sourceSessionId, payload = "") => set((state) => {
    const id = state.nextId;
    return { request: { id, sourceSessionId, payload }, nextId: id + 1 };
  }),
  consume: (id) => set((state) => state.request?.id === id ? { request: null } : state),
}));
