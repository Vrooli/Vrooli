import { Link, useLocation } from 'react-router-dom'
import { ClipboardList, Layers, StickyNote, Home } from 'lucide-react'
import { cn } from '../../lib/utils'

interface NavItem {
  to: string
  label: string
  icon: typeof Home
}

const NAV_ITEMS: NavItem[] = [
  { to: '/', label: 'Catalog', icon: Home },
  { to: '/backlog', label: 'Backlog', icon: StickyNote },
  { to: '/drafts', label: 'Drafts', icon: Layers },
]

export function TopNav() {
  const location = useLocation()

  return (
    <nav className="sticky top-0 z-50 mb-6 rounded-2xl border bg-white/95 px-6 py-3 shadow-sm backdrop-blur-sm">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className="rounded-lg bg-violet-100 p-2 text-violet-600">
            <ClipboardList size={20} strokeWidth={2.5} />
          </span>
          <span className="text-lg font-semibold text-slate-900">PRD Control Tower</span>
        </div>

        <div className="flex items-center gap-1">
          {NAV_ITEMS.map((item) => {
            const Icon = item.icon
            const isActive = location.pathname === item.to

            return (
              <Link
                key={item.to}
                to={item.to}
                className={cn(
                  'flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-medium transition-colors',
                  isActive
                    ? 'bg-violet-100 text-violet-700'
                    : 'text-slate-600 hover:bg-slate-100 hover:text-slate-900'
                )}
              >
                <Icon size={16} />
                {item.label}
              </Link>
            )
          })}
        </div>
      </div>
    </nav>
  )
}
