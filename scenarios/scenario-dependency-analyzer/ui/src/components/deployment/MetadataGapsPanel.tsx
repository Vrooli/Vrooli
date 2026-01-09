import { AlertTriangle, CheckCircle2, FileText, Layers, Package } from "lucide-react";
import { Badge } from "../ui/badge";
import { Button } from "../ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "../ui/card";
import type { DeploymentMetadataGaps, ScenarioGapInfo } from "../../types";

interface MetadataGapsPanelProps {
  gaps: DeploymentMetadataGaps | null | undefined;
  onFixGap?: (scenarioName: string, gapType: string) => void;
}

function GapItem({ gap, onFix }: { gap: ScenarioGapInfo; onFix?: (scenarioName: string, gapType: string) => void }) {
  const hasGaps = !gap.has_deployment_block ||
                  gap.missing_dependency_catalog ||
                  (gap.missing_tier_definitions && gap.missing_tier_definitions.length > 0) ||
                  (gap.missing_resource_metadata && gap.missing_resource_metadata.length > 0) ||
                  (gap.missing_scenario_metadata && gap.missing_scenario_metadata.length > 0);

  if (!hasGaps) {
    return null;
  }

  return (
    <div className="space-y-3 rounded-lg border border-amber-500/40 bg-amber-500/5 p-4">
      <div className="flex items-start justify-between">
        <div>
          <p className="font-medium text-foreground">{gap.scenario_name}</p>
          <p className="text-xs text-muted-foreground">{gap.scenario_path}</p>
        </div>
        {!gap.has_deployment_block && (
          <Badge variant="outline" className="border-red-500/50 bg-red-500/10 text-red-100">
            Critical
          </Badge>
        )}
      </div>

      {/* Missing deployment block */}
      {!gap.has_deployment_block && (
        <div className="space-y-2 rounded border border-red-500/30 bg-red-500/5 p-3">
          <div className="flex items-center gap-2 text-sm">
            <AlertTriangle className="h-4 w-4 text-red-400" />
            <span className="font-medium text-red-100">No deployment block</span>
          </div>
          <p className="text-xs text-red-200/80">
            This scenario is missing a deployment configuration block in .vrooli/service.json
          </p>
          <Button
            size="sm"
            variant="outline"
            className="h-7 border-red-500/50 bg-red-500/10 text-xs text-red-100 hover:bg-red-500/20"
            onClick={() => onFix?.(gap.scenario_name, "deployment_block")}
          >
            <FileText className="mr-1 h-3 w-3" />
            Add deployment block
          </Button>
        </div>
      )}

      {/* Missing dependency catalog */}
      {gap.missing_dependency_catalog && (
        <div className="space-y-2 rounded border border-amber-500/30 bg-amber-500/5 p-3">
          <div className="flex items-center gap-2 text-sm">
            <Package className="h-4 w-4 text-amber-400" />
            <span className="font-medium text-amber-100">Missing dependency catalog</span>
          </div>
          <p className="text-xs text-amber-200/80">
            Define deployment.dependencies to specify platform support and alternatives
          </p>
          <Button
            size="sm"
            variant="outline"
            className="h-7 border-amber-500/50 bg-amber-500/10 text-xs text-amber-100 hover:bg-amber-500/20"
            onClick={() => onFix?.(gap.scenario_name, "dependency_catalog")}
          >
            <Package className="mr-1 h-3 w-3" />
            Add dependency catalog
          </Button>
        </div>
      )}

      {/* Missing tier definitions */}
      {gap.missing_tier_definitions && gap.missing_tier_definitions.length > 0 && (
        <div className="space-y-2 rounded border border-amber-500/30 bg-amber-500/5 p-3">
          <div className="flex items-center gap-2 text-sm">
            <Layers className="h-4 w-4 text-amber-400" />
            <span className="font-medium text-amber-100">Missing tier definitions</span>
          </div>
          <div className="flex flex-wrap gap-1">
            {gap.missing_tier_definitions.map((tier) => (
              <Badge key={tier} variant="outline" className="border-amber-500/50 bg-amber-500/10 text-[10px] text-amber-100">
                {tier}
              </Badge>
            ))}
          </div>
          <Button
            size="sm"
            variant="outline"
            className="h-7 border-amber-500/50 bg-amber-500/10 text-xs text-amber-100 hover:bg-amber-500/20"
            onClick={() => onFix?.(gap.scenario_name, "tier_definitions")}
          >
            <Layers className="mr-1 h-3 w-3" />
            Define tiers
          </Button>
        </div>
      )}

      {/* Missing resource metadata */}
      {gap.missing_resource_metadata && gap.missing_resource_metadata.length > 0 && (
        <div className="space-y-2 rounded border border-amber-500/30 bg-amber-500/5 p-3">
          <div className="flex items-center gap-2 text-sm">
            <Package className="h-4 w-4 text-amber-400" />
            <span className="font-medium text-amber-100">
              Missing resource metadata ({gap.missing_resource_metadata.length})
            </span>
          </div>
          <div className="flex flex-wrap gap-1">
            {gap.missing_resource_metadata.slice(0, 5).map((resource) => (
              <Badge key={resource} variant="outline" className="border-amber-500/50 bg-amber-500/10 text-[10px] text-amber-100">
                {resource}
              </Badge>
            ))}
            {gap.missing_resource_metadata.length > 5 && (
              <Badge variant="outline" className="border-amber-500/50 bg-amber-500/10 text-[10px] text-amber-100">
                +{gap.missing_resource_metadata.length - 5} more
              </Badge>
            )}
          </div>
          <Button
            size="sm"
            variant="outline"
            className="h-7 border-amber-500/50 bg-amber-500/10 text-xs text-amber-100 hover:bg-amber-500/20"
            onClick={() => onFix?.(gap.scenario_name, "resource_metadata")}
          >
            <Package className="mr-1 h-3 w-3" />
            Add resource metadata
          </Button>
        </div>
      )}

      {/* Missing scenario metadata */}
      {gap.missing_scenario_metadata && gap.missing_scenario_metadata.length > 0 && (
        <div className="space-y-2 rounded border border-amber-500/30 bg-amber-500/5 p-3">
          <div className="flex items-center gap-2 text-sm">
            <FileText className="h-4 w-4 text-amber-400" />
            <span className="font-medium text-amber-100">
              Missing scenario metadata ({gap.missing_scenario_metadata.length})
            </span>
          </div>
          <div className="flex flex-wrap gap-1">
            {gap.missing_scenario_metadata.slice(0, 5).map((scenario) => (
              <Badge key={scenario} variant="outline" className="border-amber-500/50 bg-amber-500/10 text-[10px] text-amber-100">
                {scenario}
              </Badge>
            ))}
            {gap.missing_scenario_metadata.length > 5 && (
              <Badge variant="outline" className="border-amber-500/50 bg-amber-500/10 text-[10px] text-amber-100">
                +{gap.missing_scenario_metadata.length - 5} more
              </Badge>
            )}
          </div>
          <Button
            size="sm"
            variant="outline"
            className="h-7 border-amber-500/50 bg-amber-500/10 text-xs text-amber-100 hover:bg-amber-500/20"
            onClick={() => onFix?.(gap.scenario_name, "scenario_metadata")}
          >
            <FileText className="mr-1 h-3 w-3" />
            Add scenario metadata
          </Button>
        </div>
      )}

      {/* Suggested actions */}
      {gap.suggested_actions && gap.suggested_actions.length > 0 && (
        <div className="space-y-1 text-xs text-muted-foreground">
          <p className="font-medium">Suggested actions:</p>
          <ul className="list-inside list-disc space-y-0.5">
            {gap.suggested_actions.map((action, idx) => (
              <li key={idx}>{action}</li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}

export function MetadataGapsPanel({ gaps, onFixGap }: MetadataGapsPanelProps) {
  if (!gaps || gaps.total_gaps === 0) {
    return (
      <Card className="border border-border/40 bg-background/40">
        <CardHeader className="pb-2">
          <CardTitle className="text-sm">Metadata Gaps</CardTitle>
        </CardHeader>
        <CardContent className="flex items-center gap-2 text-sm text-muted-foreground">
          <CheckCircle2 className="h-4 w-4 text-green-400" />
          <span>No metadata gaps detected. All scenarios have complete deployment configuration.</span>
        </CardContent>
      </Card>
    );
  }

  const gapEntries = Object.entries(gaps.gaps_by_scenario || {});

  return (
    <Card className="border border-amber-500/40 bg-background/40">
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <div>
            <CardTitle className="flex items-center gap-2 text-sm">
              <AlertTriangle className="h-4 w-4 text-amber-400" />
              Metadata Gaps Detected
            </CardTitle>
            <p className="mt-1 text-xs text-muted-foreground">
              {gaps.total_gaps} gap{gaps.total_gaps !== 1 ? "s" : ""} across {gapEntries.length} scenario
              {gapEntries.length !== 1 ? "s" : ""}
            </p>
          </div>
          <Badge variant="outline" className="border-amber-500/50 bg-amber-500/10 text-amber-100">
            {gaps.scenarios_missing_all} critical
          </Badge>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Recommendations */}
        {gaps.recommendations && gaps.recommendations.length > 0 && (
          <div className="rounded-lg border border-amber-500/30 bg-amber-500/5 p-3">
            <p className="mb-2 text-xs font-medium uppercase tracking-wide text-amber-100">Recommendations</p>
            <ul className="list-inside list-disc space-y-1 text-xs text-amber-200/90">
              {gaps.recommendations.map((rec, idx) => (
                <li key={idx}>{rec}</li>
              ))}
            </ul>
          </div>
        )}

        {/* Gap items */}
        <div className="space-y-3">
          {gapEntries.map(([scenarioName, gapInfo]) => (
            <GapItem key={scenarioName} gap={gapInfo} onFix={onFixGap} />
          ))}
        </div>

        {/* Missing tiers summary */}
        {gaps.missing_tiers && gaps.missing_tiers.length > 0 && (
          <div className="rounded-lg border border-border/30 bg-background/20 p-3">
            <p className="mb-2 text-xs font-medium uppercase tracking-wide text-muted-foreground">
              Commonly missing tiers
            </p>
            <div className="flex flex-wrap gap-1">
              {gaps.missing_tiers.map((tier) => (
                <Badge key={tier} variant="secondary" className="text-[10px]">
                  {tier}
                </Badge>
              ))}
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
