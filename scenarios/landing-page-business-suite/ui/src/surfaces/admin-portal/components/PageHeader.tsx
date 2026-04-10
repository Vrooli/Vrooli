import { BookOpen, type LucideIcon } from 'lucide-react';
import type { ReactNode } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { Button } from '../../../shared/ui/button';
import { getAdminPageDocLink } from '../config/adminPages';

export interface PageHeaderProps {
  /** Page title */
  title: string;
  /** Optional description shown below title */
  description?: string;
  /** Lucide icon component */
  icon: LucideIcon;
  /** Tailwind background class for icon container (e.g., 'bg-emerald-500/10') */
  iconBgClass: string;
  /** Tailwind text color class for icon (e.g., 'text-emerald-400') */
  iconColorClass: string;
  /** Optional action buttons rendered in header */
  actions?: ReactNode;
  /** Test ID for the header container */
  testId?: string;
  /**
   * @deprecated No longer needed - all pages now use icon-title styling.
   * Kept temporarily for backwards compatibility during migration.
   */
  variant?: 'icon-title';
}

/**
 * Consistent page header component for admin portal pages.
 *
 * Displays a colored icon alongside the page title and optional description.
 */
export function PageHeader(props: PageHeaderProps) {
  const { title, description, icon: Icon, iconBgClass, iconColorClass, actions, testId } = props;
  const location = useLocation();
  const docLink = getAdminPageDocLink(location.pathname);
  const docAction = docLink ? (
    <Button
      asChild
      variant="ghost"
      size="icon"
      className="text-slate-300 hover:text-white"
      data-testid="page-docs-link"
      aria-label={docLink.label}
      title={docLink.label}
    >
      <Link to={docLink.url}>
        <BookOpen className="h-4 w-4" />
      </Link>
    </Button>
  ) : null;

  return (
    <div className="flex items-center gap-4 mb-8" data-testid={testId}>
      <div className={`p-3 rounded-xl ${iconBgClass}`}>
        <Icon className={`h-8 w-8 ${iconColorClass}`} />
      </div>
      <div className="flex-1">
        <h1 className="text-3xl font-semibold">{title}</h1>
        {description && (
          <p className="text-slate-400 mt-1">{description}</p>
        )}
      </div>
      {(docAction || actions) && (
        <div className="flex flex-wrap items-center gap-2">
          {docAction}
          {actions}
        </div>
      )}
    </div>
  );
}
