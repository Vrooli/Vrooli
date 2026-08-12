import { useState } from "react";
import { Link, useLocation, useSearchParams } from "react-router-dom";

import { ExperienceSurface } from "../components/experience/ExperienceSurface";
import { EmptyState } from "../components/ui/empty-state";
import { DataTable, type DataTableColumn } from "../components/ui/data-table";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { StatusBadge } from "../components/ui/status-badge";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

type StyleRow = { id: string; strategy: string; lineage: string; threshold: number };
type Candidate = { id: string; altKey: "candidateOneAlt" | "candidateTwoAlt"; ratio: number; passes: boolean; tone: string };

const starterStyles: StyleRow[] = [
  { id: "horizon-ink", strategy: "procedural-treated", lineage: "wpa_poster", threshold: 4.5 },
  { id: "arcade-noir", strategy: "procedural", lineage: "metaphysical", threshold: 4.5 },
  { id: "terrain-riso", strategy: "procedural-treated", lineage: "riso_zine", threshold: 4.5 },
  { id: "field-guided", strategy: "guided", lineage: "technical_minimalism", threshold: 4.5 },
];

const candidates: Candidate[] = [
  { id: "horizon-ink / seed 401", altKey: "candidateOneAlt", ratio: 5.2, passes: true, tone: "linear-gradient(135deg, #18212b 0%, #9b6d48 48%, #e3c79a 100%)" },
  { id: "terrain-riso / seed 117", altKey: "candidateTwoAlt", ratio: 3.2, passes: false, tone: "linear-gradient(135deg, #26332d 0%, #a77656 48%, #d8c9a9 100%)" },
];

const resolvedPlan = JSON.stringify({
  strategy: "procedural-treated",
  operations: ["duotone", "halftone", "scrim"],
  placement: "hero_left_copy_safe",
  expected_execution_path: "procedural → image-tools treatments",
}, null, 2);

function CandidateCard({ candidate, imageAltText, altText, plan, copied, onCopy }: { candidate: Candidate; imageAltText: string; altText: string; plan: string; copied: boolean; onCopy: () => void }) {
  const { t } = useTranslation();
  return (
    <article className="flex flex-col gap-4 rounded-panel border border-app-border bg-app-surface p-4" aria-labelledby={`${candidate.id}-heading`}>
      <div className="flex items-start justify-between gap-3">
        <div>
          <h4 id={`${candidate.id}-heading`} className="font-semibold">{candidate.id}</h4>
          <p className="text-sm text-app-muted-foreground">{t(strings.pages.workbench.candidateDescription)}</p>
        </div>
        <StatusBadge>{candidate.passes ? t(strings.pages.workbench.passVerdict) : t(strings.pages.workbench.failVerdict)}</StatusBadge>
      </div>
      <div className="grid gap-3 md:grid-cols-2">
        <figure className="flex flex-col gap-2">
          <figcaption className="text-xs font-medium uppercase tracking-wide text-app-muted-foreground">{t(strings.pages.workbench.standaloneLabel)}</figcaption>
          <div role="img" aria-label={imageAltText} className="aspect-[16/9] rounded-control border border-app-border" style={{ background: candidate.tone }} />
        </figure>
        <figure className="flex flex-col gap-2">
          <figcaption className="text-xs font-medium uppercase tracking-wide text-app-muted-foreground">{t(strings.pages.workbench.contextLabel)}</figcaption>
          <div role="img" aria-label={imageAltText} className="relative aspect-[16/9] overflow-hidden rounded-control border border-app-border" style={{ background: candidate.tone }}>
            <div className="absolute inset-x-4 top-1/2 -translate-y-1/2 rounded-control border border-white/30 bg-black/35 px-3 py-2 text-center text-sm font-semibold text-white">{t(strings.pages.workbench.foregroundLabel)}</div>
          </div>
        </figure>
      </div>
      <p className="text-sm text-app-foreground">
        <span className="font-semibold">{candidate.ratio.toFixed(1)}:1</span>{" — "}
        {candidate.passes ? t(strings.pages.workbench.passVerdict) : `${t(strings.pages.workbench.failVerdict)} 4.5:1`}
      </p>
      <div className="rounded-control border border-app-border bg-app-surface-muted p-3">
        <div className="mb-2 flex items-center justify-between gap-3">
          <h5 className="text-sm font-semibold">{t(strings.pages.workbench.resolvedPlan)}</h5>
          <Button variant="secondary" size="sm" onClick={onCopy}>{copied ? t(strings.pages.workbench.planCopied) : t(strings.pages.workbench.copyPlan)}</Button>
        </div>
        <pre className="overflow-x-auto whitespace-pre-wrap text-xs text-app-muted-foreground">{plan}</pre>
      </div>
      <div className="grid gap-2">
        <label htmlFor={`${candidate.id}-alt`} className="text-sm font-medium">{t(strings.pages.workbench.altTextLabel)}</label>
        <Input id={`${candidate.id}-alt`} placeholder={t(strings.pages.workbench.altTextPlaceholder)} value={altText} readOnly />
        <Button disabled={!altText.trim()} aria-describedby={`${candidate.id}-release-help`}>{t(strings.pages.workbench.releaseAction)}</Button>
        <p id={`${candidate.id}-release-help`} className="text-xs text-app-muted-foreground">{t(strings.pages.workbench.releaseBlocked)}</p>
      </div>
    </article>
  );
}

export function WorkbenchPage() {
  const { t } = useTranslation();
  const { pathname } = useLocation();
  const [searchParams] = useSearchParams();
  const requestedState = searchParams.get("state");
  const state = requestedState === "loading" || requestedState === "error" || requestedState === "empty" ? requestedState : "partial";
  const [copied, setCopied] = useState(false);
  const [forkAxis, setForkAxis] = useState("lineage");
  const [forked, setForked] = useState(false);
  const columns: Array<DataTableColumn<StyleRow>> = [
    { id: "id", header: t(strings.pages.workbench.styleHeading), accessor: (row) => <span className="font-mono text-sm">{row.id}</span>, sortValue: (row) => row.id },
    { id: "strategy", header: t(strings.pages.workbench.strategyHeading), accessor: (row) => <StatusBadge>{row.strategy}</StatusBadge>, sortValue: (row) => row.strategy },
    { id: "lineage", header: t(strings.pages.workbench.lineageHeading), accessor: (row) => row.lineage, sortValue: (row) => row.lineage },
    { id: "threshold", header: t(strings.pages.workbench.contrastLabel), accessor: (row) => `${row.threshold}:1`, sortValue: (row) => row.threshold },
  ];

  return (
    <ExperienceSurface
      surfaceId="backdrops"
      state={state}
      statusMessage={state === "loading" ? t(strings.pages.workbench.loadingState) : state === "error" ? t(strings.pages.workbench.errorState) : t(strings.pages.workbench.state)}
      data-testid={selectors.pages.workbench}
      data-workbench-route={pathname}
      data-workbench-state={state}
      aria-labelledby="workbench-heading"
      className="flex flex-col gap-6"
    >
      {state === "loading" ? <section role="status" className="rounded-panel border border-app-border p-6">{t(strings.pages.workbench.loadingState)}</section> : null}
      {state === "error" ? <section role="alert" className="flex flex-col gap-3 rounded-panel border border-app-border p-6"><p>{t(strings.pages.workbench.errorState)}</p><Button variant="secondary">{t(strings.pages.workbench.errorRetry)}</Button></section> : null}
      {state === "empty" ? <EmptyState title={t(strings.pages.workbench.emptyTitle)} description={t(strings.pages.workbench.emptyState)} action={<Link to="/catalog" className="inline-flex min-h-11 items-center rounded-control border border-app-border px-4 text-sm font-medium">{t(strings.pages.workbench.reviewAction)}</Link>} /> : null}
      {state !== "loading" && state !== "error" && state !== "empty" ? <>
      <header className="flex flex-col gap-2">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <h2 id="workbench-heading" className="text-2xl font-semibold">{t(strings.pages.workbench.title)}</h2>
          <StatusBadge>{t(strings.pages.workbench.state)}</StatusBadge>
        </div>
        <p className="max-w-3xl text-app-muted-foreground">{t(strings.pages.workbench.description)}</p>
      </header>

      <section aria-labelledby="catalog-heading" className="flex flex-col gap-3">
        <h3 id="catalog-heading" className="text-lg font-semibold">{t(strings.pages.workbench.catalogHeading)}</h3>
        <DataTable
          rows={starterStyles}
          columns={columns}
          getRowKey={(row) => row.id}
          caption={t(strings.pages.workbench.catalogCaption)}
          emptyMessage={t(strings.pages.workbench.emptyTitle)}
          tableTestId="backdrop-style-catalog"
        />
      </section>

      <section aria-labelledby="candidate-heading" className="flex flex-col gap-3">
        <h3 id="candidate-heading" className="text-lg font-semibold">{t(strings.pages.workbench.candidateHeading)}</h3>
        <p className="max-w-3xl text-sm text-app-muted-foreground">{t(strings.pages.workbench.candidateDescription)}</p>
        <div className="grid gap-4 xl:grid-cols-2">
          {candidates.map((candidate) => <CandidateCard key={candidate.id} candidate={candidate} imageAltText={t(strings.pages.workbench[candidate.altKey])} altText="" plan={resolvedPlan} copied={copied} onCopy={() => { setCopied(true); if (navigator.clipboard) void navigator.clipboard.writeText(resolvedPlan); }} />)}
        </div>
      </section>

      <section aria-labelledby="contact-sheet-heading" className="flex flex-col gap-3">
        <h3 id="contact-sheet-heading" className="text-lg font-semibold">{t(strings.pages.workbench.contactSheetHeading)}</h3>
        <p className="max-w-3xl text-sm text-app-muted-foreground">{t(strings.pages.workbench.contactSheetDescription)}</p>
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4" data-testid="backdrop-contact-sheet">
          {starterStyles.map((style, index) => <figure key={style.id} className="overflow-hidden rounded-panel border border-app-border bg-app-surface">
            <div role="img" aria-label={`${style.id} contact sheet preview`} className="aspect-[4/3]" style={{ background: `linear-gradient(${110 + index * 18}deg, #172033, ${index % 2 ? "#bb6b58" : "#557b83"}, #e5cf9b)` }} />
            <figcaption className="p-3 text-xs font-medium"><span className="block text-app-muted-foreground">{style.lineage}</span>{style.strategy}</figcaption>
          </figure>)}
        </div>
      </section>

      <section aria-labelledby="placement-heading" className="flex flex-col gap-3">
        <h3 id="placement-heading" className="text-lg font-semibold">{t(strings.pages.workbench.placementHeading)}</h3>
        <p className="max-w-3xl text-sm text-app-muted-foreground">{t(strings.pages.workbench.placementDescription)}</p>
        <div className="overflow-x-auto rounded-panel border border-app-border">
          <table className="w-full min-w-[38rem] text-left text-sm" data-testid="backdrop-placement-matrix">
            <caption className="sr-only">{t(strings.pages.workbench.placementHeading)}</caption>
            <thead><tr className="border-b border-app-border text-app-muted-foreground"><th className="p-3">{t(strings.pages.workbench.styleHeading)}</th><th className="p-3">{t(strings.pages.workbench.fullBleedLabel)} / {t(strings.pages.workbench.desktopLabel)}</th><th className="p-3">{t(strings.pages.workbench.fullBleedLabel)} / {t(strings.pages.workbench.mobileLabel)}</th><th className="p-3">{t(strings.pages.workbench.splitPanelLabel)} / {t(strings.pages.workbench.mobileLabel)}</th></tr></thead>
            <tbody>{candidates.map((candidate) => <tr key={`${candidate.id}-placement`} className="border-b border-app-border last:border-0"><th className="p-3 font-medium">{candidate.id}</th><td className="p-3"><StatusBadge>{t(strings.pages.workbench.passVerdict)}</StatusBadge><span className="ml-2 text-xs">{candidate.ratio.toFixed(1)}:1</span></td><td className="p-3"><StatusBadge>{candidate.passes ? t(strings.pages.workbench.passVerdict) : t(strings.pages.workbench.failVerdict)}</StatusBadge></td><td className="p-3"><StatusBadge>{t(strings.pages.workbench.failVerdict)}</StatusBadge></td></tr>)}</tbody>
          </table>
        </div>
      </section>

      <section aria-labelledby="fork-heading" className="flex flex-col gap-3 rounded-panel border border-app-border bg-app-surface p-4">
        <h3 id="fork-heading" className="text-lg font-semibold">{t(strings.pages.workbench.forkHeading)}</h3>
        <p className="max-w-3xl text-sm text-app-muted-foreground">{t(strings.pages.workbench.forkDescription)}</p>
        <div className="flex flex-wrap items-end gap-3">
          <label className="flex flex-col gap-1 text-sm font-medium" htmlFor="fork-axis">{t(strings.pages.workbench.lineageHeading)}<select id="fork-axis" className="min-h-11 rounded-control border border-app-border bg-app-surface px-3" value={forkAxis} onChange={(event) => setForkAxis(event.target.value)}><option value="lineage">{t(strings.pages.workbench.axisLineage)}</option><option value="subject">{t(strings.pages.workbench.axisSubject)}</option><option value="treatment">{t(strings.pages.workbench.axisTreatment)}</option></select></label>
          <Button variant="secondary" onClick={() => setForked(true)}>{t(strings.pages.workbench.forkAction)}</Button>
          {forked ? <StatusBadge>{t(strings.pages.workbench.forkedLabel)} / {forkAxis}</StatusBadge> : null}
        </div>
        <div className="flex flex-wrap gap-2 text-xs text-app-muted-foreground"><span>{t(strings.pages.workbench.productChrome)}</span><span>{t(strings.pages.workbench.storeChrome)}</span><Button variant="secondary" size="sm">{t(strings.pages.workbench.packAction)}</Button></div>
      </section>

      <EmptyState
        title={t(strings.pages.workbench.emptyTitle)}
        description={t(strings.pages.workbench.emptyDescription)}
        action={<Link to="/compose" className="inline-flex min-h-11 items-center rounded-control bg-app-primary px-4 text-sm font-medium text-app-primary-foreground">{t(strings.pages.workbench.reviewAction)}</Link>}
      />
      </> : null}
    </ExperienceSurface>
  );
}
