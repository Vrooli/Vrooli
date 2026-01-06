import { useState } from "react";
import {
  AlertCircle,
  AlertTriangle,
  ChevronDown,
  ChevronRight,
  CheckCircle2,
  Copy,
  ExternalLink,
  FileText,
  Lightbulb,
  Search,
  TrendingUp,
} from "lucide-react";
import { Badge } from "./ui/badge";
import { Button } from "./ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "./ui/card";
import { ScrollArea } from "./ui/scroll-area";
import { MarkdownRenderer } from "./markdown";
import type {
  ApplyFixesRequest,
  Investigation,
  MetricsData,
  Recommendation,
} from "../types";
import { cn } from "../lib/utils";

interface InvestigationReportProps {
  investigation: Investigation;
  onApplyFixes?: (request: ApplyFixesRequest) => Promise<void>;
  onViewRun?: (runId: string) => void;
  applyingFixes?: boolean;
}

export function InvestigationReport({
  investigation,
  onApplyFixes,
  onViewRun,
  applyingFixes = false,
}: InvestigationReportProps) {
  const [selectedRecommendations, setSelectedRecommendations] = useState<Set<string>>(new Set());
  const [expandedSections, setExpandedSections] = useState<Set<string>>(
    new Set(["summary", "root_cause", "recommendations", "metrics"])
  );

  const findings = investigation.findings;
  const metrics = investigation.metrics;

  const toggleSection = (section: string) => {
    setExpandedSections((prev) => {
      const next = new Set(prev);
      if (next.has(section)) {
        next.delete(section);
      } else {
        next.add(section);
      }
      return next;
    });
  };

  const toggleRecommendation = (id: string) => {
    setSelectedRecommendations((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  const handleApplyFixes = async () => {
    if (!onApplyFixes || selectedRecommendations.size === 0) return;
    await onApplyFixes({
      recommendation_ids: Array.from(selectedRecommendations),
    });
  };

  const copyReport = () => {
    const text = formatReportAsText(investigation);
    navigator.clipboard.writeText(text);
  };

  if (!findings) {
    return (
      <Card>
        <CardContent className="py-8 text-center text-muted-foreground">
          <FileText className="h-12 w-12 mx-auto mb-4 opacity-50" />
          <p>No findings available yet</p>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Search className="h-5 w-5 text-primary" />
          <h3 className="font-semibold">Investigation Report</h3>
          <Badge variant={getStatusVariant(investigation.status)}>
            {investigation.status}
          </Badge>
        </div>
        <Button variant="outline" size="sm" onClick={copyReport} className="gap-2">
          <Copy className="h-4 w-4" />
          Copy Report
        </Button>
      </div>

      <ScrollArea className="h-[600px]">
        <div className="space-y-4 pr-4">
          {/* Summary */}
          <CollapsibleSection
            title="Summary"
            icon={<FileText className="h-4 w-4" />}
            expanded={expandedSections.has("summary")}
            onToggle={() => toggleSection("summary")}
          >
            <div className="text-sm">
              <MarkdownRenderer content={findings.summary} />
            </div>
          </CollapsibleSection>

          {/* Root Cause */}
          {findings.root_cause && (
            <CollapsibleSection
              title="Root Cause Analysis"
              icon={<AlertTriangle className="h-4 w-4" />}
              expanded={expandedSections.has("root_cause")}
              onToggle={() => toggleSection("root_cause")}
              badge={
                <Badge variant={getConfidenceVariant(findings.root_cause.confidence)}>
                  {findings.root_cause.confidence} confidence
                </Badge>
              }
            >
              <div className="space-y-4">
                <div className="text-sm">
                  <MarkdownRenderer content={findings.root_cause.description} />
                </div>

                {findings.root_cause.evidence.length > 0 && (
                  <div className="space-y-2">
                    <h4 className="text-sm font-medium">Evidence</h4>
                    {findings.root_cause.evidence.map((evidence, i) => (
                      <div
                        key={i}
                        className="rounded-md border border-border bg-muted/50 p-3 text-sm"
                      >
                        <div className="flex items-center justify-between mb-2">
                          <span className="text-muted-foreground">
                            Run: <code className="text-xs">{evidence.run_id.slice(0, 8)}...</code>
                            {evidence.event_seq !== undefined && (
                              <> | Event #{evidence.event_seq}</>
                            )}
                          </span>
                          {onViewRun && (
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => onViewRun(evidence.run_id)}
                              className="gap-1 h-6 px-2"
                            >
                              <ExternalLink className="h-3 w-3" />
                              View
                            </Button>
                          )}
                        </div>
                        <div className="text-sm">
                          <MarkdownRenderer content={evidence.description} />
                        </div>
                        {evidence.snippet && (
                          <pre className="mt-2 text-[11px] bg-background rounded p-2 overflow-x-auto">
                            {evidence.snippet}
                          </pre>
                        )}
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </CollapsibleSection>
          )}

          {/* Recommendations */}
          {findings.recommendations && findings.recommendations.length > 0 && (
            <CollapsibleSection
              title="Recommendations"
              icon={<Lightbulb className="h-4 w-4" />}
              expanded={expandedSections.has("recommendations")}
              onToggle={() => toggleSection("recommendations")}
              badge={<Badge variant="secondary">{findings.recommendations.length}</Badge>}
            >
              <div className="space-y-3">
                {findings.recommendations.map((rec) => (
                  <RecommendationCard
                    key={rec.id}
                    recommendation={rec}
                    selected={selectedRecommendations.has(rec.id)}
                    onToggle={() => toggleRecommendation(rec.id)}
                    selectable={onApplyFixes !== undefined}
                  />
                ))}

                {onApplyFixes && selectedRecommendations.size > 0 && (
                  <div className="pt-2 border-t border-border">
                    <Button
                      onClick={handleApplyFixes}
                      disabled={applyingFixes}
                      className="gap-2"
                    >
                      {applyingFixes ? (
                        "Applying..."
                      ) : (
                        <>
                          <CheckCircle2 className="h-4 w-4" />
                          Apply {selectedRecommendations.size} Fix{selectedRecommendations.size !== 1 ? "es" : ""}
                        </>
                      )}
                    </Button>
                  </div>
                )}
              </div>
            </CollapsibleSection>
          )}

          {/* Metrics */}
          {metrics && (
            <CollapsibleSection
              title="Metrics Summary"
              icon={<TrendingUp className="h-4 w-4" />}
              expanded={expandedSections.has("metrics")}
              onToggle={() => toggleSection("metrics")}
            >
              <MetricsDisplay metrics={metrics} />
            </CollapsibleSection>
          )}
        </div>
      </ScrollArea>
    </div>
  );
}

interface CollapsibleSectionProps {
  title: string;
  icon: React.ReactNode;
  expanded: boolean;
  onToggle: () => void;
  badge?: React.ReactNode;
  children: React.ReactNode;
}

function CollapsibleSection({
  title,
  icon,
  expanded,
  onToggle,
  badge,
  children,
}: CollapsibleSectionProps) {
  return (
    <Card>
      <CardHeader
        className="py-3 cursor-pointer hover:bg-muted/50 transition-colors"
        onClick={onToggle}
      >
        <div className="flex items-center justify-between">
          <CardTitle className="flex items-center gap-2 text-sm font-medium">
            {expanded ? (
              <ChevronDown className="h-4 w-4" />
            ) : (
              <ChevronRight className="h-4 w-4" />
            )}
            {icon}
            {title}
          </CardTitle>
          {badge}
        </div>
      </CardHeader>
      {expanded && <CardContent className="pt-0">{children}</CardContent>}
    </Card>
  );
}

interface RecommendationCardProps {
  recommendation: Recommendation;
  selected: boolean;
  onToggle: () => void;
  selectable: boolean;
}

function RecommendationCard({
  recommendation,
  selected,
  onToggle,
  selectable,
}: RecommendationCardProps) {
  return (
    <div
      className={cn(
        "rounded-md border p-3 transition-colors",
        selected
          ? "border-primary bg-primary/5"
          : "border-border hover:bg-muted/50"
      )}
    >
      <div className="flex items-start gap-3">
        {selectable && (
          <input
            type="checkbox"
            checked={selected}
            onChange={onToggle}
            className="mt-1 h-4 w-4 rounded border-border text-primary focus:ring-primary"
          />
        )}
        <div className="flex-1 space-y-2">
          <div className="flex items-center gap-2">
            <Badge variant={getPriorityVariant(recommendation.priority)}>
              {recommendation.priority}
            </Badge>
            <Badge variant="outline">{recommendation.action_type.replace("_", " ")}</Badge>
          </div>
          <h4 className="font-medium text-sm">{recommendation.title}</h4>
          <div className="text-sm">
            <MarkdownRenderer content={recommendation.description} />
          </div>
        </div>
      </div>
    </div>
  );
}

function MetricsDisplay({ metrics }: { metrics: MetricsData }) {
  return (
    <div className="space-y-4">
      {/* Summary Stats */}
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        <MetricCard label="Total Runs" value={metrics.total_runs} />
        <MetricCard
          label="Success Rate"
          value={`${(metrics.success_rate * 100).toFixed(1)}%`}
          variant={metrics.success_rate >= 0.8 ? "success" : metrics.success_rate >= 0.5 ? "warning" : "destructive"}
        />
        <MetricCard
          label="Avg Duration"
          value={formatDuration(metrics.avg_duration_seconds)}
        />
        <MetricCard
          label="Total Tokens"
          value={metrics.total_tokens_used.toLocaleString()}
        />
        <MetricCard
          label="Total Cost"
          value={`$${metrics.total_cost.toFixed(4)}`}
        />
      </div>

      {/* Tool Usage Breakdown */}
      {Object.keys(metrics.tool_usage_breakdown).length > 0 && (
        <div className="space-y-2">
          <h4 className="text-sm font-medium">Tool Usage</h4>
          <div className="grid gap-2 sm:grid-cols-2">
            {Object.entries(metrics.tool_usage_breakdown)
              .sort(([, a], [, b]) => b - a)
              .slice(0, 10)
              .map(([tool, count]) => (
                <div
                  key={tool}
                  className="flex items-center justify-between rounded border border-border px-3 py-2 text-sm"
                >
                  <span className="font-mono text-xs">{tool}</span>
                  <Badge variant="secondary">{count}</Badge>
                </div>
              ))}
          </div>
        </div>
      )}

      {/* Error Breakdown */}
      {Object.keys(metrics.error_type_breakdown).length > 0 && (
        <div className="space-y-2">
          <h4 className="text-sm font-medium">Error Types</h4>
          <div className="grid gap-2 sm:grid-cols-2">
            {Object.entries(metrics.error_type_breakdown)
              .sort(([, a], [, b]) => b - a)
              .map(([errorType, count]) => (
                <div
                  key={errorType}
                  className="flex items-center justify-between rounded border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm"
                >
                  <span>{errorType}</span>
                  <Badge variant="destructive">{count}</Badge>
                </div>
              ))}
          </div>
        </div>
      )}

      {/* Custom Metrics */}
      {Object.keys(metrics.custom_metrics).length > 0 && (
        <div className="space-y-2">
          <h4 className="text-sm font-medium">Additional Metrics</h4>
          <div className="grid gap-2 sm:grid-cols-2">
            {Object.entries(metrics.custom_metrics).map(([name, value]) => (
              <div
                key={name}
                className="flex items-center justify-between rounded border border-border px-3 py-2 text-sm"
              >
                <span>{name}</span>
                <span className="font-medium">{value.toFixed(2)}</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

function MetricCard({
  label,
  value,
  variant = "default",
}: {
  label: string;
  value: string | number;
  variant?: "default" | "success" | "warning" | "destructive";
}) {
  return (
    <div className="rounded-md border border-border p-3">
      <p className="text-xs text-muted-foreground">{label}</p>
      <p
        className={cn(
          "text-lg font-semibold",
          variant === "success" && "text-success",
          variant === "warning" && "text-warning",
          variant === "destructive" && "text-destructive"
        )}
      >
        {value}
      </p>
    </div>
  );
}

function getStatusVariant(status: string): "default" | "secondary" | "success" | "destructive" | "outline" {
  switch (status) {
    case "completed":
      return "success";
    case "failed":
      return "destructive";
    case "running":
    case "pending":
      return "secondary";
    default:
      return "outline";
  }
}

function getConfidenceVariant(confidence: string): "default" | "success" | "secondary" | "destructive" {
  switch (confidence) {
    case "high":
      return "success";
    case "medium":
      return "secondary";
    case "low":
      return "destructive";
    default:
      return "default";
  }
}

function getPriorityVariant(priority: string): "default" | "destructive" | "secondary" | "outline" {
  switch (priority) {
    case "critical":
      return "destructive";
    case "high":
      return "destructive";
    case "medium":
      return "secondary";
    case "low":
      return "outline";
    default:
      return "default";
  }
}

function formatDuration(seconds: number): string {
  if (seconds < 60) {
    return `${seconds.toFixed(1)}s`;
  }
  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = seconds % 60;
  return `${minutes}m ${remainingSeconds.toFixed(0)}s`;
}

function formatReportAsText(investigation: Investigation): string {
  const lines: string[] = [];
  const findings = investigation.findings;

  lines.push("# Investigation Report");
  lines.push(`Status: ${investigation.status}`);
  lines.push(`Runs: ${investigation.run_ids.length}`);
  lines.push("");

  if (findings) {
    lines.push("## Summary");
    lines.push(findings.summary);
    lines.push("");

    if (findings.root_cause) {
      lines.push("## Root Cause Analysis");
      lines.push(`Confidence: ${findings.root_cause.confidence}`);
      lines.push(findings.root_cause.description);
      lines.push("");

      if (findings.root_cause.evidence.length > 0) {
        lines.push("### Evidence");
        findings.root_cause.evidence.forEach((e, i) => {
          lines.push(`${i + 1}. [Run ${e.run_id.slice(0, 8)}] ${e.description}`);
          if (e.snippet) {
            lines.push("```");
            lines.push(e.snippet);
            lines.push("```");
          }
        });
        lines.push("");
      }
    }

    if (findings.recommendations && findings.recommendations.length > 0) {
      lines.push("## Recommendations");
      findings.recommendations.forEach((rec, i) => {
        lines.push(`### ${i + 1}. ${rec.title} [${rec.priority}]`);
        lines.push(`Type: ${rec.action_type}`);
        lines.push(rec.description);
        lines.push("");
      });
    }
  }

  if (investigation.metrics) {
    const m = investigation.metrics;
    lines.push("## Metrics");
    lines.push(`- Total Runs: ${m.total_runs}`);
    lines.push(`- Success Rate: ${(m.success_rate * 100).toFixed(1)}%`);
    lines.push(`- Avg Duration: ${formatDuration(m.avg_duration_seconds)}`);
    lines.push(`- Total Tokens: ${m.total_tokens_used.toLocaleString()}`);
    lines.push(`- Total Cost: $${m.total_cost.toFixed(4)}`);
  }

  return lines.join("\n");
}
