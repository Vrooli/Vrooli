// Profile Activity Table - sortable table with profile metrics

import { useState } from "react";
import { Link } from "react-router-dom";
import { useProfileBreakdown } from "../../hooks/useProfileBreakdown";
import {
  formatPercent,
  formatNumber,
  formatTokens,
} from "../../utils/formatters";
import { ArrowUpDown, ArrowUp, ArrowDown } from "lucide-react";
import { formatUsdFixed } from "../../../../lib/currency";
import { MeasureFrame } from "../measure/MeasureFrame";
import { useMeasureDefinitions } from "../../hooks/useMeasureDefinitions";

type SortField = "profileName" | "runCount" | "successRate" | "totalCostUsd" | "totalTokens";
type SortDirection = "asc" | "desc";

export function ProfileActivityTable() {
  const { data, isLoading, error } = useProfileBreakdown();
  const definitions = useMeasureDefinitions();
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
        aVal = a[sortField] ?? 0;
        bVal = b[sortField] ?? 0;
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
    <MeasureFrame label="Profile activity" result={data?.measure} definition={definitions.data?.find((item) => item.id === "throughput.profile_breakdown")} loading={isLoading} error={error?.message}>
    <div className="rounded-lg border border-border bg-card/50 p-4 sm:p-6 overflow-hidden">
      <h3 className="mb-2 sm:mb-4 text-sm font-semibold text-muted-foreground">
        Profile Activity
      </h3>
      {profiles.length === 0 ? (
        <p className="text-sm text-muted-foreground">No profile data available</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full min-w-[360px] text-sm">
            <thead>
              <tr className="border-b border-border text-left">
                <th className="pb-2 pr-4">
                  <button
                    onClick={() => handleSort("profileName")}
                    className="flex items-center gap-1 text-xs font-semibold text-muted-foreground hover:text-foreground"
                  >
                    Profile {getSortIcon("profileName")}
                  </button>
                </th>
                <th className="pb-2 pr-4">
                  <button
                    onClick={() => handleSort("runCount")}
                    className="flex items-center gap-1 text-xs font-semibold text-muted-foreground hover:text-foreground"
                  >
                    Runs {getSortIcon("runCount")}
                  </button>
                </th>
                <th className="pb-2 pr-4">
                  <button
                    onClick={() => handleSort("successRate")}
                    className="flex items-center gap-1 text-xs font-semibold text-muted-foreground hover:text-foreground"
                  >
                    Success {getSortIcon("successRate")}
                  </button>
                </th>
                <th className="pb-2">
                  <button
                    onClick={() => handleSort("totalCostUsd")}
                    className="flex items-center gap-1 text-xs font-semibold text-muted-foreground hover:text-foreground"
                  >
                    Cost {getSortIcon("totalCostUsd")}
                  </button>
                </th>
                <th className="pb-2">
                  <button onClick={() => handleSort("totalTokens")} className="flex items-center gap-1 text-xs font-semibold text-muted-foreground hover:text-foreground">
                    Tokens {getSortIcon("totalTokens")}
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
                    className="border-b border-border last:border-0"
                  >
                    <td className="py-2 pr-4 font-medium">
                      <Link
                        to={`/profiles?profileId=${encodeURIComponent(profile.profileId)}`}
                        className="truncate max-w-[150px] inline-block hover:underline"
                        title={profile.profileName}
                      >
                        {profile.profileName}
                      </Link>
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
                      {formatUsdFixed(profile.totalCostUsd, 2)}
                    </td>
                    <td className="py-2 tabular-nums">{formatTokens(profile.totalTokens ?? 0)}</td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
    </MeasureFrame>
  );
}
