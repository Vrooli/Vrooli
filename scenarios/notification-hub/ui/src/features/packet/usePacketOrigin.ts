import { useState } from "react";
import { useLocation } from "react-router-dom";
import type { Ask } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";

import type { NavKey } from "../../layout/navSection";
import { isAwaiting } from "../asks/askView";
import { preserveRoutedTestMode } from "../routedTestMode";

export interface PacketOrigin {
  /** The list "back" returns to, with its search when the operator came from it. */
  returnTo: string;
  /** The destination the navigation shows as current while this decision is open, or null while it is not yet known. */
  section: Extract<NavKey, "asks" | "history"> | null;
}

/**
 * Where a decision sits in the Hub, from which "back" and the highlighted
 * destination follow, so the two always agree.
 *
 * 1. When the operator opened it from a list, that list: `location.state.from`
 *    (the contract with Needs you and History; anything else is ignored).
 * 2. Otherwise (a push link, a bookmark) the decision's own state: waiting is
 *    Needs you, closed is History.
 *
 * The decision's state is read once, when the page opens. Answering it must
 * not move the operator from Needs you to History while they read the outcome.
 */
export function usePacketOrigin(ask?: Pick<Ask, "state">): PacketOrigin {
  const location = useLocation();
  const [closedOnOpen] = useState(() => (ask ? !isAwaiting(ask) : null));
  const from = (location.state as { from?: unknown } | null)?.from;
  if (typeof from === "string" && /^\/(?:asks|history)(?:\?|$)/.test(from)) {
    return { returnTo: from, section: from.startsWith("/history") ? "history" : "asks" };
  }
  if (closedOnOpen) return { returnTo: preserveRoutedTestMode("/history", location.search), section: "history" };
  return { returnTo: preserveRoutedTestMode("/asks", location.search), section: closedOnOpen === false ? "asks" : null };
}
