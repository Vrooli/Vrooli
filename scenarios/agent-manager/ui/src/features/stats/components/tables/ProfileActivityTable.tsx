// Profile Activity Table - sortable table with profile metrics

import { Link } from "react-router-dom";
import { useProfileBreakdown } from "../../hooks/useProfileBreakdown";
import {
  formatPercent,
  formatNumber,
  formatTokens,
} from "../../utils/formatters";
import { formatUsdFixed } from "../../../../lib/currency";
import { MeasureFrame } from "../measure/MeasureFrame";
import { compareSortValues, SortHeaderCell, useTableSort } from "./SortableHeader";
import { useMeasureDefinitions } from "../../hooks/useMeasureDefinitions";

type SortField = "profileName" | "runCount" | "successRate" | "totalCostUsd" | "totalTokens";

export function ProfileActivityTable() {
  const { data, isLoading, error } = useProfileBreakdown();
  const definitions = useMeasureDefinitions();
  const sort = useTableSort<SortField>("runCount");

  const profiles = data?.profiles ?? [];

  // Sort data
  const sortedProfiles = [...profiles].sort((a, b) => {
    let aVal: number | string;
    let bVal: number | string;

    switch (sort.field) {
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
        aVal = a[sort.field] ?? 0;
        bVal = b[sort.field] ?? 0;
    }

    return compareSortValues(aVal, bVal, sort.direction);
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
                <SortHeaderCell sort={sort} field="profileName" label="Profile" />
                <SortHeaderCell sort={sort} field="runCount" label="Runs" />
                <SortHeaderCell sort={sort} field="successRate" label="Success" />
                <SortHeaderCell sort={sort} field="totalCostUsd" label="Cost" />
                <SortHeaderCell sort={sort} field="totalTokens" label="Tokens" last />
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
                        className="inline-flex min-h-11 max-w-[150px] items-center truncate hover:underline sm:min-h-8"
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
