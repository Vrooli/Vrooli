import { StatCard } from "../components/ui/StatCard";
import { SeverityBadge } from "../components/ui/SeverityBadge";
import { HelpDialog } from "../components/ui/HelpDialog";
import { LoadingStatCard, Skeleton } from "../components/ui/LoadingStates";

interface VulnerabilitySummary {
  critical: number;
  high: number;
  medium: number;
  low: number;
}

interface ComplianceOverviewProps {
  overallScore?: number;
  configuredComponents?: number;
  securityScore?: number;
  vaultHealth?: number;
  vulnerabilitySummary: VulnerabilitySummary;
  isComplianceLoading: boolean;
}

export const ComplianceOverview = ({
  overallScore,
  configuredComponents,
  securityScore,
  vaultHealth,
  vulnerabilitySummary,
  isComplianceLoading
}: ComplianceOverviewProps) => (
  <section id="anchor-compliance">
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
      <div className="rounded-2xl border border-white/5 bg-black/30 p-4">
        <p className="text-xs uppercase tracking-[0.2em] text-white/60">Vulnerability Mix</p>
        {isComplianceLoading ? (
          <div className="mt-4 grid gap-4 sm:grid-cols-2 md:grid-cols-4">
            <Skeleton className="h-16 w-full" />
            <Skeleton className="h-16 w-full" />
            <Skeleton className="h-16 w-full" />
            <Skeleton className="h-16 w-full" />
          </div>
        ) : (
          <div className="mt-4 grid gap-4 sm:grid-cols-2 md:grid-cols-4">
            {(["critical", "high", "medium", "low"] as const).map((level) => (
              <div key={level} className="flex flex-col items-center rounded-xl border border-white/10 bg-white/5 p-3">
                <SeverityBadge severity={level} />
                <span className="mt-2 text-2xl font-semibold text-white">
                  {vulnerabilitySummary[level] ? vulnerabilitySummary[level].toString() : "0"}
                </span>
                <span className="text-xs capitalize text-white/60">{level}</span>
              </div>
            ))}
          </div>
        )}
        <p className="mt-4 text-xs text-white/50">
          View the Security Findings table below for detailed vulnerability information and remediation actions.
        </p>
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
