import { useMemo, useState } from 'react';
import { timestampDate } from '@bufbuild/protobuf/wkt';
import type { ChartDataPoint, Investigation, MetricHistory } from '../../../types';
import { InvestigationStatus } from '../../../types';
import { statusEnumToString } from '../../../shared/api/proto-converters';
import { useTimeRange } from '../../../shared/time/TimeRangeContext';
import { ReportBody } from '../../../shared/report/ReportBody';
import { reportToPlainText } from '../../../shared/report/parseReport';

const SEVERITY_LABEL: Record<TimelineEntry['severity'], string> = {
  critical: 'Critical',
  warning: 'Warning',
  info: 'Info',
};

/** Turn the proto enum into prose: `in_progress` -> `in progress`. */
function investigationStatusLabel(status: InvestigationStatus): string {
  return statusEnumToString(status).replace(/_/g, ' ');
}

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

/**
 * A series this timeline is willing to raise a threshold crossing for.
 *
 * The thresholds and the unit are declared PER SERIES rather than assumed.
 * Previously every series was compared against 80/95 and formatted with a
 * trailing `%`, which was correct for the three percentage series and wrong
 * for network: its values are a CONNECTION COUNT, so a host with 558 open
 * connections was reported as "558.0% measured" and — because any count above
 * 95 cleared the critical bar — the timeline raised a permanent false CRITICAL
 * that never cleared. Comparing a count against a percentage bar is a unit
 * error, and the fix is to make the unit explicit rather than to pick a
 * kinder number.
 *
 * Network has no entry here ON PURPOSE. There is no authored attention
 * threshold for a connection count anywhere in this app — `MetricCard`'s
 * `defaultThresholds` covers cpu, memory, disk and gpu and deliberately omits
 * network. Inventing one here so the series "has" a bar would fabricate a
 * judgement nobody made. A series with no authored threshold raises nothing,
 * and that silence is honest: it means nobody has said what "too many" is.
 */
interface ThresholdSeries {
  label: string;
  points: ChartDataPoint[];
  unit: string;
  warn: number;
  critical: number;
}

function metricEntries(history: MetricHistory | null): TimelineEntry[] {
  if (!history) return [];
  const entries: TimelineEntry[] = [];
  const series: ThresholdSeries[] = [
    { label: 'CPU', points: history.cpu, unit: '%', warn: 80, critical: 95 },
    { label: 'Memory', points: history.memory, unit: '%', warn: 80, critical: 95 },
    { label: 'Disk', points: history.diskUsage ?? [], unit: '%', warn: 80, critical: 95 },
  ];
  for (const { label, points, unit, warn, critical } of series) {
    const point = [...points].reverse().find((candidate) => candidate.value >= warn);
    if (!point) continue;
    entries.push({
      id: `metric-${label}-${point.timestamp}`,
      at: point.timestamp,
      severity: point.value >= critical ? 'critical' : 'warning',
      source: 'metrics',
      title: `${label} crossed the attention threshold`,
      detail: `${point.value.toFixed(1)}${unit} measured in the shared observation window`,
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
      // `investigation.status` is a numeric proto enum. Interpolating it
      // directly rendered the ordinal — the timeline literally read
      // "Investigation 3" — so it goes through the existing converter and is
      // then written as words rather than as a token.
      title: `Investigation ${investigationStatusLabel(investigation.status)}`,
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
        {/* This states a property of the panel, not a state of the plant, so
            it must not wear the badge treatment — a badge here reads as a
            status the operator should react to. It is a caption. */}
        <span className="eyebrow">shared time axis</span>
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
                {/* Severity reads as a written label as well as a coloured rule,
                    so it survives without colour perception. */}
                <span className={`incident-severity incident-severity-${entry.severity}`}>
                  <span className="incident-severity-label">{SEVERITY_LABEL[entry.severity]}</span>
                  <span className="incident-severity-rule" aria-hidden="true" />
                </span>
                <span className="incident-entry-body">
                  <time className="incident-entry-time" dateTime={entry.at}>{new Date(entry.at).toLocaleString()}</time>
                  <span className="incident-entry-title">{entry.title}</span>
                  {/* The row gets a flattened one-liner; the structured render
                      of the report lives in the correlated view below. */}
                  <span className="incident-entry-summary">{reportToPlainText(entry.detail)}</span>
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
          <ReportBody text={selected.detail} className="report-body-scroll" />
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
