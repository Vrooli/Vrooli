import { useEffect } from "react";
import { ListOrdered, RefreshCw, CheckCircle2 } from "lucide-react";
import { Button } from "../ui/button";
import { Alert } from "../ui/alert";
import { LoadingState } from "../ui/spinner";
import { selectors } from "../../consts/selectors";
import type { useDeployment } from "../../hooks/useDeployment";

interface StepPlanProps {
  deployment: ReturnType<typeof useDeployment>;
}

export function StepPlan({ deployment }: StepPlanProps) {
  const {
    plan,
    planError,
    isPlanning,
    generatePlan,
    parsedManifest,
  } = deployment;

  // Auto-generate plan when entering this step
  useEffect(() => {
    if (plan === null && !planError && !isPlanning) {
      generatePlan();
    }
  }, []); // Only on mount

  return (
    <div className="space-y-6">
      {/* Loading State */}
      {isPlanning && (
        <LoadingState message="Generating deployment plan..." />
      )}

      {/* Error */}
      {planError && (
        <Alert variant="error" title="Plan Generation Failed">
          <div className="space-y-2">
            <p>{planError}</p>
            <Button
              data-testid={selectors.manifest.planButton}
              variant="outline"
              size="sm"
              onClick={generatePlan}
              disabled={!parsedManifest.ok}
            >
              <RefreshCw className="h-4 w-4 mr-1.5" />
              Retry
            </Button>
          </div>
        </Alert>
      )}

      {/* Plan Steps */}
      {plan !== null && !isPlanning && (
        <div data-testid={selectors.manifest.planResult} className="space-y-4">
          <Alert variant="info" title="Deployment Plan Ready">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <ListOrdered className="h-4 w-4" />
                {plan.length} step{plan.length !== 1 ? "s" : ""} will be executed during deployment.
              </div>
              <Button
                data-testid={selectors.manifest.planButton}
                variant="ghost"
                size="sm"
                onClick={generatePlan}
                disabled={!parsedManifest.ok}
                className="text-slate-400 hover:text-slate-200"
              >
                <RefreshCw className="h-3.5 w-3.5 mr-1" />
                Regenerate
              </Button>
            </div>
          </Alert>

          <div className="space-y-3">
            <h4 className="text-sm font-medium text-slate-300">Deployment Steps</h4>
            <ol className="space-y-3">
              {plan.map((step, index) => (
                <li
                  key={step.id}
                  className="flex gap-4 p-4 rounded-lg bg-slate-800/50 border border-slate-700"
                >
                  {/* Step Number */}
                  <div className="flex-shrink-0">
                    <div className="w-8 h-8 rounded-full bg-slate-700 flex items-center justify-center">
                      <span className="text-sm font-medium text-slate-300">
                        {index + 1}
                      </span>
                    </div>
                  </div>

                  {/* Step Content */}
                  <div className="min-w-0 flex-1">
                    <h5 className="text-sm font-medium text-slate-200">
                      {step.title}
                    </h5>
                    <p className="mt-1 text-sm text-slate-400">
                      {step.description}
                    </p>
                  </div>

                  {/* Status (will be used during actual deployment) */}
                  <div className="flex-shrink-0">
                    <div className="w-6 h-6 rounded-full bg-slate-700/50 flex items-center justify-center">
                      <CheckCircle2 className="h-4 w-4 text-slate-500" />
                    </div>
                  </div>
                </li>
              ))}
            </ol>
          </div>
        </div>
      )}
    </div>
  );
}
