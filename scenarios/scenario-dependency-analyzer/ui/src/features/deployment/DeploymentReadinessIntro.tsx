import { Info, LifeBuoy } from "lucide-react";

import { Button } from "../../components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "../../components/ui/card";
import { statusTone } from "../../theme/status";
import type { DeploymentTierOption } from "./deploymentStatus";

interface DeploymentReadinessIntroProps {
  apiError: string | null;
  targetTier: string;
  tierOptions: DeploymentTierOption[];
  onSelectTargetTier: (tier: string) => void;
}

export function DeploymentReadinessIntro({
  apiError,
  targetTier,
  tierOptions,
  onSelectTargetTier
}: DeploymentReadinessIntroProps) {
  return (
    <Card className="border border-border/60 bg-background/50">
      <CardHeader className="pb-3">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <CardTitle className="text-lg">Deployment Readiness</CardTitle>
            <p className="text-xs text-muted-foreground">
              Goal: prepare scenarios for <strong>Tier 2 desktop</strong> (portable app with UI+API+resources) and beyond.
              Pick a target tier, scan, then fix blockers with the inline guide.
            </p>
          </div>
          <div className="flex items-center gap-2 text-xs text-muted-foreground">
            <Info className="h-3.5 w-3.5" aria-hidden="true" />
            <span>Need a refresher later? Hover the tips and help buttons for definitions.</span>
          </div>
        </div>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        {apiError ? (
          <div className={`rounded border p-3 text-xs ${statusTone("danger").panel}`}>
            <p className="font-medium">API unreachable</p>
            <p className="mt-1">
              {apiError} Start the scenario via <code>vrooli scenario run scenario-dependency-analyzer</code>. If you use a custom port,
              set <code>VITE_API_PORT</code> before running <code>npm run dev</code>, or configure proxy metadata to point the UI at the right base.
            </p>
          </div>
        ) : null}
        <div className="flex flex-wrap items-center gap-2">
          <label className="text-xs text-muted-foreground">Target tier</label>
          <div className="flex flex-wrap gap-2">
            {tierOptions.map((tier) => (
              <Button
                key={tier.value}
                size="sm"
                variant={targetTier === tier.value ? "secondary" : "outline"}
                className="h-8 text-xs"
                onClick={() => onSelectTargetTier(tier.value)}
              >
                {tier.label}
              </Button>
            ))}
          </div>
          <div className="flex items-center gap-2 text-xs text-muted-foreground">
            <LifeBuoy className="h-3.5 w-3.5" aria-hidden="true" />
            <button
              className="underline-offset-2 hover:underline"
              onClick={() => window.open("/docs/deployment/tiers", "_blank")}
            >
              View tier definitions
            </button>
          </div>
        </div>
        <div className="flex flex-wrap gap-2 text-xs text-muted-foreground">
          <span className="font-medium text-foreground">What “ready” means:</span>
          <span>Deployment metadata exists for the target tier, no blocking dependencies, and tier fitness is acceptable.</span>
          <span className="text-primary">“Scan & Apply” writes inferred metadata to .vrooli/service.json.</span>
        </div>
      </CardContent>
    </Card>
  );
}
