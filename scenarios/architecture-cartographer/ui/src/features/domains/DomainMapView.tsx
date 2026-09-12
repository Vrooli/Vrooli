import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { Button } from "../../components/ui/button";
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
          variant="default"
          size="sm"
          data-testid={selectors.features.domains.view.extractButton}
          onClick={() => extract.mutate()}
          disabled={extract.isPending}
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
