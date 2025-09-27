import { LucideIcon } from 'lucide-react';

interface StatsCardProps {
  title: string;
  value: number | string;
  subtitle?: string;
  icon: LucideIcon;
  tone?: 'default' | 'success' | 'warning' | 'danger';
}

export function StatsCard({ title, value, subtitle, icon: Icon, tone = 'default' }: StatsCardProps) {
  return (
    <div className={`stats-card ${tone}`}>
      <div className="stats-icon">
        <Icon size={20} />
      </div>
      <div className="stats-content">
        <p className="stats-title">{title}</p>
        <p className="stats-value">{value}</p>
        {subtitle && <p className="stats-subtitle">{subtitle}</p>}
      </div>
    </div>
  );
}
