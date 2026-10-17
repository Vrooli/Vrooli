import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import {
  fetchControlCenterOverview,
  runMonitoringCheck,
  upsertMonitoringSchedule,
} from "../api/network";
import { useTranslation } from "../i18n";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useMemo, useState } from "react";

const panelClass = "rounded-panel border border-app-border bg-app-surface p-4";
const labelClass = "text-xs font-semibold uppercase text-app-muted-foreground";

function StatusValue({ value }: { value: string }) {
  return <span className="font-medium text-app-foreground">{value || "unknown"}</span>;
}

export function DashboardPage() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [scheduleName, setScheduleName] = useState("Home baseline watch");
  const [intervalMinutes, setIntervalMinutes] = useState("60");
  const [selectedScheduleID, setSelectedScheduleID] = useState("");
  const { data, isLoading, error } = useQuery({
    queryKey: ["network", "overview"],
    queryFn: fetchControlCenterOverview,
  });

  const latestSnapshot = data?.snapshots[0];
  const baselineSnapshot = useMemo(
    () => data?.snapshots.find((snapshot) => snapshot.status === "baseline") ?? latestSnapshot,
    [data?.snapshots, latestSnapshot],
  );
  const supported = data?.capabilities.filter((capability) => capability.supported).length ?? 0;
  const unsupported = (data?.capabilities.length ?? 0) - supported;
  const schedules = data?.monitoringSchedules ?? [];
  const selectedSchedule = schedules.find((schedule) => schedule.id === selectedScheduleID) ?? schedules[0];
  const alerts = data?.monitoringAlerts ?? [];
  const refreshOverview = () => {
    void queryClient.invalidateQueries({ queryKey: ["network", "overview"] });
  };
  const saveSchedule = useMutation({
    mutationFn: () =>
      upsertMonitoringSchedule({
        name: scheduleName,
        profile: baselineSnapshot?.profile ?? "home",
        baselineSnapshotId: baselineSnapshot?.id ?? "",
        intervalMinutes: Number(intervalMinutes) || 60,
        enabled: true,
        latencyThresholdMs: 100,
        unavailableThreshold: 1,
      }),
    onSuccess: (schedule) => {
      if (schedule?.id) {
        setSelectedScheduleID(schedule.id);
      }
      refreshOverview();
    },
  });
  const runCheck = useMutation({
    mutationFn: () => runMonitoringCheck(selectedSchedule?.id ?? ""),
    onSuccess: refreshOverview,
  });

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

        <section data-testid={selectors.network.monitoringPanel} className={`${panelClass} lg:col-span-3`}>
          <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
            <div>
              <p className={labelClass}>{t(strings.pages.dashboard.monitoring)}</p>
              <p className="mt-2 text-sm text-app-muted-foreground">
                {baselineSnapshot
                  ? `${t(strings.network.baseline)} ${baselineSnapshot.id}`
                  : t(strings.pages.dashboard.monitoringBaselineRequired)}
              </p>
            </div>
            <div className="grid gap-2 sm:grid-cols-[minmax(12rem,1fr)_8rem_auto_auto]">
              <label className="sr-only" htmlFor="monitoring-schedule-name">
                {t(strings.pages.dashboard.monitoringSchedule)}
              </label>
              <Input
                id="monitoring-schedule-name"
                value={scheduleName}
                onChange={(event) => setScheduleName(event.target.value)}
                disabled={!baselineSnapshot || saveSchedule.isPending}
              />
              <label className="sr-only" htmlFor="monitoring-interval">
                {t(strings.network.interval)}
              </label>
              <Input
                id="monitoring-interval"
                inputMode="numeric"
                value={intervalMinutes}
                onChange={(event) => setIntervalMinutes(event.target.value)}
                disabled={!baselineSnapshot || saveSchedule.isPending}
              />
              <Button
                type="button"
                size="sm"
                onClick={() => saveSchedule.mutate()}
                disabled={!baselineSnapshot || saveSchedule.isPending}
              >
                {t(strings.pages.dashboard.monitoringSchedule)}
              </Button>
              <Button
                type="button"
                size="sm"
                variant="outline"
                onClick={() => runCheck.mutate()}
                disabled={!selectedSchedule?.id || runCheck.isPending}
              >
                {t(strings.pages.dashboard.monitoringRun)}
              </Button>
            </div>
          </div>
          <div className="mt-4 grid gap-4 lg:grid-cols-2">
            <div className="text-sm">
              <p className="font-semibold">{t(strings.pages.dashboard.monitoringSchedule)}</p>
              {schedules.length > 0 ? (
                <ul className="mt-2 space-y-1">
                  {schedules.map((schedule) => (
                    <li key={schedule.id}>
                      <button
                        type="button"
                        className="text-left underline-offset-2 hover:underline"
                        onClick={() => setSelectedScheduleID(schedule.id)}
                      >
                        <span>{schedule.name}</span>
                        <span aria-hidden="true" className="mx-2 inline-block h-3 border-l border-app-border align-middle" />
                        <span>{t(strings.network.minutes, { count: schedule.intervalMinutes })}</span>
                        <span aria-hidden="true" className="mx-2 inline-block h-3 border-l border-app-border align-middle" />
                        <span>{schedule.enabled ? t(strings.network.enabled) : t(strings.network.disabled)}</span>
                      </button>
                    </li>
                  ))}
                </ul>
              ) : (
                <p className="mt-2 text-app-muted-foreground">{t(strings.pages.dashboard.monitoringEmpty)}</p>
              )}
            </div>
            <div className="text-sm">
              <p className="font-semibold">{t(strings.pages.dashboard.monitoringAlerts)}</p>
              {alerts.length > 0 ? (
                <ul className="mt-2 space-y-1">
                  {alerts.slice(0, 4).map((alert) => (
                    <li key={alert.id}>
                      {alert.severity} · {alert.status} · {alert.summary}
                    </li>
                  ))}
                </ul>
              ) : (
                <p className="mt-2 text-app-muted-foreground">{t(strings.network.none)}</p>
              )}
            </div>
          </div>
        </section>
      </div>
    </section>
  );
}
