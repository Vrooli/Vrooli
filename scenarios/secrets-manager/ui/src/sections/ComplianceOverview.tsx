import { CheckCircle2 } from "lucide-react";
import { StatCard } from "../components/ui/StatCard";
import { SeverityBadge } from "../components/ui/SeverityBadge";
import { HelpDialog } from "../components/ui/HelpDialog";
import { LoadingStatCard, Skeleton } from "../components/ui/LoadingStates";
import { Button } from "../components/ui/button";

interface VulnerabilitySummary {
  critical: number;
  high: number;
  medium: number;
  low: number;
}

interface MissingSecret {
  resource_name: string;
  secret_name: string;
  description: string;
}

interface ComplianceOverviewProps {
  overallScore?: number;
  configuredComponents?: number;
  securityScore?: number;
  vaultHealth?: number;
  vulnerabilitySummary: VulnerabilitySummary;
  missingSecrets: MissingSecret[];
  isComplianceLoading: boolean;
  isVaultLoading: boolean;
  onOpenResource?: (resourceName: string, secretKey?: string) => void;
}

export const ComplianceOverview = ({
  overallScore,
  configuredComponents,
  securityScore,
  vaultHealth,
  vulnerabilitySummary,
  missingSecrets,
  isComplianceLoading,
  isVaultLoading,
  onOpenResource
}: ComplianceOverviewProps) => (
  <section>
    <div className="mb-4 flex items-center gap-3">
      <div className="flex-1">
        <div className="flex items-center gap-2">
          <h2 className="text-xl font-semibold text-white">Security & Compliance Overview</h2>
          <HelpDialog title="Compliance Metrics">
        <p>
          This section provides a unified view of your security posture across all resources.
        </p>
        <div className="mt-3 space-y-2">
          <p><strong className="text-white">Key Metrics:</strong></p>
          <ul className="ml-4 space-y-1">
            <li><strong className="text-white">Overall Score:</strong> Weighted combination of security score and vault health</li>
            <li><strong className="text-white">Configured Components:</strong> Number of resources with complete secret configurations</li>
            <li><strong className="text-white">Security Score:</strong> Based on vulnerability severity distribution and remediation progress</li>
            <li><strong className="text-white">Vault Health:</strong> Percentage of required secrets properly stored and validated</li>
          </ul>
        </div>
        <div className="mt-3">
          <p><strong className="text-white">Vulnerability Severities:</strong></p>
          <ul className="ml-4 mt-1 space-y-1">
            <li><strong className="text-red-200">Critical:</strong> Immediate security risk, requires urgent attention</li>
            <li><strong className="text-amber-200">High:</strong> Significant risk, should be addressed soon</li>
            <li><strong className="text-yellow-200">Medium:</strong> Moderate risk, address in normal workflow</li>
            <li><strong className="text-emerald-200">Low:</strong> Minor improvements, low priority</li>
          </ul>
        </div>
      </HelpDialog>
        </div>
        <p className="mt-1 text-sm text-white/60">
          Aggregated health metrics and vulnerability summary
        </p>
      </div>
    </div>
    <div className="grid gap-4 lg:grid-cols-3">
    <div className="space-y-4 rounded-3xl border border-white/5 bg-white/5 p-6 lg:col-span-2">
      {isComplianceLoading ? (
        <div className="flex flex-wrap items-center gap-4">
          <LoadingStatCard />
          <LoadingStatCard />
          <LoadingStatCard />
          <LoadingStatCard />
        </div>
      ) : (
        <div className="flex flex-wrap items-center gap-4">
          <StatCard label="Overall Score" value={`${overallScore ?? 0}%`} />
          <StatCard label="Configured Components" value={`${configuredComponents ?? 0}`} />
          <StatCard label="Security Score" value={`${securityScore ?? 0}%`} />
          <StatCard label="Vault Health" value={`${vaultHealth ?? 0}%`} />
        </div>
      )}
      <div className="grid gap-4 md:grid-cols-2">
        <div className="rounded-2xl border border-white/5 bg-black/30 p-4">
          <p className="text-xs uppercase tracking-[0.2em] text-white/60">Vulnerability Mix</p>
          {isComplianceLoading ? (
            <div className="mt-4 space-y-3">
              <Skeleton className="h-6 w-full" />
              <Skeleton className="h-6 w-full" />
              <Skeleton className="h-6 w-full" />
              <Skeleton className="h-6 w-full" />
            </div>
          ) : (
            <div className="mt-4 space-y-3">
              {(["critical", "high", "medium", "low"] as const).map((level) => (
                <div key={level} className="flex items-center justify-between text-sm">
                  <div className="flex items-center gap-2">
                    <SeverityBadge severity={level} />
                    <span className="capitalize text-white/80">{level}</span>
                  </div>
                  <span className="text-lg font-semibold">
                    {vulnerabilitySummary[level] ? vulnerabilitySummary[level].toString() : "0"}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>
        <div className="rounded-2xl border border-white/5 bg-black/30 p-4">
          <p className="text-xs uppercase tracking-[0.2em] text-white/60">Missing Secrets</p>
          {isVaultLoading ? (
            <div className="mt-4 space-y-3">
              <Skeleton className="h-20 w-full" />
              <Skeleton className="h-20 w-full" />
            </div>
          ) : missingSecrets.length === 0 ? (
            <div className="mt-4 flex items-center gap-3 rounded-2xl border border-emerald-500/40 bg-emerald-500/10 px-4 py-4 text-sm text-emerald-100">
              <CheckCircle2 className="h-5 w-5" />
              All required secrets are configured.
            </div>
          ) : (
            <div className="mt-3 space-y-3">
              {missingSecrets.slice(0, 4).map((secret) => (
                <div key={`${secret.resource_name}-${secret.secret_name}`} className="rounded-xl border border-amber-400/30 bg-amber-400/5 p-3">
                  <p className="text-sm font-semibold text-white">{secret.resource_name}</p>
                  <p className="text-xs text-white/70">{secret.secret_name}</p>
                  <p className="mt-1 text-xs text-amber-100">{secret.description}</p>
                  {onOpenResource ? (
                    <div className="mt-2">
                      <Button
                        size="sm"
                        variant="outline"
                        className="text-[11px]"
                        onClick={() => onOpenResource(secret.resource_name, secret.secret_name)}
                      >
                        Fix in workbench
                      </Button>
                    </div>
                  ) : null}
                </div>
              ))}
              {missingSecrets.length > 4 ? (
                <p className="text-xs text-white/60">+{missingSecrets.length - 4} more secrets need attention</p>
              ) : null}
            </div>
          )}
        </div>
      </div>
    </div>

    <div className="rounded-3xl border border-white/10 bg-black/30 p-6">
      <p className="text-xs uppercase tracking-[0.2em] text-white/60">Scan Inputs</p>
      <div className="mt-4 space-y-4">
        <p className="text-sm text-white/70">
          Use the filters in the Security Findings section to refine vulnerability results.
        </p>
      </div>
    </div>
    </div>
  </section>
);
