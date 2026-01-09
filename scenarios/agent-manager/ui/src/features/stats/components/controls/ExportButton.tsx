// Export button component for CSV export of stats data

import { Download } from "lucide-react";
import { useState } from "react";
import { useStatsSummary } from "../../hooks/useStatsSummary";
import { useRunnerPerformance } from "../../hooks/useRunnerPerformance";
import { useProfileBreakdown } from "../../hooks/useProfileBreakdown";
import { useModelBreakdown } from "../../hooks/useModelBreakdown";
import { useToolUsage } from "../../hooks/useToolUsage";
import { useTimeWindow, getPresetLabel } from "../../hooks/useTimeWindow";

interface ExportButtonProps {
  disabled?: boolean;
}

export function ExportButton({ disabled }: ExportButtonProps) {
  const [isExporting, setIsExporting] = useState(false);
  const { preset, filter } = useTimeWindow();

  // Fetch all data for export
  const { data: summaryData } = useStatsSummary();
  const { data: runnerData } = useRunnerPerformance();
  const { data: profileData } = useProfileBreakdown();
  const { data: modelData } = useModelBreakdown();
  const { data: toolData } = useToolUsage();

  const handleExport = () => {
    setIsExporting(true);

    try {
      const csv: string[] = [];
      const timestamp = new Date().toISOString().slice(0, 19).replace(/:/g, "-");
      const timeWindow = getPresetLabel(preset);

      // Summary Section
      csv.push(`Agent Manager Stats Export - ${timeWindow}`);
      csv.push(`Generated: ${new Date().toLocaleString()}`);
      csv.push("");

      // KPI Summary
      if (summaryData?.summary) {
        const s = summaryData.summary;
        csv.push("=== Summary ===");
        csv.push(`Success Rate,${(s.successRate * 100).toFixed(1)}%`);
        csv.push(`Total Runs,${s.statusCounts.total}`);
        csv.push(`Complete,${s.statusCounts.complete}`);
        csv.push(`Failed,${s.statusCounts.failed}`);
        csv.push(`Running,${s.statusCounts.running}`);
        csv.push(`Pending,${s.statusCounts.pending}`);
        csv.push(`Cancelled,${s.statusCounts.cancelled}`);
        csv.push(`Needs Review,${s.statusCounts.needsReview}`);
        csv.push("");

        csv.push("=== Duration Stats ===");
        csv.push(`Average (ms),${s.duration.avgMs}`);
        csv.push(`P50 (ms),${s.duration.p50Ms}`);
        csv.push(`P95 (ms),${s.duration.p95Ms}`);
        csv.push(`P99 (ms),${s.duration.p99Ms}`);
        csv.push(`Min (ms),${s.duration.minMs}`);
        csv.push(`Max (ms),${s.duration.maxMs}`);
        csv.push("");

        csv.push("=== Cost Stats ===");
        csv.push(`Total Cost (USD),${s.cost.totalCostUsd.toFixed(2)}`);
        csv.push(`Avg Cost (USD),${s.cost.avgCostUsd.toFixed(4)}`);
        csv.push(`Input Tokens,${s.cost.inputTokens}`);
        csv.push(`Output Tokens,${s.cost.outputTokens}`);
        csv.push(`Cache Read Tokens,${s.cost.cacheReadTokens}`);
        csv.push(`Total Tokens,${s.cost.totalTokens}`);
        csv.push("");
      }

      // Runner Breakdown
      if (runnerData?.runners && runnerData.runners.length > 0) {
        csv.push("=== Runner Breakdown ===");
        csv.push("Runner Type,Runs,Success Count,Failed Count,Success Rate,Total Cost (USD),Avg Duration (ms)");
        for (const r of runnerData.runners) {
          const successRate = r.runCount > 0 ? ((r.successCount / r.runCount) * 100).toFixed(1) : "0.0";
          csv.push(`${r.runnerType},${r.runCount},${r.successCount},${r.failedCount},${successRate}%,${r.totalCostUsd.toFixed(2)},${r.avgDurationMs}`);
        }
        csv.push("");
      }

      // Profile Breakdown
      if (profileData?.profiles && profileData.profiles.length > 0) {
        csv.push("=== Profile Breakdown ===");
        csv.push("Profile Name,Profile ID,Runs,Success Count,Failed Count,Success Rate,Total Cost (USD)");
        for (const p of profileData.profiles) {
          const successRate = p.runCount > 0 ? ((p.successCount / p.runCount) * 100).toFixed(1) : "0.0";
          csv.push(`"${p.profileName}",${p.profileId},${p.runCount},${p.successCount},${p.failedCount},${successRate}%,${p.totalCostUsd.toFixed(2)}`);
        }
        csv.push("");
      }

      // Model Breakdown
      if (modelData?.models && modelData.models.length > 0) {
        csv.push("=== Model Breakdown ===");
        csv.push("Model,Runs,Success Count,Success Rate,Total Cost (USD),Total Tokens");
        for (const m of modelData.models) {
          const successRate = m.runCount > 0 ? ((m.successCount / m.runCount) * 100).toFixed(1) : "0.0";
          csv.push(`${m.model},${m.runCount},${m.successCount},${successRate}%,${m.totalCostUsd.toFixed(2)},${m.totalTokens}`);
        }
        csv.push("");
      }

      // Tool Usage
      if (toolData?.tools && toolData.tools.length > 0) {
        csv.push("=== Tool Usage ===");
        csv.push("Tool Name,Call Count,Success Count,Failed Count,Success Rate");
        for (const t of toolData.tools) {
          const successRate = t.callCount > 0 ? ((t.successCount / t.callCount) * 100).toFixed(1) : "0.0";
          csv.push(`${t.toolName},${t.callCount},${t.successCount},${t.failedCount},${successRate}%`);
        }
        csv.push("");
      }

      // Create and download file
      const csvContent = csv.join("\n");
      const blob = new Blob([csvContent], { type: "text/csv;charset=utf-8;" });
      const url = URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = `agent-manager-stats-${timestamp}.csv`;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      URL.revokeObjectURL(url);
    } finally {
      setIsExporting(false);
    }
  };

  const isDisabled = disabled || isExporting || !summaryData;

  return (
    <button
      onClick={handleExport}
      disabled={isDisabled}
      className="
        flex items-center gap-2 px-3 py-1.5 text-sm font-medium
        rounded-md border border-border bg-card/50
        text-muted-foreground hover:text-foreground hover:bg-muted/50
        disabled:opacity-50 disabled:cursor-not-allowed
        transition-colors
      "
    >
      <Download className={`h-4 w-4 ${isExporting ? "animate-bounce" : ""}`} />
      {isExporting ? "Exporting..." : "Export"}
    </button>
  );
}
