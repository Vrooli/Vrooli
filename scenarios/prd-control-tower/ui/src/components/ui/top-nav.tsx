import { useEffect, useState } from 'react'
import { Link, useLocation } from 'react-router-dom'
import { ClipboardList, Layers, StickyNote, Home, Compass, ShieldAlert } from 'lucide-react'
import { cn } from '../../lib/utils'
import { fetchQualitySummary } from '../../utils/quality'
import type { QualitySummary } from '../../types'

interface NavItem {
  to: string
  label: string
  icon: typeof Home
}

const NAV_ITEMS: NavItem[] = [
  { to: '/', label: 'Orientation', icon: Compass },
  { to: '/catalog', label: 'Catalog', icon: Home },
  { to: '/backlog', label: 'Backlog', icon: StickyNote },
  { to: '/drafts', label: 'Drafts', icon: Layers },
  { to: '/quality-scanner', label: 'Quality', icon: ShieldAlert },
]

export function TopNav() {
  const location = useLocation()
  const [qualitySummary, setQualitySummary] = useState<QualitySummary | null>(null)

  useEffect(() => {
    let mounted = true
    fetchQualitySummary()
      .then((data) => {
        if (mounted) {
          setQualitySummary(data)
        }
      })
      .catch(() => {
        if (mounted) {
          setQualitySummary(null)
        }
      })
    return () => {
      mounted = false
    }
  }, [])

  const outstandingIssues = qualitySummary?.with_issues ?? 0

  return (
    <nav className="sticky top-0 z-50 mb-6 rounded-2xl border bg-white/95 px-3 sm:px-6 py-3 shadow-sm backdrop-blur-sm">
      <div className="flex items-center justify-between gap-2">
        {/* Show compact logo on mobile, full branding on desktop */}
        <div className="flex items-center gap-2">
          <span className="rounded-lg bg-violet-100 p-2 text-violet-600">
            <ClipboardList size={20} strokeWidth={2.5} />
          </span>
          <span className="hidden sm:inline text-lg font-semibold text-slate-900">PRD Control Tower</span>
        </div>

        <div className="flex items-center gap-1 sm:gap-1.5">
          {NAV_ITEMS.map((item) => {
            const Icon = item.icon
            const isActive = location.pathname === item.to
            const showIndicator = item.to === '/quality-scanner' && outstandingIssues > 0

            return (
              <Link
                key={item.to}
                to={item.to}
                className={cn(
                  'flex flex-col sm:flex-row items-center gap-0.5 sm:gap-2 rounded-lg px-2 sm:px-4 py-1.5 sm:py-2 text-xs sm:text-sm font-medium transition-colors min-w-[3.5rem] sm:min-w-0',
                  isActive
                    ? 'bg-violet-100 text-violet-700'
                    : 'text-slate-600 hover:bg-slate-100 hover:text-slate-900'
                )}
              >
                <Icon size={16} className="shrink-0" />
                <span className="flex items-center gap-1 text-[10px] sm:text-sm leading-tight">
                  {item.label}
                  {showIndicator && (
                    <span
                      className="inline-flex h-2 w-2 sm:h-2.5 sm:w-2.5 rounded-full bg-amber-500"
                      aria-label={`${outstandingIssues} scenarios need review`}
                      title={`${outstandingIssues} scenarios need review`}
                    />
                  )}
                </span>
              </Link>
            )
          })}
        </div>
      </div>
    </nav>
  )
}
