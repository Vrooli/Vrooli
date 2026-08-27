/** @vrooliComponentSource visualization.cartesian-charts */
import { useQuery } from "@tanstack/react-query";
import { listVersionLedger } from "../../api/versionLedger";
import { CartesianCharts, type CartesianPoint } from "../../components/CartesianCharts";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

const percent = new Intl.NumberFormat(undefined, { style: "percent", maximumFractionDigits: 0 });

function lifecycleLabel(state: string) {
  return state.charAt(0).toUpperCase() + state.slice(1);
}

export function ProgressionPanel({ libraryId }: { libraryId: string }) {
  const { t } = useTranslation();
  const ledger = useQuery({
    queryKey: ["version-ledger", libraryId],
    queryFn: () => listVersionLedger(libraryId),
    enabled: Boolean(libraryId),
  });
  const rows = ledger.data ?? [];
  const newest = rows[rows.length - 1];
  const points: CartesianPoint[] = rows.map((row) => ({
    id: row.version,
    label: row.version,
    value: Math.round(row.testPassRate * 100),
    detail: `${row.gatePassCount ?? 0} passing gates · ${row.adoptionCurrent ?? 0} current adopters${row.presence === "evicted" ? " · evicted from disk" : ""}${row.lifecycleState === "retired" ? " · retired" : ""}`,
  }));
  return (
    <section
      data-testid={selectors.versions.progressionPanel}
      className="space-y-space-md"
      aria-labelledby="progression-panel-title"
    >
      {ledger.isLoading ? (
        <p role="status">{t(strings.componentDetail.progression.loading)}</p>
      ) : ledger.isError ? (
        <p role="alert">{t(strings.componentDetail.progression.error)}</p>
      ) : (
        <>
          <header className="rounded-panel border border-app-border bg-app-surface p-space-md">
            <p className="text-[0.7rem] font-semibold uppercase tracking-[0.16em] text-app-muted-foreground">
              Version ledger
            </p>
            <div className="mt-space-2xs flex flex-wrap items-end justify-between gap-space-sm">
              <div>
                <h2 id="progression-panel-title" className="text-lg font-semibold tracking-tight">
                  {t(strings.componentDetail.progression.title)}
                </h2>
                <p className="mt-space-3xs max-w-2xl text-sm text-app-muted-foreground">
                  {t(strings.componentDetail.progression.description)}
                </p>
              </div>
              {newest && (
                <div className="text-right">
                  <p className="text-xs text-app-muted-foreground">Current release</p>
                  <p className="font-mono text-xl font-semibold">{newest.version}</p>
                </div>
              )}
            </div>
          </header>

          <div className="grid gap-space-xs sm:grid-cols-3">
            <div className="rounded-control border border-app-border bg-app-surface p-space-sm">
              <p className="text-xs text-app-muted-foreground">Quality</p>
              <p className="mt-space-3xs text-xl font-semibold">
                {newest ? percent.format(newest.testPassRate) : "—"}
              </p>
              <p className="mt-space-3xs text-xs text-app-muted-foreground">
                latest test pass rate
              </p>
            </div>
            <div className="rounded-control border border-app-border bg-app-surface p-space-sm">
              <p className="text-xs text-app-muted-foreground">Adoption</p>
              <p className="mt-space-3xs text-xl font-semibold">{newest?.adoptionCurrent ?? 0}</p>
              <p className="mt-space-3xs text-xs text-app-muted-foreground">current consumers</p>
            </div>
            <div className="rounded-control border border-app-border bg-app-surface p-space-sm">
              <p className="text-xs text-app-muted-foreground">Lifecycle</p>
              <p className="mt-space-3xs text-xl font-semibold">
                {newest ? lifecycleLabel(newest.lifecycleState) : "—"}
              </p>
              <p className="mt-space-3xs text-xs text-app-muted-foreground">
                {rows.length} releases tracked
              </p>
            </div>
          </div>

          <CartesianCharts
            title="Test quality by release"
            description="Select a release to inspect its pass rate, gate results, and current adoption."
            data={points}
          />

          <div className="rounded-panel border border-app-border bg-app-surface p-space-md">
            <div className="flex flex-wrap items-baseline justify-between gap-space-xs">
              <div>
                <h3 className="text-sm font-semibold">Release history</h3>
                <p className="mt-space-3xs text-xs text-app-muted-foreground">
                  A compact ledger of what is safe to run and what is still in use.
                </p>
              </div>
              <span className="text-xs text-app-muted-foreground">{rows.length} releases</span>
            </div>
            <div className="mt-space-sm overflow-x-auto">
              <table className="w-full min-w-content text-left text-xs">
                <thead className="border-b border-app-border text-app-muted-foreground">
                  <tr>
                    <th className="pb-space-xs font-medium">Release</th>
                    <th className="pb-space-xs font-medium">Quality</th>
                    <th className="pb-space-xs font-medium">Adoption</th>
                    <th className="pb-space-xs font-medium">Lifecycle</th>
                    <th className="pb-space-xs font-medium">Presence</th>
                    <th className="pb-space-xs text-right font-medium">Size</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-app-border">
                  {[...rows].reverse().map((row) => (
                    <tr key={row.version}>
                      <th className="py-space-xs font-mono font-medium">{row.version}</th>
                      <td className="py-space-xs">{percent.format(row.testPassRate)}</td>
                      <td className="py-space-xs text-app-muted-foreground">
                        {row.adoptionCurrent} current · {row.adoptionPeak} peak
                      </td>
                      <td className="py-space-xs">
                        <span className="rounded-pill bg-app-surface-muted px-space-2xs py-space-3xs">
                          {lifecycleLabel(row.lifecycleState)}
                        </span>
                      </td>
                      <td className="py-space-xs">
                        <span className="rounded-pill bg-app-surface-muted px-space-2xs py-space-3xs">
                          {row.presence === "evicted" ? "⤓ evicted" : "● materialized"}
                        </span>
                      </td>
                      <td className="py-space-xs text-right font-mono text-app-muted-foreground">
                        {row.linesOfCode.toLocaleString()} LOC
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        </>
      )}
    </section>
  );
}
