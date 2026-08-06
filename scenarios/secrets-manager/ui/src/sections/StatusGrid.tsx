import { Activity, Database, TerminalSquare, ShieldCheck } from "lucide-react";
import { StatusTile } from "../components/ui/StatusTile";
import { LoadingStatusTile } from "../components/ui/LoadingStates";
import { formatTimestamp } from "../lib/formatters";
import type { HealthResponse, CredentialCoverageStatus, ComplianceResponse, VulnerabilityResponse } from "../lib/api";

interface StatusGridProps {
  healthData?: HealthResponse;
  credentialData?: CredentialCoverageStatus;
  complianceData?: ComplianceResponse;
  vulnerabilityData?: VulnerabilityResponse;
  isHealthLoading: boolean;
  isCredentialLoading: boolean;
  isComplianceLoading: boolean;
  isVulnerabilityLoading: boolean;
}

export const StatusGrid = ({
  healthData,
  credentialData,
  complianceData,
  vulnerabilityData,
  isHealthLoading,
  isCredentialLoading,
  isComplianceLoading,
  isVulnerabilityLoading
}: StatusGridProps) => {
  const missingSecrets = credentialData?.missing_secrets ?? [];

  const databaseConnected = healthData?.dependencies?.database?.connected ?? false;
  const lastScanTimestamp = complianceData?.last_updated ?? credentialData?.last_updated;
  const riskScore = vulnerabilityData?.risk_score;

  const tiles: Array<{
    key: string;
    loading: boolean;
    props: Parameters<typeof StatusTile>[0];
  }> = [
    {
      key: "api",
      loading: isHealthLoading,
      props: {
        icon: TerminalSquare,
        label: "API Terminal",
        value: String(healthData?.status ?? "unknown").toUpperCase(),
        meta: healthData?.service,
        intent: healthData?.status === "healthy" ? "good" : "warn"
      }
    },
    {
      key: "database",
      loading: isHealthLoading,
      props: {
        icon: Database,
        label: "Database",
        value: databaseConnected ? "CONNECTED" : "DISCONNECTED",
        meta: `v${healthData?.version ?? "1.0"}`,
        intent: databaseConnected ? "good" : "warn"
      }
    },
    {
      key: "credentials",
      loading: isCredentialLoading,
      props: {
        icon: ShieldCheck,
        label: "Credential Coverage",
        value: `${credentialData?.configured_resources ?? 0}/${credentialData?.total_resources ?? 0}`,
        meta: `${missingSecrets.length} missing`,
        intent: missingSecrets.length > 0 ? "warn" : "good"
      }
    },
    {
      key: "scan",
      loading: isComplianceLoading || isVulnerabilityLoading,
      props: {
        icon: Activity,
        label: "Last Scan",
        value: formatTimestamp(lastScanTimestamp),
        meta: `${vulnerabilityData?.scan_duration ?? 0} ms`,
        intent: riskScore && riskScore > 60 ? "danger" : "info"
      }
    }
  ];

  return (
    <section className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
      {tiles.map((tile) =>
        tile.loading ? <LoadingStatusTile key={tile.key} /> : <StatusTile key={tile.key} {...tile.props} />
      )}
    </section>
  );
};
