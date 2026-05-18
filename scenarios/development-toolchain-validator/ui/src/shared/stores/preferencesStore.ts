import { create } from "zustand";
import { persist } from "zustand/middleware";

/**
 * App-wide UI preferences.
 *
 * Scope check (react-coherence §1): these settings are shared across every
 * surface (theme, density, sidebar collapsed, last visited golden), they
 * persist across reloads, and they're not server state — so an app-wide
 * store is the right primitive. Per-surface UI state stays as local
 * `useState`; server state lives in React Query.
 *
 * Persistence: zustand's `persist` middleware writes to `localStorage`
 * under `dtv.preferences`. The `partialize` filter keeps the schema
 * forward-compatible — non-listed keys are dropped on rehydrate.
 */
export type Theme = "dark" | "light";
export type Density = "comfortable" | "compact";

export interface PreferencesState {
  theme: Theme;
  density: Density;
  sidebarCollapsed: boolean;
  lastVisitedGoldenSlug: string | null;
  setTheme: (theme: Theme) => void;
  setDensity: (density: Density) => void;
  setSidebarCollapsed: (collapsed: boolean) => void;
  setLastVisitedGoldenSlug: (slug: string | null) => void;
  toggleSidebar: () => void;
}

const STORAGE_KEY = "dtv.preferences";

export const usePreferencesStore = create<PreferencesState>()(
  persist(
    (set) => ({
      theme: "dark",
      density: "comfortable",
      sidebarCollapsed: false,
      lastVisitedGoldenSlug: null,
      setTheme: (theme) => set({ theme }),
      setDensity: (density) => set({ density }),
      setSidebarCollapsed: (sidebarCollapsed) => set({ sidebarCollapsed }),
      setLastVisitedGoldenSlug: (lastVisitedGoldenSlug) => set({ lastVisitedGoldenSlug }),
      toggleSidebar: () => set((s) => ({ sidebarCollapsed: !s.sidebarCollapsed })),
    }),
    {
      name: STORAGE_KEY,
      partialize: (s) => ({
        theme: s.theme,
        density: s.density,
        sidebarCollapsed: s.sidebarCollapsed,
        lastVisitedGoldenSlug: s.lastVisitedGoldenSlug,
      }),
    },
  ),
);

/**
 * Mirror the active theme + density onto `<html>` so design-tokens.css and
 * shared/theme/tokens.css can branch on `[data-resolved-theme]` and
 * `[data-density]`. Called from main.tsx after rehydrate.
 */
export function applyPreferencesToDocument(state: PreferencesState): void {
  if (typeof document === "undefined") return;
  const root = document.documentElement;
  root.setAttribute("data-resolved-theme", state.theme);
  root.setAttribute("data-theme", state.theme);
  root.setAttribute("data-density", state.density);
}
