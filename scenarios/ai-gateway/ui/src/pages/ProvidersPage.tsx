import { useMutation, useQuery } from "@tanstack/react-query";
import { RefreshCw } from "lucide-react";

import { listProviderRoles, smokeProvider } from "../api/gateway";
import { StatusChip } from "../components/StatusChip";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { errorMessage } from "../lib/errorMessage";

const statusTone = (status: string) => {
  const normalized = status.toLowerCase();
  if (normalized.includes("ready") || normalized.includes("ok")) return "success";
  if (normalized.includes("missing") || normalized.includes("error")) return "danger";
  if (normalized.includes("stale") || normalized.includes("warn")) return "warning";
  return "neutral";
};

export function ProvidersPage() {
  const { t } = useTranslation();
  const rolesQuery = useQuery({
    queryKey: ["provider-roles"],
    queryFn: () => listProviderRoles(),
  });
  const smokeMutation = useMutation({
    mutationFn: smokeProvider,
  });

  const providers = Array.from(new Set((rolesQuery.data?.roles ?? []).map((role) => role.provider)));

  return (
    <section
      data-testid={selectors.pages.providers}
      aria-labelledby="providers-heading"
      className="flex flex-col gap-5"
    >
      <header className="flex flex-col gap-2">
        <p className="text-xs font-semibold uppercase text-app-muted-foreground">
          {t(strings.pages.providers.eyebrow)}
        </p>
        <div className="flex flex-wrap items-end justify-between gap-3">
          <div>
            <h2 id="providers-heading" className="text-2xl font-semibold">
              {t(strings.pages.providers.title)}
            </h2>
            <p className="mt-1 max-w-3xl text-sm text-app-muted-foreground">
              {t(strings.pages.providers.description)}
            </p>
          </div>
          <button
            type="button"
            onClick={() => void rolesQuery.refetch()}
            className="inline-flex min-h-10 items-center gap-2 rounded-control border border-app-border px-3 text-sm font-medium hover:bg-app-surface-muted"
          >
            <RefreshCw aria-hidden="true" size={16} />
            {t(strings.actions.refresh)}
          </button>
        </div>
      </header>

      {rolesQuery.isLoading ? (
        <div data-testid={selectors.providers.loading} className="rounded-panel border border-app-border bg-app-surface p-4 text-sm">
          {t(strings.states.loading)}
        </div>
      ) : rolesQuery.isError ? (
        <div data-testid={selectors.providers.error} className="rounded-panel border border-red-200 bg-red-50 p-4 text-sm text-red-700">
          {errorMessage(rolesQuery.error, t)}
        </div>
      ) : (rolesQuery.data?.roles.length ?? 0) === 0 ? (
        <div data-testid={selectors.providers.empty} className="rounded-panel border border-app-border bg-app-surface p-4 text-sm text-app-muted-foreground">
          {t(strings.pages.providers.empty)}
        </div>
      ) : (
        <div data-testid={selectors.providers.table} className="overflow-hidden rounded-panel border border-app-border bg-app-surface">
          <table className="w-full min-w-[760px] text-left text-sm">
            <thead className="bg-app-surface-muted text-xs uppercase text-app-muted-foreground">
              <tr>
                <th className="px-4 py-3">{t(strings.pages.providers.columns.provider)}</th>
                <th className="px-4 py-3">{t(strings.pages.providers.columns.role)}</th>
                <th className="px-4 py-3">{t(strings.pages.providers.columns.locality)}</th>
                <th className="px-4 py-3">{t(strings.pages.providers.columns.status)}</th>
                <th className="px-4 py-3">{t(strings.pages.providers.columns.capabilities)}</th>
                <th className="px-4 py-3">{t(strings.pages.providers.columns.policy)}</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-app-border">
              {rolesQuery.data?.roles.map((role) => (
                <tr key={`${role.provider}-${role.role}`} data-testid={selectors.providers.roleRow({ provider: role.provider, role: role.role })}>
                  <td className="px-4 py-3 font-medium">{role.provider}</td>
                  <td className="px-4 py-3 font-mono text-xs">{role.role}</td>
                  <td className="px-4 py-3">{role.locality}</td>
                  <td className="px-4 py-3">
                    <StatusChip tone={statusTone(role.status)}>{role.status || t(strings.states.unknown)}</StatusChip>
                  </td>
                  <td className="px-4 py-3 text-app-muted-foreground">
                    {role.capabilities.length > 0 ? role.capabilities.join(", ") : t(strings.states.none)}
                  </td>
                  <td className="px-4 py-3 font-mono text-xs">{role.policySchemaVersion || t(strings.states.none)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <div className="grid gap-3 md:grid-cols-2">
        {providers.map((provider) => (
          <div key={provider} className="rounded-panel border border-app-border bg-app-surface p-4">
            <div className="flex items-start justify-between gap-3">
              <div>
                <h3 className="font-semibold">{provider}</h3>
                <p className="mt-1 text-sm text-app-muted-foreground">
                  {t(strings.pages.providers.smokeDescription)}
                </p>
              </div>
              <button
                type="button"
                onClick={() => smokeMutation.mutate(provider)}
                className="inline-flex min-h-10 items-center gap-2 rounded-control bg-app-primary px-3 text-sm font-medium text-app-primary-foreground"
              >
                <RefreshCw aria-hidden="true" size={16} />
                {t(strings.pages.providers.smoke)}
              </button>
            </div>
            {smokeMutation.variables === provider && smokeMutation.data ? (
              <p className="mt-3 text-sm">
                <StatusChip tone={statusTone(smokeMutation.data.status)}>
                  {smokeMutation.data.status || t(strings.states.unknown)}
                </StatusChip>{" "}
                <span className="text-app-muted-foreground">{smokeMutation.data.message}</span>
              </p>
            ) : null}
            {smokeMutation.variables === provider && smokeMutation.isError ? (
              <p className="mt-3 text-sm text-red-700">{errorMessage(smokeMutation.error, t)}</p>
            ) : null}
          </div>
        ))}
      </div>
    </section>
  );
}
