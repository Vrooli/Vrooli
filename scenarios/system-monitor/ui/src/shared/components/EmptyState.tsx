import { Inbox } from 'lucide-react';

interface EmptyStateProps {
  icon?: React.ElementType;
  message: string;
  description?: string;
}

export const EmptyState = ({ icon: Icon = Inbox, message, description }: EmptyStateProps) => (
  <div className="empty-state">
    <Icon size={32} className="text-muted" />
    <span className="text-sm font-bold text-muted">{message}</span>
    {description && <span className="text-xs text-muted">{description}</span>}
  </div>
);
