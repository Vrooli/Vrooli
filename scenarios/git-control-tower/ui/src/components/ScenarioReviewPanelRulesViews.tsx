import { useState } from "react";
import { CheckCircle2, ChevronDown, ChevronRight } from "lucide-react";
import type { TidinessIssue, AgentContextItem } from "../lib/api";
import { AttachToAgentButton } from "./AgentTab";
import { codeQualityContextItems } from "../lib/agentContext";

export function ChangedFilesView({
  issues,
  issuesByFile,
  changedFiles,
  scoreData,
  isLoading,
  agentManagerAvailable,
  onAttachToAgent,
  scenarioSlug,
}: {
  issues: TidinessIssue[];
  issuesByFile: Map<string, TidinessIssue[]>;
  changedFiles: string[];
  scoreData?: { score: number; violations: number } | null;
  isLoading: boolean;
  agentManagerAvailable?: boolean;
  onAttachToAgent?: (item: AgentContextItem) => void;
  scenarioSlug: string;
}) {
  const [expandedFile, setExpandedFile] = useState<string | null>(null);

  if (isLoading) {
    return (
      <div className="space-y-3">
        <div className="h-16 animate-pulse bg-slate-800 rounded" />
        <div className="h-16 animate-pulse bg-slate-800 rounded" />
      </div>
    );
  }

  if (changedFiles.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-8 text-slate-500">
        <p className="text-sm">No changed files to analyze</p>
      </div>
    );
  }

  return (
    <div className="space-y-3">
      {/* Scenario-wide score badge for context */}
      {scoreData && (
        <div className="flex items-center gap-2 text-xs text-slate-400">
          <span>Scenario score:</span>
          <span className={`font-medium ${
            scoreData.score >= 70 ? "text-emerald-400" :
            scoreData.score >= 40 ? "text-amber-400" : "text-red-400"
          }`}>
            {Math.round(scoreData.score)}/100
          </span>
        </div>
      )}

      {issues.length === 0 ? (
        <div className="rounded-lg border border-slate-800 bg-slate-900/50 p-4">
          <div className="flex items-center gap-2">
            <CheckCircle2 className="h-4 w-4 text-emerald-400" />
            <span className="text-xs text-emerald-300 font-medium">No issues in changed files</span>
          </div>
        </div>
      ) : (
        <div className="space-y-1">
          {Array.from(issuesByFile.entries()).map(([filePath, fileIssues]) => (
            <div key={filePath} className="rounded border border-slate-800/50 bg-slate-900/30">
              <div className="flex items-center">
                <button
                  type="button"
                  onClick={() => setExpandedFile(expandedFile === filePath ? null : filePath)}
                  className="flex-1 flex items-center gap-2 px-3 py-2 text-xs cursor-pointer hover:bg-slate-800/30"
                >
                  {expandedFile === filePath ? (
                    <ChevronDown className="h-3 w-3 text-slate-500" />
                  ) : (
                    <ChevronRight className="h-3 w-3 text-slate-500" />
                  )}
                  <code className="text-slate-200">{filePath}</code>
                  <span className="text-slate-500">({fileIssues.length} issue{fileIssues.length !== 1 ? "s" : ""})</span>
                </button>
                {agentManagerAvailable && onAttachToAgent && (
                  <div className="pr-2">
                    <AttachToAgentButton onClick={() => {
                      for (const item of codeQualityContextItems(fileIssues, scenarioSlug)) {
                        onAttachToAgent(item);
                      }
                    }} />
                  </div>
                )}
              </div>
              {expandedFile === filePath && (
                <div className="px-3 pb-3 pt-1 border-t border-slate-800/30 space-y-1.5">
                  {fileIssues.map(issue => (
                    <div key={issue.id} className="flex items-start gap-2 text-[11px]">
                      <div className={`h-1.5 w-1.5 rounded-full mt-1.5 shrink-0 ${
                        issue.severity === "critical" || issue.severity === "high" ? "bg-red-500" :
                        issue.severity === "medium" ? "bg-amber-500" : "bg-blue-500"
                      }`} />
                      <div className="min-w-0 flex-1">
                        <span className="text-slate-500">{issue.category}:</span>{" "}
                        <span className="text-slate-300">{issue.title}</span>
                        {issue.line_number != null && (
                          <span className="text-slate-600 ml-1">L:{issue.line_number}</span>
                        )}
                      </div>
                      {agentManagerAvailable && onAttachToAgent && (
                        <AttachToAgentButton onClick={() => { const items = codeQualityContextItems([issue], scenarioSlug); if (items[0]) onAttachToAgent(items[0]); }} />
                      )}
                    </div>
                  ))}
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export function ScenarioWideView({
  scoreData,
  isLoading,
  agentManagerAvailable,
  onOpenPicker,
}: {
  scoreData?: {
    score: number;
    violations: number;
    breakdown?: {
      lint_issues: number;
      type_issues: number;
      long_files: number;
      complex_functions: number;
      tech_debt_markers: number;
      duplication_issues: number;
    };
    metrics?: {
      total_files: number;
      total_lines: number;
      avg_file_length: number;
      max_complexity: number;
      avg_complexity: number;
      duplication_pct: number;
    };
  } | null;
  isLoading: boolean;
  agentManagerAvailable?: boolean;
  onOpenPicker?: () => void;
}) {
  if (isLoading) {
    return (
      <div className="space-y-3">
        <div className="h-24 animate-pulse bg-slate-800 rounded" />
        <div className="h-32 animate-pulse bg-slate-800 rounded" />
      </div>
    );
  }

  if (!scoreData) {
    return (
      <div className="flex flex-col items-center justify-center py-8 text-slate-500">
        <p className="text-sm">No quality data available</p>
        <p className="text-xs mt-1">Run a scan to generate quality metrics</p>
      </div>
    );
  }

  const scoreColor = scoreData.score >= 70 ? "bg-emerald-500" :
    scoreData.score >= 40 ? "bg-amber-500" : "bg-red-500";
  const scoreTextColor = scoreData.score >= 70 ? "text-emerald-300" :
    scoreData.score >= 40 ? "text-amber-300" : "text-red-300";

  return (
    <div className="space-y-4">
      {/* Score bar */}
      <div className="rounded-lg border border-slate-800 bg-slate-900/50 p-4 space-y-3">
        <div className="flex items-center justify-between">
          <span className={`text-2xl font-bold ${scoreTextColor}`}>
            {Math.round(scoreData.score)}
          </span>
          <span className="text-xs text-slate-500">/100</span>
        </div>
        <div className="w-full h-2 bg-slate-800 rounded-full overflow-hidden">
          <div
            className={`h-full rounded-full transition-all ${scoreColor}`}
            style={{ width: `${Math.min(100, Math.max(0, scoreData.score))}%` }}
          />
        </div>
        <div className="flex justify-between text-xs">
          <span className="text-slate-400">Violations</span>
          <div className="flex items-center gap-2">
            <span className="text-slate-200">{scoreData.violations}</span>
            {agentManagerAvailable && onOpenPicker && scoreData.violations > 0 && (
              <AttachToAgentButton onClick={onOpenPicker} />
            )}
          </div>
        </div>
      </div>

      {/* Breakdown */}
      {scoreData.breakdown && (
        <div className="rounded-lg border border-slate-800 bg-slate-900/50 p-4">
          <h4 className="text-xs font-medium text-slate-400 mb-3">Breakdown</h4>
          <div className="space-y-2">
            {([
              ["Lint issues", scoreData.breakdown.lint_issues],
              ["Type issues", scoreData.breakdown.type_issues],
              ["Long files", scoreData.breakdown.long_files],
              ["Complex functions", scoreData.breakdown.complex_functions],
              ["Tech debt markers", scoreData.breakdown.tech_debt_markers],
              ["Duplication", scoreData.breakdown.duplication_issues],
            ] as const).map(([label, value]) => (
              <div key={label} className="flex justify-between text-xs">
                <span className="text-slate-400">{label}</span>
                <span className={value > 0 ? "text-slate-200" : "text-slate-600"}>{value}</span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Metrics */}
      {scoreData.metrics && (
        <div className="rounded-lg border border-slate-800 bg-slate-900/50 p-4">
          <h4 className="text-xs font-medium text-slate-400 mb-3">Metrics</h4>
          <div className="space-y-2">
            {([
              ["Total files", String(scoreData.metrics.total_files)],
              ["Total lines", scoreData.metrics.total_lines.toLocaleString()],
              ["Avg file length", String(Math.round(scoreData.metrics.avg_file_length))],
              ["Max complexity", String(scoreData.metrics.max_complexity)],
              ["Avg complexity", scoreData.metrics.avg_complexity.toFixed(1)],
              ["Duplication", `${scoreData.metrics.duplication_pct.toFixed(1)}%`],
            ] as const).map(([label, value]) => (
              <div key={label} className="flex justify-between text-xs">
                <span className="text-slate-400">{label}</span>
                <span className="text-slate-200">{value}</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

