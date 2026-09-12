import type { CPUMetrics, MetricValue, ProcessInfo } from '../../../../types';
import { formatOptionalNumber } from '../../../../shared/utils/formatters';

interface CpuExpansionProps {
  details: CPUMetrics;
}

export const CpuExpansion = ({ details }: CpuExpansionProps) => (
  <div className="metric-details" data-sm-style="sm-style-323fdcc1e0">
    {(details.topProcesses?.length ?? 0) > 0 && (
      <div className="detail-section" data-sm-style="sm-style-91394348ef">
        <h4 data-sm-style="sm-style-c8a6493830">
          Top Processes by CPU:
        </h4>
        <div className="process-list">
          {(details.topProcesses ?? []).slice(0, 5).map((process: ProcessInfo) => (
            <div key={process.pid} data-sm-style="sm-style-d820a0b3ab">
              <span>{process.name} ({process.pid})</span>
              <span data-sm-style="sm-style-392c7463c7">
                {formatOptionalNumber(process.cpuPercent)}%
              </span>
            </div>
          ))}
        </div>
      </div>
    )}

    <div className="metric-grid-2col" data-sm-style="sm-style-c08663b577">
      <div className="detail-item">
        <span className="detail-label">Utilization:</span>
        <span className="detail-value">{metricLabel(details.usageState)}</span>
      </div>
      <div className="detail-item">
        <span className="detail-label">Steal:</span>
        <span className="detail-value">{metricLabel(details.modeBreakdown?.steal, '%')}</span>
      </div>
      <div className="detail-item">
        <span className="detail-label">Run Queue:</span>
        <span className="detail-value">{metricLabel(details.runQueueDepth, ' processes')}</span>
      </div>
      <div className="detail-item">
        <span className="detail-label">CPU Stall:</span>
        <span className="detail-value">{metricLabel(details.cpuPsiSomeAvg10, '% some')}</span>
      </div>
      <div className="detail-item">
        <span className="detail-label">Core imbalance:</span>
        <span className="detail-value">{metricLabel(details.coreImbalanceIndex, ' pp')}</span>
      </div>
    </div>

  </div>
);

const metricLabel = (metric?: MetricValue, suffix = ''): string => {
  if (!metric?.state?.case) return 'not yet sampled';
  if (metric.state.case === 'measured') return `${metric.state.value.toFixed(1)}${suffix}`;
  if (metric.state.case === 'notYetSampledReason') return `waiting for next sample: ${metric.state.value}`;
  return `${metric.state.case}: ${String(metric.state.value || 'no value')}${metric.provenance ? ` (${metric.provenance})` : ''}`;
};
