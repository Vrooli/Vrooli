import { CheckCircle2 } from "lucide-react";
import { StatCard } from "../components/ui/StatCard";
import { SeverityBadge } from "../components/ui/SeverityBadge";
import { LoadingStatCard, Skeleton } from "../components/ui/LoadingStates";

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
}

export const ComplianceOverview = ({
  overallScore,
  configuredComponents,
  securityScore,
  vaultHealth,
  vulnerabilitySummary,
  missingSecrets,
  isComplianceLoading,
  isVaultLoading
}: ComplianceOverviewProps) => (
  <section className="grid gap-4 lg:grid-cols-3">
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
  </section>
);
