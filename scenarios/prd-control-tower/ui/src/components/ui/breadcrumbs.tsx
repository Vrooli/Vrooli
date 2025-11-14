import { Link } from 'react-router-dom'
import { ChevronRight } from 'lucide-react'
import { cn } from '../../lib/utils'

export interface BreadcrumbItem {
  label: string
  to?: string
}

export interface BreadcrumbsProps {
  items: BreadcrumbItem[]
  className?: string
}

export function Breadcrumbs({ items, className }: BreadcrumbsProps) {
  return (
    <nav className={cn('flex items-center gap-2 text-sm', className)} aria-label="Breadcrumb">
      {items.map((item, index) => {
        const isLast = index === items.length - 1

        return (
          <div key={index} className="flex items-center gap-2">
            {index > 0 && <ChevronRight size={14} className="text-slate-400" />}
            {item.to && !isLast ? (
              <Link
                to={item.to}
                className="text-slate-600 hover:text-slate-900 hover:underline transition-colors"
              >
                {item.label}
              </Link>
            ) : (
              <span className={cn(isLast ? 'font-medium text-slate-900' : 'text-slate-600')}>
                {item.label}
              </span>
            )}
          </div>
        )
      })}
    </nav>
  )
}
