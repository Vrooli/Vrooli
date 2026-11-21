import { CheckCircle2, AlertCircle, XCircle, Info } from 'lucide-react';

export type StatusBadgeStatus = 'success' | 'warning' | 'error' | 'info';

export interface StatusBadgeProps {
  /** Label text displayed prominently */
  label: string;
  /** Status determines the color scheme and icon */
  status: StatusBadgeStatus;
  /** Optional description shown below the label */
  description?: string;
  /** Test ID for the badge */
  testId?: string;
}

/**
 * Style configurations for each status type.
 */
const STATUS_STYLES: Record<
  StatusBadgeStatus,
  { border: string; bg: string; icon: typeof CheckCircle2; iconColor: string }
> = {
  success: {
    border: 'border-emerald-500/30',
    bg: 'bg-emerald-500/10',
    icon: CheckCircle2,
    iconColor: 'text-emerald-300',
  },
  warning: {
    border: 'border-amber-500/30',
    bg: 'bg-amber-500/10',
    icon: AlertCircle,
    iconColor: 'text-amber-300',
  },
  error: {
    border: 'border-rose-500/30',
    bg: 'bg-rose-500/10',
    icon: XCircle,
    iconColor: 'text-rose-300',
  },
  info: {
    border: 'border-blue-500/30',
    bg: 'bg-blue-500/10',
    icon: Info,
    iconColor: 'text-blue-300',
  },
};

/**
 * StatusBadge - A configurable status indicator badge.
 *
 * Replaces the BrandingHealthBadge and similar patterns across admin pages.
 *
 * @example
 * ```tsx
 * <StatusBadge
 *   label="Site identity"
 *   status="success"
 *   description="Name and logo set"
 * />
 *
 * <StatusBadge
 *   label="SEO defaults"
 *   status="warning"
 *   description="Add page title and description"
 * />
 * ```
 */
export function StatusBadge({ label, status, description, testId }: StatusBadgeProps) {
  const styles = STATUS_STYLES[status];
  const Icon = styles.icon;

  return (
    <div
      className={`flex items-center gap-3 rounded-xl border px-4 py-3 ${styles.border} ${styles.bg}`}
      data-testid={testId}
    >
      <Icon className={`h-5 w-5 ${styles.iconColor}`} />
      <div>
        <p className="text-sm font-semibold text-white">{label}</p>
        {description && <p className="text-xs text-slate-400">{description}</p>}
      </div>
    </div>
  );
}

export interface StatusBadgeGridProps {
  /** Array of badge configurations */
  badges: StatusBadgeProps[];
  /** Number of columns in the grid (2, 3, or 4) */
  columns?: 2 | 3 | 4;
  /** Test ID for the grid container */
  testId?: string;
  /** Additional className for the grid container */
  className?: string;
}

/**
 * Column configurations for the grid.
 */
const COLUMN_STYLES: Record<2 | 3 | 4, string> = {
  2: 'sm:grid-cols-2',
  3: 'sm:grid-cols-2 lg:grid-cols-3',
  4: 'sm:grid-cols-2 lg:grid-cols-4',
};

/**
 * StatusBadgeGrid - A grid layout for multiple status badges.
 *
 * @example
 * ```tsx
 * <StatusBadgeGrid
 *   badges={[
 *     { label: 'Site identity', status: brandingHealth.checks.identity ? 'success' : 'warning', description: '...' },
 *     { label: 'Favicon', status: brandingHealth.checks.favicon ? 'success' : 'warning', description: '...' },
 *     { label: 'SEO defaults', status: brandingHealth.checks.seo ? 'success' : 'warning', description: '...' },
 *     { label: 'Social preview', status: brandingHealth.checks.ogImage ? 'success' : 'warning', description: '...' },
 *   ]}
 *   columns={4}
 *   testId="branding-health"
 * />
 * ```
 */
export function StatusBadgeGrid({ badges, columns = 4, testId, className }: StatusBadgeGridProps) {
  const columnClass = COLUMN_STYLES[columns];

  return (
    <div className={`grid gap-4 ${columnClass} ${className ?? ''}`} data-testid={testId}>
      {badges.map((badge, index) => (
        <StatusBadge
          key={badge.label}
          {...badge}
          testId={badge.testId ?? (testId ? `${testId}-badge-${String(index)}` : undefined)}
        />
      ))}
    </div>
  );
}
