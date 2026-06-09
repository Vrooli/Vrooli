import { useQuery } from "@tanstack/react-query";

import { fetchHealth } from "../../api/health";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useLiveSearchHealth } from "../../lib/liveSearchHealth";
import { useTranslation } from "../../i18n";
import type { DependencyStatus } from "@vrooli/proto-types/web-search/v1/health/health_pb";

/**
 * OpsPanel is the operations readout. It surfaces two honest signals:
 *
 *   1. Dependency status from GET /health — SearXNG reachability, the findings
 *      store, etc. (whatever the API reports under `dependencies`).
 *   2. The most-recent live-search response metadata (cached / degraded /
 *      degraded_reason / result count) as a "last query health" readout.
 *
 * There is no dedicated cache-hit-rate or budget RPC, so we do NOT fabricate
 * aggregate numbers — we report exactly the per-dependency and last-response
 * signals we actually have.
 */
export function OpsPanel() {
  const { t } = useTranslation();
  const { data, isLoading, error } = useQuery({
    queryKey: ["health"],
    queryFn: fetchHealth,
  });
  const lastQuery = useLiveSearchHealth();

  const dependencies = data ? Object.entries(data.dependencies) : [];

  return (
    <div data-testid={selectors.ops.panel} className="flex flex-col gap-6">
      <section
        data-testid={selectors.ops.dependencies}
        aria-labelledby="ops-deps-heading"
        className="rounded-panel border border-app-border bg-app-surface p-4"
      >
        <h3 id="ops-deps-heading" className="text-sm font-semibold uppercase text-app-muted-foreground">
          {t(strings.ops.dependenciesHeading)}
        </h3>
        {isLoading && (
          <p data-testid={selectors.ops.dependenciesLoading} className="mt-2 text-sm text-app-muted-foreground">
            {t(strings.ops.dependenciesLoading)}
          </p>
        )}
        {error != null && (
          <p data-testid={selectors.ops.dependenciesError} className="mt-2 text-sm text-app-danger">
            {t(strings.ops.dependenciesError)}
          </p>
        )}
        {data && dependencies.length === 0 && (
          <p data-testid={selectors.ops.dependenciesEmpty} className="mt-2 text-sm text-app-muted-foreground">
            {t(strings.ops.noDependencies)}
          </p>
        )}
        {dependencies.length > 0 && (
          <ul className="mt-2 flex flex-col gap-2">
            {dependencies.map(([name, status]) => (
              <DependencyRow key={name} name={name} status={status} />
            ))}
          </ul>
        )}
      </section>

      <section
        data-testid={selectors.ops.lastQuery}
        aria-labelledby="ops-last-heading"
        className="rounded-panel border border-app-border bg-app-surface p-4"
      >
        <h3 id="ops-last-heading" className="text-sm font-semibold uppercase text-app-muted-foreground">
          {t(strings.ops.lastQueryHeading)}
        </h3>
        {!lastQuery ? (
          <p data-testid={selectors.ops.lastQueryEmpty} className="mt-2 text-sm text-app-muted-foreground">
            {t(strings.ops.lastQueryEmpty)}
          </p>
        ) : (
          <dl className="mt-2 grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 text-sm">
            <dt className="text-app-muted-foreground">{t(strings.ops.lastQueryLabel)}</dt>
            <dd className="truncate text-app-foreground">{lastQuery.query}</dd>

            <dt className="text-app-muted-foreground">{t(strings.ops.resultCountLabel)}</dt>
            <dd className="text-app-foreground">{lastQuery.resultCount}</dd>

            <dt className="text-app-muted-foreground">{t(strings.ops.cachedLabel)}</dt>
            <dd className="text-app-foreground">
              {lastQuery.cached ? t(strings.ops.cachedYes) : t(strings.ops.cachedNo)}
            </dd>

            <dt className="text-app-muted-foreground">{t(strings.ops.degradedLabel)}</dt>
            <dd className={lastQuery.degraded ? "text-app-warning" : "text-app-success"}>
              {lastQuery.degraded ? t(strings.ops.degradedYes) : t(strings.ops.degradedNo)}
            </dd>

            {lastQuery.degraded && lastQuery.degradedReason && (
              <>
                <dt className="text-app-muted-foreground">{t(strings.ops.degradedReasonLabel)}</dt>
                <dd className="text-app-foreground">{lastQuery.degradedReason}</dd>
              </>
            )}
          </dl>
        )}
      </section>
    </div>
  );
}

function DependencyRow({ name, status }: { name: string; status: DependencyStatus }) {
  const { t } = useTranslation();

  return (
    <li
      data-testid={selectors.ops.dependency}
      data-dependency={name}
      className="flex items-center justify-between gap-3 rounded-control bg-app-surface-muted px-3 py-2"
    >
      <span className="font-mono text-sm text-app-foreground">{name}</span>
      <span className="flex items-center gap-2 text-xs">
        {status.latencyMs > 0 && (
          <span className="text-app-muted-foreground">
            {t(strings.ops.dependencyLatency, { latency: status.latencyMs.toFixed(0) })}
          </span>
        )}
        <span
          className={
            status.connected
              ? "rounded-pill bg-app-success/15 px-2 py-0.5 text-app-success"
              : "rounded-pill bg-app-danger/15 px-2 py-0.5 text-app-danger"
          }
        >
          {status.connected
            ? t(strings.ops.dependencyConnected)
            : t(strings.ops.dependencyDisconnected)}
        </span>
      </span>
    </li>
  );
}
