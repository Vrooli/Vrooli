/**
 * PriorityIndicator Component
 * Visual indicator for task priority level
 */

import { memo } from 'react';
import { AlertCircle, AlertTriangle, Info, Minus } from 'lucide-react';
import type { Priority } from '../../types/api';

interface PriorityIndicatorProps {
  priority: Priority;
  showLabel?: boolean;
}

export const PriorityIndicator = memo(function PriorityIndicator({ priority, showLabel = false }: PriorityIndicatorProps) {
  const config = {
    critical: {
      icon: AlertCircle,
      color: 'text-red-500',
      bg: 'bg-red-500/10',
      label: 'Critical',
    },
    high: {
      icon: AlertTriangle,
      color: 'text-orange-500',
      bg: 'bg-orange-500/10',
      label: 'High',
    },
    medium: {
      icon: Info,
      color: 'text-yellow-500',
      bg: 'bg-yellow-500/10',
      label: 'Medium',
    },
    low: {
      icon: Minus,
      color: 'text-slate-400',
      bg: 'bg-slate-500/10',
      label: 'Low',
    },
  };

  const { icon: Icon, color, bg, label } = config[priority];

  if (showLabel) {
    return (
      <div className={`flex items-center gap-1.5 px-2 py-1 rounded ${bg}`}>
        <Icon className={`h-3.5 w-3.5 ${color}`} />
        <span className={`text-xs font-medium ${color}`}>{label}</span>
      </div>
    );
  }

  return (
    <div className={`p-1 rounded ${bg}`} title={`Priority: ${label}`}>
      <Icon className={`h-3.5 w-3.5 ${color}`} />
    </div>
  );
});
