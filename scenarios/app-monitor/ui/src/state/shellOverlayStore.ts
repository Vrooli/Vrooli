import { create } from 'zustand';

type OverlayView = 'tabs' | 'actions' | null;

interface ShellOverlayState {
  activeView: OverlayView;
  overlayHost: HTMLElement | null;
  openView: (view: Exclude<OverlayView, null>) => void;
  closeView: () => void;
  registerHost: (host: HTMLElement | null) => void;
}

export const useShellOverlayStore = create<ShellOverlayState>((set) => ({
  activeView: null,
  overlayHost: null,
  openView: (view) => set({ activeView: view }),
  closeView: () => set({ activeView: null }),
  registerHost: (host) => set((state) => (
    state.overlayHost === host
      ? state
      : { overlayHost: host }
  )),
}));
