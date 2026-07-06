import { useQuery } from "@tanstack/react-query";
import { useMemo, useState } from "react";
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
  fetchEvidence,
  fetchFleet,
  fetchScenarioSpec,
  recaptureScenario,
  type ExperienceClaimSpec,
  type ReconciliationEvidenceRow,
  type ScenarioSpecPage,
} from "../api/experience";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Textarea } from "../components/ui/textarea";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

const scenarios = [
  { name: "business-health", coverage: "L3", debt: 8, pages: 1, status: "8 advisory" },
  { name: "web-console", coverage: "L3", debt: 0, pages: 1, status: "green" },
  { name: "experience-manager", coverage: "L2", debt: 0, pages: 5, status: "dogfood" },
];

const findings = [
  {
    title: "Matrix priority copy diverges from declared intent",
    severity: "warning",
    evidence: "/scenarios/business-health/pages/matrix/evidence",
  },
  {
    title: "Binding suggestion available for findings table",
    severity: "info",
    evidence: "/scenarios/experience-manager/pages/findings/evidence",
  },
];

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
    <section data-testid={testId} aria-labelledby={`${testId}-heading`} className="flex flex-col gap-5">
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
  return [row.claimType, row.checkedAt, row.verdict].filter(Boolean).join(" · ");
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

export function FleetPage() {
  const { t } = useTranslation();
  const { data, refetch } = useQuery({
    queryKey: ["experience-fleet"],
    queryFn: fetchFleet,
  });
  const rows =
    data?.scenarios.map((scenario) => ({
      name: scenario.scenario,
      coverage: scenario.maxDepth,
      debt: scenario.debtScore,
      pages: scenario.pageCount,
      status: scenario.status,
    })) ?? scenarios;
  const covered = data ? data.withExperienceCount : scenarios.length;
  const total = data ? data.scenarioCount : scenarios.length;
  const coveragePercent = total > 0 ? Math.round((covered / total) * 100) : 0;

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
            {data
              ? t(strings.experience.fleet.pagesTracked, { count: data.totalPages })
              : t(strings.experience.fleet.loadingData)}
          </p>
        </div>
        <Button
          data-testid={selectors.experience.fleet.refreshAction}
          type="button"
          onClick={() => void refetch()}
        >
          <RefreshCw className="mr-2 size-4" aria-hidden="true" />
          {t(strings.experience.fleet.refresh)}
        </Button>
      </div>
      <div className="overflow-x-auto rounded-panel border border-app-border bg-app-surface">
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
            {rows.map((scenario) => (
              <tr key={scenario.name} className="border-t border-app-border">
                <td className="px-4 py-3">
                  <Link
                    data-testid={selectors.experience.fleet.scenarioLink}
                    className="font-medium text-app-primary underline-offset-4 hover:underline"
                    to={`/scenarios/${scenario.name}`}
                  >
                    {scenario.name}
                  </Link>
                </td>
                <td className="px-4 py-3">{scenario.coverage}</td>
                <td className="px-4 py-3">{scenario.debt}</td>
                <td className="px-4 py-3">{scenario.status}</td>
              </tr>
            ))}
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
      <div className="grid gap-4 xl:grid-cols-[1fr_20rem]">
        <div className="overflow-x-auto rounded-panel border border-app-border bg-app-surface">
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
          className="rounded-panel border border-app-border bg-app-surface p-4"
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
                className="text-xs font-semibold uppercase text-app-primary"
              >
                {claim.tier}
              </span>
              <p className="mt-2 font-medium">{claim.id}</p>
              <p className="mt-1 text-sm text-app-muted-foreground">{claim.type}</p>
              <Link
                data-testid={selectors.experience.explorer.evidenceLink}
                to={claimEvidencePath(scenario, page.document.id)}
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
      <div className="grid gap-4 xl:grid-cols-[1.2fr_0.8fr]">
        <div className="rounded-panel border border-app-border bg-app-surface p-4">
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
              {captureRef ? <code className="mt-2 break-all text-xs">{captureRef}</code> : null}
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
          className="rounded-panel border border-app-border bg-app-surface p-4"
        >
          <h3 className="font-semibold">{t(strings.experience.evidence.treeLabel)}</h3>
          <pre className="mt-3 overflow-auto rounded-control bg-app-surface-muted p-3 text-xs">
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

  return (
    <PageFrame
      testId={selectors.pages.studio}
      title={t(strings.experience.studio.title)}
      description={t(strings.experience.studio.description)}
    >
      <div className="grid gap-4 xl:grid-cols-[minmax(20rem,0.85fr)_1.15fr]">
        <form
          data-testid={selectors.experience.studio.specForm}
          aria-label={t(strings.experience.studio.formLabel)}
          className="rounded-panel border border-app-border bg-app-surface p-4"
        >
          <label className="block text-sm font-semibold" htmlFor="studio-page-title">
            {t(strings.experience.common.page)}
          </label>
          <Input id="studio-page-title" className="mt-2 bg-app-surface text-app-foreground" defaultValue={t(strings.experience.studio.defaultPage)} />
          <label className="mt-4 block text-sm font-semibold" htmlFor="studio-claim">
            {t(strings.experience.common.claims)}
          </label>
          <Textarea
            id="studio-claim"
            className="mt-2 min-h-28 bg-app-surface text-app-foreground"
            defaultValue={t(strings.experience.studio.defaultClaim)}
          />
          <div
            data-testid={selectors.experience.studio.validationSummary}
            role="alert"
            aria-label={t(strings.experience.studio.validationLabel)}
            className="mt-4 rounded-control border border-app-border bg-app-surface-muted p-3 text-sm"
          >
            {t(strings.experience.studio.validationCopy)}
          </div>
          <Button data-testid={selectors.experience.studio.saveAction} type="button" className="mt-4">
            <Save className="mr-2 size-4" aria-hidden="true" />
            {t(strings.experience.studio.save)}
          </Button>
        </form>
        <section
          data-testid={selectors.experience.studio.wireframePreview}
          aria-label={t(strings.experience.studio.wireframeLabel)}
          className="rounded-panel border border-app-border bg-app-surface p-4"
        >
          <h3 className="font-semibold">{t(strings.experience.studio.wireframeLabel)}</h3>
          <div className="mt-4 grid min-h-64 gap-3 rounded-control border border-dashed border-app-border p-4 md:grid-cols-3">
            <div className="rounded-control bg-app-surface-muted p-3">{t(strings.experience.studio.previewDepthSummary)}</div>
            <div className="rounded-control bg-app-surface-muted p-3">{t(strings.experience.studio.previewCoverageMeter)}</div>
            <div className="rounded-control bg-app-surface-muted p-3">{t(strings.experience.studio.previewDebtTable)}</div>
          </div>
          <ul
            data-testid={selectors.experience.studio.variantRail}
            aria-label={t(strings.experience.studio.variantsLabel)}
            className="mt-4 grid gap-3 md:grid-cols-2"
          >
            {[t(strings.experience.studio.variantCompactTable), t(strings.experience.studio.variantEvidenceForward)].map((variant) => (
              <li key={variant} className="rounded-control border border-app-border p-3 text-sm">
                <GitCompare className="mb-2 size-4 text-app-primary" aria-hidden="true" />
                {variant}
              </li>
            ))}
          </ul>
          <Button data-testid={selectors.experience.studio.promoteAction} type="button" className="mt-4">
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
        {findings.map((finding) => (
          <li key={finding.title} className="rounded-panel border border-app-border bg-app-surface p-4">
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
                <p className="mt-2 font-medium">{finding.title}</p>
                <Link
                  data-testid={selectors.experience.findings.evidenceLink}
                  to={finding.evidence}
                  className="mt-3 inline-flex text-sm text-app-primary underline-offset-4 hover:underline"
                >
                  {t(strings.experience.common.viewEvidence)}
                </Link>
              </div>
              <div className="flex gap-2">
                <Button data-testid={selectors.experience.findings.previewAction} type="button" variant="outline">
                  <FileSearch className="mr-2 size-4" aria-hidden="true" />
                  {t(strings.experience.findings.preview)}
                </Button>
                <Button data-testid={selectors.experience.findings.applyAction} type="button">
                  <Gauge className="mr-2 size-4" aria-hidden="true" />
                  {t(strings.experience.findings.apply)}
                </Button>
              </div>
            </div>
          </li>
        ))}
      </ul>
    </PageFrame>
  );
}
