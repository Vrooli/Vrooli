import { useQuery } from "@tanstack/react-query";
import { useEffect, useMemo, useState } from "react";
import {
	AlertTriangle,
	ClipboardCheck,
	FileSearch,
  Gauge,
  GitCompare,
  RefreshCw,
	Save,
	Wand2,
} from "lucide-react";
import { Link, useParams } from "react-router-dom";

import {
  applyFindingsFixes,
  applyStudioDraft,
  compareStudioVariants,
  fetchEvidence,
  fetchFindings,
  fetchFleet,
  fetchScenarioSpec,
  previewFindingsFixes,
  recaptureScenario,
  renderStudioSpec,
  promoteStudioVariant,
  suggestStudioBindings,
  type ExperienceClaimSpec,
  type ReconciliationEvidenceRow,
  type ScenarioSpecPage,
  type StudioApplyResult,
  type StudioPageDraft,
} from "../api/experience";
import type { FixResponse } from "@vrooli/proto-types/scenario-validation/v1/validation_pb";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Textarea } from "../components/ui/textarea";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

function PageFrame({
  testId,
  title,
  description,
  children,
}: {
  testId: string;
  title: string;
  description: string;
  children: React.ReactNode;
}) {
  return (
    <section data-testid={testId} aria-labelledby={`${testId}-heading`} className="flex min-w-0 flex-col gap-5">
      <div className="flex flex-col gap-2">
        <h2 id={`${testId}-heading`} className="text-2xl font-semibold text-app-foreground">
          {title}
        </h2>
        <p className="max-w-3xl text-sm text-app-muted-foreground">{description}</p>
      </div>
      {children}
    </section>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-panel border border-app-border bg-app-surface p-4">
      <p className="text-xs font-semibold uppercase text-app-muted-foreground">{label}</p>
      <p className="mt-2 text-2xl font-semibold">{value}</p>
    </div>
  );
}

function displayDepth(page: ScenarioSpecPage) {
  const spec = page.spec;
  if ((spec.states?.length ?? 0) > 1) {
    return "L3";
  }
  if ((spec.elements?.length ?? 0) > 0 && (spec.claims?.length ?? 0) > 0) {
    return "L2";
  }
  if ((spec.priorities?.length ?? 0) > 0) {
    return "L1";
  }
  return "L0";
}

function stateDepth(page: ScenarioSpecPage, state: string) {
  return page.spec.states?.some((entry) => entry.id === state) ? displayDepth(page) : "-";
}

function machineClaims(pages: ScenarioSpecPage[]) {
  return pages.flatMap((page) =>
    (page.spec.claims ?? [])
      .filter((claim) => claim.tier === "machine")
      .map((claim) => ({ page, claim })),
  );
}

function claimEvidencePath(scenario: string, pageID: string) {
  return `/scenarios/${scenario}/pages/${pageID}/evidence`;
}

function newestEvidence(rows: ReconciliationEvidenceRow[]) {
  return [...rows].sort((a, b) => Date.parse(b.checkedAt) - Date.parse(a.checkedAt))[0];
}

function captureImageSource(captureRef: string) {
  if (/^(https?:|data:|blob:|\/)/.test(captureRef)) {
    return captureRef;
  }
  return "";
}

function evidenceIsStale(row: ReconciliationEvidenceRow | undefined) {
  if (!row?.checkedAt) {
    return false;
  }
  const checked = Date.parse(row.checkedAt);
  return Number.isFinite(checked) && Date.now() - checked > 5 * 60_000;
}

function formatEvidenceMeta(row: ReconciliationEvidenceRow) {
  const viewport = row.viewport
    ? `${row.viewport}${row.viewportWidth && row.viewportHeight ? ` ${row.viewportWidth}x${row.viewportHeight}` : ""}`
    : "";
  return [row.claimType, viewport, row.checkedAt, row.verdict].filter(Boolean).join(" · ");
}

function formatAXNode(json: string) {
  const trimmed = json.trim();
  if (trimmed === "" || trimmed === "{}") {
    return "";
  }
  try {
    return JSON.stringify(JSON.parse(trimmed), null, 2);
  } catch {
    return trimmed;
  }
}

function firstMachineClaim(page: ScenarioSpecPage | undefined) {
  return page?.spec.claims?.find((claim) => claim.tier === "machine") ?? page?.spec.claims?.[0];
}

function pageDraftFromSpec(page: ScenarioSpecPage | undefined, title: string, claimStatement: string): StudioPageDraft {
  const spec = page?.spec;
  const pageID = spec?.page.id || page?.document.id || "new-page";
  const baseClaim = firstMachineClaim(page);
  const claimID = baseClaim?.id || `${pageID}-draft-claim`;
  return {
    id: pageID,
    title,
    purpose: spec?.page.purpose ?? "",
    routes: spec?.page.routes ?? [],
    prdRefs: spec?.page.prd_refs ?? [],
    status: page?.document.status || "draft",
    priorities: (spec?.priorities ?? []).map((priority) => ({
      statement: priority.statement,
      notes: priority.notes ?? "",
    })),
    states: (spec?.states ?? [{ id: "default", description: "" }]).map((state) => ({
      id: state.id,
      description: state.description ?? "",
    })),
    elements: (spec?.elements ?? []).map((element) => ({
      id: element.id,
      role: element.role ?? "",
      name: element.name ?? "",
      description: element.description ?? "",
    })),
    claims: [
      {
        id: claimID,
        type: baseClaim?.type ?? "custom",
        statement: claimStatement,
        tier: baseClaim?.tier ?? "machine",
        elements: baseClaim?.elements ?? [],
        states: baseClaim?.states ?? ["default"],
        viewports: [],
        locales: [],
        rationale: "",
      },
    ],
    bindings: [],
    sketchRegions: [],
  };
}

function validationText(result: StudioApplyResult | undefined, fallback: string) {
  const findings = result?.validation?.report?.findings ?? [];
  if (findings.length === 0) {
    return fallback;
  }
  return findings.map((finding) => `${finding.severity}: ${finding.title}`).join("\n");
}

function uniqueRuleIDs(response: FixResponse | undefined) {
  return Array.from(new Set(response?.candidates.map((candidate) => candidate.ruleId).filter(Boolean) ?? []));
}

function fixPreviewText(response: FixResponse | undefined, emptyCopy: string) {
  if (!response) {
    return emptyCopy;
  }
  if (response.candidates.length === 0) {
    return response.messages.join("\n") || emptyCopy;
  }
  return response.candidates
    .map((candidate) => {
      const beforeLines = candidate.before ? candidate.before.split("\n").length : 0;
      const afterLines = candidate.after ? candidate.after.split("\n").length : 0;
      return `${candidate.ruleId} ${candidate.filePath}\n${candidate.description}\n-${beforeLines} +${afterLines}`;
    })
    .join("\n\n");
}

export function FleetPage() {
  const { t } = useTranslation();
  const {
    data,
    dataUpdatedAt,
    isError,
    isFetching,
    isLoading,
    refetch,
  } = useQuery({
    queryKey: ["experience-fleet"],
    queryFn: fetchFleet,
    staleTime: 60_000,
  });
  const rows = data?.scenarios ?? [];
  const covered = data?.withExperienceCount ?? 0;
  const total = data?.scenarioCount ?? 0;
  const coveragePercent = total > 0 ? Math.round((covered / total) * 100) : 0;
  const stale = Boolean(data && dataUpdatedAt && Date.now() - dataUpdatedAt > 60_000);

  return (
    <PageFrame
      testId={selectors.pages.fleet}
      title={t(strings.experience.fleet.title)}
      description={t(strings.experience.fleet.description)}
    >
      <div
        data-testid={selectors.experience.fleet.depthSummary}
        role="region"
        aria-label={t(strings.experience.fleet.depthLabel)}
        className="grid gap-3 md:grid-cols-[1fr_1fr_auto]"
      >
        <Metric label={t(strings.experience.fleet.specCoverage)} value={`${covered} / ${total}`} />
        <div
          data-testid={selectors.experience.fleet.coverageMeter}
          role="meter"
          aria-label={t(strings.experience.fleet.specCoverage)}
          aria-valuemin={0}
          aria-valuemax={100}
          aria-valuenow={coveragePercent}
          className="rounded-panel border border-app-border bg-app-surface p-4"
        >
          <p className="text-xs font-semibold uppercase text-app-muted-foreground">
            {t(strings.experience.fleet.depthDistribution)}
          </p>
          <div className="mt-3 h-3 rounded-full bg-app-surface-muted">
            <div className="h-3 rounded-full bg-app-primary" style={{ width: `${coveragePercent}%` }} />
          </div>
          <p className="mt-2 text-sm text-app-muted-foreground">
            {isLoading
              ? t(strings.experience.fleet.loadingData)
              : stale
                ? t(strings.experience.fleet.staleData)
                : t(strings.experience.fleet.pagesTracked, { count: data?.totalPages ?? 0 })}
          </p>
        </div>
        <Button
          data-testid={selectors.experience.fleet.refreshAction}
          type="button"
          onClick={() => void refetch()}
        >
          <RefreshCw className="mr-2 size-4" aria-hidden="true" />
          {isFetching ? t(strings.experience.explorer.refreshing) : t(strings.experience.fleet.refresh)}
        </Button>
      </div>
      {isError ? (
        <div role="alert" className="rounded-panel border border-app-border bg-app-surface p-4 text-sm text-app-muted-foreground">
          {t(strings.experience.fleet.loadError)}
        </div>
      ) : null}
      <div className="min-w-0 max-w-full overflow-x-auto rounded-panel border border-app-border bg-app-surface">
        <table
          data-testid={selectors.experience.fleet.debtTable}
          aria-label={t(strings.experience.fleet.tableLabel)}
          className="min-w-full text-left text-sm"
        >
          <thead className="bg-app-surface-muted text-xs uppercase text-app-muted-foreground">
            <tr>
              <th className="px-4 py-3">{t(strings.experience.common.scenario)}</th>
              <th className="px-4 py-3">{t(strings.experience.common.depth)}</th>
              <th className="px-4 py-3">{t(strings.experience.common.debt)}</th>
              <th className="px-4 py-3">{t(strings.experience.common.status)}</th>
            </tr>
          </thead>
          <tbody>
            {isLoading ? (
              <tr className="border-t border-app-border">
                <td className="px-4 py-3 text-app-muted-foreground" colSpan={4}>
                  {t(strings.experience.fleet.loadingData)}
                </td>
              </tr>
            ) : rows.length === 0 ? (
              <tr className="border-t border-app-border">
                <td className="px-4 py-3 text-app-muted-foreground" colSpan={4}>
                  {t(strings.experience.fleet.emptyFleet)}
                </td>
              </tr>
            ) : (
              rows.map((scenario) => (
              <tr key={scenario.scenario} className="border-t border-app-border">
                <td className="px-4 py-3">
                  <Link
                    data-testid={selectors.experience.fleet.scenarioLink}
                    className="inline-flex min-h-11 items-center font-medium text-app-primary underline-offset-4 hover:underline"
                    to={`/scenarios/${scenario.scenario}`}
                  >
                    {scenario.scenario}
                  </Link>
                </td>
                <td className="px-4 py-3">{scenario.maxDepth}</td>
                <td className="px-4 py-3">{scenario.debtScore}</td>
                <td className="px-4 py-3">{scenario.status}</td>
              </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </PageFrame>
  );
}

export function ScenarioExplorerPage() {
  const { t } = useTranslation();
  const params = useParams();
  const scenario = params.scenario ?? "experience-manager";
  const {
    data: pages,
    dataUpdatedAt,
    isError,
    isFetching,
    isLoading,
    refetch,
  } = useQuery({
    queryKey: ["experience-scenario-spec", scenario],
    queryFn: () => fetchScenarioSpec(scenario),
    staleTime: 60_000,
  });
  const rows = pages ?? [];
  const claims = machineClaims(rows);
  const stale = Boolean(dataUpdatedAt && Date.now() - dataUpdatedAt > 60_000);

  return (
    <PageFrame
      testId={selectors.pages.explorer}
      title={t(strings.experience.explorer.title)}
      description={t(strings.experience.explorer.description)}
    >
      <div className="grid min-w-0 gap-4 xl:grid-cols-[minmax(0,1fr)_20rem]">
        <div className="min-w-0 max-w-full overflow-x-auto rounded-panel border border-app-border bg-app-surface">
          <table
            data-testid={selectors.experience.explorer.depthGrid}
            aria-label={t(strings.experience.explorer.gridLabel)}
            className="min-w-full text-left text-sm"
          >
            <thead className="bg-app-surface-muted text-xs uppercase text-app-muted-foreground">
              <tr>
                <th className="px-4 py-3">{t(strings.experience.common.page)}</th>
                <th className="px-4 py-3">{t(strings.experience.explorer.defaultState)}</th>
                <th className="px-4 py-3">{t(strings.experience.explorer.emptyState)}</th>
                <th className="px-4 py-3">{t(strings.experience.explorer.staleState)}</th>
                <th className="px-4 py-3">{t(strings.experience.common.claims)}</th>
              </tr>
            </thead>
            <tbody>
              {isLoading ? (
                <tr className="border-t border-app-border">
                  <td className="px-4 py-3 text-app-muted-foreground" colSpan={5}>
                    {t(strings.experience.explorer.loadingSpec)}
                  </td>
                </tr>
              ) : rows.length === 0 ? (
                <tr className="border-t border-app-border">
                  <td className="px-4 py-3 text-app-muted-foreground" colSpan={5}>
                    {t(strings.experience.explorer.emptySpec)}
                  </td>
                </tr>
              ) : (
                rows.map((row) => (
                  <tr key={row.document.id} className="border-t border-app-border">
                    <td className="px-4 py-3 font-medium">{row.document.title || row.spec.page.title}</td>
                    <td className="px-4 py-3">{stateDepth(row, "default")}</td>
                    <td className="px-4 py-3">{stateDepth(row, "empty")}</td>
                    <td className="px-4 py-3">{stateDepth(row, "stale")}</td>
                    <td className="px-4 py-3">{row.spec.claims?.length ?? 0}</td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
        <aside
          data-testid={selectors.experience.explorer.gapPanel}
          role={isError ? "alert" : "region"}
          aria-label={t(strings.experience.explorer.gapsLabel)}
          className="min-w-0 rounded-panel border border-app-border bg-app-surface p-4"
        >
          <h3 className="font-semibold">{t(strings.experience.explorer.gapsLabel)}</h3>
          {isError ? (
            <p className="mt-2 text-sm text-app-muted-foreground">
              {t(strings.experience.explorer.loadError)}
            </p>
          ) : rows.length === 0 && !isLoading ? (
            <p className="mt-2 text-sm text-app-muted-foreground">
              {t(strings.experience.explorer.emptyGap)}
            </p>
          ) : (
            <p className="mt-2 text-sm text-app-muted-foreground">
              {stale
                ? t(strings.experience.explorer.staleData)
                : t(strings.experience.explorer.summary, {
                    claims: claims.length,
                    pages: rows.length,
                  })}
            </p>
          )}
          <Button
            data-testid={selectors.experience.explorer.studioAction}
            type="button"
            className="mt-4"
            onClick={() => void refetch()}
          >
            <Wand2 className="mr-2 size-4" aria-hidden="true" />
            {isFetching ? t(strings.experience.explorer.refreshing) : t(strings.experience.explorer.openStudio)}
          </Button>
        </aside>
      </div>
      <ul
        data-testid={selectors.experience.explorer.claimList}
        aria-label={t(strings.experience.explorer.claimsLabel)}
        className="grid gap-3 md:grid-cols-2"
      >
        {isLoading ? (
          <li className="rounded-panel border border-app-border bg-app-surface p-4 text-sm text-app-muted-foreground">
            {t(strings.experience.explorer.loadingClaims)}
          </li>
        ) : claims.length === 0 ? (
          <li className="rounded-panel border border-app-border bg-app-surface p-4 text-sm text-app-muted-foreground">
            {t(strings.experience.explorer.emptyClaims)}
          </li>
        ) : (
          claims.map(({ page, claim }: { page: ScenarioSpecPage; claim: ExperienceClaimSpec }) => (
            <li key={`${page.document.id}:${claim.id}`} className="rounded-panel border border-app-border bg-app-surface p-4">
              <span
                data-testid={selectors.experience.explorer.tierLabel}
                role="note"
                aria-label={`${t(strings.experience.common.tier)} ${claim.tier}`}
                className="text-xs font-semibold uppercase text-app-primary"
              >
                {claim.tier}
              </span>
              <p className="mt-2 font-medium">{claim.id}</p>
              <p className="mt-1 text-sm text-app-muted-foreground">{claim.type}</p>
              <Link
                data-testid={selectors.experience.explorer.evidenceLink}
                to={claimEvidencePath(scenario, page.document.id)}
                aria-label={`${t(strings.experience.common.viewEvidence)} ${page.document.title} ${claim.id}`}
                className="mt-3 inline-flex text-sm text-app-primary underline-offset-4 hover:underline"
              >
                {t(strings.experience.common.viewEvidence)}
              </Link>
            </li>
          ))
        )}
      </ul>
    </PageFrame>
  );
}

export function EvidencePage() {
  const { t } = useTranslation();
  const params = useParams();
  const scenario = params.scenario ?? "experience-manager";
  const page = params.page ?? "fleet";
  const [selectedID, setSelectedID] = useState("");
  const {
    data: evidence,
    isError,
    isFetching,
    isLoading,
    refetch,
  } = useQuery({
    queryKey: ["experience-evidence", scenario, page],
    queryFn: () => fetchEvidence({ scenario, page }),
    staleTime: 60_000,
  });
  const rows = useMemo(() => evidence ?? [], [evidence]);
  const selected = useMemo(
    () => rows.find((row) => row.id === selectedID) ?? newestEvidence(rows),
    [rows, selectedID],
  );
  const captureRef = selected?.captureRef ?? "";
  const captureSrc = captureImageSource(captureRef);
  const axNode = formatAXNode(selected?.axNodeJson ?? "");
  const stale = evidenceIsStale(selected);
  const recapture = async () => {
    await recaptureScenario(scenario);
    await refetch();
  };

  return (
    <PageFrame
      testId={selectors.pages.evidence}
      title={t(strings.experience.evidence.title)}
      description={t(strings.experience.evidence.description)}
    >
      <div className="grid min-w-0 gap-4 xl:grid-cols-[minmax(0,1.2fr)_minmax(0,0.8fr)]">
        <div className="min-w-0 rounded-panel border border-app-border bg-app-surface p-4">
          {isLoading ? (
            <div
              data-testid={selectors.experience.evidence.captureImage}
              role="img"
              aria-label={t(strings.experience.evidence.captureLabel)}
              className="flex min-h-72 w-full items-center justify-center rounded-control border border-dashed border-app-border bg-app-surface-muted text-sm text-app-muted-foreground"
            >
              {t(strings.experience.evidence.loadingEvidence)}
            </div>
          ) : captureSrc ? (
            <img
              data-testid={selectors.experience.evidence.captureImage}
              src={captureSrc}
              alt={t(strings.experience.evidence.captureLabel)}
              className="min-h-72 w-full rounded-control border border-dashed border-app-border bg-app-surface-muted object-cover"
            />
          ) : (
            <div
              data-testid={selectors.experience.evidence.captureImage}
              role="img"
              aria-label={t(strings.experience.evidence.captureLabel)}
              className="flex min-h-72 w-full flex-col items-center justify-center rounded-control border border-dashed border-app-border bg-app-surface-muted p-4 text-center text-sm text-app-muted-foreground"
            >
              <span>{rows.length === 0 ? t(strings.experience.evidence.emptyEvidence) : t(strings.experience.evidence.captureReference)}</span>
              {captureRef ? <code className="mt-2 max-w-full break-all text-xs">{captureRef}</code> : null}
            </div>
          )}
          <Button
            data-testid={selectors.experience.evidence.recaptureAction}
            type="button"
            className="mt-4"
            onClick={() => void recapture()}
          >
            <RefreshCw className="mr-2 size-4" aria-hidden="true" />
            {isFetching ? t(strings.experience.evidence.refreshing) : t(strings.experience.evidence.recapture)}
          </Button>
          {stale ? <p className="mt-2 text-sm text-app-warning">{t(strings.experience.evidence.staleEvidence)}</p> : null}
        </div>
        <div
          data-testid={selectors.experience.evidence.treePanel}
          role={isError ? "alert" : "region"}
          aria-label={t(strings.experience.evidence.treeLabel)}
          className="min-w-0 rounded-panel border border-app-border bg-app-surface p-4"
        >
          <h3 className="font-semibold">{t(strings.experience.evidence.treeLabel)}</h3>
          <pre className="mt-3 max-w-full overflow-auto rounded-control bg-app-surface-muted p-3 text-xs">
            {isError
              ? t(strings.experience.evidence.loadError)
              : axNode || t(strings.experience.evidence.emptyTree)}
          </pre>
        </div>
      </div>
      <ul
        data-testid={selectors.experience.evidence.verdictList}
        aria-label={t(strings.experience.evidence.verdictsLabel)}
        className="grid gap-3 md:grid-cols-2"
      >
        {isLoading ? (
          <li className="rounded-panel border border-app-border bg-app-surface p-4 text-sm text-app-muted-foreground">
            {t(strings.experience.evidence.loadingEvidence)}
          </li>
        ) : rows.length === 0 ? (
          <li className="rounded-panel border border-app-border bg-app-surface p-4 text-sm text-app-muted-foreground">
            {t(strings.experience.evidence.emptyVerdicts)}
          </li>
        ) : (
          rows.map((row) => (
            <li key={row.id} className="rounded-panel border border-app-border bg-app-surface p-4">
              <p className="font-medium">{row.claim}</p>
              <p className="text-sm text-app-muted-foreground">{formatEvidenceMeta(row)}</p>
              {row.message ? <p className="mt-2 text-sm text-app-muted-foreground">{row.message}</p> : null}
              <button
                data-testid={selectors.experience.evidence.evidenceLink}
                type="button"
                onClick={() => setSelectedID(row.id)}
                aria-label={`${t(strings.experience.common.viewEvidence)} ${row.claim}`}
                className="mt-3 inline-flex text-sm text-app-primary underline-offset-4 hover:underline"
              >
                {t(strings.experience.common.viewEvidence)}
              </button>
            </li>
          ))
        )}
      </ul>
    </PageFrame>
  );
}

export function StudioPage() {
  const { t } = useTranslation();
  const params = useParams();
  const scenario = params.scenario ?? "experience-manager";
  const [selectedPageID, setSelectedPageID] = useState(params.page ?? "");
  const [initializedPageID, setInitializedPageID] = useState("");
  const [title, setTitle] = useState("");
  const [claimStatement, setClaimStatement] = useState("");
  const [result, setResult] = useState<StudioApplyResult>();
  const [isSaving, setIsSaving] = useState(false);
  const [saveError, setSaveError] = useState("");
  const {
    data: pages,
    isError: specError,
    isLoading: specLoading,
  } = useQuery({
    queryKey: ["experience-studio-spec", scenario],
    queryFn: () => fetchScenarioSpec(scenario),
    staleTime: 60_000,
  });
  const rows = pages ?? [];
  const selectedPage = rows.find((row) => row.document.id === selectedPageID) ?? rows[0];
  const selectedID = selectedPage?.document.id ?? selectedPageID;

  useEffect(() => {
    if (!selectedPage) {
      return;
    }
    if (initializedPageID === selectedPage.document.id) {
      return;
    }
    setSelectedPageID(selectedPage.document.id);
    setTitle(selectedPage.spec.page.title || selectedPage.document.title);
    setClaimStatement(firstMachineClaim(selectedPage)?.statement ?? "");
    setResult(undefined);
    setSaveError("");
    setInitializedPageID(selectedPage.document.id);
  }, [initializedPageID, selectedPage]);

  const draft = useMemo(
    () => pageDraftFromSpec(selectedPage, title, claimStatement),
    [claimStatement, selectedPage, title],
  );
  const variants = useMemo(
    () => [
      { id: "draft", title: title || selectedPage?.document.title || "draft", page: draft },
      {
        id: "evidence-forward",
        title: t(strings.experience.studio.evidenceForwardVariant),
        page: {
          ...draft,
          title: `${title || selectedPage?.document.title || "draft"} evidence`,
          claims: draft.claims.map((claim) => ({
            ...claim,
            statement: claim.statement || t(strings.experience.studio.emptyClaim),
          })),
        },
      },
    ],
    [draft, selectedPage?.document.title, t, title],
  );
  const renderQuery = useQuery({
    queryKey: ["experience-studio-render", scenario, selectedID],
    queryFn: () => renderStudioSpec(scenario, selectedID),
    enabled: Boolean(selectedID),
    staleTime: 60_000,
  });
  const compareQuery = useQuery({
    queryKey: ["experience-studio-compare", scenario, selectedID, title, claimStatement],
    queryFn: () => compareStudioVariants(scenario, selectedID, variants),
    enabled: Boolean(selectedID && title),
    staleTime: 5_000,
  });
  const suggestionsQuery = useQuery({
    queryKey: ["experience-studio-bindings", scenario, selectedID],
    queryFn: () => suggestStudioBindings(scenario, selectedID),
    enabled: Boolean(selectedID),
    staleTime: 60_000,
  });
  const previewHTML = compareQuery.data?.html || renderQuery.data?.html || "";
  const renderedVariants = compareQuery.data?.variants ?? [];
  const validationCopy = saveError || validationText(result, t(strings.experience.studio.validationCopy));
  const saveDraft = async () => {
    setIsSaving(true);
    setSaveError("");
    try {
      setResult(await applyStudioDraft(scenario, draft));
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : t(strings.errors.unknown));
    } finally {
      setIsSaving(false);
    }
  };
  const promoteDraft = async () => {
    const primaryVariant = variants[0];
    if (!primaryVariant) {
      return;
    }
    setIsSaving(true);
    setSaveError("");
    try {
      setResult(await promoteStudioVariant(scenario, selectedID, primaryVariant));
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : t(strings.errors.unknown));
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <PageFrame
      testId={selectors.pages.studio}
      title={t(strings.experience.studio.title)}
      description={t(strings.experience.studio.description)}
    >
      <div className="grid min-w-0 gap-4 xl:grid-cols-[minmax(0,0.85fr)_minmax(0,1.15fr)]">
        <form
          data-testid={selectors.experience.studio.specForm}
          aria-label={t(strings.experience.studio.formLabel)}
          className="min-w-0 rounded-panel border border-app-border bg-app-surface p-4"
        >
          <label className="block text-sm font-semibold" htmlFor="studio-page-select">
            {t(strings.experience.common.page)}
          </label>
          <select
            id="studio-page-select"
            className="mt-2 min-h-11 w-full rounded-control border border-app-border bg-app-surface px-3 text-sm text-app-foreground"
            value={selectedID}
            onChange={(event) => {
              setInitializedPageID("");
              setSelectedPageID(event.target.value);
            }}
            disabled={specLoading || rows.length === 0}
          >
            {specLoading ? <option value="">{t(strings.experience.studio.loadingSpec)}</option> : null}
            {!specLoading && rows.length === 0 ? <option value="">{t(strings.experience.studio.emptySpec)}</option> : null}
            {rows.map((row) => (
              <option key={row.document.id} value={row.document.id}>
                {row.document.title || row.spec.page.title}
              </option>
            ))}
          </select>
          <label className="mt-4 block text-sm font-semibold" htmlFor="studio-page-title">
            {t(strings.experience.studio.pageTitle)}
          </label>
          <Input
            id="studio-page-title"
            className="mt-2 bg-app-surface text-app-foreground"
            value={title}
            onChange={(event) => setTitle(event.target.value)}
            placeholder={t(strings.experience.studio.defaultPage)}
          />
          <label className="mt-4 block text-sm font-semibold" htmlFor="studio-claim">
            {t(strings.experience.common.claims)}
          </label>
          <Textarea
            id="studio-claim"
            className="mt-2 min-h-28 bg-app-surface text-app-foreground"
            value={claimStatement}
            onChange={(event) => setClaimStatement(event.target.value)}
            placeholder={t(strings.experience.studio.emptyClaim)}
          />
          <div
            data-testid={selectors.experience.studio.validationSummary}
            role={saveError ? "alert" : "status"}
            aria-label={t(strings.experience.studio.validationLabel)}
            className="mt-4 whitespace-pre-line rounded-control border border-app-border bg-app-surface-muted p-3 text-sm"
          >
            {specError ? t(strings.experience.studio.loadError) : validationCopy}
          </div>
          <div className="mt-4 rounded-control border border-app-border p-3">
            <p className="text-xs font-semibold uppercase text-app-muted-foreground">
              {t(strings.experience.studio.suggestionsLabel)}
            </p>
            <ul className="mt-2 grid gap-2 text-sm text-app-muted-foreground">
              {(suggestionsQuery.data ?? []).length === 0 ? (
                <li>{suggestionsQuery.isLoading ? t(strings.experience.studio.loadingSuggestions) : t(strings.experience.studio.emptySuggestions)}</li>
              ) : (
                suggestionsQuery.data?.map((suggestion) => (
                  <li key={`${suggestion.elementId}:${suggestion.testid || suggestion.role}`}>
                    <span className="font-medium text-app-foreground">{suggestion.elementId}</span>{" "}
                    {suggestion.testid || suggestion.role || suggestion.accessibleName}
                  </li>
                ))
              )}
            </ul>
          </div>
          <Button
            data-testid={selectors.experience.studio.saveAction}
            type="button"
            className="mt-4"
            onClick={() => void saveDraft()}
            disabled={!selectedID || isSaving}
          >
            <Save className="mr-2 size-4" aria-hidden="true" />
            {isSaving ? t(strings.experience.studio.saving) : t(strings.experience.studio.save)}
          </Button>
        </form>
        <section
          data-testid={selectors.experience.studio.wireframePreview}
          aria-label={t(strings.experience.studio.wireframeLabel)}
          className="min-w-0 rounded-panel border border-app-border bg-app-surface p-4"
        >
          <h3 className="font-semibold">{t(strings.experience.studio.wireframeLabel)}</h3>
          <div className="mt-4 min-h-64 overflow-auto rounded-control border border-dashed border-app-border bg-app-surface-muted p-4">
            {renderQuery.isLoading || compareQuery.isLoading ? (
              <p className="text-sm text-app-muted-foreground">{t(strings.experience.studio.loadingPreview)}</p>
            ) : previewHTML ? (
              <div className="prose prose-sm max-w-full overflow-x-auto text-app-foreground" dangerouslySetInnerHTML={{ __html: previewHTML }} />
            ) : (
              <p className="text-sm text-app-muted-foreground">{t(strings.experience.studio.emptyPreview)}</p>
            )}
          </div>
          <ul
            data-testid={selectors.experience.studio.variantRail}
            aria-label={t(strings.experience.studio.variantsLabel)}
            className="mt-4 grid gap-3 md:grid-cols-2"
          >
            {renderedVariants.length === 0 ? (
              <li className="rounded-control border border-app-border p-3 text-sm text-app-muted-foreground">
                {compareQuery.isError ? t(strings.experience.studio.variantError) : t(strings.experience.studio.emptyVariants)}
              </li>
            ) : (
              renderedVariants.map((variant) => (
              <li key={variant.id} className="rounded-control border border-app-border p-3 text-sm">
                <GitCompare className="mb-2 size-4 text-app-primary" aria-hidden="true" />
                <span className="font-medium">{variant.title}</span>
              </li>
              ))
            )}
          </ul>
          <Button
            data-testid={selectors.experience.studio.promoteAction}
            type="button"
            className="mt-4"
            onClick={() => void promoteDraft()}
            disabled={!selectedID || isSaving}
          >
            <ClipboardCheck className="mr-2 size-4" aria-hidden="true" />
            {t(strings.experience.studio.promote)}
          </Button>
        </section>
      </div>
    </PageFrame>
  );
}

export function FindingsPage() {
  const { t } = useTranslation();
  const params = useParams();
  const scenario = params.scenario ?? "experience-manager";
  const [preview, setPreview] = useState<FixResponse>();
  const [fixError, setFixError] = useState("");
  const [isFixing, setIsFixing] = useState(false);
  const {
    data: findings,
    isError,
    isFetching,
    isLoading,
    refetch,
  } = useQuery({
    queryKey: ["experience-findings", scenario],
    queryFn: () => fetchFindings(scenario),
    staleTime: 60_000,
  });
  const rows = findings ?? [];
  const previewFixes = async () => {
    setIsFixing(true);
    setFixError("");
    try {
      setPreview(await previewFindingsFixes(scenario));
    } catch (err) {
      setFixError(err instanceof Error ? err.message : t(strings.experience.findings.previewError));
    } finally {
      setIsFixing(false);
    }
  };
  const applyFixes = async () => {
    setIsFixing(true);
    setFixError("");
    try {
      const result = await applyFindingsFixes(scenario, uniqueRuleIDs(preview));
      setPreview(result.preview);
      await refetch();
    } catch (err) {
      setFixError(err instanceof Error ? err.message : t(strings.experience.findings.applyError));
    } finally {
      setIsFixing(false);
    }
  };

  return (
    <PageFrame
      testId={selectors.pages.findings}
      title={t(strings.experience.findings.title)}
      description={t(strings.experience.findings.description)}
    >
      <ul
        data-testid={selectors.experience.findings.findingsList}
        aria-label={t(strings.experience.findings.listLabel)}
        className="grid gap-3"
      >
        {isLoading ? (
          <li className="rounded-panel border border-app-border bg-app-surface p-4 text-sm text-app-muted-foreground">
            {t(strings.experience.findings.loadingFindings)}
          </li>
        ) : isError ? (
          <li role="alert" className="rounded-panel border border-app-border bg-app-surface p-4 text-sm text-app-muted-foreground">
            {t(strings.experience.findings.loadError)}
          </li>
        ) : rows.length === 0 ? (
          <li className="rounded-panel border border-app-border bg-app-surface p-4 text-sm text-app-muted-foreground">
            {t(strings.experience.findings.emptyFindings)}
          </li>
        ) : (
          rows.map((finding) => (
          <li key={`${finding.code}:${finding.location}:${finding.message}`} className="rounded-panel border border-app-border bg-app-surface p-4">
            <div className="flex flex-wrap items-start justify-between gap-3">
              <div>
                <span
                  data-testid={selectors.experience.findings.severityLabel}
                  role="note"
                  className="inline-flex items-center gap-2 text-xs font-semibold uppercase text-app-warning"
                >
                  <AlertTriangle className="size-4" aria-hidden="true" />
                  {finding.severity}
                </span>
                <p className="mt-2 font-medium">{finding.code}</p>
                <p className="mt-1 text-sm text-app-muted-foreground">{finding.message || finding.remediation}</p>
                <Link
                  data-testid={selectors.experience.findings.evidenceLink}
                  to={`/scenarios/${scenario}/pages/findings/evidence`}
                  aria-label={`${t(strings.experience.common.viewEvidence)} ${finding.code}`}
                  className="mt-3 inline-flex text-sm text-app-primary underline-offset-4 hover:underline"
                >
                  {t(strings.experience.common.viewEvidence)}
                </Link>
              </div>
              <div className="flex gap-2">
                <Button
                  data-testid={selectors.experience.findings.previewAction}
                  type="button"
                  variant="outline"
                  onClick={() => void previewFixes()}
                  disabled={isFixing || isFetching}
                >
                  <FileSearch className="mr-2 size-4" aria-hidden="true" />
                  {isFixing ? t(strings.experience.explorer.refreshing) : t(strings.experience.findings.preview)}
                </Button>
                <Button
                  data-testid={selectors.experience.findings.applyAction}
                  type="button"
                  onClick={() => void applyFixes()}
                  disabled={isFixing || uniqueRuleIDs(preview).length === 0}
                >
                  <Gauge className="mr-2 size-4" aria-hidden="true" />
                  {t(strings.experience.findings.apply)}
                </Button>
              </div>
            </div>
          </li>
          ))
        )}
      </ul>
      <section
        role={fixError ? "alert" : "status"}
        aria-label={t(strings.experience.findings.fixPreviewLabel)}
        className="rounded-panel border border-app-border bg-app-surface p-4"
      >
        <h3 className="font-semibold">{t(strings.experience.findings.fixPreviewLabel)}</h3>
        <pre className="mt-3 overflow-auto whitespace-pre-wrap rounded-control bg-app-surface-muted p-3 text-xs">
          {fixError || fixPreviewText(preview, t(strings.experience.findings.previewEmpty))}
        </pre>
      </section>
    </PageFrame>
  );
}
