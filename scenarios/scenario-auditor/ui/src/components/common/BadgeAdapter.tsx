import type { CSSProperties } from 'react';
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1.2.5";

interface BadgeProps {
  children: React.ReactNode;
  variant?: 'default' | 'primary' | 'success' | 'warning' | 'danger' | 'info';
  size?: 'sm' | 'md';
  className?: string;
}

const toneByVariant = {
  default: 'neutral',
  primary: 'info',
  success: 'success',
  warning: 'warning',
  danger: 'danger',
  info: 'info',
} as const;

export function Badge({ children, variant = 'default', size = 'md', className }: BadgeProps) {
  const style: CSSProperties = { fontSize: size === 'sm' ? '12px' : '14px' };
  return <StatusBadge tone={toneByVariant[variant]} className={className} style={style}>{children}</StatusBadge>;
}
