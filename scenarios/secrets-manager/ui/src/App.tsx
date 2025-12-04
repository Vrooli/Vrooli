import { useMemo, useState, useEffect } from "react";
import { RefreshCcw } from "lucide-react";
import { Header } from "./sections/Header";
import { OrientationHub } from "./sections/OrientationHub";
import { DeploymentReadinessPanel, TierReadiness } from "./sections/TierReadiness";
import { ComplianceOverview } from "./sections/ComplianceOverview";
import { SecurityTables } from "./sections/SecurityTables";
import { ResourcePanel } from "./features/resource-panel/ResourcePanel";
import { useSecretsData } from "./hooks/useSecretsData";
import { useVulnerabilities } from "./hooks/useVulnerabilities";
import { useResourcePanel } from "./hooks/useResourcePanel";
import { useJourneys } from "./hooks/useJourneys";
import { useScenarios } from "./hooks/useScenarios";
import { useCampaigns } from "./hooks/useCampaigns";
import { TabNav } from "./components/ui/TabNav";
import { TabTip } from "./components/ui/TabTip";
import { SnapshotPanel } from "./sections/SnapshotPanel";
import { ResourceTable } from "./sections/ResourceTable";
import { CampaignsPanel } from "./sections/CampaignsPanel";
import { DeploymentStepper } from "./sections/DeploymentStepper";
import type { JourneyId } from "./features/journeys/journeySteps";
import { TutorialOverlay } from "./components/ui/TutorialOverlay";

export default function App() {
  type ExperienceTab = "dashboard" | "resources" | "compliance" | "deployment";

  const [activeTab, setActiveTab] = useState<ExperienceTab>("dashboard");
  const [resourceTab, setResourceTab] = useState<"tier" | "resource">("tier");
  const [campaignStep, setCampaignStep] = useState(0);
  const [showTutorialOverlay, setShowTutorialOverlay] = useState(false);
  const [tutorialAnchor, setTutorialAnchor] = useState<string | undefined>(undefined);
  const [selectedScenario, setSelectedScenario] = useState<string>("secrets-manager");
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

  const {
    search: scenarioSearch,
    setSearch: setScenarioSearch,
    query: scenarioQuery,
    scenarios,
    filtered
  } = useScenarios();
  const {
    search: campaignSearch,
    setSearch: setCampaignSearch,
    query: campaignQuery,
    readinessQuery: campaignReadinessQuery,
    filtered: filteredCampaigns
  } = useCampaigns(selectedScenario);

  const topResourceNeedingAttention = useMemo(() => {
    if (resourceInsights.length > 0) {
      return resourceInsights[0].resource_name;
    }
    const resourceStatuses = vaultQuery.data?.resource_statuses ?? [];
    return resourceStatuses.find((status) => status.secrets_missing > 0)?.resource_name;
  }, [resourceInsights, vaultQuery.data]);

  const vulnerabilitySummary = {
    critical: complianceQuery.data?.vulnerability_summary?.critical ?? 0,
    high: complianceQuery.data?.vulnerability_summary?.high ?? 0,
    medium: complianceQuery.data?.vulnerability_summary?.medium ?? 0,
    low: complianceQuery.data?.vulnerability_summary?.low ?? 0
  };

  const missingSecrets = vaultQuery.data?.missing_secrets ?? [];
  const resourceStatuses = vaultQuery.data?.resource_statuses ?? [];
  const vulnerabilities = vulnerabilityQuery.data?.vulnerabilities ?? [];

  const blockedTiers = tierReadiness.filter((tier) => tier.ready_percent < 100 || tier.strategized < tier.total);
  const readinessCount = blockedTiers.length;
  const complianceCount =
    vulnerabilitySummary.critical + vulnerabilitySummary.high + vulnerabilitySummary.medium + vulnerabilitySummary.low;
  const missingSecretsCount = heroStats?.missing_secrets ?? missingSecrets.length ?? 0;

  const scrollToAnchor = (anchorId?: string) => {
    if (typeof document === "undefined" || !anchorId) return;
    const el = document.getElementById(anchorId);
    if (el) {
      el.scrollIntoView({ behavior: "smooth", block: "start" });
    }
  };

  const {
    activeJourney,
    journeyStep,
    journeySteps,
    handleJourneySelect,
    handleJourneyExit,
    handleJourneyNext,
    handleJourneyBack,
    setJourneyStep,
    journeyNextDisabled,
    deploymentFlow
  } = useJourneys({
    selectedScenario,
    onDeploymentScenarioChange: setSelectedScenario,
    scenarioSelection: {
      scenarios,
      filtered,
      search: scenarioSearch,
      isLoading: scenarioQuery.isLoading,
      selectedScenario,
      onSearchChange: setScenarioSearch,
      onSelect: setSelectedScenario
    },
    heroStats,
    vulnerabilitySummary,
    orientationData,
    tierReadiness,
    topResourceNeedingAttention,
    onOpenResource: openResourcePanel,
    onRefetchVulnerabilities: () => vulnerabilityQuery.refetch(),
    onNavigateTab: (tab) => setActiveTab(tab as ExperienceTab),
    onStartTutorial: (journey, startStep = 1) => {
      const config: Record<JourneyId, { tab: ExperienceTab; anchor?: string }> = {
        "configure-secrets": { tab: "resources", anchor: "anchor-resources" },
        "fix-vulnerabilities": { tab: "compliance", anchor: "anchor-compliance" },
        "prep-deployment": { tab: "deployment", anchor: "anchor-deployment" },
        orientation: { tab: "dashboard" }
      };
      const target = config[journey];
      if (!target) return;
      setActiveTab(target.tab);
      const startIndex = Math.min(startStep, Math.max(journeySteps.length - 1, 0));
      setJourneyStep(startIndex);
      setTutorialAnchor(target.anchor);
      setShowTutorialOverlay(true);
      scrollToAnchor(target.anchor);
    }
  });

  useEffect(() => {
    if (!scenarioQuery.data?.scenarios?.length) return;
    // Prefer secrets-manager if present; otherwise first scenario
    const preferred = scenarioQuery.data.scenarios.find((scenario) => scenario.name === "secrets-manager");
    const fallback = scenarioQuery.data.scenarios[0];
    const nextScenario = preferred?.name || fallback?.name;
    if (nextScenario && selectedScenario === "secrets-manager" && nextScenario !== selectedScenario) {
      setSelectedScenario(nextScenario);
    }
  }, [scenarioQuery.data?.scenarios, selectedScenario]);

  const activeJourneyCard = journeyCards.find((card) => card.id === activeJourney);

  const handleJourneySelectTyped = (journeyId: JourneyId) => {
    closeTutorialOverlay();
    setActiveTab("dashboard");
    handleJourneySelect(journeyId);
  };

  const startPrepDeploymentJourney = () => {
    setActiveTab("dashboard");
    closeTutorialOverlay();
    handleJourneySelectTyped("prep-deployment");
  };

  const closeTutorialOverlay = () => {
    setShowTutorialOverlay(false);
    setTutorialAnchor(undefined);
  };

  const tabs: Array<{ id: ExperienceTab; label: string; badgeCount?: number }> = [
    {
      id: "dashboard",
      label: "Dashboard",
      badgeCount: missingSecretsCount > 0 ? missingSecretsCount : undefined
    },
    {
      id: "resources",
      label: "Resources",
      badgeCount: readinessCount > 0 ? readinessCount : undefined
    },
    {
      id: "compliance",
      label: "Compliance",
      badgeCount: complianceCount > 0 ? complianceCount : undefined
    },
    {
      id: "deployment",
      label: "Deployment",
      badgeCount: readinessCount > 0 ? readinessCount : undefined
    }
  ];

  const handleCampaignStepChange = (index: number) => {
    setCampaignStep(index);
    const anchorMap: Record<number, string | undefined> = {
      0: "anchor-campaigns",
      1: "anchor-deployment",
      2: "anchor-resources",
      3: "anchor-manifest"
    };
    scrollToAnchor(anchorMap[index]);
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
          onRefresh={refreshAll}
        />

        <div className="rounded-3xl border border-white/10 bg-white/5 p-4">
          <TabNav tabs={tabs} activeTab={activeTab} onChange={(id) => setActiveTab(id as ExperienceTab)} />
        </div>

        {activeTab === "dashboard" && missingSecretsCount > 0 ? (
          <TabTip
            title={`${missingSecretsCount} secret${missingSecretsCount === 1 ? "" : "s"} need attention`}
            description="Open the Resources tab to see which services are missing secrets and how to fix them."
            actionLabel="Go to Resources"
            onAction={() => setActiveTab("resources")}
          />
        ) : null}
        {activeTab === "resources" && readinessCount > 0 ? (
          <TabTip
            title={`${readinessCount} tier${readinessCount === 1 ? "" : "s"} need strategies`}
            description="Start with By Tier to clear blockers, then verify per-resource coverage."
            actionLabel="View By Tier"
            onAction={() => setResourceTab("tier")}
          />
        ) : null}
        {activeTab === "compliance" && complianceCount > 0 ? (
          <TabTip
            title={`${complianceCount} compliance finding${complianceCount === 1 ? "" : "s"}`}
            description="Use filters to triage critical and high issues first. Journeys can jump you straight to the right tab."
          />
        ) : null}
        {activeTab === "deployment" && readinessCount > 0 ? (
          <TabTip
            title="Deployment prep needs strategies"
            description="Clear blocked tiers, then generate and export a manifest for the selected campaign."
            actionLabel="Run readiness"
            onAction={() => handleCampaignStepChange(1)}
          />
        ) : null}

        <div className="space-y-8">
          {activeTab === "dashboard" && (
            <>
              <SnapshotPanel
                heroStats={heroStats}
                updatedAt={orientationData?.updated_at}
                healthData={healthQuery.data}
                vaultData={vaultQuery.data}
                complianceData={complianceQuery.data}
                vulnerabilityData={vulnerabilityQuery.data}
                isLoading={
                  healthQuery.isLoading || vaultQuery.isLoading || complianceQuery.isLoading || vulnerabilityQuery.isLoading
                }
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
                onJourneySelect={handleJourneySelectTyped}
                onJourneyExit={handleJourneyExit}
                onJourneyNext={handleJourneyNext}
                onJourneyBack={handleJourneyBack}
                journeyNextDisabled={journeyNextDisabled}
                onShowReadiness={() => setActiveTab("resources")}
              />
            </>
          )}

          {activeTab === "resources" && (
            <div className="space-y-4">
              <TabNav
                tabs={[
                  { id: "tier", label: "By Tier" },
                  { id: "resource", label: "Per Resource" }
                ]}
                activeTab={resourceTab}
                onChange={(id) => setResourceTab(id as "tier" | "resource")}
              />
              {resourceTab === "tier" ? (
                <TierReadiness
                  tierReadiness={tierReadiness}
                  isLoading={orientationQuery.isLoading}
                  onOpenResource={openResourcePanel}
                  resourceInsights={resourceInsights}
                  resourceStatuses={resourceStatuses}
                />
              ) : (
                <ResourceTable
                  resourceStatuses={resourceStatuses}
                  isLoading={vaultQuery.isLoading}
                  onOpenResource={openResourcePanel}
                />
              )}
            </div>
          )}

          {activeTab === "deployment" && (
            <>
              <CampaignsPanel
                campaigns={filteredCampaigns}
                isLoading={campaignQuery.isLoading || campaignReadinessQuery.isLoading}
                search={campaignSearch}
                onSearchChange={setCampaignSearch}
                selectedScenario={selectedScenario}
                onSelectScenario={(scenario) => {
                  setSelectedScenario(scenario);
                  handleCampaignStepChange(1);
                }}
                defaultBlockedTiers={readinessCount}
              />

              <DeploymentStepper
                activeStep={campaignStep}
                onStepChange={handleCampaignStepChange}
                onOpenResource={
                  topResourceNeedingAttention ? () => openResourcePanel(topResourceNeedingAttention) : undefined
                }
                onGenerateManifest={deploymentFlow.onGenerateManifest}
                hasManifest={!!deploymentFlow.manifestData}
              />

              <DeploymentReadinessPanel
                tierReadiness={tierReadiness}
                resourceInsights={resourceInsights}
                manifestState={deploymentFlow}
                onOpenResource={openResourcePanel}
                onStartJourney={startPrepDeploymentJourney}
              />
            </>
          )}

          {activeTab === "compliance" && (
            <>
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
            </>
          )}
        </div>

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

      {showTutorialOverlay && activeJourney && activeJourney !== "orientation" && journeySteps[journeyStep] ? (
        <TutorialOverlay
          title={activeJourneyCard?.title || "Tutorial"}
          subtitle={activeJourneyCard?.description}
          stepLabel={`Step ${journeyStep + 1} of ${journeySteps.length}`}
          content={journeySteps[journeyStep].content}
          anchorId={tutorialAnchor}
          onClose={closeTutorialOverlay}
          onNext={handleJourneyNext}
          onBack={journeyStep > 0 ? handleJourneyBack : undefined}
          disableNext={journeyNextDisabled}
        />
      ) : null}
    </div>
  );
}
