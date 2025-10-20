import { create } from 'zustand';

type OverlayView = 'tabs' | 'actions' | null;

interface ShellOverlayState {
  activeView: OverlayView;
  openView: (view: Exclude<OverlayView, null>) => void;
  closeView: () => void;
}

export const useShellOverlayStore = create<ShellOverlayState>((set) => ({
  activeView: null,
  openView: (view) => set({ activeView: view }),
  closeView: () => set({ activeView: null }),
}));
