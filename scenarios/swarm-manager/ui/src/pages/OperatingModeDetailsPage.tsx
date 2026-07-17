/**
 * Operating Mode Details Page
 *
 * Shows the catalog metadata for one operating mode (label, description,
 * scope, run strategy, phases) and the list of initiatives currently bound
 * to that mode. Editable fields (label, description) persist via the API
 * overlay store.
 */

import { type ReactNode, useEffect, useMemo, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate, useParams } from "react-router-dom";
import {
  ArrowUpRight,
  CheckCircle2,
  HelpCircle,
  Info,
  Layers,
  List,
  Network,
  Pencil,
  Play,
  RotateCcw,
  Scale,
  StepForward,
  Workflow,
  X,
  XCircle,
} from "lucide-react";
import { Button } from "../components/ui/button";
import { cn } from "../lib/utils";
import { CompactTabBar, type CompactTabItem } from "../components/ui/compact-tab-bar";
import { DetailPageHeader } from "../components/detail/DetailPageHeader";
import { DetailPageLayout } from "../components/detail/DetailPageLayout";
import { DetailSection } from "../components/detail/DetailSection";
import { ErrorState } from "../components/ui/error-state";
import { PageLoadingState } from "../components/ui/loading-states";
import { ConceptExplainerDialog } from "../components/ui/concept-explainer-dialog";
import { HowToChooseDialog } from "../components/initiative/operating-mode/how-to-choose-dialog";
import { initiativeModeService } from "../services";
import { useDocsUrl } from "../services/external-links";
import { initiativeDetailPath } from "../app/routes/route-paths";
import { selectors } from "../consts/selectors";
import type {
  OperatingModeDetail,
  OperatingModeLinkedInitiative,
  OperatingModeRound,
  OperatingModeSimulation,
  OperatingModeSimulationPreset,
  OperatingModeSimulationStep,
  OperatingModeWorkspace,
} from "../types/operating-mode";
import { PhaseGraph } from "../components/initiative/operating-mode/phase-graph";
import { PhaseList } from "../components/initiative/operating-mode/phase-list";
import { PhaseViewer } from "../components/initiative/operating-mode/phase-viewer";
import {
  contractPhaseView,
  livePhaseView,
  simulationPhaseView,
  type PhaseView,
  type PhaseViewSource,
} from "../components/initiative/operating-mode/phase-view";
import { CapabilityList } from "../components/initiative/operating-mode/capability-list";
import {
  CAPABILITY_EXPLAINER,
  DEFAULT_FLAG_EXPLAINER,
  FLOW_GUIDE_EXPLAINER,
  RUN_STRATEGY_EXPLAINER,
  TARGET_KIND_EXPLAINER,
  type ConceptExplainer,
} from "../components/initiative/operating-mode/concept-explainers";
import {
  humanizeRunStrategy,
  humanizeTargetKind,
  phaseCardDomId,
} from "../components/initiative/operating-mode/utils";
import { useUrlState } from "../hooks/use-url-state";
import { isMemberItemStrategy, presentModeLabel } from "../lib/member-item-strategy";
import { useAttachToSessionAction } from "../components/session/context/useAttachToSessionAction";
import { operatingModeOption } from "../components/session/context/session-context-refs";

const EMPTY_LENSES: never[] = [];

type PhasesView = "list" | "graph";
type OperatingModeTab = "overview" | "phases" | "flow" | "guidance";

const HIGHLIGHT_DURATION_MS = 1500;
const SIMULATION_STEP_MS = 900;
const LIVE_REFRESH_MS = 5000;

export function OperatingModeDetailsPage() {
  const params = useParams();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const mode = params.mode ?? "";

  const { data, isLoading, error, refetch } = useQuery<OperatingModeDetail>({
    queryKey: ["operating-modes", "detail", mode],
    queryFn: () => initiativeModeService.getMode(mode),
    enabled: Boolean(mode),
  });

  const catalogQuery = useQuery({
    queryKey: ["operating-modes", "catalog"],
    queryFn: () => initiativeModeService.catalog(),
  });
  const catalogModes = useMemo(
    () => catalogQuery.data?.modes ?? [],
    [catalogQuery.data],
  );
  const [howToChooseOpen, setHowToChooseOpen] = useState(false);

  const [isEditing, setEditing] = useState(false);
  const [labelDraft, setLabelDraft] = useState("");
  const [descriptionDraft, setDescriptionDraft] = useState("");
  const [selectedLiveInitiative, setSelectedLiveInitiative] = useState("");
  const [activePreset, setActivePreset] = useState("");
  const [flowSource, setFlowSource] = useState<PhaseViewSource>("simulation");
  const [flowGuideOpen, setFlowGuideOpen] = useState(false);

  useEffect(() => {
    if (data) {
      setLabelDraft(data.entry.label);
      setDescriptionDraft(data.entry.description ?? "");
    }
  }, [data]);

  useEffect(() => {
    if (!data?.linkedInitiatives.length) {
      setSelectedLiveInitiative("");
      return;
    }
    setSelectedLiveInitiative((current) =>
      data.linkedInitiatives.some((initiative) => initiative.name === current)
        ? current
        : data.linkedInitiatives[0]?.name ?? "",
    );
  }, [data?.linkedInitiatives]);

  const updateMutation = useMutation({
    mutationFn: (args: { label?: string; description?: string }) =>
      initiativeModeService.updateMode(mode, args),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["operating-modes"] });
      setEditing(false);
    },
  });

  const phases = useMemo(() => data?.entry.phases ?? [], [data]);
  const hasPhaseGraph = Boolean(data?.entry.phaseGraph && phases.length > 0);
  const simulationQuery = useQuery<OperatingModeSimulation>({
    queryKey: ["operating-modes", "simulation", mode, activePreset],
    queryFn: () => initiativeModeService.simulateMode(mode, activePreset || undefined),
    enabled: Boolean(mode && hasPhaseGraph),
  });
  const liveWorkspaceQuery = useQuery<OperatingModeWorkspace>({
    queryKey: ["operating-modes", "live-workspace", selectedLiveInitiative],
    queryFn: () => initiativeModeService.workspace(selectedLiveInitiative),
    enabled: Boolean(hasPhaseGraph && selectedLiveInitiative),
  });

  const [phasesView, setPhasesView] = useUrlState<PhasesView>("view", "graph", {
    validate: (value): value is PhasesView => value === "list" || value === "graph",
  });
  const [activeTab, setActiveTab] = useUrlState<OperatingModeTab>("tab", "overview", {
    validate: (value): value is OperatingModeTab =>
      value === "overview" || value === "phases" || value === "flow" || value === "guidance",
  });
  const [highlightedPhaseId, setHighlightedPhaseId] = useState<string | null>(null);
  const [simulationIndex, setSimulationIndex] = useState(0);
  const [simulationPlaying, setSimulationPlaying] = useState(false);
  const highlightTimerRef = useRef<number | null>(null);
  const [activeExplainer, setActiveExplainer] = useState<ConceptExplainer | null>(null);
  const docsExecutionModesUrl = useDocsUrl("/docs/concepts/EXECUTION-MODES.md");
  const docsHolisticLoopUrl = useDocsUrl("/docs/guides/holistic-loop-mode.md");
  const docsPhasedPlanDrainUrl = useDocsUrl("/docs/guides/phased-plan-drain-mode.md");
  const attachToSession = useAttachToSessionAction(data ? operatingModeOption(data.entry) : null);

  useEffect(() => {
    return () => {
      if (highlightTimerRef.current !== null) {
        window.clearTimeout(highlightTimerRef.current);
      }
    };
  }, []);

  useEffect(() => {
    if (!simulationPlaying) return;
    const traceLength = simulationQuery.data?.trace.length ?? 0;
    if (traceLength === 0) return;
    const timer = window.setInterval(() => {
      setSimulationIndex((current) => {
        if (current >= traceLength - 1) {
          window.clearInterval(timer);
          setSimulationPlaying(false);
          return current;
        }
        return current + 1;
      });
    }, SIMULATION_STEP_MS);
    return () => window.clearInterval(timer);
  }, [simulationPlaying, simulationQuery.data?.trace.length]);

  useEffect(() => {
    setSimulationIndex(0);
    setSimulationPlaying(false);
  }, [mode, simulationQuery.data]);

  const liveWorkspace = liveWorkspaceQuery.data;
  const liveActiveRound = useMemo(
    () => selectLiveRound(liveWorkspace?.rounds ?? []),
    [liveWorkspace?.rounds],
  );

  useEffect(() => {
    if (!selectedLiveInitiative || !liveActiveRound || !isLiveRoundActive(liveActiveRound)) return;
    const timer = window.setInterval(() => {
      void initiativeModeService
        .refreshRound(selectedLiveInitiative, liveActiveRound.mode, liveActiveRound.round)
        .then(() => queryClient.invalidateQueries({
          queryKey: ["operating-modes", "live-workspace", selectedLiveInitiative],
        }));
    }, LIVE_REFRESH_MS);
    return () => window.clearInterval(timer);
  }, [liveActiveRound, queryClient, selectedLiveInitiative]);

  const handleSelectPhase = (phase: string) => {
    setHighlightedPhaseId(phase);
    if (highlightTimerRef.current !== null) {
      window.clearTimeout(highlightTimerRef.current);
    }
    highlightTimerRef.current = window.setTimeout(() => {
      setHighlightedPhaseId(null);
      highlightTimerRef.current = null;
    }, HIGHLIGHT_DURATION_MS);
    if (typeof document !== "undefined") {
      scrollElementIntoNearestContainer(document.getElementById(phaseCardDomId(phase)));
    }
  };

  if (!mode) {
    return <ErrorState title="Missing mode" message="The URL does not include a mode identifier." />;
  }
  if (isLoading) return <PageLoadingState label="Loading operating mode..." />;
  if (error || !data) {
    const technicalDetails = (error as Error | undefined)?.message;
    return (
      <div className="space-y-3">
        <ErrorState
          title="We couldn't load this operating mode"
          message="The mode catalog endpoint did not respond. Try again, or check that the operating-mode service is running."
          onRetry={() => void refetch()}
        />
        {technicalDetails && (
          <details
            className="rounded-md border border-slate-800/80 bg-slate-900/50 px-3 py-2 text-xs text-slate-400"
            data-testid={selectors.initiativeDetails.operatingModeErrorTechnicalDetails}
          >
            <summary className="cursor-pointer select-none">Technical details</summary>
            <pre className="mt-2 whitespace-pre-wrap break-words font-mono text-[11px] text-slate-500">
              {technicalDetails}
            </pre>
          </details>
        )}
      </div>
    );
  }

  const { entry, linkedInitiatives } = data;
  // Deep-link normalization: /operating-modes/item-level stays addressable
  // (the wire value persists until Phase 8), but the PRESENTATION is the
  // member-item workflow strategy, not an operating mode — see
  // lib/member-item-strategy.ts for the mapping contract.
  const isStrategy = isMemberItemStrategy(entry.mode);
  const transitions = entry.phaseGraph?.transitions ?? [];
  const subModeLookup: Record<string, typeof entry> = {};
  for (const catalogMode of catalogModes) subModeLookup[catalogMode.mode] = catalogMode;
  const simulationTrace = simulationQuery.data?.trace ?? [];
  const activeSimulationStep = simulationTrace[simulationIndex] ?? null;
  const liveCatalogPhase = liveActiveRound
    ? phases.find((candidate) => candidate.phase === liveActiveRound.phase)
    : undefined;
  const liveView = liveWorkspace && liveActiveRound
    ? livePhaseView(liveActiveRound, liveWorkspace, transitions, selectedLiveInitiative, liveCatalogPhase)
    : null;
  const activePresetId = simulationQuery.data?.activePreset ?? activePreset;
  const flowPhaseView: PhaseView | null = (() => {
    if (flowSource === "live") return liveView;
    if (!activeSimulationStep) return null;
    const stepCatalogPhase = phases.find((candidate) => candidate.phase === activeSimulationStep.phase);
    if (flowSource === "contract") {
      return stepCatalogPhase ? contractPhaseView(stepCatalogPhase, transitions) : null;
    }
    return simulationPhaseView(activeSimulationStep, mode, activePresetId, stepCatalogPhase);
  })();
  const selectedPhaseId = highlightedPhaseId ?? liveView?.phase ?? activeSimulationStep?.phase ?? null;
  const tabs: CompactTabItem<OperatingModeTab>[] = [
    { value: "overview", label: "Overview", icon: Layers },
    { value: "phases", label: "Phases", icon: Network, count: phases.length },
    { value: "flow", label: "Flow", icon: Workflow, badge: linkedInitiatives.length > 0 ? (
      <span className="ml-1.5 inline-flex h-4 min-w-4 items-center justify-center rounded-full bg-cyan-500/25 px-1 text-[10px] font-semibold text-cyan-300">
        {linkedInitiatives.length}
      </span>
    ) : null },
    { value: "guidance", label: "Guidance", icon: CheckCircle2 },
  ];
  const tabBar = (
    <div className="border-t border-slate-800/50" data-testid={selectors.initiativeDetails.modeDetailsTabRow}>
      <CompactTabBar
        items={tabs}
        activeValue={activeTab}
        onValueChange={setActiveTab}
        aria-label="Operating mode detail sections"
        className="px-3"
        tabTestIdPrefix="operating-mode-details-tab"
      />
    </div>
  );

  const handleSave = () => {
    const trimmedLabel = labelDraft.trim();
    if (!trimmedLabel) return;
    const patch: { label?: string; description?: string } = {};
    if (trimmedLabel !== entry.label) patch.label = trimmedLabel;
    if (descriptionDraft !== (entry.description ?? "")) patch.description = descriptionDraft;
    if (Object.keys(patch).length === 0) {
      setEditing(false);
      return;
    }
    updateMutation.mutate(patch);
  };

  const handleCancel = () => {
    setLabelDraft(entry.label);
    setDescriptionDraft(entry.description ?? "");
    setEditing(false);
  };

  return (
    <DetailPageLayout
      header={
        <DetailPageHeader
          entityType={isStrategy ? "Workflow Strategy" : "Operating Mode"}
          title={presentModeLabel(entry.mode, entry.label)}
          subtitle={entry.mode}
          nodeId={null}
          lenses={EMPTY_LENSES}
          menuActions={[attachToSession.actionItem]}
          tabBar={tabBar}
        />
      }
    >
      {attachToSession.sheet}
      {activeTab === "overview" && (
        <>
          {isStrategy && (
            <div
              className="mb-3 rounded-lg border border-cyan-500/30 bg-cyan-500/5 px-3 py-2.5 text-sm text-cyan-100"
              data-testid={selectors.initiativeDetails.memberItemStrategyNotice}
            >
              This is the member-item workflow strategy, not an operating mode: items run their
              own workflows and the initiative provides strategy configuration. It is still
              stored under the legacy <code className="font-mono text-[12px]">item-level</code>{" "}
              id, so existing links keep working.
            </div>
          )}
          <DetailSection
            title="Overview"
            icon={Layers}
            hideDivider
            action={
              isEditing ? (
                <div className="flex gap-2">
                  <Button variant="ghost" size="sm" onClick={handleCancel} disabled={updateMutation.isPending}>
                    <X className="mr-1 h-3.5 w-3.5" />Cancel
                  </Button>
                  <Button size="sm" onClick={handleSave} disabled={updateMutation.isPending || !labelDraft.trim()}>
                    {updateMutation.isPending ? "Saving..." : "Save"}
                  </Button>
                </div>
              ) : (
                <Button variant="ghost" size="sm" onClick={() => setEditing(true)}>
                  <Pencil className="mr-1 h-3.5 w-3.5" />Edit
                </Button>
              )
            }
          >
            {isEditing ? (
              <div className="space-y-3">
                <label className="block text-sm">
                  <span className="mb-1 block text-xs font-medium text-slate-400">Display label</span>
                  <input
                    type="text"
                    value={labelDraft}
                    onChange={(e) => setLabelDraft(e.target.value)}
                    className="w-full rounded-md border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-100 focus:border-cyan-500 focus:outline-none"
                    data-testid="operating-mode-label-input"
                  />
                </label>
                <label className="block text-sm">
                  <span className="mb-1 block text-xs font-medium text-slate-400">Description</span>
                  <textarea
                    value={descriptionDraft}
                    onChange={(e) => setDescriptionDraft(e.target.value)}
                    rows={4}
                    className="w-full rounded-md border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-100 focus:border-cyan-500 focus:outline-none"
                    data-testid="operating-mode-description-input"
                  />
                </label>
                {updateMutation.isError && (
                  <p className="text-sm text-red-400">{(updateMutation.error).message}</p>
                )}
              </div>
            ) : (
              <div className="space-y-3 text-sm text-slate-200">
                {entry.description ? (
                  <p className="whitespace-pre-wrap">{entry.description}</p>
                ) : (
                  <p className="italic text-slate-500">No description set.</p>
                )}
                <dl className="grid grid-cols-2 gap-x-4 gap-y-1 text-xs text-slate-400 md:grid-cols-4">
                  <div>
                    <ExplainableLabel
                      label="Target"
                      onOpen={() => setActiveExplainer(TARGET_KIND_EXPLAINER)}
                      testId={selectors.initiativeDetails.modeDetailsScopeInfoIcon}
                    />
                    <dd className="text-slate-200">{humanizeTargetKind(entry.targetKind)}</dd>
                  </div>
                  <div>
                    <ExplainableLabel
                      label="Run strategy"
                      onOpen={() => setActiveExplainer(RUN_STRATEGY_EXPLAINER)}
                      testId={selectors.initiativeDetails.modeDetailsRunStrategyInfoIcon}
                    />
                    <dd className="text-slate-200">{humanizeRunStrategy(entry.runStrategy)}</dd>
                  </div>
                  <div>
                    <ExplainableLabel
                      label="Default"
                      onOpen={() => setActiveExplainer(DEFAULT_FLAG_EXPLAINER)}
                      testId={selectors.initiativeDetails.modeDetailsDefaultInfoIcon}
                    />
                    <dd className="text-slate-200">{entry.default ? "yes" : "no"}</dd>
                  </div>
                  <div>
                    <dt className="text-slate-500">Usage</dt>
                    <dd className="text-slate-200">{entry.usageCount} initiative{entry.usageCount === 1 ? "" : "s"}</dd>
                  </div>
                </dl>
              </div>
            )}
          </DetailSection>
          <DetailSection
            title="Capabilities"
            action={
              <button
                type="button"
                onClick={() => setActiveExplainer(CAPABILITY_EXPLAINER)}
                className="rounded p-1 text-slate-400 transition-colors hover:bg-white/5 hover:text-slate-200"
                aria-label="What capability flags mean"
              >
                <Info className="h-4 w-4" />
              </button>
            }
          >
            <div data-testid={selectors.initiativeDetails.modeDetailsCapabilitiesSection}>
              <CapabilityList capabilities={entry.capabilities} variant="full" />
            </div>
          </DetailSection>
          <LinkedInitiativesSection linkedInitiatives={linkedInitiatives} onNavigate={navigate} />
        </>
      )}

      {activeTab === "phases" && phases.length > 0 && (
        <DetailSection
          title="Phases"
          action={
            hasPhaseGraph ? (
              <div className="flex gap-0.5" data-testid="operating-mode-phases-view-toggle">
                <button
                  type="button"
                  onClick={() => setPhasesView("list")}
                  className={`rounded p-1 transition-colors ${
                    phasesView === "list"
                      ? "bg-slate-700/50 text-slate-200"
                      : "text-slate-500 hover:text-slate-300"
                  }`}
                  title="List view"
                  aria-pressed={phasesView === "list"}
                >
                  <List className="h-4 w-4" />
                </button>
                <button
                  type="button"
                  onClick={() => setPhasesView("graph")}
                  className={`rounded p-1 transition-colors ${
                    phasesView === "graph"
                      ? "bg-slate-700/50 text-slate-200"
                      : "text-slate-500 hover:text-slate-300"
                  }`}
                  title="Phase graph"
                  aria-pressed={phasesView === "graph"}
                >
                  <Network className="h-4 w-4" />
                </button>
              </div>
            ) : null
          }
        >
          <div className="space-y-4">
            {hasPhaseGraph && phasesView === "graph" && (
              <PhaseGraph
                entry={entry}
                selectedPhaseId={selectedPhaseId}
                onSelectPhase={handleSelectPhase}
              />
            )}
            <PhaseList
              phases={phases}
              transitions={entry.phaseGraph?.transitions}
              highlightedPhaseId={selectedPhaseId}
              subModes={subModeLookup}
            />
          </div>
        </DetailSection>
      )}

      {activeTab === "flow" && (
        <div className="space-y-4">
          {hasPhaseGraph ? (
            <>
              <FlowIntro onOpenGuide={() => setFlowGuideOpen(true)} />
              <FlowControls
                source={flowSource}
                onChangeSource={(next) => {
                  setSimulationPlaying(false);
                  setFlowSource(next);
                }}
                hasLive={linkedInitiatives.length > 0}
                stepControls={
                  flowSource !== "live" ? (
                    <SimulationStepControls
                      isPlaying={simulationPlaying}
                      isLoading={simulationQuery.isLoading}
                      hasError={Boolean(simulationQuery.error)}
                      activeIndex={simulationIndex}
                      traceLength={simulationTrace.length}
                      onPlay={() => setSimulationPlaying(true)}
                      onPause={() => setSimulationPlaying(false)}
                      onStep={() => {
                        setSimulationPlaying(false);
                        setSimulationIndex((current) =>
                          Math.min(current + 1, Math.max(0, simulationTrace.length - 1)),
                        );
                      }}
                      onReset={() => {
                        setSimulationPlaying(false);
                        setSimulationIndex(0);
                      }}
                    />
                  ) : null
                }
              >
                {flowSource === "simulation" && (
                  <SimulationPresetSelector
                    presets={simulationQuery.data?.presets ?? []}
                    selected={(simulationQuery.data?.presets ?? []).find((preset) => preset.id === activePresetId)}
                    disabled={simulationQuery.isLoading || Boolean(simulationQuery.error)}
                    onSelect={(preset) => {
                      setSimulationPlaying(false);
                      setSimulationIndex(0);
                      setActivePreset(preset);
                    }}
                  />
                )}
                {flowSource === "live" && linkedInitiatives.length > 0 && (
                  <LiveInitiativeSelector
                    linkedInitiatives={linkedInitiatives}
                    selected={selectedLiveInitiative}
                    onSelect={setSelectedLiveInitiative}
                    onRefresh={() => void liveWorkspaceQuery.refetch()}
                  />
                )}
              </FlowControls>
              {flowPhaseView ? (
                <PhaseViewer view={flowPhaseView} subtitle={flowSubtitle(flowSource, activeSimulationStep, simulationTrace.length, selectedLiveInitiative, liveWorkspace?.rounds ?? [])} />
              ) : (
                <FlowEmptyState
                  source={flowSource}
                  simulationLoading={simulationQuery.isLoading}
                  simulationError={simulationQuery.error}
                  liveLoading={liveWorkspaceQuery.isLoading}
                  liveError={liveWorkspaceQuery.error}
                  hasLinked={linkedInitiatives.length > 0}
                  selectedInitiative={selectedLiveInitiative}
                />
              )}
            </>
          ) : (
            <DetailSection title="Flow" hideDivider>
              <p className="text-sm italic text-slate-500">This mode does not run as a phase graph, so it has no flow trace.</p>
            </DetailSection>
          )}
        </div>
      )}

      {activeTab === "guidance" && (
        <>
          <DetailSection title="When to use" hideDivider>
            <ul
              className="space-y-2"
              data-testid={selectors.initiativeDetails.modeDetailsBestForSection}
            >
              {entry.bestFor.map((item, idx) => (
                <li key={`best-${idx}`} className="flex items-start gap-2 text-sm text-slate-200">
                  <CheckCircle2 className="mt-0.5 h-4 w-4 shrink-0 text-emerald-400" aria-hidden="true" />
                  <span>{item}</span>
                </li>
              ))}
            </ul>
          </DetailSection>

          <DetailSection title="When not to use">
            <ul
              className="space-y-2"
              data-testid={selectors.initiativeDetails.modeDetailsNotForSection}
            >
              {entry.notFor.map((item, idx) => (
                <li key={`not-${idx}`} className="flex items-start gap-2 text-sm text-slate-200">
                  <XCircle className="mt-0.5 h-4 w-4 shrink-0 text-amber-400" aria-hidden="true" />
                  <span>{item}</span>
                </li>
              ))}
            </ul>
          </DetailSection>

          <DetailSection title="Tradeoffs">
            <ul
              className="space-y-2"
              data-testid={selectors.initiativeDetails.modeDetailsTradeoffsSection}
            >
              {entry.tradeoffs.map((item, idx) => (
                <li key={`trade-${idx}`} className="flex items-start gap-2 text-sm text-slate-200">
                  <Scale className="mt-0.5 h-4 w-4 shrink-0 text-cyan-400" aria-hidden="true" />
                  <span>{item}</span>
                </li>
              ))}
            </ul>
          </DetailSection>

          <DetailSection title="Learn more">
            <LearnMoreLinks
              mode={entry.mode}
              executionModesUrl={docsExecutionModesUrl}
              holisticLoopUrl={docsHolisticLoopUrl}
              phasedPlanDrainUrl={docsPhasedPlanDrainUrl}
            />
          </DetailSection>

          <DetailSection title="Compare to other modes" hideDivider>
            <Button
              variant="outline"
              size="sm"
              onClick={() => setHowToChooseOpen(true)}
              data-testid={selectors.initiativeDetails.modeDetailsHowToChooseButton}
            >
              How does this compare to other modes?
            </Button>
          </DetailSection>
        </>
      )}

      <ConceptExplainerDialog
        isOpen={activeExplainer !== null}
        onClose={() => setActiveExplainer(null)}
        title={activeExplainer?.title ?? ""}
        intro={activeExplainer?.intro}
        sections={activeExplainer?.sections ?? []}
      />

      <HowToChooseDialog
        isOpen={howToChooseOpen}
        onClose={() => setHowToChooseOpen(false)}
        catalog={catalogModes}
      />

      <ConceptExplainerDialog
        isOpen={flowGuideOpen}
        onClose={() => setFlowGuideOpen(false)}
        title={FLOW_GUIDE_EXPLAINER.title}
        intro={FLOW_GUIDE_EXPLAINER.intro}
        sections={FLOW_GUIDE_EXPLAINER.sections}
        testId={selectors.initiativeDetails.flowGuideDialog}
      />
    </DetailPageLayout>
  );
}

function LinkedInitiativesSection({
  linkedInitiatives,
  onNavigate,
}: {
  linkedInitiatives: OperatingModeLinkedInitiative[];
  onNavigate: (to: string) => void;
}) {
  return (
    <DetailSection title="Linked Initiatives">
      {linkedInitiatives.length === 0 ? (
        <p className="text-sm italic text-slate-500">No initiatives currently use this mode.</p>
      ) : (
        <ul className="space-y-1.5">
          {linkedInitiatives.map((init) => (
            <li key={init.name}>
              <button
                type="button"
                onClick={() => onNavigate(initiativeDetailPath(init.name))}
                className="flex w-full items-start justify-between gap-2 rounded-md border border-slate-800 bg-slate-900/40 px-3 py-2 text-left text-sm transition-colors hover:border-slate-700 hover:bg-slate-800/60"
                data-testid="operating-mode-linked-initiative"
              >
                <div>
                  <p className="font-medium text-slate-100">{init.title || init.name}</p>
                  <p className="text-xs text-slate-500">{init.name}</p>
                </div>
                {init.status && (
                  <span className="rounded-full bg-slate-800 px-2 py-0.5 text-[11px] font-medium text-slate-300">
                    {init.status}
                  </span>
                )}
              </button>
            </li>
          ))}
        </ul>
      )}
    </DetailSection>
  );
}

function scrollElementIntoNearestContainer(element: HTMLElement | null) {
  if (!element) return;
  let parent = element.parentElement;
  while (parent) {
    const style = window.getComputedStyle(parent);
    const scrollableY = /(auto|scroll)/.test(style.overflowY) && parent.scrollHeight > parent.clientHeight;
    if (scrollableY) {
      const containerRect = parent.getBoundingClientRect();
      const elementRect = element.getBoundingClientRect();
      const top = elementRect.top - containerRect.top + parent.scrollTop - (parent.clientHeight / 2) + (elementRect.height / 2);
      parent.scrollTo({ top: Math.max(0, top), behavior: "smooth" });
      return;
    }
    parent = parent.parentElement;
  }
  const absoluteTop = element.getBoundingClientRect().top + window.scrollY - (window.innerHeight / 2) + (element.offsetHeight / 2);
  window.scrollTo({ top: Math.max(0, absoluteTop), behavior: "smooth" });
}

function FlowIntro({ onOpenGuide }: { onOpenGuide: () => void }) {
  return (
    <div className="flex flex-wrap items-start justify-between gap-2 rounded-lg border border-slate-800 bg-slate-900/40 px-3 py-2.5">
      <div className="min-w-0">
        <h2 className="flex items-center gap-1.5 text-sm font-semibold text-slate-100">
          <Workflow className="h-4 w-4 text-cyan-300" aria-hidden="true" />
          How this mode flows
        </h2>
        <p className="mt-0.5 text-xs leading-relaxed text-slate-400">
          One phase viewer, three data sources. See the agent's <span className="text-slate-200">Instructions</span>,
          Reads, Emits, and Transition for the <span className="text-slate-200">Contract</span> template, a stepped
          <span className="text-slate-200"> Simulation</span> preset, or a <span className="text-slate-200">Live</span> round.
        </p>
      </div>
      <Button
        type="button"
        variant="outline"
        size="sm"
        onClick={onOpenGuide}
        data-testid={selectors.initiativeDetails.flowGuideButton}
      >
        <HelpCircle className="mr-1 h-3.5 w-3.5" />
        Guide
      </Button>
    </div>
  );
}

const FLOW_SOURCES: Array<{ value: PhaseViewSource; label: string; testId: string }> = [
  { value: "contract", label: "Contract", testId: selectors.initiativeDetails.flowSourceContract },
  { value: "simulation", label: "Simulation", testId: selectors.initiativeDetails.flowSourceSimulation },
  { value: "live", label: "Live", testId: selectors.initiativeDetails.flowSourceLive },
];

// FlowControls is the single data-source control that replaces the two stacked
// Simulation + Live panels: a Contract / Simulation / Live toggle plus the
// source-specific step controls and selectors, all feeding one PhaseViewer.
function FlowControls({
  source,
  onChangeSource,
  hasLive,
  stepControls,
  children,
}: {
  source: PhaseViewSource;
  onChangeSource: (next: PhaseViewSource) => void;
  hasLive: boolean;
  stepControls?: ReactNode;
  children?: ReactNode;
}) {
  return (
    <div className="rounded-lg border border-slate-800 bg-slate-900/40 p-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div
          className="inline-flex rounded-md border border-slate-800 bg-slate-950/60 p-0.5"
          role="tablist"
          aria-label="Flow data source"
          data-testid={selectors.initiativeDetails.flowSourceToggle}
        >
          {FLOW_SOURCES.map((option) => {
            const disabled = option.value === "live" && !hasLive;
            return (
              <button
                key={option.value}
                type="button"
                role="tab"
                aria-selected={source === option.value}
                disabled={disabled}
                data-testid={option.testId}
                onClick={() => onChangeSource(option.value)}
                title={disabled ? "No initiatives use this mode yet" : undefined}
                className={cn(
                  "rounded px-2.5 py-1 text-xs font-medium transition-colors",
                  source === option.value
                    ? "bg-cyan-500/20 text-cyan-200"
                    : "text-slate-400 hover:text-slate-200",
                  disabled && "cursor-not-allowed opacity-40 hover:text-slate-400",
                )}
              >
                {option.label}
              </button>
            );
          })}
        </div>
        {stepControls}
      </div>
      {children}
    </div>
  );
}

function SimulationStepControls({
  isPlaying,
  isLoading,
  hasError,
  activeIndex,
  traceLength,
  onPlay,
  onPause,
  onStep,
  onReset,
}: {
  isPlaying: boolean;
  isLoading: boolean;
  hasError: boolean;
  activeIndex: number;
  traceLength: number;
  onPlay: () => void;
  onPause: () => void;
  onStep: () => void;
  onReset: () => void;
}) {
  const atEnd = traceLength === 0 || activeIndex >= traceLength - 1;
  return (
    <div className="flex items-center gap-1">
      <Button
        type="button"
        variant="ghost"
        size="sm"
        onClick={isPlaying ? onPause : onPlay}
        disabled={isLoading || hasError || traceLength === 0 || (isPlaying ? false : atEnd)}
      >
        <Play className="mr-1 h-3.5 w-3.5" />
        {isPlaying ? "Pause" : "Play"}
      </Button>
      <Button type="button" variant="ghost" size="sm" onClick={onStep} disabled={isLoading || hasError || atEnd}>
        <StepForward className="mr-1 h-3.5 w-3.5" />
        Step
      </Button>
      <Button
        type="button"
        variant="ghost"
        size="sm"
        onClick={onReset}
        disabled={isLoading || hasError || activeIndex === 0}
      >
        <RotateCcw className="mr-1 h-3.5 w-3.5" />
        Reset
      </Button>
    </div>
  );
}

function LiveInitiativeSelector({
  linkedInitiatives,
  selected,
  onSelect,
  onRefresh,
}: {
  linkedInitiatives: Array<{ name: string; title: string }>;
  selected: string;
  onSelect: (name: string) => void;
  onRefresh: () => void;
}) {
  return (
    <div className="mt-3 flex flex-wrap items-center gap-2">
      <label className="text-[11px] font-medium uppercase tracking-wide text-slate-400" htmlFor="flow-live-select">
        Initiative
      </label>
      <select
        id="flow-live-select"
        value={selected}
        onChange={(event) => onSelect(event.target.value)}
        className="min-w-0 flex-1 rounded border border-slate-700 bg-slate-900 px-2 py-1 text-xs text-slate-200"
        aria-label="Live initiative"
      >
        {linkedInitiatives.map((initiative) => (
          <option key={initiative.name} value={initiative.name}>
            {initiative.title || initiative.name}
          </option>
        ))}
      </select>
      <Button type="button" variant="ghost" size="sm" onClick={onRefresh} disabled={!selected}>
        <RotateCcw className="mr-1 h-3.5 w-3.5" />
        Refresh
      </Button>
    </div>
  );
}

function flowSubtitle(
  source: PhaseViewSource,
  activeStep: OperatingModeSimulationStep | null,
  traceLength: number,
  selectedInitiative: string,
  rounds: OperatingModeRound[],
): string {
  if (source === "live") {
    if (!selectedInitiative) return "Real data · select a linked initiative";
    const round = rounds.length > 0 ? rounds[rounds.length - 1] : undefined;
    return `Real data · ${selectedInitiative}${round ? ` · round ${round.round}` : ""}`;
  }
  if (!activeStep) return source === "contract" ? "Template with unfilled slots" : "Preset data · no trace";
  const position = `step ${activeStep.index + 1} / ${traceLength}`;
  return source === "contract"
    ? `Template with unfilled slots · ${position}`
    : `Preset data · ${position}`;
}

function FlowEmptyState({
  source,
  simulationLoading,
  simulationError,
  liveLoading,
  liveError,
  hasLinked,
  selectedInitiative,
}: {
  source: PhaseViewSource;
  simulationLoading: boolean;
  simulationError: Error | null;
  liveLoading: boolean;
  liveError: Error | null;
  hasLinked: boolean;
  selectedInitiative: string;
}) {
  const message = (() => {
    if (source === "live") {
      if (liveLoading) return "Loading live rounds…";
      if (liveError) return liveError.message;
      if (!hasLinked) return "No initiatives use this mode yet. Switch an initiative to this mode to replay its real rounds here.";
      if (!selectedInitiative) return "Select a linked initiative to render its live phase prompt.";
      return `${selectedInitiative} has no live or completed rounds yet.`;
    }
    if (simulationLoading) return "Loading simulation…";
    if (simulationError) return simulationError.message;
    return "No phase to show.";
  })();
  const tone = source === "live" && liveError ? "text-red-400" : "text-slate-500";
  return (
    <section className="rounded-lg border border-slate-800 bg-slate-950/50 p-4">
      <p className={cn("text-sm italic", tone)}>{message}</p>
    </section>
  );
}

function SimulationPresetSelector({
  presets,
  selected,
  disabled,
  onSelect,
}: {
  presets: OperatingModeSimulationPreset[];
  selected?: OperatingModeSimulationPreset;
  disabled: boolean;
  onSelect: (preset: string) => void;
}) {
  if (presets.length === 0) return null;
  return (
    <div className="mt-3 rounded-md border border-slate-800 bg-slate-900/40 p-2.5">
      <div className="flex flex-wrap items-center gap-2">
        <label className="text-[11px] font-medium uppercase tracking-wide text-slate-400" htmlFor="flow-preset-select">
          Scenario
        </label>
        <select
          id="flow-preset-select"
          value={selected?.id ?? ""}
          disabled={disabled}
          onChange={(event) => onSelect(event.target.value)}
          className="min-w-0 flex-1 rounded border border-slate-700 bg-slate-900 px-2 py-1 text-xs text-slate-200 disabled:opacity-60"
          data-testid={selectors.initiativeDetails.flowPresetSelect}
          aria-label="Simulation scenario preset"
        >
          {presets.map((preset) => (
            <option key={preset.id} value={preset.id}>
              {preset.label}
            </option>
          ))}
        </select>
      </div>
      {selected && (
        <div className="mt-1.5" data-testid={selectors.initiativeDetails.flowPresetScenario}>
          <p className="text-[11px] leading-relaxed text-slate-400">{selected.description}</p>
          {selected.branch && (
            <p className="mt-0.5 text-[11px] font-medium text-cyan-300/90">Demonstrates: {selected.branch}</p>
          )}
        </div>
      )}
    </div>
  );
}

function selectLiveRound(rounds: OperatingModeRound[]): OperatingModeRound | undefined {
  return rounds.find(isLiveRoundActive)
    ?? [...rounds].reverse().find((round) => round.status === "pending_evidence" || round.status === "needs_attention" || round.status === "completed");
}

function isLiveRoundActive(round: OperatingModeRound): boolean {
  return round.status === "reserved" || round.status === "agent_running";
}

function ExplainableLabel({
  label,
  onOpen,
  testId,
}: {
  label: string;
  onOpen: () => void;
  testId: string;
}) {
  return (
    <div className="flex items-center gap-1">
      <dt className="text-slate-500">{label}</dt>
      <button
        type="button"
        onClick={onOpen}
        className="rounded p-0.5 text-slate-500 transition-colors hover:bg-white/5 hover:text-slate-200"
        aria-label={`What ${label.toLowerCase()} means`}
        data-testid={testId}
      >
        <Info className="h-3 w-3" />
      </button>
    </div>
  );
}

function LearnMoreLinks({
  mode,
  executionModesUrl,
  holisticLoopUrl,
  phasedPlanDrainUrl,
}: {
  mode: string;
  executionModesUrl: string | null;
  holisticLoopUrl: string | null;
  phasedPlanDrainUrl: string | null;
}) {
  // Always offer the canonical EXECUTION-MODES doc; mode-specific guides are
  // surfaced when applicable.
  const links: Array<{ key: string; label: string; url: string | null }> = [
    {
      key: "execution-modes",
      label: "Execution modes (concept)",
      url: executionModesUrl,
    },
  ];
  if (mode === "holistic-loop") {
    links.push({ key: "holistic", label: "Holistic loop guide", url: holisticLoopUrl });
  }
  if (mode === "phased-plan-drain") {
    links.push({ key: "phased", label: "Phased plan drain guide", url: phasedPlanDrainUrl });
  }
  const docsUnavailable = links.every((link) => link.url === null);
  return (
    <div
      className="space-y-2"
      data-testid={selectors.initiativeDetails.modeDetailsLearnMoreSection}
    >
      {docsUnavailable ? (
        <p className="text-sm italic text-slate-500">Docs server unavailable.</p>
      ) : null}
      <ul className="space-y-1.5">
        {links.map((link) => (
          <li key={link.key}>
            {link.url ? (
              <a
                href={link.url}
                target="_blank"
                rel="noreferrer"
                className="inline-flex items-center gap-1.5 text-sm text-cyan-300 hover:text-cyan-200"
              >
                {link.label}
                <ArrowUpRight className="h-3.5 w-3.5" aria-hidden="true" />
              </a>
            ) : (
              <span className="inline-flex items-center gap-1.5 text-sm text-slate-500">
                {link.label}
                <ArrowUpRight className="h-3.5 w-3.5" aria-hidden="true" />
              </span>
            )}
          </li>
        ))}
      </ul>
    </div>
  );
}
