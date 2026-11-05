import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  RouteSource,
  PublicExposure,
  type Route,
} from "@vrooli/proto-types/tunnel-manager/v1/routes/routes_pb";

import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { QueryState } from "../../components/ui/QueryState";
import { StatusBadge, type BadgeTone } from "../../components/ui/StatusBadge";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { routesClient } from "../../api/routes";
import {
  PUBLIC_EXPOSURE_OPTIONS,
  normalizeExposure,
  publicExposureLabel,
  publicExposureTone,
} from "./labels";

const ROUTES_QUERY_KEY = ["routes"] as const;

type SourceKey = (typeof strings.routes.source)[keyof typeof strings.routes.source];

// Static enum→key map so the unused-key scan sees literal references and the
// helper stays typed to the strings-subtree union (never `string`).
const SOURCE_LABEL: Record<RouteSource, SourceKey> = {
  [RouteSource.UNSPECIFIED]: strings.routes.source.unknown,
  [RouteSource.SCENARIO]: strings.routes.source.scenario,
  [RouteSource.EXTERNAL]: strings.routes.source.external,
};

function sourceLabel(source: RouteSource): SourceKey {
  return SOURCE_LABEL[source];
}

function sourceTone(source: RouteSource): BadgeTone {
  return source === RouteSource.EXTERNAL ? "info" : "neutral";
}

/** A route is external when its provenance is RouteSource.EXTERNAL. */
function isExternal(route: Route): boolean {
  return route.source === RouteSource.EXTERNAL;
}

/**
 * ExternalRoutesPanel manages routes that point at an arbitrary local service
 * target rather than a known scenario's UI port. Add (subdomain + target URL)
 * and delete; the source badge distinguishes external from scenario routes.
 */
export function ExternalRoutesPanel() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [subdomain, setSubdomain] = useState("");
  const [target, setTarget] = useState("");
  const [domain, setDomain] = useState("");
  const [publicExposure, setPublicExposure] = useState<PublicExposure>(PublicExposure.INHERIT);

  const routesQuery = useQuery({ queryKey: ROUTES_QUERY_KEY, queryFn: () => routesClient.listRoutes({}) });
  const invalidate = () => void queryClient.invalidateQueries({ queryKey: ROUTES_QUERY_KEY });

  const addMutation = useMutation({
    mutationFn: () =>
      routesClient.createRoute({
        subdomain: subdomain.trim(),
        serviceTarget: target.trim(),
        domain: domain.trim(),
        source: RouteSource.EXTERNAL,
        publicExposure,
      }),
    onSuccess: () => {
      setSubdomain("");
      setTarget("");
      setDomain("");
      setPublicExposure(PublicExposure.INHERIT);
      invalidate();
    },
  });

  // UpdateRoute is a full replace, so changing only the per-route /public
  // override re-sends the route's current identity + routing fields alongside
  // the new public_exposure value.
  const exposureMutation = useMutation({
    mutationFn: ({ route, value }: { route: Route; value: PublicExposure }) =>
      routesClient.updateRoute({
        id: route.id,
        subdomain: route.subdomain,
        scenario: route.scenario,
        domain: route.domain,
        localPort: route.localPort,
        tier: route.tier,
        healthPath: route.healthPath,
        enabled: route.enabled,
        source: route.source,
        serviceTarget: route.serviceTarget,
        publicExposure: value,
      }),
    onSuccess: invalidate,
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => routesClient.deleteRoute({ id }),
    onSuccess: invalidate,
  });

  const externalRoutes = (routesQuery.data?.routes ?? []).filter(isExternal);
  const canSubmit = subdomain.trim() !== "" && target.trim() !== "" && !addMutation.isPending;

  const handleAdd = (e: React.FormEvent) => {
    e.preventDefault();
    if (canSubmit) addMutation.mutate();
  };

  return (
    <section data-testid={selectors.routes.panel} className="flex flex-col gap-4">
      <div className="flex flex-col gap-1">
        <h2 className="text-lg font-semibold">{t(strings.routes.externalHeading)}</h2>
        <p className="text-sm text-app-muted-foreground">{t(strings.routes.externalDescription)}</p>
      </div>

      <form
        data-testid={selectors.routes.addForm}
        onSubmit={handleAdd}
        className="grid gap-3 rounded-panel border border-app-border bg-app-surface p-4 sm:grid-cols-[minmax(0,1fr)_minmax(0,1.5fr)_minmax(0,1fr)_minmax(0,1fr)_auto] sm:items-end"
      >
        <label className="flex flex-col gap-1 text-sm">
          <span className="font-medium">{t(strings.routes.subdomainLabel)}</span>
          <Input
            data-testid={selectors.routes.subdomainInput}
            value={subdomain}
            onChange={(e) => setSubdomain(e.target.value)}
            placeholder={t(strings.routes.subdomainPlaceholder)}
            aria-label={t(strings.routes.subdomainLabel)}
          />
        </label>
        <label className="flex flex-col gap-1 text-sm">
          <span className="font-medium">{t(strings.routes.targetLabel)}</span>
          <Input
            data-testid={selectors.routes.targetInput}
            value={target}
            onChange={(e) => setTarget(e.target.value)}
            placeholder={t(strings.routes.targetPlaceholder)}
            aria-label={t(strings.routes.targetLabel)}
          />
        </label>
        <label className="flex flex-col gap-1 text-sm">
          <span className="font-medium">{t(strings.routes.domainLabel)}</span>
          <Input
            data-testid={selectors.routes.domainInput}
            value={domain}
            onChange={(e) => setDomain(e.target.value)}
            placeholder={t(strings.routes.domainPlaceholder)}
            aria-label={t(strings.routes.domainLabel)}
          />
        </label>
        <label className="flex flex-col gap-1 text-sm">
          <span className="font-medium">{t(strings.routes.publicExposure.label)}</span>
          <select
            data-testid={selectors.routes.publicExposureSelect}
            value={publicExposure}
            onChange={(e) => setPublicExposure(Number(e.target.value))}
            aria-label={t(strings.routes.publicExposure.label)}
            className="h-11 rounded-control border border-app-border bg-app-surface px-3 text-sm text-app-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50"
          >
            {PUBLIC_EXPOSURE_OPTIONS.map((option) => (
              <option key={option} value={option}>
                {t(publicExposureLabel(option))}
              </option>
            ))}
          </select>
        </label>
        <Button type="submit" data-testid={selectors.routes.addButton} disabled={!canSubmit}>
          {t(strings.routes.addButton)}
        </Button>
      </form>
      <p className="text-xs text-app-muted-foreground">{t(strings.routes.publicExposure.help)}</p>
      {addMutation.error && (
        <p data-testid={selectors.routes.addError} role="alert" className="text-sm text-app-danger">
          {t(strings.routes.addError)}
        </p>
      )}
      {deleteMutation.error && (
        <p data-testid={selectors.routes.deleteError} role="alert" className="text-sm text-app-danger">
          {t(strings.routes.deleteError)}
        </p>
      )}
      {exposureMutation.error && (
        <p data-testid={selectors.routes.publicExposureError} role="alert" className="text-sm text-app-danger">
          {t(strings.routes.publicExposure.error)}
        </p>
      )}

      <QueryState
        isLoading={routesQuery.isLoading}
        error={routesQuery.error}
        isEmpty={externalRoutes.length === 0}
        loadingLabel={t(strings.routes.loading)}
        errorLabel={t(strings.routes.error)}
        emptyLabel={t(strings.routes.empty)}
      >
        <div className="overflow-x-auto rounded-panel border border-app-border">
          <table data-testid={selectors.routes.table} className="w-full text-left text-sm">
            <thead className="border-b border-app-border bg-app-surface-muted text-xs uppercase text-app-muted-foreground">
              <tr>
                <th className="px-3 py-2">{t(strings.routes.colSubdomain)}</th>
                <th className="px-3 py-2">{t(strings.routes.colTarget)}</th>
                <th className="px-3 py-2">{t(strings.routes.colSource)}</th>
                <th className="px-3 py-2">{t(strings.routes.colPublicExposure)}</th>
                <th className="px-3 py-2">{t(strings.routes.colUrl)}</th>
                <th className="px-3 py-2">{t(strings.routes.colActions)}</th>
              </tr>
            </thead>
            <tbody>
              {externalRoutes.map((route: Route) => (
                <tr
                  key={route.id}
                  data-testid={selectors.routes.row}
                  className="border-b border-app-border last:border-0"
                >
                  <td className="px-3 py-2 font-medium">{route.subdomain}</td>
                  <td className="px-3 py-2 text-app-muted-foreground">{route.serviceTarget || "—"}</td>
                  <td className="px-3 py-2">
                    <StatusBadge tone={sourceTone(route.source)} data-testid={selectors.routes.sourceBadge}>
                      {t(sourceLabel(route.source))}
                    </StatusBadge>
                  </td>
                  <td className="px-3 py-2">
                    <div className="flex items-center gap-2">
                      <StatusBadge
                        tone={publicExposureTone(normalizeExposure(route.publicExposure))}
                        data-testid={selectors.routes.publicExposureBadge}
                      >
                        {t(publicExposureLabel(normalizeExposure(route.publicExposure)))}
                      </StatusBadge>
                      <select
                        data-testid={selectors.routes.publicExposureRowSelect({ id: route.id })}
                        value={normalizeExposure(route.publicExposure)}
                        disabled={exposureMutation.isPending}
                        aria-label={t(strings.routes.publicExposure.rowLabel)}
                        onChange={(e) =>
                          exposureMutation.mutate({
                            route,
                            value: Number(e.target.value),
                          })
                        }
                        className="h-9 rounded-control border border-app-border bg-app-surface px-2 text-sm text-app-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50"
                      >
                        {PUBLIC_EXPOSURE_OPTIONS.map((option) => (
                          <option key={option} value={option}>
                            {t(publicExposureLabel(option))}
                          </option>
                        ))}
                      </select>
                    </div>
                  </td>
                  <td className="px-3 py-2">
                    <a
                      data-testid={selectors.routes.url}
                      href={route.publicUrl}
                      target="_blank"
                      rel="noreferrer"
                      className="text-app-primary underline-offset-2 hover:underline"
                    >
                      {route.publicUrl}
                    </a>
                  </td>
                  <td className="px-3 py-2">
                    <Button
                      variant="outline"
                      data-testid={selectors.routes.deleteButton}
                      disabled={deleteMutation.isPending}
                      onClick={() => deleteMutation.mutate(route.id)}
                    >
                      {t(strings.routes.deleteButton)}
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </QueryState>
    </section>
  );
}
