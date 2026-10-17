import { useEffect, useMemo, useState } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../../shared/ui/card';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../../../shared/ui/select';
import {
  getTrafficBreakdown,
  getTrafficSeries,
  type TrafficBreakdown,
  type TrafficDimensionKey,
  type TrafficSeries,
} from '../../../shared/api';
import type { AnalyticsDateRange } from '../controllers/analyticsController';

const DIMENSIONS: Array<{ value: TrafficDimensionKey; label: string }> = [
  { value: 'country', label: 'Country' },
  { value: 'referrer_kind', label: 'Referrer kind' },
  { value: 'utm_source', label: 'Campaign source' },
  { value: 'utm_campaign', label: 'Campaign' },
  { value: 'device_class', label: 'Device' },
  { value: 'landing_path', label: 'Landing page' },
  { value: 'variant', label: 'Variant' },
];

interface Props { range: AnalyticsDateRange; }

export function TrafficAttributionSection({ range }: Props) {
  const [dimension, setDimension] = useState<TrafficDimensionKey>('country');
  const [breakdown, setBreakdown] = useState<TrafficBreakdown | null>(null);
  const [series, setSeries] = useState<TrafficSeries | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    setError(null);
    void Promise.all([
      getTrafficBreakdown(dimension, range.startDate, range.endDate, 10),
      getTrafficSeries('visitors', range.startDate, range.endDate),
    ]).then(([nextBreakdown, nextSeries]) => {
      if (cancelled) return;
      setBreakdown(nextBreakdown);
      setSeries(nextSeries);
    }).catch((cause: unknown) => {
      if (!cancelled) setError(cause instanceof Error ? cause.message : 'Traffic attribution is unavailable');
    });
    return () => { cancelled = true; };
  }, [dimension, range.startDate, range.endDate]);

  const maxSeriesValue = useMemo(() => Math.max(...(series?.points.map((point) => point.value) ?? [0]), 1), [series]);
  const hasEnrichedData = Boolean(breakdown?.rows.length);

  return (
    <section className="mb-8 grid gap-6 xl:grid-cols-[1.15fr_0.85fr]" data-testid="traffic-attribution">
      <Card className="border-white/10 bg-white/5">
        <CardHeader className="flex flex-row items-start justify-between gap-4">
          <div>
            <CardTitle>Traffic over time</CardTitle>
            <CardDescription className="text-slate-400">Unique visitors by day in the selected window.</CardDescription>
          </div>
          <span className="text-xs text-slate-500">{series?.unit ?? 'count'}</span>
        </CardHeader>
        <CardContent>
          {error ? <p className="text-sm text-amber-200">Traffic observations are temporarily unavailable.</p> : series?.points.length ? (
            <div className="flex h-44 items-end gap-1" data-testid="traffic-series">
              {series.points.map((point) => (
                <div key={point.bucket_start} className="group flex h-full flex-1 flex-col justify-end" title={`${new Date(point.bucket_start).toLocaleDateString()}: ${point.value.toLocaleString()}`}>
                  <div className="min-h-1 rounded-t bg-sky-400/80" style={{ height: `${String(Math.max((point.value / maxSeriesValue) * 100, 2))}%` }} />
                </div>
              ))}
            </div>
          ) : (
            <div className="py-16 text-center text-sm text-slate-400" data-testid="traffic-series-empty">No enriched traffic data exists for this window.</div>
          )}
        </CardContent>
      </Card>

      <Card className="border-white/10 bg-white/5">
        <CardHeader className="flex flex-row items-start justify-between gap-4">
          <div>
            <CardTitle>Traffic attribution</CardTitle>
            <CardDescription className="text-slate-400">Ranked visitors, conversions, and revenue.</CardDescription>
          </div>
          <Select value={dimension} onValueChange={(value) => { setDimension(value as TrafficDimensionKey); }}>
            <SelectTrigger className="w-44" data-testid="traffic-dimension-select"><SelectValue /></SelectTrigger>
            <SelectContent>{DIMENSIONS.map((item) => <SelectItem key={item.value} value={item.value}>{item.label}</SelectItem>)}</SelectContent>
          </Select>
        </CardHeader>
        <CardContent>
          {error ? <p className="text-sm text-amber-200">Traffic observations are temporarily unavailable.</p> : !hasEnrichedData ? (
            <div className="py-12 text-center text-sm text-slate-400" data-testid="traffic-breakdown-empty">
              No enriched traffic data exists for this window. Older events may predate attribution enrichment.
            </div>
          ) : (
            <div className="space-y-4" data-testid="traffic-breakdown-rows">
              {breakdown?.rows.map((row) => (
                <div key={row.key}>
                  <div className="mb-1 flex items-center justify-between gap-3 text-sm">
                    <span className="truncate text-slate-200">{row.label}</span>
                    <span className="shrink-0 text-slate-400">{row.sessions.toLocaleString()} · {(row.share * 100).toFixed(1)}%</span>
                  </div>
                  <div className="h-2 overflow-hidden rounded-full bg-white/10"><div className="h-full rounded-full bg-sky-400" style={{ width: `${String(Math.min(row.share * 100, 100))}%` }} /></div>
                  <div className="mt-1 text-xs text-slate-500">{row.conversions.toLocaleString()} conversions · {breakdown.currency.toUpperCase()} {(row.revenue_minor / 100).toFixed(2)}</div>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
      <p className="xl:col-span-2 text-xs text-slate-500" data-testid="traffic-count-semantics">Visitors: unique stable visitor IDs. Sessions: distinct session IDs ({breakdown?.total_sessions.toLocaleString() ?? '—'} in this attribution view).</p>
    </section>
  );
}

export function RevenueByCampaign({ range }: Props) {
  const [breakdown, setBreakdown] = useState<TrafficBreakdown | null>(null);
  const [unavailable, setUnavailable] = useState(false);

  useEffect(() => {
    let cancelled = false;
    setUnavailable(false);
    void getTrafficBreakdown('utm_campaign', range.startDate, range.endDate, 10).then((result) => {
      if (!cancelled) setBreakdown(result);
    }).catch(() => {
      if (!cancelled) setUnavailable(true);
    });
    return () => { cancelled = true; };
  }, [range.startDate, range.endDate]);

  return (
    <Card className="border-white/10 bg-white/5" data-testid="revenue-by-campaign">
      <CardHeader>
        <CardTitle>Revenue by campaign</CardTitle>
        <CardDescription className="text-slate-400">Successful checkout revenue joined to first-touch campaign attribution.</CardDescription>
      </CardHeader>
      <CardContent>
        {unavailable ? <p className="text-sm text-amber-200">Campaign revenue observations are temporarily unavailable.</p> : !breakdown?.rows.length ? (
          <p className="py-8 text-center text-sm text-slate-400">No enriched campaign revenue exists for this window.</p>
        ) : (
          <div className="space-y-3">
            {breakdown.rows.map((row) => <div key={row.key} className="flex items-center justify-between gap-4 border-b border-white/5 py-2 text-sm"><span className="truncate text-slate-200">{row.label}</span><span className="shrink-0 font-medium text-emerald-300">{breakdown.currency.toUpperCase()} {(row.revenue_minor / 100).toFixed(2)}</span></div>)}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

export { DIMENSIONS };
