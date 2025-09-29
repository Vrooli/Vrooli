import { Activity, AlertTriangle, CheckCircle2, Clock3 } from 'lucide-react';
import type { ComponentHealth } from '../types';

interface ComponentHealthCardProps {
  data: ComponentHealth;
}

function getStatusTone(status: string) {
  if (status === 'healthy') return 'healthy';
  if (status === 'degraded') return 'degraded';
  return 'critical';
}

export function ComponentHealthCard({ data }: ComponentHealthCardProps) {
  const tone = getStatusTone(data.status);
  const StatusIcon = tone === 'healthy' ? CheckCircle2 : tone === 'degraded' ? AlertTriangle : Activity;
  const lastChecked = data.last_check ? new Date(data.last_check).toLocaleTimeString() : 'no data';
  const responseTime = Number.isFinite(Number(data.response_time_ms))
    ? `${Math.round(Number(data.response_time_ms))} ms`
    : 'n/a';

  return (
    <article className={`component-card ${tone}`}>
      <div className="component-header">
        <div className="component-icon" aria-hidden="true">
          <StatusIcon size={18} strokeWidth={2.4} />
        </div>
        <div>
          <h3>{data.component}</h3>
          <p>Last check · {lastChecked}</p>
        </div>
      </div>
      <div className="component-metrics" role="list">
        <div className="metric" role="listitem">
          <Clock3 size={16} aria-hidden="true" />
          <span>{responseTime}</span>
        </div>
        {typeof data.error_count === 'number' ? (
          <div className="metric" role="listitem">
            <AlertTriangle size={16} aria-hidden="true" />
            <span>{data.error_count} errors (1h)</span>
          </div>
        ) : null}
      </div>
    </article>
  );
}
