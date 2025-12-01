import { useMemo } from "react";
import { RefreshCcw } from "lucide-react";
import { Header } from "./sections/Header";
import { OrientationHub } from "./sections/OrientationHub";
import { TierReadiness } from "./sections/TierReadiness";
import { ResourceWorkbench } from "./sections/ResourceWorkbench";
import { StatusGrid } from "./sections/StatusGrid";
import { ComplianceOverview } from "./sections/ComplianceOverview";
import { SecurityTables } from "./sections/SecurityTables";
import { ResourcePanel } from "./features/resource-panel/ResourcePanel";
import { useSecretsData } from "./hooks/useSecretsData";
import { useVulnerabilities } from "./hooks/useVulnerabilities";
import { useResourcePanel } from "./hooks/useResourcePanel";
import { useJourneys } from "./hooks/useJourneys";
import type { JourneyId } from "./features/journeys/journeySteps";

export default function App() {
  const {
    healthQuery,
    vaultQuery,
    complianceQuery,
    orientationQuery,
    isRefreshing,
    isInitialLoading,
    refreshAll
  } = useSecretsData();

  const {
    vulnerabilityQuery,
    componentType,
    componentFilter,
    severityFilter,
    componentOptions,
    setComponentType,
    setComponentFilter,
    setSeverityFilter
  } = useVulnerabilities(vaultQuery.data);

  const {
    activeResource,
    selectedSecretKey,
    strategyTier,
    strategyHandling,
    strategyPrompt,
    strategyDescription,
    resourceDetailQuery,
    openResourcePanel,
    closeResourcePanel,
    setSelectedSecretKey,
    setStrategyTier,
    setStrategyHandling,
    setStrategyPrompt,
    setStrategyDescription,
    handleSecretUpdate,
    handleStrategyApply,
    handleVulnerabilityStatus
  } = useResourcePanel();

  const orientationData = orientationQuery.data;
  const heroStats = orientationData?.hero_stats;
  const journeyCards = orientationData?.journeys ?? [];
  const tierReadiness = orientationData?.tier_readiness ?? [];
  const resourceInsights = orientationData?.resource_insights ?? [];

  const topResourceNeedingAttention = useMemo(() => {
    if (resourceInsights.length > 0) {
      return resourceInsights[0].resource_name;
    }
    const resourceStatuses = vaultQuery.data?.resource_statuses ?? [];
    return resourceStatuses.find((status) => status.secrets_missing > 0)?.resource_name;
  }, [resourceInsights, vaultQuery.data]);

  const {
    activeJourney,
    journeyStep,
    journeySteps,
    handleJourneySelect,
    handleJourneyExit,
    handleJourneyNext,
    handleJourneyBack
  } = useJourneys({
    heroStats,
    orientationData,
    tierReadiness,
    topResourceNeedingAttention,
    onOpenResource: openResourcePanel,
    onRefetchVulnerabilities: () => vulnerabilityQuery.refetch()
  });

  const vulnerabilitySummary = {
    critical: complianceQuery.data?.vulnerability_summary?.critical ?? 0,
    high: complianceQuery.data?.vulnerability_summary?.high ?? 0,
    medium: complianceQuery.data?.vulnerability_summary?.medium ?? 0,
    low: complianceQuery.data?.vulnerability_summary?.low ?? 0
  };

  const missingSecrets = vaultQuery.data?.missing_secrets ?? [];
  const resourceStatuses = vaultQuery.data?.resource_statuses ?? [];
  const vulnerabilities = vulnerabilityQuery.data?.vulnerabilities ?? [];
  const topMissingSecret = missingSecrets[0];
  const priorityResource = topMissingSecret?.resource_name ?? topResourceNeedingAttention ?? null;
  const prioritySecretKey = topMissingSecret?.secret_name;
  const priorityMissingCount = heroStats?.missing_secrets ?? missingSecrets.length;

  const activeJourneyCard = journeyCards.find((card) => card.id === activeJourney);

  const handleJourneySelectTyped = (journeyId: JourneyId) => {
    handleJourneySelect(journeyId);
  };

  return (
    <div className="relative min-h-screen bg-slate-950 text-slate-50">
      <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_top,_rgba(16,185,129,0.15),_transparent_50%),radial-gradient(circle_at_bottom,_rgba(59,130,246,0.15),_transparent_60%)]" />

      {/* Initial loading overlay */}
      {isInitialLoading && !healthQuery.data && !vaultQuery.data && !complianceQuery.data && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/95 backdrop-blur-sm">
          <div className="rounded-3xl border border-emerald-400/40 bg-emerald-400/5 px-8 py-6 shadow-2xl shadow-emerald-500/20">
            <div className="flex items-center gap-4">
              <RefreshCcw className="h-8 w-8 animate-spin text-emerald-400" />
              <div>
                <p className="text-lg font-semibold text-white">Loading Security Dashboard</p>
                <p className="text-sm text-white/60">Fetching vault status, compliance data, and vulnerability scans...</p>
              </div>
            </div>
          </div>
        </div>
      )}

      <main className="relative mx-auto flex max-w-6xl flex-col gap-8 px-6 py-10">
        <Header
          isInitialLoading={isInitialLoading}
          isRefreshing={isRefreshing}
          totalFindings={vulnerabilityQuery.data?.total_count}
          onRefresh={refreshAll}
        />

        <OrientationHub
          heroStats={heroStats}
          vulnerabilityInsights={orientationData?.vulnerability_insights ?? []}
          journeyCards={journeyCards}
          activeJourneyCard={activeJourneyCard}
          activeJourney={activeJourney}
          journeySteps={journeySteps}
          journeyStep={journeyStep}
          updatedAt={orientationData?.updated_at}
          isLoading={orientationQuery.isLoading}
          priorityResource={priorityResource}
          prioritySecretKey={prioritySecretKey}
          missingCount={priorityMissingCount}
          onOpenResource={openResourcePanel}
          onJourneySelect={handleJourneySelectTyped}
          onJourneyExit={handleJourneyExit}
          onJourneyNext={handleJourneyNext}
          onJourneyBack={handleJourneyBack}
        />

        <TierReadiness
          tierReadiness={tierReadiness}
          isLoading={orientationQuery.isLoading}
          onOpenResource={openResourcePanel}
          resourceInsights={resourceInsights}
        />

        <ResourceWorkbench
          resourceInsights={resourceInsights}
          isLoading={orientationQuery.isLoading}
          onOpenResource={openResourcePanel}
        />

        <StatusGrid
          healthData={healthQuery.data}
          vaultData={vaultQuery.data}
          complianceData={complianceQuery.data}
          vulnerabilityData={vulnerabilityQuery.data}
          isHealthLoading={healthQuery.isLoading}
          isVaultLoading={vaultQuery.isLoading}
          isComplianceLoading={complianceQuery.isLoading}
          isVulnerabilityLoading={vulnerabilityQuery.isLoading}
        />

        <ComplianceOverview
          overallScore={complianceQuery.data?.overall_score}
          configuredComponents={complianceQuery.data?.configured_components}
          securityScore={complianceQuery.data?.remediation_progress?.security_score}
          vaultHealth={complianceQuery.data?.vault_secrets_health}
          vulnerabilitySummary={vulnerabilitySummary}
          missingSecrets={missingSecrets}
          isComplianceLoading={complianceQuery.isLoading}
          isVaultLoading={vaultQuery.isLoading}
          onOpenResource={openResourcePanel}
        />

        <SecurityTables
          resourceStatuses={resourceStatuses}
          vulnerabilities={vulnerabilities}
          isVaultLoading={vaultQuery.isLoading}
          isVulnerabilityLoading={vulnerabilityQuery.isLoading}
          componentType={componentType}
          componentFilter={componentFilter}
          severityFilter={severityFilter}
          componentOptions={componentOptions}
          scanId={vulnerabilityQuery.data?.scan_id}
          riskScore={vulnerabilityQuery.data?.risk_score}
          scanDuration={vulnerabilityQuery.data?.scan_duration}
          onOpenResource={openResourcePanel}
          onComponentTypeChange={setComponentType}
          onComponentFilterChange={setComponentFilter}
          onSeverityFilterChange={setSeverityFilter}
        />

        <footer className="pb-6 text-center text-xs text-white/40">
          Powered by the Vrooli lifecycle · API base: {import.meta.env.VITE_API_BASE_URL || "lifecycle-managed"}
        </footer>
      </main>

      {activeResource && (
        <ResourcePanel
          activeResource={activeResource}
          resourceDetail={resourceDetailQuery.data}
          isLoading={resourceDetailQuery.isLoading}
          isFetching={resourceDetailQuery.isFetching}
          selectedSecretKey={selectedSecretKey}
          strategyTier={strategyTier}
          strategyHandling={strategyHandling}
          strategyPrompt={strategyPrompt}
          strategyDescription={strategyDescription}
          tierReadiness={tierReadiness}
          onClose={closeResourcePanel}
          onSelectSecret={setSelectedSecretKey}
          onUpdateSecret={handleSecretUpdate}
          onApplyStrategy={handleStrategyApply}
          onUpdateVulnerabilityStatus={handleVulnerabilityStatus}
          onSetStrategyTier={setStrategyTier}
          onSetStrategyHandling={setStrategyHandling}
          onSetStrategyPrompt={setStrategyPrompt}
          onSetStrategyDescription={setStrategyDescription}
        />
      )}
    </div>
  );
}
