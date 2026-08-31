/** @vrooliComponentSource data-display.data-table */
import { useQuery } from "@tanstack/react-query";

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/Card";
import { DataTable, type DataTableColumn } from "../components/DataTable";
import { EmptyState } from "../components/EmptyState";
import { StatusBadge } from "../components/StatusBadge";
import { getCatalogCoverage, listCatalogNextWork, type CoverageReport } from "../api/catalog";
import { CaptureGrid } from "../../../library/components/CaptureGrid/versions/1.0.0/CaptureGrid";
import { FindingList } from "../../../library/components/FindingList/versions/1.0.7/FindingList";
import { HealthIndicator } from "../../../library/components/HealthIndicator/versions/1.0.0/HealthIndicator";
import { MetricBreakdown } from "../../../library/components/MetricBreakdown/versions/1.0.0/MetricBreakdown";
import { NetworkGraph } from "../../../library/components/NetworkGraph/versions/1.0.5/NetworkGraph";
import { ProgressLadder } from "../../../library/components/ProgressLadder/versions/1.0.0/ProgressLadder";
import { ScoreGauge } from "@vrooli/react-component-library/ScoreGauge/1";

const maturityLabels: Record<string, string> = {
  missing: "Missing",
  scaffolded: "Scaffolded",
  verified: "Verified",
  production_ready: "Production ready",
};

const columns: Array<DataTableColumn<CoverageReport["rows"][number]>> = [
  {
    id: "asset",
    header: "Asset",
    accessor: (row) => <span className="font-medium">{row.name || row.assetId}</span>,
    searchValue: (row) => `${row.name} ${row.assetId}`,
    sortValue: (row) => row.name || row.assetId,
  },
  { id: "domain", header: "Domain", accessor: (row) => row.domain, sortValue: (row) => row.domain },
  { id: "target", header: "Target", accessor: (row) => row.target, sortValue: (row) => row.target },
  {
    id: "achieved",
    header: "Achieved",
    accessor: (row) => (
      <StatusBadge
        tone={
          row.achieved === "production-ready"
            ? "success"
            : row.achieved === "missing"
              ? "danger"
              : "warning"
        }
      >
        {maturityLabels[row.achieved] ?? row.achieved}
      </StatusBadge>
    ),
    sortValue: (row) => row.achieved,
  },
  {
    id: "blocks",
    header: "Downstream",
    accessor: (row) => row.blocksDownstream || "—",
    sortValue: (row) => row.blocksDownstream,
  },
];

function MaturityCard({ label, value, total }: { label: string; value: number; total: number }) {
  return (
    <Card>
      <CardContent className="grid gap-space-3xs">
        <span className="text-label text-app-muted-foreground">{label}</span>
        <strong className="text-title">
          {value}
          <span className="text-body text-app-muted-foreground"> / {total}</span>
        </strong>
      </CardContent>
    </Card>
  );
}

export function CoveragePage() {
  const coverage = useQuery({
    queryKey: ["catalog", "coverage"],
    queryFn: getCatalogCoverage,
    staleTime: 30_000,
    retry: false,
  });
  const nextWork = useQuery({
    queryKey: ["catalog", "next-work"],
    queryFn: () => listCatalogNextWork(8),
    staleTime: 30_000,
    retry: false,
  });
  const report = coverage.data;
  const maturity = report?.maturity;
  const rows = report?.rows ?? [];
  const nextRows = nextWork.data?.rows ?? [];
  const score = maturity?.weightedAssetScore ?? 0;
  const findings = rows.flatMap((row) =>
    (row.failedGates ?? []).map((gate) => ({
      id: `${row.assetId}-${gate}`,
      assetId: row.assetId,
      severity: "error",
      message: `${gate} is not passing for ${row.name || row.assetId}.`,
      remediation: `Promote ${row.name || row.assetId} through the ${gate} gate.`,
    })),
  );

  if (coverage.isLoading)
    return (
      <div
        data-testid="coverage-page"
        role="status"
        className="text-body text-app-muted-foreground"
      >
        Loading catalog coverage…
      </div>
    );
  if (coverage.isError || !report || !maturity)
    return (
      <EmptyState
        title="Coverage unavailable"
        description="The catalog coverage service did not return a report. Try again when the API is healthy."
      />
    );

  return (
    <div data-testid="coverage-page" className="grid gap-space-lg">
      <header className="grid gap-space-3xs">
        <p className="text-label uppercase text-app-muted-foreground">Operator view</p>
        <h1 className="text-title">Catalog coverage</h1>
        <p className="text-body text-app-muted-foreground">
          See maturity distribution, evidence-backed targets, and the next highest-leverage work
          without leaving the workspace.
        </p>
      </header>
      <section
        aria-labelledby="coverage-summary"
        className="grid gap-space-sm sm:grid-cols-2 xl:grid-cols-4"
      >
        <h2 id="coverage-summary" className="sr-only">
          Coverage summary
        </h2>
        <MaturityCard
          label="At or above target"
          value={maturity.atOrAboveTarget}
          total={maturity.total}
        />
        {Object.entries(maturity.byRung).map(([rung, value]) => (
          <MaturityCard
            key={rung}
            label={maturityLabels[rung] ?? rung}
            value={value}
            total={maturity.total}
          />
        ))}
      </section>
      <section
        aria-labelledby="coverage-score"
        className="grid gap-space-sm lg:grid-cols-[minmax(14rem,1fr)_2fr]"
      >
        <h2 id="coverage-score" className="sr-only">
          Weighted score
        </h2>
        <ScoreGauge value={score} label="Weighted catalog score" />
        <MetricBreakdown
          items={[
            {
              id: "assets",
              label: "Assets at target",
              value: maturity.atOrAboveTarget,
              total: maturity.total,
            },
            {
              id: "gates",
              label: "Mandatory gate evidence",
              value: maturity.mandatoryGateCoverage?.numerator ?? 0,
              total: maturity.mandatoryGateCoverage?.denominator ?? 0,
            },
            {
              id: "ready",
              label: "Production ready",
              value: maturity.productionReadyCoverage?.numerator ?? 0,
              total: maturity.productionReadyCoverage?.denominator ?? 0,
            },
          ]}
        />
      </section>
      <section aria-labelledby="coverage-health" className="grid gap-space-sm lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle id="coverage-health">Asset health</CardTitle>
            <CardDescription>
              Server-computed health labels stay readable without chart-only status.
            </CardDescription>
          </CardHeader>
          <CardContent className="grid gap-space-xs">
            {rows.slice(0, 8).map((row) => (
              <div
                key={row.assetId}
                aria-label={`${row.name || row.assetId} health`}
                className="flex items-center justify-between gap-space-xs"
              >
                <HealthIndicator
                  state={
                    row.assetScore >= 90 ? "healthy" : row.assetScore >= 50 ? "degraded" : "blocked"
                  }
                  score={row.assetScore}
                />
              </div>
            ))}
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>Progress to target</CardTitle>
          </CardHeader>
          <CardContent>
            <ProgressLadder
              rungs={["scaffolded", "implemented", "verified", "production-ready"].map((label) => ({
                id: label,
                label:
                  label === "production-ready"
                    ? "Production ready"
                    : `${label.charAt(0).toUpperCase()}${label.slice(1)}`,
                complete: (maturity.byRung[label] ?? 0) > 0,
                current: label === "verified",
              }))}
            />
          </CardContent>
        </Card>
      </section>
      <section
        aria-label="Catalog relationships and capture evidence"
        className="grid gap-space-sm lg:grid-cols-2"
      >
        <Card>
          <CardHeader>
            <CardTitle>Dependency health</CardTitle>
          </CardHeader>
          <CardContent>
            <NetworkGraph
              nodes={rows
                .filter((row) => row.assetId)
                .slice(0, 40)
                .map((row) => ({
                  id: row.assetId,
                  label: row.name || row.assetId,
                  health: row.achieved,
                }))}
              edges={[]}
            />
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>Evidence capture matrix</CardTitle>
          </CardHeader>
          <CardContent>
            <CaptureGrid
              cells={rows.slice(0, 8).flatMap((row) =>
                ["light", "dark"].map((theme) => ({
                  id: `${row.assetId}-${theme}`,
                  viewport: "declared",
                  theme: theme as "light" | "dark",
                  status: row.visualEvidence ? ("pass" as const) : ("missing" as const),
                })),
              )}
            />
          </CardContent>
        </Card>
      </section>
      <Card>
        <CardHeader>
          <CardTitle>Findings</CardTitle>
        </CardHeader>
        <CardContent>
          <FindingList findings={findings} />
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>Ranked next work</CardTitle>
          <CardDescription>
            Rows are ordered by maturity gap and downstream leverage.
          </CardDescription>
        </CardHeader>
        <CardContent>
          {nextWork.isLoading ? (
            <p className="text-body text-app-muted-foreground">Calculating next work…</p>
          ) : nextRows.length ? (
            <ol className="grid gap-space-xs">
              {nextRows.map((row) => (
                <li
                  key={row.assetId}
                  className="flex flex-wrap items-center justify-between gap-space-xs border-b border-app-border pb-space-xs last:border-0"
                >
                  <span>
                    <span className="font-medium">{row.name || row.assetId}</span>
                    <span className="ms-space-xs text-body text-app-muted-foreground">
                      {row.achieved} → {row.target}
                    </span>
                  </span>
                  <StatusBadge tone="info">blocks {row.blocksDownstream}</StatusBadge>
                </li>
              ))}
            </ol>
          ) : (
            <p className="text-body text-app-muted-foreground">
              No below-target work is currently ranked.
            </p>
          )}
        </CardContent>
      </Card>
      {rows.length ? (
        <DataTable
          rows={rows}
          columns={columns}
          getRowKey={(row) => row.assetId}
          caption="Catalog maturity rows"
          searchLabel="Search coverage"
          searchPlaceholder="Search assets, domains, or maturity"
          emptyMessage="No coverage rows match this search."
          tableTestId="coverage-table"
        />
      ) : (
        <EmptyState
          title="No catalog rows"
          description="The catalog did not return any planned assets."
        />
      )}
    </div>
  );
}
