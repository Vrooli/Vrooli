import { useQuery } from "@tanstack/react-query";

import { getCatalogStructure } from "../../api/catalogGraph";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";

export function StructurePanel() {
  const query = useQuery({
    queryKey: ["catalog", "structure"],
    queryFn: getCatalogStructure,
    retry: false,
  });
  if (query.isLoading)
    return (
      <section
        data-testid="dashboard-structure"
        data-state="loading"
        className="rounded-panel border border-app-border bg-app-surface p-space-sm"
      >
        <p className="text-sm text-app-muted-foreground">Loading catalog structure…</p>
      </section>
    );
  if (query.error)
    return (
      <section
        data-testid="dashboard-structure"
        data-state="error"
        className="rounded-panel border border-app-border bg-app-surface p-space-sm"
      >
        <p role="alert" className="text-sm text-app-danger">
          Catalog structure is unavailable.
        </p>
      </section>
    );
  if (!query.data)
    return (
      <section
        data-testid="dashboard-structure"
        data-state="empty"
        className="rounded-panel border border-app-border bg-app-surface p-space-sm"
      >
        <EmptyState
          title="No structure projection"
          description="The catalog has not returned a structural read model yet."
        />
      </section>
    );
  return (
    <section
      data-testid="dashboard-structure"
      data-state="success"
      className="rounded-panel border border-app-border bg-app-surface p-space-sm space-y-space-sm"
      aria-labelledby="dashboard-structure-heading"
    >
      <div>
        <h2 id="dashboard-structure-heading" className="font-semibold">
          Structure
        </h2>
        <p className="text-xs text-app-muted-foreground">
          Population and dependency pressure by rung.
        </p>
      </div>
      <div className="grid gap-space-2xs sm:grid-cols-2 lg:grid-cols-3">
        {query.data.population.map((row) => (
          <div
            key={row.rung}
            className="rounded-control border border-app-border px-space-2xs py-space-2xs"
          >
            <div className="text-xs text-app-muted-foreground">
              Rung {row.rung} · {row.rungName}
            </div>
            <div className="text-lg font-semibold">{row.count}</div>
          </div>
        ))}
      </div>
      <div className="space-y-space-2xs">
        <h3 className="text-sm font-medium">Invariants</h3>
        {query.data.invariants.map((invariant) => (
          <div key={invariant.id} className="flex items-start justify-between gap-space-xs text-xs">
            <span>{invariant.label}</span>
            <span className="shrink-0 font-mono text-app-muted-foreground">{invariant.status}</span>
          </div>
        ))}
      </div>
      <div className="space-y-space-2xs">
        <h3 className="text-sm font-medium">Largest blast radius</h3>
        {query.data.blastRadius.slice(0, 5).map((row) => (
          <div
            key={row.asset?.assetId}
            className="flex items-center justify-between gap-space-xs text-xs"
          >
            <span className="truncate">{row.asset?.assetId}</span>
            <span className="shrink-0 font-mono text-app-muted-foreground">
              {row.transitiveDependentCount}
            </span>
          </div>
        ))}
      </div>
    </section>
  );
}
