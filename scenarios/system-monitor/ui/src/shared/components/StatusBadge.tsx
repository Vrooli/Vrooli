import type { ReactNode } from 'react';

interface StatusBadgeProps {
  variant: 'success' | 'warning' | 'error' | 'info';
  children: ReactNode;
}

export const StatusBadge = ({ variant, children }: StatusBadgeProps) => (
  <span className={`badge badge-${variant}`}>
    {children}
  </span>
);
