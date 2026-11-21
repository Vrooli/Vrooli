import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { exportSnapshotReport, fetchControlCenterOverview, runSnapshot } from "../api/network";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

const panelClass = "rounded-panel border border-app-border bg-app-surface p-4";
const buttonClass = "rounded-control bg-app-primary px-3 py-2 text-sm font-medium text-app-primary-foreground";
const secondaryButtonClass = "rounded-control border border-app-border px-3 py-2 text-sm font-medium hover:bg-app-surface-muted";

export function SnapshotsPage() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { data, isLoading, error } = useQuery({
    queryKey: ["network", "overview"],
    queryFn: fetchControlCenterOverview,
  });
  const snapshotMutation = useMutation({
    mutationFn: () => runSnapshot("home"),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["network"] }),
  });
  const reportMutation = useMutation({
    mutationFn: (id: string) => exportSnapshotReport(id),
  });

  const latest = snapshotMutation.data ?? data?.snapshots[0];

  return (
    <section data-testid={selectors.pages.snapshots} aria-labelledby="snapshots-heading" className="flex flex-col gap-4">
      <div>
        <h2 id="snapshots-heading" className="text-2xl font-semibold">
          {t(strings.pages.snapshots.title)}
        </h2>
        <p className="text-app-muted-foreground">{t(strings.pages.snapshots.description)}</p>
      </div>

      {isLoading && <p data-testid={selectors.network.loading}>{t(strings.network.loading)}</p>}
      {error && <p data-testid={selectors.network.error}>{t(strings.network.error)}</p>}

      <div className="flex flex-wrap gap-2">
        <button type="button" className={buttonClass} onClick={() => snapshotMutation.mutate()}>
          {t(strings.pages.snapshots.run)}
        </button>
        <button
          type="button"
          className={secondaryButtonClass}
          disabled={!latest}
          onClick={() => latest && reportMutation.mutate(latest.id)}
        >
          {t(strings.pages.snapshots.export)}
        </button>
      </div>

      {!latest && !isLoading && (
        <p data-testid={selectors.network.empty} className={panelClass}>
          {t(strings.pages.snapshots.empty)}
        </p>
      )}

      {latest && (
        <div className="grid gap-4 xl:grid-cols-[minmax(0,1.4fr)_minmax(18rem,0.6fr)]">
          <section data-testid={selectors.network.latestSnapshot} className={panelClass}>
            <p className="text-lg font-semibold">{latest.summary}</p>
            <p className="text-sm text-app-muted-foreground">
              {t(strings.network.status)}: {latest.status} · {t(strings.network.created)}: {latest.createdAt}
            </p>
            <h3 className="mt-5 text-sm font-semibold uppercase text-app-muted-foreground">
              {t(strings.pages.snapshots.metrics)}
            </h3>
            <div className="mt-2 overflow-x-auto">
              <table className="w-full min-w-[34rem] text-left text-sm">
                <thead className="text-app-muted-foreground">
                  <tr>
                    <th className="py-2 pe-3 font-medium">{t(strings.network.metric)}</th>
                    <th className="py-2 pe-3 font-medium">{t(strings.network.value)}</th>
                    <th className="py-2 pe-3 font-medium">{t(strings.network.unit)}</th>
                    <th className="py-2 font-medium">{t(strings.network.status)}</th>
                  </tr>
                </thead>
                <tbody>
                  {latest.metrics.map((metric) => (
                    <tr key={`${metric.name}-${metric.status}`} className="border-t border-app-border">
                      <td className="py-2 pe-3">{metric.name}</td>
                      <td className="py-2 pe-3">{metric.value || t(strings.network.none)}</td>
                      <td className="py-2 pe-3">{metric.unit || t(strings.network.none)}</td>
                      <td className="py-2">{metric.status}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </section>

          <aside className={panelClass}>
            <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">
              {t(strings.pages.snapshots.findings)}
            </h3>
            <ul className="mt-3 list-disc space-y-2 ps-5 text-sm">
              {(latest.findings.length > 0 ? latest.findings : [t(strings.network.none)]).map((finding) => (
                <li key={finding}>{finding}</li>
              ))}
            </ul>
          </aside>
        </div>
      )}

      {reportMutation.data && (
        <section className={panelClass}>
          <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">
            {t(strings.pages.snapshots.report)}
          </h3>
          <pre className="mt-3 max-h-96 overflow-auto whitespace-pre-wrap text-sm">{reportMutation.data}</pre>
        </section>
      )}
    </section>
  );
}
