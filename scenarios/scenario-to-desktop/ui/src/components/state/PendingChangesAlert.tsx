/**
 * Alert component showing detected configuration changes that may invalidate cached state.
 * Provides actions to re-run affected stages or dismiss the alert.
 */

import { AlertTriangle, RefreshCw, X } from "lucide-react";
import { Button } from "../ui/button";
import type { StateChange } from "../../lib/api";

interface PendingChangesAlertProps {
  changes: StateChange[];
  onRerun?: () => void;
  onDismiss?: () => void;
  isRerunning?: boolean;
}

const changeTypeLabels: Record<string, string> = {
  manifest_path: "Manifest path",
  manifest_content: "Manifest content",
  manifest_touched: "Manifest file",
  preflight_secrets: "Secrets configuration",
  preflight_timeout: "Preflight timeout",
  template_type: "Template type",
  framework: "Framework",
  deployment_mode: "Deployment mode",
  app_display_name: "App display name",
  app_description: "App description",
  icon_path: "Icon path",
  platforms: "Target platforms",
  signing_enabled: "Signing toggle",
  signing_config: "Signing certificates",
  output_location: "Output location",
};

const stageLabels: Record<string, string> = {
  bundle: "Bundle",
  preflight: "Preflight",
  generate: "Generate",
  build: "Build",
  smoke_test: "Smoke Test",
};

export function PendingChangesAlert({
  changes,
  onRerun,
  onDismiss,
  isRerunning = false,
}: PendingChangesAlertProps) {
  if (!changes || changes.length === 0) {
    return null;
  }

  // Group changes by affected stage
  const changesByStage = changes.reduce(
    (acc, change) => {
      const stage = change.affected_stage;
      if (!acc[stage]) {
        acc[stage] = [];
      }
      acc[stage].push(change);
      return acc;
    },
    {} as Record<string, StateChange[]>
  );

  // Find earliest affected stage for display
  const stageOrder = ["bundle", "preflight", "generate", "build", "smoke_test"];
  const affectedStages = Object.keys(changesByStage);
  const earliestStage = stageOrder.find((s) => affectedStages.includes(s)) || affectedStages[0];

  return (
    <div className="rounded-lg border border-yellow-700/50 bg-yellow-950/30 p-4">
      <div className="flex items-start gap-3">
        <AlertTriangle className="mt-0.5 h-5 w-5 flex-shrink-0 text-yellow-400" />
        <div className="flex-1 min-w-0">
          <h4 className="text-sm font-medium text-yellow-200">
            Configuration changes detected
          </h4>
          <p className="mt-1 text-xs text-yellow-300/80">
            The following changes may invalidate cached results from{" "}
            <span className="font-medium">{stageLabels[earliestStage] || earliestStage}</span>{" "}
            onward:
          </p>

          <ul className="mt-2 space-y-1">
            {changes.map((change, idx) => (
              <li
                key={`${change.change_type}-${idx}`}
                className="flex items-start gap-2 text-xs text-yellow-200/90"
              >
                <span className="text-yellow-400">-</span>
                <span>
                  <span className="font-medium">
                    {changeTypeLabels[change.change_type] || change.change_type}
                  </span>
                  {change.reason && (
                    <span className="text-yellow-300/70">: {change.reason}</span>
                  )}
                  {change.old_value && change.new_value && (
                    <span className="text-yellow-300/60">
                      {" "}
                      ({change.old_value} → {change.new_value})
                    </span>
                  )}
                </span>
              </li>
            ))}
          </ul>

          <div className="mt-3 flex items-center gap-2">
            {onRerun && (
              <Button
                size="sm"
                variant="outline"
                onClick={onRerun}
                disabled={isRerunning}
                className="border-yellow-600 bg-yellow-900/30 text-yellow-200 hover:bg-yellow-900/50"
              >
                <RefreshCw
                  className={`mr-1.5 h-3 w-3 ${isRerunning ? "animate-spin" : ""}`}
                />
                {isRerunning ? "Re-running..." : "Re-run Now"}
              </Button>
            )}
            {onDismiss && (
              <Button
                size="sm"
                variant="ghost"
                onClick={onDismiss}
                className="text-yellow-300/70 hover:bg-yellow-900/20 hover:text-yellow-200"
              >
                <X className="mr-1 h-3 w-3" />
                Later
              </Button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
