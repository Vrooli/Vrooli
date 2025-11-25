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
    <nav className={cn('flex flex-wrap items-center gap-2 text-sm', className)} aria-label="Breadcrumb">
      {items.map((item, index) => {
        const isLast = index === items.length - 1

        return (
          <div key={index} className="flex items-center gap-2">
            {index > 0 && <ChevronRight size={16} className="text-slate-400" />}
            {item.to && !isLast ? (
              <Link
                to={item.to}
                className="rounded-lg px-2 py-1 text-violet-600 font-medium hover:bg-violet-50 hover:text-violet-700 transition-all min-h-[32px] flex items-center"
              >
                {item.label}
              </Link>
            ) : (
              <span className={cn(
                'rounded-lg px-2 py-1 min-h-[32px] flex items-center',
                isLast ? 'font-semibold text-slate-900 bg-slate-100' : 'text-slate-600'
              )}>
                {item.label}
              </span>
            )}
          </div>
        )
      })}
    </nav>
  )
}
