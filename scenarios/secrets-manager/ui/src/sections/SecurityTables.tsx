import { ShieldAlert, ShieldCheck } from "lucide-react";
import { SecretsRow } from "../components/ui/SecretsRow";
import { VulnerabilityItem } from "../components/ui/VulnerabilityItem";
import { LoadingSecretsRow, Skeleton } from "../components/ui/LoadingStates";
import type { VaultResourceStatus, SecurityVulnerability } from "../lib/api";

interface SecurityTablesProps {
  resourceStatuses: VaultResourceStatus[];
  vulnerabilities: SecurityVulnerability[];
  isVaultLoading: boolean;
  isVulnerabilityLoading: boolean;
  componentType: string;
  componentFilter: string;
  severityFilter: string;
  componentOptions: string[];
  scanId?: string;
  riskScore?: number;
  scanDuration?: number;
  onOpenResource?: (resourceName: string, secretKey?: string) => void;
  onComponentTypeChange: (value: string) => void;
  onComponentFilterChange: (value: string) => void;
  onSeverityFilterChange: (value: string) => void;
}

export const SecurityTables = ({
  resourceStatuses,
  vulnerabilities,
  isVaultLoading,
  isVulnerabilityLoading,
  componentType,
  componentFilter,
  severityFilter,
  componentOptions,
  scanId,
  riskScore,
  scanDuration,
  onOpenResource,
  onComponentTypeChange,
  onComponentFilterChange,
  onSeverityFilterChange
}: SecurityTablesProps) => (
  <section id="anchor-vulns" className="grid gap-6 lg:grid-cols-2">
    <div className="rounded-3xl border border-white/5 bg-white/5 p-6">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-white">Component Security Status</h2>
        <div className="flex items-center gap-2 text-xs text-white/60">
          <ShieldCheck className="h-4 w-4" />
          {isVaultLoading ? <Skeleton className="h-3 w-32" variant="text" /> : "Live data from vault and file scans"}
        </div>
      </div>
      <div className="mt-4 space-y-3">
        {isVaultLoading ? (
          <>
            <LoadingSecretsRow />
            <LoadingSecretsRow />
            <LoadingSecretsRow />
          </>
        ) : resourceStatuses.length === 0 ? (
          <div className="rounded-2xl border border-white/10 bg-white/5 p-6 text-center text-sm text-white/60">
            No resource scan results available yet.
          </div>
        ) : (
          resourceStatuses.slice(0, 6).map((status) => (
            <SecretsRow key={status.resource_name} status={status} onOpenResource={onOpenResource} />
          ))
        )}
      </div>
    </div>

    <div className="rounded-3xl border border-white/5 bg-white/5 p-6">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-white">Security Findings</h2>
        <div className="flex items-center gap-2 text-xs text-white/60">
          <ShieldAlert className="h-4 w-4" />
          {isVulnerabilityLoading ? <Skeleton className="h-3 w-24" variant="text" /> : `Showing ${vulnerabilities.length} results`}
        </div>
      </div>

      <div className="mt-4 space-y-4">
        <div className="grid gap-3 md:grid-cols-3">
          <div>
            <label htmlFor="componentType" className="text-xs text-white/60">
              Component type
            </label>
            <select
              id="componentType"
              className="mt-2 w-full rounded-2xl border border-white/10 bg-white/5 px-3 py-2 text-sm text-white"
              value={componentType}
              onChange={(event) => onComponentTypeChange(event.target.value)}
            >
              <option value="">All components</option>
              <option value="resource">Resources</option>
              <option value="scenario">Scenarios</option>
            </select>
          </div>
          <div>
            <label htmlFor="component" className="text-xs text-white/60">
              Component
            </label>
            <select
              id="component"
              className="mt-2 w-full rounded-2xl border border-white/10 bg-white/5 px-3 py-2 text-sm text-white"
              value={componentFilter}
              onChange={(event) => onComponentFilterChange(event.target.value)}
            >
              <option value="">All</option>
              {componentOptions.map((option) => (
                <option key={option} value={option}>
                  {option}
                </option>
              ))}
            </select>
          </div>
          <div>
            <label htmlFor="severity" className="text-xs text-white/60">
              Severity
            </label>
            <select
              id="severity"
              className="mt-2 w-full rounded-2xl border border-white/10 bg-white/5 px-3 py-2 text-sm text-white"
              value={severityFilter}
              onChange={(event) => onSeverityFilterChange(event.target.value)}
            >
              <option value="">All levels</option>
              <option value="critical">Critical</option>
              <option value="high">High</option>
              <option value="medium">Medium</option>
              <option value="low">Low</option>
            </select>
          </div>
        </div>

        <div className="rounded-2xl border border-white/10 bg-white/5 p-4 text-sm text-white/70">
          <p>
            Scan ID: <span className="font-mono text-white">{scanId ?? "pending"}</span>
          </p>
          <p>Risk score: {riskScore ?? 0}</p>
          <p>Duration: {scanDuration ?? 0} ms</p>
        </div>
      </div>

      <div className="mt-4 space-y-3">
        {isVulnerabilityLoading ? (
          <>
            <Skeleton className="h-32 w-full" />
            <Skeleton className="h-32 w-full" />
            <Skeleton className="h-32 w-full" />
          </>
        ) : vulnerabilities.length === 0 ? (
          <div className="rounded-2xl border border-emerald-500/40 bg-emerald-500/10 p-6 text-center text-sm text-emerald-100">
            <ShieldCheck className="mx-auto mb-2 h-6 w-6" />
            No vulnerabilities found for the selected filters.
          </div>
        ) : (
          vulnerabilities.slice(0, 6).map((vuln) => (
            <VulnerabilityItem key={vuln.id} vuln={vuln} />
          ))
        )}
      </div>
    </div>
  </section>
);
