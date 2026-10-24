import { priorityColor, priorityLabel } from '../utils/formatPriority';

interface PriorityBadgeProps {
  priority: number;
}

export const PriorityBadge = ({ priority }: PriorityBadgeProps) => (
  <span
    className="priority-badge"
    style={{
      display: 'inline-block',
      minWidth: '4em',
      textAlign: 'center',
      padding: '0 0.4em',
      fontSize: 'var(--text-xs)',
      fontWeight: 'bold',
      color: priorityColor(priority),
      border: `1px solid ${priorityColor(priority)}`,
      borderRadius: '3px',
    }}
  >
    {priorityLabel(priority)}
  </span>
);
