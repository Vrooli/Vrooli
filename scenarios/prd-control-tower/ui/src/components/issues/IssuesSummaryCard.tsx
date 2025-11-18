import { ReactNode } from 'react'
import { AlertCircle, AlertTriangle, Loader2, RefreshCw, ShieldAlert } from 'lucide-react'
import { Link } from 'react-router-dom'
import { Card, CardContent, CardHeader, CardTitle } from '../ui/card'
import { Button } from '../ui/button'
import { Badge } from '../ui/badge'
import { cn } from '../../lib/utils'

export type IssueTone = 'critical' | 'warning' | 'info' | 'muted' | 'success'

const TONE_STYLES: Record<IssueTone, { container: string; title: string; list: string; border: string }> = {
  critical: {
    container: 'border-rose-200 bg-rose-50 text-rose-900',
    title: 'text-rose-900',
    list: 'text-rose-800',
    border: 'border-rose-200',
  },
  warning: {
    container: 'border-amber-200 bg-amber-50 text-amber-900',
    title: 'text-amber-900',
    list: 'text-amber-800',
    border: 'border-amber-200',
  },
  info: {
    container: 'border-slate-200 bg-slate-50 text-slate-900',
    title: 'text-slate-900',
    list: 'text-slate-700',
    border: 'border-slate-200',
  },
  muted: {
    container: 'border-slate-200 bg-white text-slate-800',
    title: 'text-slate-900',
    list: 'text-slate-700',
    border: 'border-slate-200',
  },
  success: {
    container: 'border-emerald-200 bg-emerald-50 text-emerald-900',
    title: 'text-emerald-900',
    list: 'text-emerald-800',
    border: 'border-emerald-200',
  },
}

export interface IssueCategorySection {
  id?: string
  title?: ReactNode
  items: ReactNode[]
  maxVisible?: number
}

export interface IssueCategory {
  id?: string
  title: ReactNode
  icon?: ReactNode
  tone?: IssueTone
  description?: ReactNode
  items?: ReactNode[]
  sections?: IssueCategorySection[]
  maxVisible?: number
  badgeValue?: ReactNode
  footer?: ReactNode
  className?: string
}

export interface IssueFooterAction {
  label: string
  to?: string
  href?: string
  onClick?: () => void
}

interface IssuesSummaryCardProps {
  title?: string
  subtitle?: string
  icon?: ReactNode
  metadata?: ReactNode
  overview?: ReactNode
  loading?: boolean
  error?: string | null
  onRefresh?: () => void
  refreshing?: boolean
  refreshLabel?: string
  statusLabel?: string
  statusTone?: IssueTone
  issueCount?: number
  statusMessage?: ReactNode
  categories?: IssueCategory[]
  emptyState?: ReactNode
  footerActions?: IssueFooterAction[]
  className?: string
}

const DEFAULT_ICON = <ShieldAlert size={18} className="text-violet-600" />

export function IssuesSummaryCard({
  title = 'Issues',
  subtitle,
  icon = DEFAULT_ICON,
  metadata,
  overview,
  loading,
  error,
  onRefresh,
  refreshing,
  refreshLabel = 'Refresh',
  statusLabel,
  statusTone = 'info',
  issueCount,
  statusMessage,
  categories,
  emptyState,
  footerActions,
  className,
}: IssuesSummaryCardProps) {
  const showCategories = (categories?.length ?? 0) > 0
  const tone = TONE_STYLES[statusTone]

  const statusPill = statusLabel ? (
    <div className={`inline-flex items-center gap-2 rounded-full px-3 py-1 text-sm font-medium ${tone.container}`}>
      <span className={tone.title}>{statusLabel}</span>
      {typeof issueCount === 'number' && (
        <Badge variant="secondary" className="bg-white text-slate-700">
          {issueCount} issue{issueCount === 1 ? '' : 's'}
        </Badge>
      )}
    </div>
  ) : null

  return (
    <Card className={cn('mt-6', className)}>
      {(title || subtitle) && (
        <CardHeader className="flex flex-row items-center justify-between space-y-0">
          <div>
            <CardTitle className="flex items-center gap-2 text-xl text-slate-900">
              {icon}
              {title}
            </CardTitle>
            {subtitle && <p className="text-sm text-muted-foreground">{subtitle}</p>}
          </div>
          {(metadata || onRefresh) && (
            <div className="flex items-center gap-2 text-xs text-muted-foreground">
              {metadata}
              {onRefresh && (
                <Button size="sm" variant="outline" onClick={onRefresh} disabled={refreshing} className="gap-2">
                  {refreshing ? <Loader2 size={14} className="animate-spin" /> : <RefreshCw size={14} />}
                  {refreshing ? 'Refreshing' : refreshLabel}
                </Button>
              )}
            </div>
          )}
        </CardHeader>
      )}
      <CardContent className="space-y-4">
        {error && (
          <div className="flex items-start gap-2 rounded-md border border-rose-200 bg-rose-50 p-3 text-sm text-rose-900">
            <AlertCircle size={16} className="mt-0.5" />
            <div>
              <p className="font-medium">Unable to load issues</p>
              <p>{error}</p>
            </div>
          </div>
        )}

        {loading && !showCategories && !error && (
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <Loader2 size={16} className="animate-spin" /> Gathering latest scan...
          </div>
        )}

        {statusPill}

        {overview}

        {statusMessage && (!issueCount || issueCount === 0) && (
          <div className="flex items-center gap-2 rounded-md border border-emerald-200 bg-emerald-50 p-3 text-sm text-emerald-800">
            <AlertTriangle size={16} className="text-emerald-600" />
            {statusMessage}
          </div>
        )}

        {showCategories && (
          <div className={cn('grid gap-4', categories!.length > 1 ? 'md:grid-cols-2' : 'grid-cols-1')}>
            {categories!.map((category, idx) => (
              <IssueCategoryCard key={category.id ?? idx} {...category} />
            ))}
          </div>
        )}

        {!showCategories && !loading && !error && emptyState}

        {footerActions && footerActions.length > 0 && (
          <div className="flex flex-wrap gap-4 text-sm text-muted-foreground">
            {footerActions.map((action) => {
              if (action.to) {
                return (
                  <Link key={action.label} to={action.to} className="text-primary hover:underline">
                    {action.label}
                  </Link>
                )
              }
              if (action.href) {
                return (
                  <a key={action.label} href={action.href} className="text-primary hover:underline" target="_blank" rel="noreferrer">
                    {action.label}
                  </a>
                )
              }
              return (
                <button key={action.label} onClick={action.onClick} className="text-primary hover:underline" type="button">
                  {action.label}
                </button>
              )
            })}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

export function IssueCategoryCard({
  title,
  icon,
  tone = 'info',
  description,
  items,
  sections,
  maxVisible = 4,
  badgeValue,
  footer,
  className,
}: IssueCategory) {
  const toneStyles = TONE_STYLES[tone]
  const lists = sections?.length
    ? sections
    : items
      ? [{ items, maxVisible }]
      : []

  return (
    <div className={cn('space-y-3 rounded-lg border p-3 text-sm', toneStyles.container, className)}>
      <p className={cn('flex items-center gap-2 font-semibold', toneStyles.title)}>
        {icon ?? <AlertTriangle size={16} />}
        {title}
        {badgeValue && <Badge variant="secondary">{badgeValue}</Badge>}
      </p>
      {description && <div className="text-xs text-muted-foreground">{description}</div>}
      {lists.length > 0 && (
        <div className="space-y-3">
          {lists.map((section, idx) => {
            const visible = section.maxVisible ?? maxVisible
            const displayed = section.items.slice(0, visible)
            const hiddenCount = section.items.length - displayed.length
            return (
              <div key={section.id ?? idx} className="space-y-1">
                {section.title && <p className="text-xs font-semibold uppercase tracking-wide">{section.title}</p>}
                <ul className={cn('list-inside list-disc text-xs', toneStyles.list)}>
                  {displayed.map((item, itemIdx) => (
                    <li key={itemIdx}>{item}</li>
                  ))}
                  {hiddenCount > 0 && (
                    <li className="text-xs text-muted-foreground">
                      +{hiddenCount} more...
                    </li>
                  )}
                </ul>
              </div>
            )
          })}
        </div>
      )}
      {footer && <div className="border-t border-dashed pt-2 text-xs text-muted-foreground">{footer}</div>}
    </div>
  )
}
