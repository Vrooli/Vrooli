import { create } from "zustand";

/**
 * Approvals inbox store.
 *
 * Two concerns, both intentionally tiny:
 *
 *  1. `pendingCount` — the ONE primitive the badge and the mobile banner
 *     subscribe to. A polling query (useApprovals) writes it; the
 *     FloatingToolbar badge and the Workspace banner each read only this number
 *     so a change re-renders just those, never the whole tree. This mirrors the
 *     TabUnreadBadge isolation pattern (components/TabBar.tsx).
 *
 *  2. The deep-link slice — web-console has NO router, so a push/chat link that
 *     should land on one decision is parsed at boot (parseApprovalsDeepLink
 *     below, called from App.tsx) into `pendingDeepLinkId` + `shouldOpenDrawer`.
 *     Workspace consumes it with the same deferred-intent discipline it uses for
 *     a reopened session (Workspace.tsx) — open the drawer, select the decision,
 *     then clear the intent so a later navigation doesn't reopen it.
 */
export interface ApprovalsState {
  /** Count of still-pending decisions; the badge/banner source. */
  pendingCount: number;
  setPendingCount: (count: number) => void;

  /** A decision id the inbox should open and select, from a deep link. */
  pendingDeepLinkId: string | null;
  /** True until the drawer has honored the deep link (or it was cleared). */
  shouldOpenDrawer: boolean;
  /** Record a deep-link intent (also flips shouldOpenDrawer on). */
  requestOpen: (decisionId: string | null) => void;
  /** Consume the intent once the drawer has acted on it. */
  clearDeepLinkIntent: () => void;
}

export const useApprovalsStore = create<ApprovalsState>((set) => ({
  pendingCount: 0,
  setPendingCount: (count) => { set({ pendingCount: Math.max(0, count | 0) }); },

  pendingDeepLinkId: null,
  shouldOpenDrawer: false,
  requestOpen: (decisionId) => { set({ pendingDeepLinkId: decisionId, shouldOpenDrawer: true }); },
  clearDeepLinkIntent: () => { set({ pendingDeepLinkId: null, shouldOpenDrawer: false }); },
}));

/**
 * Parse an approvals deep link out of the boot URL, WITHOUT a router.
 *
 * Supported shapes (any one triggers the inbox to open):
 *   ?approval=<askId>         canonical
 *   ?decision=<askId>         alias
 *   ?authorization=<askId>    an agent-manager authorization push lands here
 *   ?run=<askId>              an agent-manager run-review push lands here
 *   ?approvals=1 / ?approvals (open the inbox with no specific decision)
 *
 * It returns the intent without mutating anything so it is trivially testable;
 * App.tsx applies it via useApprovalsStore.requestOpen and (optionally) strips
 * the query with history.replaceState so a refresh doesn't reopen it.
 */
export interface ApprovalsDeepLink {
  open: boolean;
  decisionId: string | null;
}

export function parseApprovalsDeepLink(search: string): ApprovalsDeepLink {
  let params: URLSearchParams;
  try {
    params = new URLSearchParams(search || "");
  } catch {
    return { open: false, decisionId: null };
  }
  for (const key of ["approval", "decision", "authorization", "run"]) {
    const value = params.get(key);
    if (value && value.trim()) return { open: true, decisionId: value.trim() };
  }
  if (params.has("approvals")) {
    const flag = params.get("approvals");
    if (flag === null || flag === "" || flag === "1" || flag === "true") {
      return { open: true, decisionId: null };
    }
  }
  return { open: false, decisionId: null };
}
