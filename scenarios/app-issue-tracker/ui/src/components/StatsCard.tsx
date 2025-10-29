import { LucideIcon } from 'lucide-react';

interface StatsCardProps {
  title: string;
  value: number | string;
  subtitle?: string;
  icon: LucideIcon;
  tone?: 'default' | 'success' | 'warning' | 'danger';
  onClick?: () => void;
}

export function StatsCard({ title, value, subtitle, icon: Icon, tone = 'default', onClick }: StatsCardProps) {
  const Component = onClick ? 'button' : 'div';
  const componentProps = onClick ? { type: 'button' as const, onClick } : {};
  const classNames = [`stats-card`, tone];

  if (onClick) {
    classNames.push('stats-card--interactive');
  }

  return (
    <Component className={classNames.join(' ')} {...componentProps}>
      <span className="stats-icon">
        <Icon size={20} />
      </span>
      <span className="stats-content">
        <span className="stats-title">{title}</span>
        <span className="stats-value">{value}</span>
        {subtitle && <span className="stats-subtitle">{subtitle}</span>}
      </span>
    </Component>
  );
}
