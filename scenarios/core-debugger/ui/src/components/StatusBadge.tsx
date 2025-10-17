import { AlertTriangle, CheckCircle2, OctagonX } from 'lucide-react';
import type { HealthStatus } from '../types';

interface StatusBadgeProps {
  status: HealthStatus;
}

const STATUS_META: Record<HealthStatus, { label: string; tone: 'healthy' | 'degraded' | 'critical'; Icon: typeof CheckCircle2 }> = {
  healthy: {
    label: 'All Systems Operational',
    tone: 'healthy',
    Icon: CheckCircle2
  },
  degraded: {
    label: 'Partial Degradation',
    tone: 'degraded',
    Icon: AlertTriangle
  },
  critical: {
    label: 'Major Outage',
    tone: 'critical',
    Icon: OctagonX
  }
};

export function StatusBadge({ status }: StatusBadgeProps) {
  const meta = STATUS_META[status] ?? STATUS_META.healthy;
  const { Icon } = meta;

  return (
    <div className={`status-badge ${meta.tone}`} role="status" aria-live="polite">
      <Icon size={18} strokeWidth={2.5} />
      <span>{meta.label}</span>
    </div>
  );
}
