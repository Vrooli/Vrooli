import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { fetchControlCenterOverview } from "../api/network";
import { useTranslation } from "../i18n";
import { useQuery } from "@tanstack/react-query";

const panelClass = "rounded-panel border border-app-border bg-app-surface p-4";
const labelClass = "text-xs font-semibold uppercase text-app-muted-foreground";

function StatusValue({ value }: { value: string }) {
  return <span className="font-medium text-app-foreground">{value || "unknown"}</span>;
}

export function DashboardPage() {
  const { t } = useTranslation();
  const { data, isLoading, error } = useQuery({
    queryKey: ["network", "overview"],
    queryFn: fetchControlCenterOverview,
  });

  const latestSnapshot = data?.snapshots[0];
  const supported = data?.capabilities.filter((capability) => capability.supported).length ?? 0;
  const unsupported = (data?.capabilities.length ?? 0) - supported;

  return (
    <section
      data-testid={selectors.pages.dashboard}
      aria-labelledby="dashboard-heading"
      className="flex flex-col gap-4"
    >
      <h2 id="dashboard-heading" className="text-2xl font-semibold">
        {t(strings.pages.dashboard.title)}
      </h2>
      <p className="text-app-muted-foreground">{t(strings.pages.dashboard.description)}</p>

      {isLoading && (
        <p data-testid={selectors.network.loading} className="text-app-muted-foreground">
          {t(strings.network.loading)}
        </p>
      )}
      {error && (
        <p data-testid={selectors.network.error} className="text-app-muted-foreground">
          {t(strings.network.error)}
        </p>
      )}

      <div className="grid gap-4 lg:grid-cols-3">
        <section data-testid={selectors.network.latestSnapshot} className={panelClass}>
          <p className={labelClass}>{t(strings.pages.dashboard.latestSnapshot)}</p>
          {latestSnapshot ? (
            <div className="mt-3 space-y-2 text-sm">
              <p className="text-lg font-semibold">{latestSnapshot.summary}</p>
              <p>
                {t(strings.network.status)} <StatusValue value={latestSnapshot.status} />
              </p>
              <p>
                {t(strings.network.profile)} <StatusValue value={latestSnapshot.profile} />
              </p>
            </div>
          ) : (
            <p data-testid={selectors.network.empty} className="mt-3 text-sm text-app-muted-foreground">
              {t(strings.pages.dashboard.noSnapshots)}
            </p>
          )}
        </section>

        <section data-testid={selectors.network.resolverStatus} className={panelClass}>
          <p className={labelClass}>{t(strings.pages.dashboard.resolverStatus)}</p>
          <div className="mt-3 space-y-2 text-sm">
            <p>
              {t(strings.network.backend)}{" "}
              <StatusValue value={data?.resolverStatus?.backend ?? t(strings.network.unknown)} />
            </p>
            <p>
              {t(strings.network.status)}{" "}
              <StatusValue value={data?.resolverStatus?.status ?? t(strings.network.unknown)} />
            </p>
            <p>
              {t(strings.network.filtering)}{" "}
              <StatusValue
                value={
                  data?.resolverStatus?.filteringEnabled
                    ? t(strings.network.enabled)
                    : t(strings.network.disabled)
                }
              />
            </p>
          </div>
        </section>

        <section data-testid={selectors.network.capabilitySummary} className={panelClass}>
          <p className={labelClass}>{t(strings.pages.dashboard.adapterCapabilities)}</p>
          <div className="mt-3 grid grid-cols-2 gap-3 text-sm">
            <div>
              <p className="text-2xl font-semibold">{supported}</p>
              <p className="text-app-muted-foreground">{t(strings.pages.dashboard.supported)}</p>
            </div>
            <div>
              <p className="text-2xl font-semibold">{unsupported}</p>
              <p className="text-app-muted-foreground">{t(strings.pages.dashboard.unsupported)}</p>
            </div>
          </div>
        </section>

        <section className={panelClass}>
          <p className={labelClass}>{t(strings.pages.dashboard.optimizationReadiness)}</p>
          <p className="mt-3 text-sm text-app-muted-foreground">
            {latestSnapshot ? t(strings.pages.dashboard.ready) : t(strings.pages.dashboard.blocked)}
          </p>
        </section>

        <section data-testid={selectors.network.privacySummary} className={panelClass}>
          <p className={labelClass}>{t(strings.pages.dashboard.privacyPosture)}</p>
          <dl className="mt-3 grid grid-cols-2 gap-3 text-sm">
            <div>
              <dt className="text-app-muted-foreground">{t(strings.pages.settings.queryLogDays)}</dt>
              <dd className="font-semibold">{data?.retention?.queryLogDays ?? 0}</dd>
            </div>
            <div>
              <dt className="text-app-muted-foreground">{t(strings.pages.settings.householdMode)}</dt>
              <dd className="font-semibold">
                {data?.visibility?.householdMode ? t(strings.network.enabled) : t(strings.network.disabled)}
              </dd>
            </div>
          </dl>
        </section>
      </div>
    </section>
  );
}
