import type { DisplayMode } from "../stores/useWorkspaceStore";

export function isTabLikeDisplayMode(mode: DisplayMode): boolean {
  return mode === "tabs" || mode === "sidebar";
}
