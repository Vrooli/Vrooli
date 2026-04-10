import { useMemo } from "react";
import type { RepoStatus, HealthResponse, SyncStatusResponse } from "../lib/api";

export interface HeaderState {
  isHealthy: boolean;
  ahead: number;
  behind: number;
  branchName: string;
  upstreamRef: string;
  upstreamBranch: string;
  trackingMismatch: boolean;
  cleanDetails: string;
}

export function useHeaderState(
  status?: RepoStatus,
  health?: HealthResponse,
  syncStatus?: SyncStatusResponse
): HeaderState {
  return useMemo(() => {
    const isHealthy = health?.readiness ?? false;
    const ahead = syncStatus?.ahead ?? status?.branch.ahead ?? 0;
    const behind = syncStatus?.behind ?? status?.branch.behind ?? 0;
    const branchName = syncStatus?.branch ?? status?.branch.head ?? "";
    const upstreamRef = syncStatus?.upstream ?? status?.branch.upstream ?? "";
    const upstreamBranch = upstreamRef ? upstreamRef.split("/").slice(1).join("/") : "";
    const trackingMismatch = Boolean(
      branchName && upstreamBranch && branchName !== upstreamBranch
    );
    const cleanDetails = [
      ahead > 0 ? `${ahead} ahead` : "",
      behind > 0 ? `${behind} behind` : ""
    ]
      .filter(Boolean)
      .join(", ");

    return {
      isHealthy,
      ahead,
      behind,
      branchName,
      upstreamRef,
      upstreamBranch,
      trackingMismatch,
      cleanDetails
    };
  }, [status, health, syncStatus]);
}
