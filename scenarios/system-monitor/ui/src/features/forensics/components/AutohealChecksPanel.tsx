import type { AutohealEnvelope } from '../types';
import { NotProvisionedCard } from './NotProvisionedCard';

interface AutohealChecksPanelProps {
  envelope: AutohealEnvelope;
}

const statusClass = (status: string): string => {
  const s = status.toLowerCase();
  if (s === 'ok' || s === 'pass' || s === 'passed') return 'text-success';
  if (s === 'critical' || s === 'fail' || s === 'failed') return 'text-error';
  if (s === 'warning' || s === 'warn') return 'text-warning';
  return 'text-muted';
};

export const AutohealChecksPanel = ({ envelope }: AutohealChecksPanelProps) => {
  if (!envelope.available) {
    return <NotProvisionedCard title="Autoheal Checks" reason={envelope.reason || 'autoheal offline'} />;
  }
  const checks = envelope.checks ?? [];
  return (
    <div className="card" data-sm-style="sm-style-7b635e08e2">
      <div className="font-bold" data-sm-style="sm-style-b113dc3b73">
        Autoheal Checks (forensics)
      </div>
      {checks.length === 0 ? (
        <div className="text-sm text-muted">No forensics-relevant checks reported.</div>
      ) : (
        <ul data-sm-style="sm-style-0d21d4c312">
          {checks.map((c) => (
            <li
              key={c.checkId}
              data-sm-style="sm-style-980f9b6819"
            >
              <div>
                <div className="text-sm" data-sm-style="sm-style-51316ccfb7">
                  {c.checkId}
                </div>
                {c.message && <div className="text-xs text-muted">{c.message}</div>}
              </div>
              <span
                className={`text-xs font-bold uppercase ${statusClass(c.status)}`}
              >
                {c.status}
              </span>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
};
