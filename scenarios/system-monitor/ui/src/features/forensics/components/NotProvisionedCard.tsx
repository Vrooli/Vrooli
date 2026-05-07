import { AlertTriangle } from 'lucide-react';

interface NotProvisionedCardProps {
  title: string;
  reason?: string;
}

export const NotProvisionedCard = ({ title, reason }: NotProvisionedCardProps) => (
  <div className="card" style={{ padding: 'var(--spacing-md)' }}>
    <div className="flex-row-center gap-sm" style={{ marginBottom: '0.5rem' }}>
      <AlertTriangle size={16} className="text-muted" />
      <span className="font-bold">{title}</span>
    </div>
    <div className="text-sm text-muted">
      {reason || 'Not provisioned on this host.'}
    </div>
  </div>
);
