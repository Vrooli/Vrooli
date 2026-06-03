import { useQuery } from "@tanstack/react-query";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { fetchFederationStatus } from "../../api/search";

/**
 * FederationStatusBar surfaces live federation health above the results:
 * classifier/reranker model availability (which gate auto-routing and unified
 * rerank) plus a per-provider reachability chip carrying each leaf's freshness
 * note. It reads RoutingService.Status; a failed probe degrades to a quiet
 * inline message and never blocks searching.
 */
export function FederationStatusBar() {
  const { t } = useTranslation();
  const { data, isLoading, error } = useQuery({
    queryKey: ["federation-status"],
    queryFn: fetchFederationStatus,
  });

  if (isLoading) {
    return (
      <p className="text-xs text-app-muted-foreground" data-testid={selectors.search.statusBar}>
        {t(strings.search.statusHeading)}…
      </p>
    );
  }

  if (error || !data) {
    return (
      <p className="text-xs text-app-muted-foreground" data-testid={selectors.search.statusBar}>
        {t(strings.search.statusError)}
      </p>
    );
  }

  const reachable = data.providers.filter((p) => p.reachable).length;

  return (
    <div
      data-testid={selectors.search.statusBar}
      className="flex flex-wrap items-center gap-2 text-xs"
    >
      <span className="font-medium text-app-muted-foreground">{t(strings.search.statusHeading)}:</span>
      <ModelChip label={t(strings.search.classifier)} ok={data.classifierAvailable} okLabel={t(strings.search.available)} offLabel={t(strings.search.unavailable)} />
      <ModelChip label={t(strings.search.reranker)} ok={data.rerankerAvailable} okLabel={t(strings.search.available)} offLabel={t(strings.search.unavailable)} />
      <span className="text-app-muted-foreground">
        {t(strings.search.providersReachable, { reachable, total: data.providers.length })}
      </span>
      <div className="flex flex-wrap gap-1.5">
        {data.providers.map((p) => (
          <span
            key={p.providerId}
            data-testid={selectors.search.providerChip({ providerId: p.providerId })}
            title={p.freshness}
            className={
              "rounded-full border px-2 py-0.5 " +
              (p.reachable
                ? "border-app-border text-app-foreground"
                : "border-app-destructive/40 text-app-destructive")
            }
          >
            {p.reachable ? "✓" : "✗"} {p.providerId}
          </span>
        ))}
      </div>
    </div>
  );
}

function ModelChip({
  label,
  ok,
  okLabel,
  offLabel,
}: {
  label: string;
  ok: boolean;
  okLabel: string;
  offLabel: string;
}) {
  return (
    <span
      className={
        "rounded-full border px-2 py-0.5 " +
        (ok ? "border-app-border text-app-foreground" : "border-app-destructive/40 text-app-destructive")
      }
    >
      {label}: {ok ? okLabel : offLabel}
    </span>
  );
}
