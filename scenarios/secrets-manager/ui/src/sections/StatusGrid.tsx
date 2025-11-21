import { Activity, Database, TerminalSquare, ShieldCheck } from "lucide-react";
import { StatusTile } from "../components/ui/StatusTile";
import { LoadingStatusTile } from "../components/ui/LoadingStates";
import { formatTimestamp } from "../lib/formatters";
import type { HealthResponse, VaultSecretsStatus, ComplianceResponse, VulnerabilityResponse } from "../lib/api";

interface StatusGridProps {
  healthData?: HealthResponse;
  vaultData?: VaultSecretsStatus;
  complianceData?: ComplianceResponse;
  vulnerabilityData?: VulnerabilityResponse;
  isHealthLoading: boolean;
  isVaultLoading: boolean;
  isComplianceLoading: boolean;
  isVulnerabilityLoading: boolean;
}

export const StatusGrid = ({
  healthData,
  vaultData,
  complianceData,
  vulnerabilityData,
  isHealthLoading,
  isVaultLoading,
  isComplianceLoading,
  isVulnerabilityLoading
}: StatusGridProps) => {
  const missingSecrets = vaultData?.missing_secrets ?? [];

  return (
    <section className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
      {isHealthLoading ? (
        <LoadingStatusTile />
      ) : (
        <StatusTile
          icon={TerminalSquare}
          label="API Terminal"
          value={String(healthData?.status ?? "unknown").toUpperCase()}
          meta={healthData?.service}
          intent={healthData?.status === "healthy" ? "good" : "warn"}
        />
      )}
      {isHealthLoading ? (
        <LoadingStatusTile />
      ) : (
        <StatusTile
          icon={Database}
          label="Database"
          value={
            healthData?.dependencies?.database?.connected
              ? "CONNECTED"
              : "DISCONNECTED"
          }
          meta={`v${healthData?.version ?? "1.0"}`}
          intent={
            healthData?.dependencies?.database?.connected ? "good" : "warn"
          }
        />
      )}
      {isVaultLoading ? (
        <LoadingStatusTile />
      ) : (
        <StatusTile
          icon={ShieldCheck}
          label="Vault Coverage"
          value={`${vaultData?.configured_resources ?? 0}/${vaultData?.total_resources ?? 0}`}
          meta={`${missingSecrets.length} missing`}
          intent={missingSecrets.length > 0 ? "warn" : "good"}
        />
      )}
      {isComplianceLoading || isVulnerabilityLoading ? (
        <LoadingStatusTile />
      ) : (
        <StatusTile
          icon={Activity}
          label="Last Scan"
          value={formatTimestamp(complianceData?.last_updated ?? vaultData?.last_updated)}
          meta={`${vulnerabilityData?.scan_duration ?? 0} ms`}
          intent={vulnerabilityData?.risk_score && vulnerabilityData.risk_score > 60 ? "danger" : "info"}
        />
      )}
    </section>
  );
};
