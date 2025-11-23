import React, { useState } from "react";
import {
  CheckCircle2,
  Circle,
  Download,
  FileSearch,
  AlertTriangle,
  Wrench,
  Play,
  RefreshCw,
  Shield,
  FileCode,
  ChevronDown,
  ChevronRight
} from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "../ui/card";
import { Badge } from "../ui/badge";
import { Button } from "../ui/button";

interface RecommendedFlowPanelProps {
  onScanAll?: () => void;
  onExportDAG?: () => void;
  onOpenDocs?: () => void;
  onJumpToStatus?: () => void;
  dagHelp?: string | null;
  targetTierLabel?: string;
}

interface FlowStep {
  number: number;
  title: string;
  description: string;
  icon: React.ComponentType<{ className?: string }>;
  action?: string;
  completed?: boolean;
  ctaLabel?: string;
  onClick?: () => void;
}

/**
 * RecommendedFlowPanel - displays a guided workflow for deployment readiness
 * Helps users understand the sequential steps to prepare scenarios for deployment
 */
export function RecommendedFlowPanel({
  onScanAll,
  onExportDAG,
  onOpenDocs,
  onJumpToStatus,
  dagHelp,
  targetTierLabel
}: RecommendedFlowPanelProps) {
  const [open, setOpen] = useState<boolean>(false);
  const flowSteps: FlowStep[] = [
    {
      number: 1,
      title: "Scan scenario dependencies",
      description: "Detect resources and scenario dependencies from code analysis",
      icon: FileSearch,
      action: "Select a scenario below and click 'Scan' or 'Scan & Apply'",
      ctaLabel: "Scan non-ready",
      onClick: onScanAll
    },
    {
      number: 2,
      title: "Review tier fitness & blockers",
      description: "Check deployment tier compatibility and identify blocking dependencies",
      icon: AlertTriangle,
      action: "Review tier fitness scores and blocker counts in the table",
      ctaLabel: "Jump to statuses",
      onClick: onJumpToStatus
    },
    {
      number: 3,
      title: "Export DAG for deployment-manager",
      description: "Generate recursive dependency tree for deployment orchestration",
      icon: Download,
      action: "Use CLI: scenario-dependency-analyzer dag export <scenario> --recursive",
      ctaLabel: "Show CLI",
      onClick: onExportDAG
    },
    {
      number: 4,
      title: "Fix metadata gaps",
      description: "Add missing deployment.tiers and deployment.dependencies metadata",
      icon: Wrench,
      action: "Edit .vrooli/service.json files to add missing tier/dependency metadata",
      ctaLabel: "Open docs",
      onClick: onOpenDocs
    }
  ];

  return (
    <Card className="border border-primary/30 bg-primary/5">
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <CardTitle className="text-base flex items-center gap-2">
            <Shield className="h-4 w-4 text-primary" /> Deployment Readiness Workflow
          </CardTitle>
          <div className="flex items-center gap-2">
            <Badge variant="secondary" className="text-xs">Recommended</Badge>
            <Button
              size="sm"
              variant="outline"
              className="h-8 gap-2 text-xs"
              onClick={() => setOpen((prev) => !prev)}
            >
              {open ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
              {open ? "Hide workflow" : "Show workflow"}
            </Button>
          </div>
        </div>
      </CardHeader>
      {open ? (
        <CardContent className="space-y-3">
          <p className="text-sm text-muted-foreground">
            Follow these steps to prepare scenarios for production deployment across tiers. Current focus: {targetTierLabel || "choose a tier above"}.
          </p>

          <div className="space-y-3">
            {flowSteps.map((step) => (
              <div
                key={step.number}
                className="flex gap-3 rounded-lg border border-border/40 bg-background/40 p-3 transition-colors hover:bg-background/60"
              >
                <div className="flex-shrink-0 pt-1">
                  <div className="flex items-center gap-2">
                    <div className="h-6 w-6 rounded-full border border-border/60 bg-background/60 text-[11px] font-semibold text-foreground flex items-center justify-center">
                      {step.number}
                    </div>
                    {step.completed ? (
                      <CheckCircle2 className="h-4 w-4 text-green-400" aria-hidden="true" />
                    ) : (
                      <Circle className="h-4 w-4 text-muted-foreground" aria-hidden="true" />
                    )}
                  </div>
                </div>
                <div className="flex-1 space-y-1">
                  <div className="flex items-start gap-2">
                    <step.icon className="h-4 w-4 text-primary mt-0.5" />
                    <div className="flex-1">
                      <p className="font-medium text-sm text-foreground">
                        {step.number}. {step.title}
                      </p>
                      <p className="text-xs text-muted-foreground mt-0.5">
                        {step.description}
                      </p>
                      {step.action && (
                        <p className="text-xs text-primary/80 mt-1 font-mono">
                          → {step.action}
                        </p>
                      )}
                    </div>
                  </div>
                  {step.ctaLabel ? (
                    <Button
                      size="sm"
                      variant="secondary"
                      className="mt-1 w-fit gap-2"
                      onClick={step.onClick}
                      disabled={!step.onClick}
                    >
                      <Play className="h-3 w-3" aria-hidden="true" />
                      {step.ctaLabel}
                    </Button>
                  ) : null}
                </div>
              </div>
            ))}
          </div>

          <div className="pt-2 space-y-2 border-t border-border/40">
            <p className="text-xs text-muted-foreground">
              <strong>Quick actions:</strong>
            </p>
            <div className="flex flex-wrap gap-2 text-xs">
              <button
                className="rounded border border-border/60 bg-background/60 px-2 py-1 hover:bg-background/80 transition-colors"
                onClick={onScanAll}
              >
                <RefreshCw className="mr-1 inline-block h-3 w-3" aria-hidden="true" />
                Scan all scenarios
              </button>
              <button
                className="rounded border border-border/60 bg-background/60 px-2 py-1 hover:bg-background/80 transition-colors"
                onClick={onExportDAG}
              >
                <FileCode className="mr-1 inline-block h-3 w-3" aria-hidden="true" />
                Export DAG JSON
              </button>
              {onOpenDocs ? (
                <button
                  className="rounded border border-border/60 bg-background/60 px-2 py-1 hover:bg-background/80 transition-colors"
                  onClick={onOpenDocs}
                >
                  <FileSearch className="mr-1 inline-block h-3 w-3" aria-hidden="true" />
                  View docs
                </button>
              ) : null}
            </div>
          </div>

          {dagHelp ? (
            <div className="pt-2 rounded bg-background/40 border border-border/30 p-2">
              <p className="text-[11px] text-muted-foreground">
                💡 <strong>How to export a DAG:</strong>
              </p>
              <pre className="mt-1 rounded bg-background/80 px-2 py-1 text-xs text-foreground">
scenario-dependency-analyzer dag export &lt;scenario&gt; --recursive
              </pre>
              <p className="mt-1 text-[11px] text-muted-foreground">
                Swap &lt;scenario&gt; for any name, then feed the JSON to deployment-manager.
              </p>
            </div>
          ) : (
            <div className="pt-2 rounded bg-background/40 border border-border/30 p-2">
              <p className="text-[11px] text-muted-foreground">
                💡 <strong>Pro tip:</strong> Use the <code className="rounded bg-background/80 px-1 py-0.5 text-primary">deployment</code> CLI command to view detailed readiness reports:
                <code className="block mt-1 rounded bg-background/80 px-2 py-1 text-xs">
                  scenario-dependency-analyzer deployment &lt;scenario&gt;
                </code>
              </p>
            </div>
          )}
        </CardContent>
      ) : null}
    </Card>
  );
}
