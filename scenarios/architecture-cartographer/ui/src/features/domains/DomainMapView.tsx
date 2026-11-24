import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { Button } from "@vrooli/react-component-library/Button/2.2.14";
import { EmptyState } from "../../components/EmptyState";
import { ErrorState } from "../../components/ErrorState";
import { LoadingState } from "../../components/LoadingState";
import { DataTable, type DataTableColumn } from "../../components/DataTable";
import type { DerivedDomain } from "@vrooli/proto-types/architecture-cartographer/v1/domains/domains_pb";
import {
  DomainSource,
  Archetype,
} from "@vrooli/proto-types/architecture-cartographer/v1/domains/domains_pb";
import type { DomainArchetype } from "@vrooli/proto-types/architecture-cartographer/v1/domains/domains_pb";
import {
  useGetDomainMap,
  useExtractDomains,
} from "./controllers/useDomainsController";
import { ConvergenceReport } from "./ConvergenceReport";
import { BoundaryHealth } from "./BoundaryHealth";
import { useSearchDomainSlice } from "../graph/controllers/useGraphController";
import { useState } from "react";
import { DocCodeDriftPanel } from "./DocCodeDriftPanel";

export interface DomainMapViewProps {
  scenario: string;
}

function domainSourceLabel(source: DomainSource): string {
  switch (source) {
    case DomainSource.API_MANIFEST:
      return "api manifest";
    case DomainSource.DOMAINS_DOC:
      return "DOMAINS.md";
    case DomainSource.API_FOLDERS:
      return "api/internal folders";
    case DomainSource.CLI_GROUPS:
      return "cli groups";
    case DomainSource.UI_FEATURES:
      return "ui features";
    default:
      return "unspecified";
  }
}

function sliceEvidenceAnchor(domain: string, rung: string, index: number): string {
  const safe = (value: string) => value.replace(/[^a-zA-Z0-9_-]/g, "-");
  return `slice-evidence-${safe(domain)}-${safe(rung)}-${index}`;
}

const ARCHETYPE_LABELS: Record<Archetype, string> = {
  [Archetype.UNSPECIFIED]: "",
  [Archetype.REPORTING]: "reporting",
  [Archetype.SERVICE]: "service",
  [Archetype.MUTATION]: "mutation",
  [Archetype.CLASSIFICATION]: "classification",
  [Archetype.ORCHESTRATION]: "orchestration",
  [Archetype.SCORING]: "scoring",
  [Archetype.QUERY]: "query",
};

function singleArchetypeLabel(archetype: DomainArchetype): string {
  return ARCHETYPE_LABELS[archetype.archetype] || archetype.declaredLabel || "";
}

function archetypeLabel(domain: DerivedDomain): string {
  const labels = (domain.archetypes ?? []).map(singleArchetypeLabel).filter(Boolean);
  return labels.length === 0 ? "—" : labels.join(", ");
}

export function DomainMapView({ scenario }: DomainMapViewProps) {
  const { t } = useTranslation();
  const domainMap = useGetDomainMap(scenario);
  const extract = useExtractDomains(scenario);

  if (domainMap.isPending) {
    return (
      <div data-testid={selectors.features.domains.view.loading}>
        <LoadingState label={t(strings.pages.targetDomains.loading)} />
      </div>
    );
  }
  if (domainMap.isError) {
    return (
      <div data-testid={selectors.features.domains.view.error}>
        <ErrorState
          title={t(strings.pages.targetDomains.errorTitle)}
          message={domainMap.error instanceof Error ? domainMap.error.message : String(domainMap.error)}
          retryLabel={t(strings.shared.error.retry)}
          onRetry={() => {
            void domainMap.refetch();
          }}
        />
      </div>
    );
  }

  const map = domainMap.data.domainMap;

  return (
    <div data-testid={selectors.features.domains.view.root} className="flex flex-col gap-4">
      <div className="flex flex-wrap gap-2">
        <Button
          type="button"
          variant="primary"
          size="sm"
          data-testid={selectors.features.domains.view.extractButton}
          onClick={() => extract.mutate()}
          disabled={extract.isPending}
          shape="square"
        >
          {extract.isPending
            ? t(strings.pages.targetDomains.extracting)
            : t(strings.pages.targetDomains.extractButton)}
        </Button>
      </div>

      {!map ? (
        <div data-testid={selectors.features.domains.view.empty}>
          <EmptyState title={t(strings.pages.targetDomains.empty)} />
        </div>
      ) : (
        <>
          <section aria-labelledby="domains-heading" className="flex flex-col gap-2">
            <h4 id="domains-heading" className="text-lg font-semibold">
              {t(strings.pages.targetDomains.domainsHeading)}
            </h4>
            <DomainMapTable domains={map.domains} scenario={scenario} />
          </section>

          {map.domains.length > 0 && <DomainSliceAnatomy scenario={scenario} domains={map.domains} />}

          <DocCodeDriftPanel scenario={scenario} />

          <section
            aria-labelledby="domains-shared-substrate-heading"
            data-testid={selectors.features.domains.sharedSubstrate.root}
            className="flex flex-col gap-2"
          >
            <h4 id="domains-shared-substrate-heading" className="text-lg font-semibold">
              {t(strings.pages.targetDomains.sharedSubstrateHeading)}
            </h4>
            {map.sharedSubstrate.length === 0 ? (
              <p className="text-sm text-app-muted-foreground">{t(strings.pages.targetDomains.noSharedSubstrate)}</p>
            ) : (
              <ul className="flex flex-wrap gap-1">
                {map.sharedSubstrate.map((path) => (
                  <li key={path}>
                    <span className="font-mono text-xs bg-app-muted px-1 py-0.5 rounded">{path}</span>
                  </li>
                ))}
              </ul>
            )}
          </section>

          <section aria-labelledby="domains-non-domains-heading" className="flex flex-col gap-2">
            <h4 id="domains-non-domains-heading" className="text-lg font-semibold">
              {t(strings.pages.targetDomains.nonDomainsHeading)}
            </h4>
            {map.nonDomains.length === 0 ? (
              <p className="text-sm text-app-muted-foreground">{t(strings.pages.targetDomains.noNonDomains)}</p>
            ) : (
              <ul className="flex flex-wrap gap-1">
                {map.nonDomains.map((name) => (
                  <li key={name}>
                    <span className="font-mono text-xs bg-app-muted px-1 py-0.5 rounded">{name}</span>
                  </li>
                ))}
              </ul>
            )}
          </section>

          <section aria-labelledby="domains-declarations-heading" className="flex flex-col gap-2">
            <h4 id="domains-declarations-heading" className="text-lg font-semibold">
              {t(strings.pages.targetDomains.declarationsHeading)}
            </h4>
            <ul className="flex flex-col gap-1">
              {map.declarations.map((decl) => (
                <li key={decl.source} className="text-sm">
                  <span className="font-medium">{domainSourceLabel(decl.source)}</span>
                  {decl.authoritative && (
                    <span className="ml-1 text-xs text-app-muted-foreground">{t(strings.pages.targetDomains.authorityLabel)}</span>
                  )}
                  {decl.domainNames.length > 0 && (
                    <span className="ml-1 font-mono text-xs text-app-muted-foreground">
                      {decl.domainNames.join(", ")}
                    </span>
                  )}
                </li>
              ))}
            </ul>
          </section>

          <ConvergenceReport scenario={scenario} />

          <BoundaryHealth scenario={scenario} />
        </>
      )}
    </div>
  );
}

function DomainSliceAnatomy({ scenario, domains }: { scenario: string; domains: readonly DerivedDomain[] }) {
  const [domainName, setDomainName] = useState(domains[0]?.name ?? "");
  const search = useSearchDomainSlice(scenario, domainName);
  const selectedDomain = domains.find((item) => item.name === domainName);
  const hit = search.data?.results.find((item) => item.domain === domainName);
  const expectedRungs = ["proto", "handler", "internal", "cli", "ui"];
  const returnedRungs = hit?.slice.rungs ?? [];
  const rungs = expectedRungs.map((name) => returnedRungs.find((item) => item.name === name));
  const attestation = hit?.slice.attestation;
  const basis = attestation?.basis === 1 ? "DERIVED" : attestation?.basis === 2 ? "VALIDATED" : attestation?.basis === 3 ? "DECLARED_UNVERIFIED" : attestation?.basis === 4 ? "CONTRADICTED" : "UNKNOWN";
  const sufficiency = attestation?.sufficiency === 1 ? "FULL" : attestation?.sufficiency === 2 ? "PARTIAL" : attestation?.sufficiency === 3 ? "INSUFFICIENT" : "UNKNOWN";
  const edges = hit?.slice.layer_edges ?? [];

  return (
    <section aria-labelledby="domain-slice-heading" className="flex flex-col gap-3" data-testid="domain-slice-anatomy">
      <h4 id="domain-slice-heading" className="text-lg font-semibold">Implementation slice anatomy</h4>
      <label className="flex items-center gap-2 text-sm">
        <span>Domain</span>
        <select aria-label="Slice domain" value={domainName} onChange={(event) => setDomainName(event.target.value)} className="rounded border border-app-border bg-app-surface px-2 py-1">
          {domains.map((item) => <option key={item.name} value={item.name}>{item.name}</option>)}
        </select>
      </label>
      <p className="text-xs text-app-muted-foreground">Read scope: scenario:{scenario}. This view searches an existing cached graph snapshot and does not initiate graph extraction. Snapshot age/freshness and current coverage are not established by this view.</p>
      {search.isPending ? <p role="status">Loading bounded slice…</p> : null}
      {search.isError ? <p role="alert">Slice search unavailable: {search.error instanceof Error ? search.error.message : String(search.error)}</p> : null}
      {search.data && !hit ? (
        <div role="status" className="rounded border border-app-border p-3">
          <p>No ranked slice was returned for {domainName}.</p>
          {search.data.degradations.map((item) => <p key={item.code} className="mt-1 text-sm">{item.code}: {item.message}</p>)}
          {!search.data.degradations.length && <p className="mt-1 text-sm">No exact domain candidate matched within this scenario-scoped result set.</p>}
          <p className="mt-1 text-xs text-app-muted-foreground">No extraction was attempted by this search view.</p>
        </div>
      ) : null}
      {hit ? (
        <div className="flex flex-col gap-3">
          <header className="rounded border border-app-border p-3">
            <h5 className="font-semibold">{hit.title}</h5>
            <p className="mt-1 text-sm">Scenario {hit.scenario} · domain {hit.domain} · cached snapshot {hit.snapshot_id} · rank score {hit.score.toFixed(3)}</p>
            <p className="mt-1 text-xs text-app-muted-foreground">Derived domain ownership: {selectedDomain?.responsibility || "responsibility not declared"}. Purpose: {selectedDomain?.purpose || "purpose not declared"}. Owns data: {selectedDomain?.ownsData || "no data-ownership claim declared"}.</p>
            <p className="mt-1 text-xs text-app-muted-foreground">Domain declaration provenance: {selectedDomain?.provenance?.length ? selectedDomain.provenance.map(domainSourceLabel).join(", ") : "unknown"}</p>
            <p className="mt-2 text-sm">Attestation: {basis} / {sufficiency}. {attestation?.claim || "Attestation claim unavailable."}</p>
            {attestation?.citations?.length ? <ul aria-label="Attestation citation links" className="mt-1 list-disc pl-5 text-xs">{attestation.citations.map((citation, index) => {
              const match = rungs.flatMap((rung) => rung?.evidence.map((item, itemIndex) => ({ rung: rung.name, item, itemIndex })) ?? []).find(({ item }) => item.value === citation.locator);
              return <li key={`${citation.locator}:${index}`}>{citation.kind}: {citation.locator}{citation.note ? ` — ${citation.note}` : ""}{match ? <> · <a className="underline" href={`#${sliceEvidenceAnchor(hit.domain, match.rung, match.itemIndex)}`}>{match.rung} rung evidence row</a></> : <span> · Not linked to a returned rung evidence row (bounded sample).</span>}</li>;
            })}</ul> : <p className="mt-1 text-xs">No attestation citations returned.</p>}
            {attestation?.gaps?.length ? <p className="mt-1 text-xs text-app-muted-foreground">Known gaps: {attestation.gaps.join("; ")}</p> : null}
          </header>
          <section aria-labelledby="slice-rungs-heading">
            <h5 id="slice-rungs-heading" className="font-medium">Five-rung source anatomy</h5>
            <ol className="mt-2 grid gap-2 md:grid-cols-2 xl:grid-cols-3">
              {rungs.map((rung, index) => <li key={expectedRungs[index]} className="rounded border border-app-border p-3">
                <h6 className="font-medium">{expectedRungs[index]} · {rung ? (rung.present ? "present" : "missing") : "not returned"}</h6>
                {rung?.files?.length ? <ul className="mt-1 list-disc pl-5 font-mono text-xs">{rung.files.slice(0, 4).map((file) => <li key={file.path}>{file.path} · {file.lines} lines{file.is_test ? " · test source" : ""}</li>)}</ul> : <p className="mt-1 text-xs text-app-muted-foreground">No file rows in this bounded slice.</p>}
                {rung?.symbols?.length ? <p className="mt-1 text-xs">Symbols: {rung.symbols.slice(0, 4).map((symbol) => `${symbol.name} (${symbol.kind}) — ${symbol.file}`).join("; ")}</p> : null}
                {rung?.evidence?.length ? <ul className="mt-1 space-y-1 text-xs text-app-muted-foreground">{rung.evidence.slice(0, 4).map((item, itemIndex) => <li id={sliceEvidenceAnchor(hit.domain, rung.name, itemIndex)} key={`${item.kind}:${item.value}:${itemIndex}`}>{item.kind} · {item.source}: {item.value}</li>)}</ul> : null}
              </li>)}
            </ol>
          </section>
          <section aria-labelledby="slice-transitions-heading" className="rounded border border-app-border p-3">
            <h5 id="slice-transitions-heading" className="font-medium">Layer handoffs</h5>
            <p className="text-xs text-app-muted-foreground">Observed edges are package-level imports from this graph snapshot, not symbol-level or runtime call flow. Missing edges mean no direct edge was reported in this slice.</p>
            <ol className="mt-2 flex flex-wrap items-center gap-2 text-xs">{expectedRungs.slice(0, -1).map((from, index) => {
              const to = expectedRungs[index + 1];
              const observed = edges.filter((edge) => edge.from_rung === from && edge.to_rung === to);
              return <li key={`${from}:${to}`} className="rounded bg-app-muted px-2 py-1">{from} → {to}: {observed.length ? observed.map((edge) => `${edge.kind} observed`).join(", ") : "no direct edge reported"}</li>;
            })}</ol>
            {edges.some((edge) => !expectedRungs.includes(edge.from_rung) || !expectedRungs.includes(edge.to_rung)) ? <p className="mt-2 text-xs">Other package transitions: {edges.filter((edge) => !expectedRungs.includes(edge.from_rung) || !expectedRungs.includes(edge.to_rung)).map((edge) => `${edge.from_rung}→${edge.to_rung} (${edge.kind})`).join(", ")}</p> : null}
            <p className="mt-2 text-xs text-app-muted-foreground">Surfaces: {hit.slice.surfaces.length ? hit.slice.surfaces.join(", ") : "none reported"} · returned evidence paths: {hit.paths.length ? hit.paths.join(", ") : "none"}</p>
          </section>
          {search.data?.degradations.map((item) => <p key={item.code} role="status" className="text-sm">{item.code}: {item.message}</p>)}
        </div>
      ) : null}
    </section>
  );
}

interface DomainMapTableProps {
  domains: readonly DerivedDomain[];
  scenario: string;
}

function DomainMapTable({ domains }: DomainMapTableProps) {
  const { t } = useTranslation();

  if (domains.length === 0) {
    return (
      <div data-testid={selectors.features.domains.table.empty}>
        <EmptyState title={t(strings.pages.targetDomains.noDomains)} />
      </div>
    );
  }

  const columns: ReadonlyArray<DataTableColumn<DerivedDomain>> = [
    {
      key: "name",
      header: t(strings.pages.targetDomains.columns.name),
      cell: (row) => <span className="font-semibold">{row.name}</span>,
    },
    {
      key: "paths",
      header: t(strings.pages.targetDomains.columns.paths),
      cell: (row) => (
        <span className="font-mono text-xs">
          {row.paths.length === 0 ? "—" : row.paths.join(", ")}
        </span>
      ),
    },
    {
      key: "archetype",
      header: t(strings.pages.targetDomains.columns.archetype),
      cell: (row) => (
        <span className="text-xs text-app-muted-foreground">
          {archetypeLabel(row)}
        </span>
      ),
    },
    {
      key: "provenance",
      header: t(strings.pages.targetDomains.columns.provenance),
      cell: (row) => (
        <span className="text-xs text-app-muted-foreground">
          {row.provenance.length === 0
            ? "—"
            : row.provenance.map((s) => domainSourceLabel(s)).join(", ")}
        </span>
      ),
    },
  ];

  return (
    <div data-testid={selectors.features.domains.table.root}>
      <DataTable
        rows={domains}
        getRowId={(d) => d.name}
        columns={columns}
        emptyMessage={t(strings.pages.targetDomains.noDomains)}
      />
    </div>
  );
}
