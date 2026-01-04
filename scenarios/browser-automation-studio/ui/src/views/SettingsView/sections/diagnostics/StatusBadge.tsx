/**
 * StatusBadge Component
 *
 * Displays a status indicator badge with icon and label.
 */

import { STATUS_COLORS, type StatusColorKey } from './utils';

interface StatusBadgeProps {
  status: StatusColorKey;
  label?: string;
}

export function StatusBadge({ status, label }: StatusBadgeProps) {
  const config = STATUS_COLORS[status] ?? STATUS_COLORS.error;
  const Icon = config.icon;

  return (
    <span className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium ${config.bg} ${config.text}`}>
      <Icon size={12} className={status === 'loading' ? 'animate-spin' : ''} />
      {label || status.charAt(0).toUpperCase() + status.slice(1)}
    </span>
  );
}

export default StatusBadge;
