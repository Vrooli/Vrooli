// Profile Activity Table - sortable table with profile metrics

import { useState } from "react";
import { useProfileBreakdown } from "../../hooks/useProfileBreakdown";
import {
  formatPercent,
  formatCurrency,
  formatNumber,
} from "../../utils/formatters";
import { ArrowUpDown, ArrowUp, ArrowDown } from "lucide-react";

type SortField = "profileName" | "runCount" | "successRate" | "totalCostUsd";
type SortDirection = "asc" | "desc";

export function ProfileActivityTable() {
  const { data, isLoading, error } = useProfileBreakdown();
  const [sortField, setSortField] = useState<SortField>("runCount");
  const [sortDirection, setSortDirection] = useState<SortDirection>("desc");

  const handleSort = (field: SortField) => {
    if (sortField === field) {
      setSortDirection((d) => (d === "asc" ? "desc" : "asc"));
    } else {
      setSortField(field);
      setSortDirection("desc");
    }
  };

  const getSortIcon = (field: SortField) => {
    if (sortField !== field) {
      return <ArrowUpDown className="h-3 w-3" />;
    }
    return sortDirection === "asc" ? (
      <ArrowUp className="h-3 w-3" />
    ) : (
      <ArrowDown className="h-3 w-3" />
    );
  };

  if (isLoading) {
    return (
      <div className="rounded-lg border border-border/60 bg-card/50 p-6">
        <div className="mb-4 h-5 w-36 animate-pulse rounded bg-muted/30" />
        <div className="space-y-2">
          {[1, 2, 3].map((i) => (
            <div key={i} className="h-10 animate-pulse rounded bg-muted/20" />
          ))}
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="rounded-lg border border-red-500/30 bg-red-500/5 p-6">
        <h3 className="text-sm font-semibold">Profile Activity</h3>
        <p className="mt-2 text-sm text-red-500">Failed to load: {error.message}</p>
      </div>
    );
  }

  const profiles = data?.profiles ?? [];

  // Sort data
  const sortedProfiles = [...profiles].sort((a, b) => {
    let aVal: number | string;
    let bVal: number | string;

    switch (sortField) {
      case "profileName":
        aVal = a.profileName;
        bVal = b.profileName;
        break;
      case "successRate":
        aVal = a.runCount > 0 ? a.successCount / a.runCount : 0;
        bVal = b.runCount > 0 ? b.successCount / b.runCount : 0;
        break;
      case "totalCostUsd":
        aVal = a.totalCostUsd;
        bVal = b.totalCostUsd;
        break;
      default:
        aVal = a[sortField];
        bVal = b[sortField];
    }

    if (typeof aVal === "string" && typeof bVal === "string") {
      return sortDirection === "asc"
        ? aVal.localeCompare(bVal)
        : bVal.localeCompare(aVal);
    }
    return sortDirection === "asc"
      ? (aVal as number) - (bVal as number)
      : (bVal as number) - (aVal as number);
  });

  return (
    <div className="rounded-lg border border-border/60 bg-card/50 p-6">
      <h3 className="mb-4 text-sm font-semibold uppercase tracking-wide text-muted-foreground">
        Profile Activity
      </h3>
      {profiles.length === 0 ? (
        <p className="text-sm text-muted-foreground">No profile data available</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border/60 text-left">
                <th className="pb-2 pr-4">
                  <button
                    onClick={() => handleSort("profileName")}
                    className="flex items-center gap-1 text-xs font-semibold uppercase text-muted-foreground hover:text-foreground"
                  >
                    Profile {getSortIcon("profileName")}
                  </button>
                </th>
                <th className="pb-2 pr-4">
                  <button
                    onClick={() => handleSort("runCount")}
                    className="flex items-center gap-1 text-xs font-semibold uppercase text-muted-foreground hover:text-foreground"
                  >
                    Runs {getSortIcon("runCount")}
                  </button>
                </th>
                <th className="pb-2 pr-4">
                  <button
                    onClick={() => handleSort("successRate")}
                    className="flex items-center gap-1 text-xs font-semibold uppercase text-muted-foreground hover:text-foreground"
                  >
                    Success {getSortIcon("successRate")}
                  </button>
                </th>
                <th className="pb-2">
                  <button
                    onClick={() => handleSort("totalCostUsd")}
                    className="flex items-center gap-1 text-xs font-semibold uppercase text-muted-foreground hover:text-foreground"
                  >
                    Cost {getSortIcon("totalCostUsd")}
                  </button>
                </th>
              </tr>
            </thead>
            <tbody>
              {sortedProfiles.map((profile) => {
                const successRate =
                  profile.runCount > 0 ? profile.successCount / profile.runCount : 0;
                return (
                  <tr
                    key={profile.profileId}
                    className="border-b border-border/30 last:border-0"
                  >
                    <td className="py-2 pr-4 font-medium">
                      <span className="truncate max-w-[150px] inline-block" title={profile.profileName}>
                        {profile.profileName}
                      </span>
                    </td>
                    <td className="py-2 pr-4 tabular-nums">
                      {formatNumber(profile.runCount)}
                    </td>
                    <td className="py-2 pr-4 tabular-nums">
                      <span
                        className={
                          successRate >= 0.9
                            ? "text-emerald-500"
                            : successRate >= 0.7
                            ? "text-amber-500"
                            : "text-red-500"
                        }
                      >
                        {formatPercent(successRate)}
                      </span>
                    </td>
                    <td className="py-2 tabular-nums">
                      {formatCurrency(profile.totalCostUsd)}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
