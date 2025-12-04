import { useState } from "react";
import { ChevronDown, ChevronRight } from "lucide-react";
import { StatusGrid } from "./StatusGrid";
import { Skeleton } from "../components/ui/LoadingStates";
import type { HealthResponse, VaultSecretsStatus, ComplianceResponse, VulnerabilityResponse } from "../lib/api";

interface SnapshotPanelProps {
  heroStats?: {
    overall_score: number;
    readiness_label: string;
    risk_score: number;
    confidence: number;
    vault_configured: number;
    vault_total: number;
    missing_secrets: number;
  };
  updatedAt?: string;
  healthData?: HealthResponse;
  vaultData?: VaultSecretsStatus;
  complianceData?: ComplianceResponse;
  vulnerabilityData?: VulnerabilityResponse;
  isLoading: boolean;
}

export const SnapshotPanel = ({
  heroStats,
  updatedAt,
  healthData,
  vaultData,
  complianceData,
  vulnerabilityData,
  isLoading
}: SnapshotPanelProps) => {
  const [expanded, setExpanded] = useState(false);
  const keyStats = [
    { label: "Overall", value: heroStats?.overall_score ? `${heroStats.overall_score}%` : "—" },
    { label: "Missing secrets", value: heroStats ? `${heroStats.missing_secrets}` : "—" },
    { label: "Risk", value: heroStats ? `${heroStats.risk_score}` : "—" }
  ];

  return (
    <section className="rounded-3xl border border-white/10 bg-white/5 p-4">
      <button
        type="button"
        className="flex w-full items-center justify-between gap-3"
        onClick={() => setExpanded((value) => !value)}
      >
        <div className="flex items-center gap-3">
          {expanded ? (
            <ChevronDown className="h-5 w-5 text-emerald-300" />
          ) : (
            <ChevronRight className="h-5 w-5 text-emerald-300" />
          )}
          <div>
            <p className="text-xs uppercase tracking-[0.3em] text-emerald-300">System Snapshot</p>
            <p className="text-sm text-white/70">Key stats collapsed by default to reduce clutter.</p>
          </div>
        </div>
        <div className="rounded-full border border-white/10 px-3 py-1 text-[11px] text-white/60">
          Updated {updatedAt ? new Date(updatedAt).toLocaleTimeString() : "—"}
        </div>
      </button>

      {expanded ? (
        <div className="mt-3 grid gap-3 sm:grid-cols-3">
          {keyStats.map((stat) => (
            <div key={stat.label} className="rounded-2xl border border-white/10 bg-black/30 px-3 py-2 text-left">
              <p className="text-[11px] uppercase tracking-[0.2em] text-white/60">{stat.label}</p>
              {isLoading ? (
                <Skeleton className="mt-1 h-6 w-16" variant="text" />
              ) : (
                <p className="mt-1 text-lg font-semibold text-white">{stat.value}</p>
              )}
            </div>
          ))}
        </div>
      ) : (
        <div className="mt-3 flex flex-wrap items-center gap-3 rounded-2xl border border-white/10 bg-black/30 px-3 py-2 text-sm text-white/80">
          {isLoading ? (
            <Skeleton className="h-6 w-32" variant="text" />
          ) : (
            <>
              <span>Overall {keyStats[0]?.value}</span>
              <span className="text-white/60">•</span>
              <span>Missing {keyStats[1]?.value}</span>
              <span className="text-white/60">•</span>
              <span>Risk {keyStats[2]?.value}</span>
            </>
          )}
        </div>
      )}

      {expanded ? (
        <div className="mt-4">
          <StatusGrid
            healthData={healthData}
            vaultData={vaultData}
            complianceData={complianceData}
            vulnerabilityData={vulnerabilityData}
            isHealthLoading={isLoading}
            isVaultLoading={isLoading}
            isComplianceLoading={isLoading}
            isVulnerabilityLoading={isLoading}
          />
        </div>
      ) : null}
    </section>
  );
};
