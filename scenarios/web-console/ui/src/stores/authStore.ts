import { create } from "zustand";
import { completeWebAuthCallback, getAccessToken, signOut as signOutAuth, startSignIn } from "../lib/auth";

interface AuthState {
  accessToken: string | null;
  loading: boolean;
  error: string | null;
  initialize: () => Promise<void>;
  signIn: () => Promise<void>;
  signOut: () => Promise<void>;
}

export const useAuthStore = create<AuthState>((set) => ({
  accessToken: null,
  loading: true,
  error: null,
  initialize: async () => {
    try {
      await completeWebAuthCallback();
      const sync = async (event?: { event: string }) => {
        const accessToken = await getAccessToken();
        set({ accessToken, loading: false, error: event?.event === "session-expired" ? "Subscription session expired" : null });
      };
      if (window.desktop?.auth?.onAuthChanged) {
        window.desktop.auth.onAuthChanged((event) => { void sync(event); });
      }
      await sync();
    } catch (error) {
      set({ loading: false, error: error instanceof Error ? error.message : "Authentication failed" });
    }
  },
  signIn: async () => {
    set({ loading: true, error: null });
    try {
      await startSignIn();
      set({ loading: false });
    } catch (error) {
      set({ loading: false, error: error instanceof Error ? error.message : "Sign-in failed" });
    }
  },
  signOut: async () => {
    set({ loading: true, error: null });
    try {
      await signOutAuth();
      set({ accessToken: null, loading: false });
    } catch (error) {
      set({ loading: false, error: error instanceof Error ? error.message : "Sign-out failed" });
    }
  },
}));
