/**
 * Operating Mode Details Page
 *
 * Shows the catalog metadata for one operating mode (label, description,
 * scope, run strategy, phases) and the list of initiatives currently bound
 * to that mode. Editable fields (label, description) persist via the API
 * overlay store.
 */

import { useEffect, useMemo, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate, useParams } from "react-router-dom";
import {
  ArrowUpRight,
  CheckCircle2,
  Info,
  Layers,
  List,
  Network,
  Pencil,
  Scale,
  X,
  XCircle,
} from "lucide-react";
import { Button } from "../components/ui/button";
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
import type { OperatingModeDetail } from "../types/operating-mode";
import { PhaseGraph } from "../components/initiative/operating-mode/phase-graph";
import { PhaseList } from "../components/initiative/operating-mode/phase-list";
import { CapabilityList } from "../components/initiative/operating-mode/capability-list";
import {
  CAPABILITY_EXPLAINER,
  DEFAULT_FLAG_EXPLAINER,
  RUN_STRATEGY_EXPLAINER,
  SCOPE_KIND_EXPLAINER,
  type ConceptExplainer,
} from "../components/initiative/operating-mode/concept-explainers";
import {
  humanizeRunStrategy,
  humanizeScopeKind,
  phaseCardDomId,
} from "../components/initiative/operating-mode/utils";
import { useUrlState } from "../hooks/use-url-state";
import { useAttachToSessionAction } from "../components/session/context/useAttachToSessionAction";
import { operatingModeOption } from "../components/session/context/session-context-refs";

const EMPTY_LENSES: never[] = [];

type PhasesView = "list" | "graph";

const HIGHLIGHT_DURATION_MS = 1500;

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

  useEffect(() => {
    if (data) {
      setLabelDraft(data.entry.label);
      setDescriptionDraft(data.entry.description ?? "");
    }
  }, [data]);

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

  const [phasesView, setPhasesView] = useUrlState<PhasesView>("view", "graph", {
    validate: (value): value is PhasesView => value === "list" || value === "graph",
  });
  const [highlightedPhaseId, setHighlightedPhaseId] = useState<string | null>(null);
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
      document.getElementById(phaseCardDomId(phase))?.scrollIntoView({
        behavior: "smooth",
        block: "center",
      });
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
          entityType="Operating Mode"
          entityIcon={Layers}
          title={entry.label}
          subtitle={entry.mode}
          nodeId={null}
          lenses={EMPTY_LENSES}
          metadata={
            <span className="rounded-full bg-slate-800 px-2 py-0.5 text-[11px] font-medium text-slate-300">
              {linkedInitiatives.length} initiative{linkedInitiatives.length === 1 ? "" : "s"}
            </span>
          }
          actions={attachToSession.button}
        />
      }
    >
      {attachToSession.sheet}
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
                  label="Scope"
                  onOpen={() => setActiveExplainer(SCOPE_KIND_EXPLAINER)}
                  testId={selectors.initiativeDetails.modeDetailsScopeInfoIcon}
                />
                <dd className="text-slate-200">{humanizeScopeKind(entry.scopeKind)}</dd>
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

      {phases.length > 0 && (
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
                selectedPhaseId={highlightedPhaseId}
                onSelectPhase={handleSelectPhase}
              />
            )}
            <PhaseList phases={phases} highlightedPhaseId={highlightedPhaseId} />
          </div>
        </DetailSection>
      )}

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

      <DetailSection title="When to use">
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

      <DetailSection title="Linked Initiatives">
        {linkedInitiatives.length === 0 ? (
          <p className="text-sm italic text-slate-500">No initiatives currently use this mode.</p>
        ) : (
          <ul className="space-y-1.5">
            {linkedInitiatives.map((init) => (
              <li key={init.name}>
                <button
                  type="button"
                  onClick={() => navigate(initiativeDetailPath(init.name))}
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
    </DetailPageLayout>
  );
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
