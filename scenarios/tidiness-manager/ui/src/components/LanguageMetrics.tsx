import { Code, AlertCircle, FileCode, Copy, AlertTriangle } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "./ui/card";
import { Badge } from "./ui/badge";
import type { LanguageMetrics as LanguageMetricsType } from "../lib/api";

interface LanguageMetricsProps {
  metrics: Record<string, LanguageMetricsType>;
}

export function LanguageMetrics({ metrics }: LanguageMetricsProps) {
  const languages = Object.values(metrics);

  if (languages.length === 0) {
    return null;
  }

  return (
    <div className="space-y-4">
      <h3 className="text-lg font-semibold flex items-center gap-2">
        <Code className="h-5 w-5" />
        Code Quality Metrics by Language
      </h3>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {languages.map((langMetrics) => (
          <Card key={langMetrics.language}>
            <CardHeader className="pb-3">
              <CardTitle className="text-sm font-medium flex items-center justify-between">
                <span className="capitalize">{langMetrics.language}</span>
                <FileCode className="h-4 w-4 text-slate-400" />
              </CardTitle>
              <CardDescription>
                {langMetrics.file_count} files · {langMetrics.total_lines.toLocaleString()} lines
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-3">
              {/* Code Metrics - Always available */}
              {langMetrics.code_metrics && (
                <div className="space-y-2">
                  <div className="text-xs font-semibold text-slate-400 uppercase">Technical Debt</div>
                  <div className="grid grid-cols-3 gap-2 text-xs">
                    <MetricBadge
                      label="TODOs"
                      value={langMetrics.code_metrics.todo_count}
                      variant={langMetrics.code_metrics.todo_count > 5 ? "warning" : "default"}
                    />
                    <MetricBadge
                      label="FIXMEs"
                      value={langMetrics.code_metrics.fixme_count}
                      variant={langMetrics.code_metrics.fixme_count > 3 ? "error" : "default"}
                    />
                    <MetricBadge
                      label="HACKs"
                      value={langMetrics.code_metrics.hack_count}
                      variant={langMetrics.code_metrics.hack_count > 2 ? "error" : "default"}
                    />
                  </div>

                  <div className="grid grid-cols-2 gap-2 text-xs pt-2">
                    <div>
                      <div className="text-slate-400">Avg Imports</div>
                      <div className="font-semibold">{langMetrics.code_metrics.avg_imports_per_file.toFixed(1)}</div>
                    </div>
                    <div>
                      <div className="text-slate-400">Avg Functions</div>
                      <div className="font-semibold">{langMetrics.code_metrics.avg_functions_per_file.toFixed(1)}</div>
                    </div>
                  </div>
                </div>
              )}

              {/* Complexity Metrics */}
              {langMetrics.complexity && !langMetrics.complexity.skipped && (
                <div className="space-y-2 pt-2 border-t border-slate-700">
                  <div className="text-xs font-semibold text-slate-400 uppercase flex items-center gap-1">
                    <AlertCircle className="h-3 w-3" />
                    Complexity ({langMetrics.complexity.tool})
                  </div>
                  <div className="grid grid-cols-2 gap-2 text-xs">
                    <div>
                      <div className="text-slate-400">Average</div>
                      <div className="font-semibold">{langMetrics.complexity.average_complexity.toFixed(1)}</div>
                    </div>
                    <div>
                      <div className="text-slate-400">Maximum</div>
                      <div className="font-semibold text-red-400">{langMetrics.complexity.max_complexity}</div>
                    </div>
                  </div>
                  {langMetrics.complexity.high_complexity_count > 0 && (
                    <div className="text-xs">
                      <Badge variant="warning">
                        {langMetrics.complexity.high_complexity_count} high complexity functions
                      </Badge>
                    </div>
                  )}
                </div>
              )}

              {langMetrics.complexity && langMetrics.complexity.skipped && (
                <div className="text-xs text-slate-500 italic pt-2 border-t border-slate-700">
                  <AlertTriangle className="h-3 w-3 inline mr-1" />
                  {langMetrics.complexity.skip_reason}
                </div>
              )}

              {/* Duplication Metrics */}
              {langMetrics.duplicates && !langMetrics.duplicates.skipped && (
                <div className="space-y-2 pt-2 border-t border-slate-700">
                  <div className="text-xs font-semibold text-slate-400 uppercase flex items-center gap-1">
                    <Copy className="h-3 w-3" />
                    Duplication ({langMetrics.duplicates.tool})
                  </div>
                  <div className="grid grid-cols-2 gap-2 text-xs">
                    <div>
                      <div className="text-slate-400">Duplicate Blocks</div>
                      <div className="font-semibold">{langMetrics.duplicates.total_duplicates}</div>
                    </div>
                    <div>
                      <div className="text-slate-400">Duplicate Lines</div>
                      <div className="font-semibold">{langMetrics.duplicates.total_lines}</div>
                    </div>
                  </div>
                  {langMetrics.duplicates.total_duplicates > 0 && (
                    <div className="text-xs">
                      <Badge variant={langMetrics.duplicates.total_duplicates > 5 ? "error" : "warning"}>
                        Refactoring opportunity
                      </Badge>
                    </div>
                  )}
                </div>
              )}

              {langMetrics.duplicates && langMetrics.duplicates.skipped && (
                <div className="text-xs text-slate-500 italic pt-2 border-t border-slate-700">
                  <AlertTriangle className="h-3 w-3 inline mr-1" />
                  {langMetrics.duplicates.skip_reason}
                </div>
              )}
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}

interface MetricBadgeProps {
  label: string;
  value: number;
  variant?: "default" | "warning" | "error";
}

function MetricBadge({ label, value, variant = "default" }: MetricBadgeProps) {
  const variantClasses = {
    default: "bg-slate-700 text-slate-300",
    warning: "bg-yellow-900/30 text-yellow-400 border border-yellow-600",
    error: "bg-red-900/30 text-red-400 border border-red-600",
  };

  return (
    <div className={`rounded px-2 py-1 text-center ${variantClasses[variant]}`}>
      <div className="font-semibold">{value}</div>
      <div className="text-[10px] opacity-75">{label}</div>
    </div>
  );
}
