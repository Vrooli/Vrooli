import { useMediaQuery } from "../../hooks/useMediaQuery";

/**
 * Wide enough to show the list of settings beside the open one: the sidebar,
 * the list and a readable form fit side by side. Below this the list is a page
 * of its own and each setting opens over it.
 */
export const WIDE_SETTINGS_QUERY = "(min-width: 68rem)";

export function useWideSettings(): boolean {
  return useMediaQuery(WIDE_SETTINGS_QUERY);
}
