import { useSelfHealth } from "../../hooks/useSelfHealth";
import { selectors } from "../../consts/selectors";
import type {
  CatalogSummary,
  ProviderConformance,
  ReliabilityLedger,
  PhaseReliability,
  SelfHealthTrendPoint,
  TrendDelta
} from "../../lib/api";

const card = "rounded-2xl border border-white/10 bg-white/[0.02] p-5";
const sectionTitle = "text-xs uppercase tracking-[0.3em] text-slate-400";

function pct(value?: number): string {
  if (typeof value !== "number" || Number.isNaN(value)) return "—";
  return `${Math.round(value * 100)}%`;
}

function num(value?: number): number {
  return typeof value === "number" ? value : 0;
}

function deltaLabel(delta?: number, asPct = false): string {
  if (typeof delta !== "number" || delta === 0) return "no change";
  const arrow = delta > 0 ? "▲" : "▼";
  const magnitude = asPct ? `${Math.abs(Math.round(delta * 100))}%` : `${Math.abs(delta)}`;
  return `${arrow} ${magnitude}`;
}

function CatalogSection({ catalog }: { catalog?: CatalogSummary }) {
  if (!catalog) return null;
  return (
    <section className={card} data-testid={selectors.health.catalog}>
      <p className={sectionTitle}>Phase Catalog</p>
      <div className="mt-3 flex flex-wrap gap-6 text-sm">
        <Stat label="Total phases" value={num(catalog.totalPhases)} />
        <Stat label="Delegated" value={num(catalog.delegatedPhases)} />
        <Stat label="Native" value={num(catalog.nativePhases)} />
      </div>
      <div className="mt-4 flex flex-wrap gap-2">
        {(catalog.phases ?? []).map((phase) => (
          <span
            key={phase.name}
            className="rounded-full border border-white/15 px-3 py-1 text-xs text-slate-300"
            title={phase.delegated ? `delegated → ${phase.provider ?? "?"}` : "native"}
          >
            {phase.name}
            <span className="ml-1 text-slate-500">{phase.delegated ? "·d" : "·n"}</span>
          </span>
        ))}
      </div>
    </section>
  );
}

function ConformanceSection({ conformance, freshness }: { conformance?: ProviderConformance[]; freshness?: string }) {
  const rows = conformance ?? [];
  return (
    <section className={card} data-testid={selectors.health.conformance}>
      <div className="flex items-center justify-between">
        <p className={sectionTitle}>Provider Conformance</p>
        <span className="text-xs text-slate-500">{freshness === "skipped" ? "scan skipped" : "live scan"}</span>
      </div>
      {rows.length === 0 ? (
        <p className="mt-3 text-sm text-slate-400">No conformance data.</p>
      ) : (
        <div className="mt-3 overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead className="text-xs uppercase tracking-wider text-slate-500">
              <tr>
                <th className="py-2 pr-4">Provider</th>
                <th className="py-2 pr-4">Phase</th>
                <th className="py-2 pr-2">Reach</th>
                <th className="py-2 pr-2">Contract</th>
                <th className="py-2 pr-2">Spec</th>
                <th className="py-2 pr-2">Identity</th>
                <th className="py-2 pr-2">Metrics</th>
                <th className="py-2 pr-4">Score</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((row) => (
                <tr key={`${row.provider}:${row.phase}`} className="border-t border-white/5">
                  <td className="py-2 pr-4 text-slate-200">{row.provider}</td>
                  <td className="py-2 pr-4 text-slate-400">{row.phase}</td>
                  <td className="py-2 pr-2">{check(row.reachable)}</td>
                  <td className="py-2 pr-2">{check(row.contractValid)}</td>
                  <td className="py-2 pr-2">{check(row.specValid)}</td>
                  <td className="py-2 pr-2">{check(row.identityOk)}</td>
                  <td className="py-2 pr-2">{check(row.metricsAdopted)}</td>
                  <td className="py-2 pr-4 text-slate-200">{pct(row.adoptionScore)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}

function LedgerSection({ ledger }: { ledger?: ReliabilityLedger }) {
  if (!ledger) return null;
  const phases = ledger.phases ?? [];
  return (
    <section className={card} data-testid={selectors.health.ledger}>
      <div className="flex items-center justify-between">
        <p className={sectionTitle}>Reliability &amp; Performance</p>
        <span className="text-xs text-slate-500">
          {num(ledger.runCount)} runs · {num(ledger.windowDays)}d window · {pct(ledger.availability)} availability
        </span>
      </div>
      <div className="mt-3 flex flex-wrap gap-2">
        {(ledger.runOutcomes ?? []).map((o) => (
          <span key={o.outcome} className="rounded-full border border-white/15 px-3 py-1 text-xs text-slate-300">
            {o.outcome}: {num(o.count)}
          </span>
        ))}
      </div>
      {phases.length === 0 ? (
        <p className="mt-3 text-sm text-slate-400">No phase observations in window.</p>
      ) : (
        <div className="mt-4 overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead className="text-xs uppercase tracking-wider text-slate-500">
              <tr>
                <th className="py-2 pr-4">Phase</th>
                <th className="py-2 pr-2">Avail</th>
                <th className="py-2 pr-2">Fail</th>
                <th className="py-2 pr-2">Obs</th>
                <th className="py-2 pr-2">Metrics</th>
                <th className="py-2 pr-2">p50</th>
                <th className="py-2 pr-2">p95</th>
                <th className="py-2 pr-4">max</th>
              </tr>
            </thead>
            <tbody>
              {phases.map((p: PhaseReliability) => (
                <tr key={p.phase} className="border-t border-white/5">
                  <td className="py-2 pr-4 text-slate-200">
                    {p.phase}
                    {p.provider ? <span className="ml-1 text-slate-500">→ {p.provider}</span> : null}
                  </td>
                  <td className="py-2 pr-2">{pct(p.availability)}</td>
                  <td className="py-2 pr-2">{pct(p.failureRate)}</td>
                  <td className="py-2 pr-2 text-slate-400">{num(p.totalObservations)}</td>
                  <td className="py-2 pr-2 text-slate-400">{num(p.metricsAdopted)}</td>
                  <td className="py-2 pr-2 text-slate-400">{num(p.duration?.p50)}s</td>
                  <td className="py-2 pr-2 text-slate-400">{num(p.duration?.p95)}s</td>
                  <td className="py-2 pr-4 text-slate-400">{num(p.duration?.max)}s</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}

function TrendSection({ trend, series }: { trend?: TrendDelta; series?: SelfHealthTrendPoint[] }) {
  const points = series ?? [];
  if (!trend && points.length === 0) return null;
  return (
    <section className={card} data-testid={selectors.health.trend}>
      <p className={sectionTitle}>Trend</p>
      {trend ? (
        <div className="mt-3 flex flex-wrap gap-6 text-sm">
          <Stat label="Availability Δ" value={deltaLabel(trend.availabilityDelta, true)} />
          <Stat label="Run count Δ" value={deltaLabel(trend.runCountDelta)} />
          <Stat label="Prev availability" value={pct(trend.previousAvailability)} />
        </div>
      ) : null}
      {points.length > 0 ? (
        <div className="mt-4 space-y-1 text-xs text-slate-400">
          {points.slice(0, 8).map((point) => (
            <div key={point.capturedAt} className="flex justify-between border-t border-white/5 py-1">
              <span>{point.capturedAt?.slice(0, 19).replace("T", " ")}</span>
              <span>
                {pct(point.availability)} avail · {num(point.runCount)} runs · {num(point.metricsAdopted)} metrics
              </span>
            </div>
          ))}
        </div>
      ) : null}
    </section>
  );
}

function Stat({ label, value }: { label: string; value: string | number }) {
  return (
    <div>
      <p className="text-2xl font-semibold text-white">{value}</p>
      <p className="text-xs text-slate-400">{label}</p>
    </div>
  );
}

function check(ok?: boolean) {
  return ok ? <span className="text-emerald-400">✓</span> : <span className="text-rose-400">✕</span>;
}

export function HealthPage() {
  const { data, isLoading, isError, error } = useSelfHealth();

  if (isLoading) {
    return (
      <div className={card} data-testid={selectors.health.page}>
        <p className="text-sm text-slate-400">Loading self-health…</p>
      </div>
    );
  }

  if (isError) {
    return (
      <div className={card} data-testid={selectors.health.page}>
        <p className="text-sm text-rose-400">Failed to load self-health: {(error as Error)?.message ?? "unknown error"}</p>
      </div>
    );
  }

  const selfHealth = data ?? {};
  const hasLedger = (selfHealth.ledger?.phases ?? []).length > 0 || num(selfHealth.ledger?.runCount) > 0;

  return (
    <div className="flex flex-col gap-5" data-testid={selectors.health.page}>
      <CatalogSection catalog={selfHealth.catalog} />
      <ConformanceSection conformance={selfHealth.conformance} freshness={selfHealth.conformanceFreshness} />
      {hasLedger ? (
        <LedgerSection ledger={selfHealth.ledger} />
      ) : (
        <div className={card} data-testid={selectors.health.empty}>
          <p className="text-sm text-slate-400">No runs recorded in the ledger window yet.</p>
        </div>
      )}
      <TrendSection trend={selfHealth.ledger?.trend} series={selfHealth.trendSeries} />
    </div>
  );
}
