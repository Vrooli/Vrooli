import { useQuery } from "@tanstack/react-query";
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
import { Link } from "react-router-dom";

import { fetchFleet } from "../api/experience";
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

const pageDepth = [
  { page: "Fleet", defaultState: "L2", emptyState: "L1", staleState: "L1", claims: 7 },
  { page: "Scenario Explorer", defaultState: "L2", emptyState: "L1", staleState: "-", claims: 7 },
  { page: "Evidence", defaultState: "L2", emptyState: "L1", staleState: "L1", claims: 7 },
  { page: "Studio", defaultState: "L2", emptyState: "L1", staleState: "-", claims: 10 },
  { page: "Findings", defaultState: "L2", emptyState: "L1", staleState: "-", claims: 6 },
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

const capturePlaceholder =
  "data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='960' height='540' viewBox='0 0 960 540'%3E%3Crect width='960' height='540' fill='%23f1f5f9'/%3E%3Crect x='80' y='72' width='800' height='80' rx='8' fill='%23cbd5e1'/%3E%3Crect x='80' y='188' width='360' height='260' rx='8' fill='%23ffffff' stroke='%23cbd5e1'/%3E%3Crect x='480' y='188' width='400' height='260' rx='8' fill='%23ffffff' stroke='%23cbd5e1'/%3E%3Ctext x='110' y='120' font-family='Inter,Arial' font-size='28' fill='%230f172a'%3EPage capture%3C/text%3E%3Ctext x='112' y='240' font-family='Inter,Arial' font-size='18' fill='%2364748b'%3EClaim region%3C/text%3E%3Ctext x='512' y='240' font-family='Inter,Arial' font-size='18' fill='%2364748b'%3EAX node%3C/text%3E%3C/svg%3E";

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
      debt: Number(scenario.debtScore),
      pages: Number(scenario.pageCount),
      status: scenario.status,
    })) ?? scenarios;
  const covered = data ? Number(data.withExperienceCount) : scenarios.length;
  const total = data ? Number(data.scenarioCount) : scenarios.length;
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
            {data ? `${data.totalPages} pages tracked` : "Loading fleet data"}
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
                <th className="px-4 py-3">Default</th>
                <th className="px-4 py-3">Empty</th>
                <th className="px-4 py-3">Stale</th>
                <th className="px-4 py-3">{t(strings.experience.common.claims)}</th>
              </tr>
            </thead>
            <tbody>
              {pageDepth.map((row) => (
                <tr key={row.page} className="border-t border-app-border">
                  <td className="px-4 py-3 font-medium">{row.page}</td>
                  <td className="px-4 py-3">{row.defaultState}</td>
                  <td className="px-4 py-3">{row.emptyState}</td>
                  <td className="px-4 py-3">{row.staleState}</td>
                  <td className="px-4 py-3">{row.claims}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <aside
          data-testid={selectors.experience.explorer.gapPanel}
          role="region"
          aria-label={t(strings.experience.explorer.gapsLabel)}
          className="rounded-panel border border-app-border bg-app-surface p-4"
        >
          <h3 className="font-semibold">{t(strings.experience.explorer.gapsLabel)}</h3>
          <p className="mt-2 text-sm text-app-muted-foreground">
            {t(strings.experience.explorer.gapsCopy)}
          </p>
          <Button data-testid={selectors.experience.explorer.studioAction} type="button" className="mt-4">
            <Wand2 className="mr-2 size-4" aria-hidden="true" />
            {t(strings.experience.explorer.openStudio)}
          </Button>
        </aside>
      </div>
      <ul
        data-testid={selectors.experience.explorer.claimList}
        aria-label={t(strings.experience.explorer.claimsLabel)}
        className="grid gap-3 md:grid-cols-2"
      >
        {["debt-table-perceivable", "summary-before-table", "drill-in-keyboard"].map((claim) => (
          <li key={claim} className="rounded-panel border border-app-border bg-app-surface p-4">
            <span
              data-testid={selectors.experience.explorer.tierLabel}
              role="note"
              className="text-xs font-semibold uppercase text-app-primary"
            >
              machine
            </span>
            <p className="mt-2 font-medium">{claim}</p>
            <Link
              data-testid={selectors.experience.explorer.evidenceLink}
              to="/scenarios/experience-manager/pages/fleet/evidence"
              className="mt-3 inline-flex text-sm text-app-primary underline-offset-4 hover:underline"
            >
              {t(strings.experience.common.viewEvidence)}
            </Link>
          </li>
        ))}
      </ul>
    </PageFrame>
  );
}

export function EvidencePage() {
  const { t } = useTranslation();

  return (
    <PageFrame
      testId={selectors.pages.evidence}
      title={t(strings.experience.evidence.title)}
      description={t(strings.experience.evidence.description)}
    >
      <div className="grid gap-4 xl:grid-cols-[1.2fr_0.8fr]">
        <div className="rounded-panel border border-app-border bg-app-surface p-4">
          <img
            data-testid={selectors.experience.evidence.captureImage}
            src={capturePlaceholder}
            alt={t(strings.experience.evidence.captureLabel)}
            className="min-h-72 w-full rounded-control border border-dashed border-app-border bg-app-surface-muted object-cover"
          />
          <Button data-testid={selectors.experience.evidence.recaptureAction} type="button" className="mt-4">
            <RefreshCw className="mr-2 size-4" aria-hidden="true" />
            {t(strings.experience.evidence.recapture)}
          </Button>
        </div>
        <div
          data-testid={selectors.experience.evidence.treePanel}
          role="region"
          aria-label={t(strings.experience.evidence.treeLabel)}
          className="rounded-panel border border-app-border bg-app-surface p-4"
        >
          <h3 className="font-semibold">{t(strings.experience.evidence.treeLabel)}</h3>
          <pre className="mt-3 overflow-auto rounded-control bg-app-surface-muted p-3 text-xs">
            role=table name="Experience debt" testid=fleet-debt-table
          </pre>
        </div>
      </div>
      <ul
        data-testid={selectors.experience.evidence.verdictList}
        aria-label={t(strings.experience.evidence.verdictsLabel)}
        className="grid gap-3 md:grid-cols-2"
      >
        {["debt-table-perceivable", "summary-before-table"].map((claim) => (
          <li key={claim} className="rounded-panel border border-app-border bg-app-surface p-4">
            <p className="font-medium">{claim}</p>
            <p className="text-sm text-app-muted-foreground">machine · 2026-07-05T16:31:16Z · passed</p>
            <Link
              data-testid={selectors.experience.evidence.evidenceLink}
              to="#tree"
              className="mt-3 inline-flex text-sm text-app-primary underline-offset-4 hover:underline"
            >
              {t(strings.experience.common.viewEvidence)}
            </Link>
          </li>
        ))}
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
          <Input id="studio-page-title" className="mt-2 bg-app-surface text-app-foreground" defaultValue="Fleet" />
          <label className="mt-4 block text-sm font-semibold" htmlFor="studio-claim">
            {t(strings.experience.common.claims)}
          </label>
          <Textarea
            id="studio-claim"
            className="mt-2 min-h-28 bg-app-surface text-app-foreground"
            defaultValue="The experience-debt table is exposed as a real table with its accessible name."
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
          role="region"
          aria-label={t(strings.experience.studio.wireframeLabel)}
          className="rounded-panel border border-app-border bg-app-surface p-4"
        >
          <h3 className="font-semibold">{t(strings.experience.studio.wireframeLabel)}</h3>
          <div className="mt-4 grid min-h-64 gap-3 rounded-control border border-dashed border-app-border p-4 md:grid-cols-3">
            <div className="rounded-control bg-app-surface-muted p-3">Depth summary</div>
            <div className="rounded-control bg-app-surface-muted p-3">Coverage meter</div>
            <div className="rounded-control bg-app-surface-muted p-3">Debt table</div>
          </div>
          <ul
            data-testid={selectors.experience.studio.variantRail}
            aria-label={t(strings.experience.studio.variantsLabel)}
            className="mt-4 grid gap-3 md:grid-cols-2"
          >
            {["Compact table", "Evidence-forward"].map((variant) => (
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
