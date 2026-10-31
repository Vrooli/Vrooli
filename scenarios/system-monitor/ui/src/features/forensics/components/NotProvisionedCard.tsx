import { AlertTriangle } from 'lucide-react';

interface NotProvisionedCardProps {
  title: string;
  reason?: string;
}

export const NotProvisionedCard = ({ title, reason }: NotProvisionedCardProps) => (
  <div className="card" data-sm-style="sm-style-7b635e08e2">
    <div className="flex-row-center gap-sm" data-sm-style="sm-style-b113dc3b73">
      <AlertTriangle size={16} className="text-muted" />
      <span className="font-bold">{title}</span>
    </div>
    <div className="text-sm text-muted">
      {reason || 'Not provisioned on this host.'}
    </div>
  </div>
);
