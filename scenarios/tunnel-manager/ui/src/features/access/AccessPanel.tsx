import { useQuery } from "@tanstack/react-query";
import type { AccessHostState } from "@vrooli/proto-types/tunnel-manager/v1/config/config_pb";

import { Button } from "../../components/ui/button";
import { QueryState } from "../../components/ui/QueryState";
import { StatusBadge, type BadgeTone } from "../../components/ui/StatusBadge";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { configClient } from "../../api/config";

const ACCESS_QUERY_KEY = ["access-status"] as const;

type OverrideKey = (typeof strings.access.override)[keyof typeof strings.access.override];

// The host-state `override` is a plain string from the server ("inherit",
// "enabled", "disabled"). Map it to a typed strings key; unknown/empty values
// fall back to inherit so the column never renders a bare server token.
const OVERRIDE_LABEL: Record<string, OverrideKey> = {
  inherit: strings.access.override.inherit,
  enabled: strings.access.override.enabled,
  disabled: strings.access.override.disabled,
};

function overrideLabel(override: string): OverrideKey {
  return OVERRIDE_LABEL[override] ?? strings.access.override.inherit;
}

function boolTone(value: boolean): BadgeTone {
  return value ? "success" : "neutral";
}

/**
 * AccessPanel renders the /public Access-bypass read model alongside the drift
 * view: the global switch state, whether the Cloudflare Access client is
 * configured, the per-host effective decisions, and the dry-run plan (which
 * bypass apps a reconcile would create or remove). It is a pure read surface —
 * GetAccessStatus mutates nothing at the Cloudflare edge.
 */
export function AccessPanel() {
  const { t } = useTranslation();
  const accessQuery = useQuery({
    queryKey: ACCESS_QUERY_KEY,
    queryFn: () => configClient.getAccessStatus({}),
  });

  const status = accessQuery.data?.status;
  const hosts = status?.hosts ?? [];
  const toCreate = status?.toCreate ?? [];
  const toRemove = status?.toRemove ?? [];

  return (
    <section data-testid={selectors.access.panel} className="flex flex-col gap-6">
      <div className="flex flex-col gap-3 rounded-panel border border-app-border bg-app-surface p-4">
        <div className="flex items-center justify-between gap-3">
          <div className="flex flex-col gap-1">
            <p className="text-xs font-semibold uppercase text-app-muted-foreground">
              {t(strings.access.heading)}
            </p>
            <p className="text-sm text-app-muted-foreground">{t(strings.access.description)}</p>
          </div>
          <Button
            variant="outline"
            data-testid={selectors.access.refreshButton}
            onClick={() => void accessQuery.refetch()}
          >
            {t(strings.common.refresh)}
          </Button>
        </div>
        <div data-testid={selectors.access.summary} className="flex flex-wrap gap-2">
          {accessQuery.error ? (
            <StatusBadge tone="warning" data-testid={selectors.access.configuredBadge}>
              {t(strings.access.summaryUnavailable)}
            </StatusBadge>
          ) : (
            <>
              <StatusBadge
                tone={status?.enabled ? "success" : "neutral"}
                data-testid={selectors.access.globalBadge}
              >
                {status?.enabled ? t(strings.access.globalEnabled) : t(strings.access.globalDisabled)}
              </StatusBadge>
              <StatusBadge
                tone={status?.configured ? "success" : "warning"}
                data-testid={selectors.access.configuredBadge}
              >
                {status?.configured ? t(strings.access.configured) : t(strings.access.notConfigured)}
              </StatusBadge>
            </>
          )}
        </div>
        <p data-testid={selectors.access.note} className="text-sm text-app-muted-foreground">
          {t(strings.access.note)}
        </p>
      </div>

      <QueryState
        isLoading={accessQuery.isLoading}
        error={accessQuery.error}
        isEmpty={hosts.length === 0}
        loadingLabel={t(strings.access.loading)}
        errorLabel={t(strings.access.error)}
        emptyLabel={t(strings.access.empty)}
      >
        <div className="grid gap-3 sm:grid-cols-2">
          <div
            data-testid={selectors.access.planCreate}
            className="flex flex-col gap-1 rounded-panel border border-app-border bg-app-surface-muted p-3 text-sm"
          >
            <span className="text-xs font-semibold uppercase text-app-muted-foreground">
              {t(strings.access.planCreateHeading)}
            </span>
            <span className="text-app-foreground">
              {toCreate.length > 0 ? toCreate.join(", ") : t(strings.access.planNone)}
            </span>
          </div>
          <div
            data-testid={selectors.access.planRemove}
            className="flex flex-col gap-1 rounded-panel border border-app-border bg-app-surface-muted p-3 text-sm"
          >
            <span className="text-xs font-semibold uppercase text-app-muted-foreground">
              {t(strings.access.planRemoveHeading)}
            </span>
            <span className="text-app-foreground">
              {toRemove.length > 0 ? toRemove.join(", ") : t(strings.access.planNone)}
            </span>
          </div>
        </div>

        <div className="overflow-x-auto rounded-panel border border-app-border">
          <table data-testid={selectors.access.table} className="w-full text-left text-sm">
            <thead className="border-b border-app-border bg-app-surface-muted text-xs uppercase text-app-muted-foreground">
              <tr>
                <th className="px-3 py-2">{t(strings.access.colHost)}</th>
                <th className="px-3 py-2">{t(strings.access.colOverride)}</th>
                <th className="px-3 py-2">{t(strings.access.colEffective)}</th>
                <th className="px-3 py-2">{t(strings.access.colManaged)}</th>
                <th className="px-3 py-2">{t(strings.access.colAppId)}</th>
              </tr>
            </thead>
            <tbody>
              {hosts.map((host: AccessHostState) => (
                <tr
                  key={host.host}
                  data-testid={selectors.access.row}
                  className="border-b border-app-border last:border-0"
                >
                  <td data-testid={selectors.access.hostName} className="px-3 py-2 font-medium">
                    {host.host}
                  </td>
                  <td className="px-3 py-2">
                    <StatusBadge tone="neutral" data-testid={selectors.access.overrideBadge}>
                      {t(overrideLabel(host.override))}
                    </StatusBadge>
                  </td>
                  <td className="px-3 py-2">
                    <StatusBadge tone={boolTone(host.effectiveBypass)} data-testid={selectors.access.bypassBadge}>
                      {host.effectiveBypass ? t(strings.access.bypassOn) : t(strings.access.bypassOff)}
                    </StatusBadge>
                  </td>
                  <td className="px-3 py-2">
                    <StatusBadge tone={boolTone(host.managed)} data-testid={selectors.access.managedBadge}>
                      {host.managed ? t(strings.access.managedYes) : t(strings.access.managedNo)}
                    </StatusBadge>
                  </td>
                  <td data-testid={selectors.access.appId} className="px-3 py-2 text-app-muted-foreground">
                    {host.appId || "—"}
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
