import { Shield, Play, CheckCircle2, Server, Globe, Key, Network } from "lucide-react";
import { Button } from "../ui/button";
import { Alert } from "../ui/alert";
import { LoadingState } from "../ui/spinner";
import type { useDeployment } from "../../hooks/useDeployment";

interface StepPreflightProps {
  deployment: ReturnType<typeof useDeployment>;
}

// Placeholder checks - will be replaced with real API data
const PREFLIGHT_CHECKS = [
  { id: "ssh", label: "SSH Access", description: "Verify SSH connection to target server", icon: Key },
  { id: "os", label: "Operating System", description: "Confirm Ubuntu 22.04+ is running", icon: Server },
  { id: "dns", label: "DNS Resolution", description: "Check domain points to server IP", icon: Globe },
  { id: "ports", label: "Port Availability", description: "Verify required ports are not in use", icon: Network },
];

export function StepPreflight({ deployment }: StepPreflightProps) {
  const {
    preflightPassed,
    preflightError,
    isRunningPreflight,
    runPreflight,
    parsedManifest,
  } = deployment;

  return (
    <div className="space-y-6">
      {/* Info Banner */}
      <Alert variant="info" title="VPS Preflight Checks">
        Before deployment, we verify your target server is properly configured and accessible.
      </Alert>

      {/* Run Preflight Button */}
      <div className="flex items-center gap-3">
        <Button
          onClick={runPreflight}
          disabled={isRunningPreflight || !parsedManifest.ok}
        >
          <Play className="h-4 w-4 mr-1.5" />
          {isRunningPreflight ? "Running Checks..." : "Run Preflight Checks"}
        </Button>
      </div>

      {/* Loading State */}
      {isRunningPreflight && (
        <LoadingState message="Running preflight checks..." />
      )}

      {/* Error */}
      {preflightError && (
        <Alert variant="error" title="Preflight Failed">
          {preflightError}
        </Alert>
      )}

      {/* Success */}
      {preflightPassed === true && !isRunningPreflight && (
        <Alert variant="success" title="Preflight Passed">
          <div className="flex items-center gap-2">
            <Shield className="h-4 w-4" />
            All checks passed. Your server is ready for deployment.
          </div>
        </Alert>
      )}

      {/* Preflight Checks List */}
      <div className="space-y-3">
        <h4 className="text-sm font-medium text-slate-300">Preflight Checks</h4>
        <ul className="space-y-2">
          {PREFLIGHT_CHECKS.map((check) => {
            const Icon = check.icon;
            const isPassed = preflightPassed === true;

            return (
              <li
                key={check.id}
                className="flex items-center gap-4 p-3 rounded-lg bg-slate-800/50 border border-slate-700"
              >
                <div className="flex-shrink-0">
                  <div className={`w-8 h-8 rounded-full flex items-center justify-center ${
                    isPassed ? "bg-emerald-500/20" : "bg-slate-700"
                  }`}>
                    <Icon className={`h-4 w-4 ${isPassed ? "text-emerald-400" : "text-slate-400"}`} />
                  </div>
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium text-slate-200">{check.label}</p>
                  <p className="text-xs text-slate-400">{check.description}</p>
                </div>
                <div className="flex-shrink-0">
                  {isPassed ? (
                    <CheckCircle2 className="h-5 w-5 text-emerald-400" />
                  ) : (
                    <div className="w-5 h-5 rounded-full border-2 border-slate-600" />
                  )}
                </div>
              </li>
            );
          })}
        </ul>
      </div>

      {/* Note about API */}
      <p className="text-xs text-slate-500">
        Note: Full preflight checks will be available when the preflight API is implemented.
        Current checks are simulated for UI demonstration.
      </p>
    </div>
  );
}
