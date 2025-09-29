import type { ReactNode } from 'react';

interface StatisticCardProps {
  title: string;
  value: ReactNode;
  caption?: ReactNode;
  tone?: 'default' | 'success' | 'warning' | 'info' | 'danger';
  icon: ReactNode;
}

export function StatisticCard({ title, value, caption, tone = 'default', icon }: StatisticCardProps) {
  return (
    <div className={`stat-card ${tone}`} role="listitem">
      <div className="stat-icon" aria-hidden="true">
        {icon}
      </div>
      <div className="stat-body">
        <div className="stat-title">{title}</div>
        <div className="stat-value">{value}</div>
        {caption ? <div className="stat-caption">{caption}</div> : null}
      </div>
    </div>
  );
}
