import { type ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { ChevronRight, Home } from 'lucide-react'
import { cn } from '../../lib/utils'

export interface BreadcrumbItem {
  label: string
  to?: string
}

export interface BreadcrumbsProps {
  items: BreadcrumbItem[]
  actions?: ReactNode
  className?: string
}

/**
 * Breadcrumbs component - Provides clear navigation context and path
 *
 * Design principles:
 * - Clear visual hierarchy (links → separators → current page)
 * - Home icon for quick return to orientation
 * - Optional action buttons for page-level operations
 * - Mobile-friendly with truncation on small screens
 * - Touch-friendly targets (min-h-[44px])
 */
export function Breadcrumbs({ items, actions, className }: BreadcrumbsProps) {
  return (
    <div className={cn('flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3', className)}>
      <nav className="flex flex-wrap items-center gap-2 text-sm" aria-label="Breadcrumb">
        {/* Home link */}
        <Link
          to="/"
          className="rounded-lg p-1.5 text-slate-500 hover:bg-slate-100 hover:text-slate-700 transition-all min-h-[32px] flex items-center active:scale-95"
          aria-label="Go to orientation page"
        >
          <Home size={16} />
        </Link>

        {items.length > 0 && <ChevronRight size={16} className="text-slate-400" />}

        {items.map((item, index) => {
          const isLast = index === items.length - 1

          return (
            <div key={index} className="flex items-center gap-2">
              {index > 0 && <ChevronRight size={16} className="text-slate-400" />}
              {item.to && !isLast ? (
                <Link
                  to={item.to}
                  className="rounded-lg px-3 py-1.5 text-violet-600 font-medium hover:bg-violet-50 hover:text-violet-700 transition-all duration-200 min-h-[32px] flex items-center active:scale-95"
                >
                  {item.label}
                </Link>
              ) : (
                <span className={cn(
                  'rounded-lg px-3 py-1.5 min-h-[32px] flex items-center',
                  isLast ? 'font-semibold text-slate-900 bg-slate-100' : 'text-slate-600'
                )}>
                  {item.label}
                </span>
              )}
            </div>
          )
        })}
      </nav>

      {actions && (
        <div className="flex items-center gap-2">
          {actions}
        </div>
      )}
    </div>
  )
}
