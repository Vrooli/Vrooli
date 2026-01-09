import { ChevronRight, Home } from "lucide-react";
import { Fragment, ReactNode } from "react";

export interface BreadcrumbItem {
  /** Label to display */
  label: string;
  /** Click handler - if not provided, item is not clickable */
  onClick?: () => void;
  /** Optional icon to show before label */
  icon?: ReactNode;
  /** If true, this is the current/active item (shown without link styling) */
  current?: boolean;
}

interface BreadcrumbsProps {
  /** Array of breadcrumb items from root to current */
  items: BreadcrumbItem[];
  /** Optional class name for the container */
  className?: string;
  /** Whether to show home icon for first item */
  showHomeIcon?: boolean;
  /** Separator icon between items (defaults to ChevronRight) */
  separator?: ReactNode;
}

/**
 * Breadcrumbs navigation component for showing hierarchy and enabling quick navigation.
 *
 * @example
 * ```tsx
 * <Breadcrumbs
 *   items={[
 *     { label: "Dashboard", onClick: () => navigateToDashboard() },
 *     { label: "My Project", onClick: () => openProject(project) },
 *     { label: "Login Workflow", current: true },
 *   ]}
 * />
 * ```
 */
export function Breadcrumbs({
  items,
  className = "",
  showHomeIcon = true,
  separator,
}: BreadcrumbsProps) {
  if (items.length === 0) return null;

  const separatorElement = separator ?? (
    <ChevronRight size={14} className="text-gray-500 flex-shrink-0" />
  );

  return (
    <nav
      aria-label="Breadcrumb"
      className={`flex items-center gap-1 text-sm ${className}`}
    >
      <ol className="flex items-center gap-1 flex-wrap">
        {items.map((item, index) => {
          const isFirst = index === 0;
          const isLast = index === items.length - 1;
          const isCurrent = item.current || isLast;
          const isClickable = !!item.onClick && !isCurrent;

          return (
            <Fragment key={`${item.label}-${index}`}>
              {!isFirst && (
                <li className="flex items-center" aria-hidden="true">
                  {separatorElement}
                </li>
              )}
              <li className="flex items-center">
                {isClickable ? (
                  <button
                    type="button"
                    onClick={item.onClick}
                    className="flex items-center gap-1.5 px-1.5 py-0.5 text-gray-400 hover:text-surface hover:bg-gray-700/50 rounded transition-colors"
                  >
                    {isFirst && showHomeIcon && !item.icon && (
                      <Home size={14} className="flex-shrink-0" />
                    )}
                    {item.icon && (
                      <span className="flex-shrink-0">{item.icon}</span>
                    )}
                    <span className="truncate max-w-[150px]">{item.label}</span>
                  </button>
                ) : (
                  <span
                    className={`flex items-center gap-1.5 px-1.5 py-0.5 ${
                      isCurrent
                        ? "text-surface font-medium"
                        : "text-gray-400"
                    }`}
                    aria-current={isCurrent ? "page" : undefined}
                  >
                    {isFirst && showHomeIcon && !item.icon && (
                      <Home size={14} className="flex-shrink-0" />
                    )}
                    {item.icon && (
                      <span className="flex-shrink-0">{item.icon}</span>
                    )}
                    <span className="truncate max-w-[200px]">{item.label}</span>
                  </span>
                )}
              </li>
            </Fragment>
          );
        })}
      </ol>
    </nav>
  );
}

export default Breadcrumbs;
