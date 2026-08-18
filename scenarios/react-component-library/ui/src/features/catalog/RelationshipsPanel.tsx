import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";

import { getAssetPortContract, getAssetRelationships } from "../../api/catalogGraph";
import { EmptyState } from "../../components/EmptyState";
import { assetPath } from "../../routes";

function AssetLink({ assetId, name, rung }: { assetId: string; name: string; rung: number }) {
  return (
    <Link
      to={assetPath(assetId, { tab: "relationships" })}
      className="flex min-w-0 items-center gap-space-2xs rounded-control border border-app-border px-space-2xs py-space-2xs text-xs hover:bg-app-surface-muted focus-visible:outline focus-visible:outline-2 focus-visible:outline-app-primary"
    >
      <span
        aria-label={`Rung ${rung}`}
        className="shrink-0 rounded-pill bg-app-surface-muted px-space-2xs py-space-3xs font-mono"
      >
        R{rung}
      </span>
      <span className="min-w-0 flex-1 truncate">{name || assetId}</span>
      <span className="shrink-0 font-mono text-app-muted-foreground">{assetId}</span>
    </Link>
  );
}

export function RelationshipsPanel({ assetId }: { assetId: string }) {
  const query = useQuery({
    queryKey: ["catalog", "relationships", assetId],
    queryFn: () => getAssetRelationships(assetId),
    enabled: Boolean(assetId),
    retry: false,
  });
  const ports = useQuery({
    queryKey: ["catalog", "ports", assetId],
    queryFn: () => getAssetPortContract(assetId),
    enabled: Boolean(assetId) && Boolean(query.data),
    retry: false,
  });

  if (query.isLoading) {
    return (
      <section data-testid="relationships-panel" data-state="loading" className="space-y-space-xs">
        <p className="text-sm text-app-muted-foreground">Loading asset relationships…</p>
      </section>
    );
  }
  if (query.error) {
    return (
      <section data-testid="relationships-panel" data-state="error" className="space-y-space-xs">
        <p role="alert" className="text-sm text-app-danger">
          Unable to load asset relationships. The catalog graph may be unavailable.
        </p>
      </section>
    );
  }
  const relationships = query.data;
  if (!relationships) {
    return (
      <section data-testid="relationships-panel" data-state="empty" className="space-y-space-xs">
        <EmptyState
          title="No relationship data"
          description="This asset does not have a graph projection yet."
        />
      </section>
    );
  }
  return (
    <section
      data-testid="relationships-panel"
      data-state="success"
      className="space-y-space-sm text-app-foreground"
    >
      <div className="grid gap-space-xs lg:grid-cols-2">
        <section className="space-y-space-2xs">
          <div className="flex items-baseline justify-between gap-space-xs">
            <h3 className="font-medium">Depends on</h3>
            <span className="text-xs text-app-muted-foreground">
              {relationships.directDependencies.length} direct · {relationships.closure.length}{" "}
              closure
            </span>
          </div>
          {relationships.directDependencies.length === 0 ? (
            <p className="text-xs text-app-muted-foreground">No declared dependencies.</p>
          ) : (
            <div className="space-y-space-3xs">
              {relationships.directDependencies.map((asset) => (
                <AssetLink
                  key={asset.assetId}
                  assetId={asset.assetId}
                  name={asset.name}
                  rung={asset.rung}
                />
              ))}
            </div>
          )}
        </section>
        <section className="space-y-space-2xs">
          <div className="flex items-baseline justify-between gap-space-xs">
            <h3 className="font-medium">Used by</h3>
            <span className="text-xs text-app-muted-foreground">
              {relationships.directDependents.length} direct ·{" "}
              {relationships.transitiveDependentCount || relationships.transitiveDependents.length}{" "}
              transitive
            </span>
          </div>
          {relationships.directDependents.length === 0 ? (
            <p className="text-xs text-app-muted-foreground">
              No catalog assets depend on this asset.
            </p>
          ) : (
            <div className="space-y-space-3xs">
              {relationships.directDependents.map((asset) => (
                <AssetLink
                  key={asset.assetId}
                  assetId={asset.assetId}
                  name={asset.name}
                  rung={asset.rung}
                />
              ))}
            </div>
          )}
        </section>
      </div>
      <section className="space-y-space-2xs">
        <div className="flex items-baseline justify-between gap-space-xs">
          <h3 className="font-medium">Closure by rung</h3>
          <span className="text-xs text-app-muted-foreground">
            {relationships.closure.length} total
          </span>
        </div>
        <div className="space-y-space-xs">
          {relationships.closureBands.map((band) => (
            <section key={band.rung} className="space-y-space-2xs">
              <h4 className="text-xs font-semibold uppercase tracking-wide">
                Rung {band.rung} · {band.rungName}{" "}
                <span className="font-normal text-app-muted-foreground">({band.count})</span>
              </h4>
              <div className="space-y-space-3xs">
                {band.assets.map((asset) => (
                  <AssetLink
                    key={asset.assetId}
                    assetId={asset.assetId}
                    name={asset.name}
                    rung={asset.rung}
                  />
                ))}
              </div>
            </section>
          ))}
        </div>
      </section>
      <section className="rounded-panel border border-app-border bg-app-surface-muted p-space-xs">
        <h3 className="font-medium">Blast radius</h3>
        <p className="mt-space-3xs text-sm">
          {relationships.transitiveDependentCount || relationships.transitiveDependents.length}{" "}
          transitive dependents
        </p>
      </section>
      <section className="space-y-space-2xs" aria-labelledby="host-contract-heading">
        <div className="flex items-baseline justify-between gap-space-xs">
          <h3 id="host-contract-heading" className="font-medium">
            Host contract
          </h3>
          {ports.data && (
            <span className="text-xs text-app-muted-foreground">
              {ports.data.closureCount} closure assets
            </span>
          )}
        </div>
        {ports.isLoading && (
          <p className="text-xs text-app-muted-foreground">Loading host obligations…</p>
        )}
        {ports.error && (
          <p role="alert" className="text-xs text-app-danger">
            Host obligations are unavailable.
          </p>
        )}
        {ports.data &&
          (ports.data.selfContained ? (
            <p className="text-xs text-app-muted-foreground">
              This closure satisfies every declared host port.
            </p>
          ) : (
            <div className="space-y-space-2xs">
              {ports.data.unmetPorts.map((port) => (
                <div
                  key={port.capabilityId}
                  className="rounded-control border border-app-border px-space-2xs py-space-2xs text-xs"
                >
                  <div className="font-medium">{port.capabilityId}</div>
                  <div className="text-app-muted-foreground">
                    Demanded by {port.demandingAssets.length} asset(s);{" "}
                    {port.candidateSatisfiers.length} candidate satisfier(s)
                  </div>
                </div>
              ))}
            </div>
          ))}
      </section>
    </section>
  );
}
