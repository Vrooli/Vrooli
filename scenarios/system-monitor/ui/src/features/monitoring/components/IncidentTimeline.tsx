import { useMemo, useState } from 'react';
import { timestampDate } from '@bufbuild/protobuf/wkt';
import type { Investigation, MetricHistory } from '../../../types';
import { InvestigationStatus } from '../../../types';
import { useTimeRange } from '../../../shared/time/TimeRangeContext';

export interface TimelineEntry {
  id: string;
  at: string;
  severity: 'critical' | 'warning' | 'info';
  source: 'metrics' | 'investigation';
  title: string;
  detail: string;
  value?: number;
}

interface IncidentTimelineProps {
  history: MetricHistory | null;
  investigations: Investigation[];
  onOpenSource: (source: 'logs' | 'forensics') => void;
  onInvestigate: (entry: TimelineEntry) => void;
}

function metricEntries(history: MetricHistory | null): TimelineEntry[] {
  if (!history) return [];
  const entries: TimelineEntry[] = [];
  const series: Array<[string, typeof history.cpu]> = [
    ['CPU', history.cpu],
    ['Memory', history.memory],
    ['Network', history.network],
    ['Disk', history.diskUsage ?? []],
  ];
  for (const [label, points] of series) {
    const point = [...points].reverse().find((candidate) => candidate.value >= 80);
    if (!point) continue;
    entries.push({
      id: `metric-${label}-${point.timestamp}`,
      at: point.timestamp,
      severity: point.value >= 95 ? 'critical' : 'warning',
      source: 'metrics',
      title: `${label} crossed the attention threshold`,
      detail: `${point.value.toFixed(1)}% measured in the shared observation window`,
      value: point.value,
    });
  }
  return entries;
}

export function IncidentTimeline({ history, investigations, onOpenSource, onInvestigate }: IncidentTimelineProps) {
  const { range } = useTimeRange();
  const [selected, setSelected] = useState<TimelineEntry | null>(null);
  const entries = useMemo(() => [
    ...metricEntries(history),
    ...investigations.slice(0, 5).map((investigation) => ({
      id: `investigation-${investigation.id}`,
      at: investigation.startTime ? timestampDate(investigation.startTime).toISOString() : new Date().toISOString(),
      severity: investigation.status === InvestigationStatus.FAILED ? 'critical' as const : 'info' as const,
      source: 'investigation' as const,
      title: `Investigation ${investigation.status}`,
      detail: investigation.findings || (investigation.details ? 'Investigation details attached' : 'Investigation activity recorded'),
    })),
  ].sort((a, b) => Date.parse(b.at) - Date.parse(a.at)), [history, investigations]);

  return (
    <section className="card incident-timeline" aria-labelledby="incident-timeline-heading">
      <div className="flex-row-center" data-sm-style="sm-style-f81ee4dad6">
        <div>
          <h2 id="incident-timeline-heading" data-sm-style="sm-style-d47aef18a0">Incident timeline</h2>
          <p className="text-sm text-muted" data-sm-style="sm-style-2a0ca8350a">Correlated signals across the last {range.label}.</p>
        </div>
        <span className="badge badge-info">shared time axis</span>
      </div>

      {entries.length === 0 ? (
        <div className="empty-state" data-sm-style="sm-style-323fdcc1e0">
          No threshold crossings or investigations in this window. Signals will appear here when attention is required.
        </div>
      ) : (
        <ol className="incident-timeline-list" data-sm-style="sm-style-c5121bd731">
          {entries.map((entry) => (
            <li key={entry.id} className="incident-timeline-entry">
              <button type="button" className="incident-timeline-entry-button" onClick={() => { setSelected(entry); }}>
                <span className={`status-dot status-${entry.severity}`} aria-hidden="true" />
                <span>
                  <strong>{entry.title}</strong>
                  <span className="text-xs text-muted">{new Date(entry.at).toLocaleString()} · {entry.detail}</span>
                </span>
              </button>
            </li>
          ))}
        </ol>
      )}

      {selected && (
        <div className="card incident-correlated-view" role="dialog" aria-label="Correlated incident view" data-sm-style="sm-style-323fdcc1e0">
          <div className="flex-row-center" data-sm-style="sm-style-53d9d4c2b2">
            <h3 data-sm-style="sm-style-2a0ca8350a">Correlated view</h3>
            <button type="button" className="header-button" onClick={() => { setSelected(null); }}>Close</button>
          </div>
          <p className="text-sm">{selected.title} · all sources scoped to {range.label} around {new Date(selected.at).toLocaleString()}.</p>
          <div className="flex-row-center" data-sm-style="sm-style-2bec9ac048">
            <button type="button" className="header-button" onClick={() => { onOpenSource('logs'); }}>Open logs</button>
            <button type="button" className="header-button" onClick={() => { onOpenSource('forensics'); }}>Open forensics</button>
            <button type="button" className="btn btn-primary" onClick={() => { onInvestigate(selected); }}>Investigate this window</button>
          </div>
        </div>
      )}
    </section>
  );
}
