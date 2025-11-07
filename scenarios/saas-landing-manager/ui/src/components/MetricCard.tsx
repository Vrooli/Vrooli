import clsx from 'clsx';

export interface MetricCardProps {
  title: string;
  value: number | string;
  helper?: string;
  accent: 'blue' | 'violet' | 'green' | 'amber';
}

export const MetricCard = ({ title, value, helper, accent }: MetricCardProps) => (
  <article className={clsx('metric-card', accent)}>
    <header>{title}</header>
    <strong>{typeof value === 'number' ? value.toLocaleString() : value}</strong>
    {helper && <p>{helper}</p>}
  </article>
);
