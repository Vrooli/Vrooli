import { useEffect } from "react";
import { useQuery } from "@tanstack/react-query";
import { listApprovals, type ApprovalDecision } from "../api/approvals";
import { useApprovalsStore } from "../stores/useApprovalsStore";

/**
 * Poll the operator's approvals and keep the store's pendingCount in sync so
 * the badge and mobile banner (which subscribe to that one primitive) update
 * without this hook being mounted everywhere.
 *
 * `openOnly=false` fetches the resolved/history view; the default fetches the
 * pending queue. The badge count is always derived from the pending set, so a
 * light always-on poll (enabled by default, 30s) drives it even while the
 * drawer is closed.
 *
 * Mirrors useCapabilities (30s refetch, enabled gate).
 */
export function useApprovals(options?: { openOnly?: boolean; enabled?: boolean }) {
  const openOnly = options?.openOnly ?? true;
  const enabled = options?.enabled ?? true;
  const setPendingCount = useApprovalsStore((s) => s.setPendingCount);

  const query = useQuery<ApprovalDecision[]>({
    queryKey: ["approvals", openOnly],
    queryFn: () => listApprovals(openOnly, 100),
    refetchInterval: 30_000,
    enabled,
  });

  // Keep the badge/banner count honest from whichever fetch is the pending one.
  useEffect(() => {
    if (openOnly && query.data) {
      setPendingCount(query.data.filter((d) => d.state === "pending").length);
    }
  }, [openOnly, query.data, setPendingCount]);

  return query;
}

/**
 * A tiny always-on count poller for the badge/banner, independent of the
 * drawer. Kept separate so mounting it is cheap: it fetches the open set every
 * 60s and writes only the pending count into the store.
 */
export function useApprovalsBadgePoll(enabled = true) {
  const setPendingCount = useApprovalsStore((s) => s.setPendingCount);
  const query = useQuery<ApprovalDecision[]>({
    queryKey: ["approvals-badge"],
    queryFn: () => listApprovals(true, 100),
    refetchInterval: 60_000,
    enabled,
    // A transient hub outage should not clear the badge to zero mid-session.
    staleTime: 55_000,
  });
  useEffect(() => {
    if (query.data) setPendingCount(query.data.filter((d) => d.state === "pending").length);
  }, [query.data, setPendingCount]);
  return query;
}
