import { type LucideIcon } from 'lucide-react'
import { Button } from './button'
import { cn } from '../../lib/utils'

export interface EmptyStateAction {
  label: string
  onClick?: () => void
  href?: string
  variant?: 'default' | 'secondary' | 'outline'
}

export interface EmptyStateProps {
  icon?: LucideIcon
  title: string
  description: string
  actions?: EmptyStateAction[]
  className?: string
}

/**
 * EmptyState component - Provides clear guidance and CTAs when no data exists
 *
 * Design principles:
 * - Clear title explaining why the view is empty
 * - Helpful description suggesting next steps
 * - Primary CTA(s) to guide user forward
 * - Visual icon to reduce text-heaviness
 * - Generous whitespace to avoid feeling cluttered
 */
export function EmptyState({ icon: Icon, title, description, actions, className }: EmptyStateProps) {
  return (
    <div className={cn(
      'flex flex-col items-center justify-center gap-6 rounded-2xl border-2 border-dashed border-slate-200 bg-slate-50/50 p-12 text-center',
      className
    )}>
      {Icon && (
        <div className="rounded-2xl bg-white p-4 shadow-sm">
          <Icon size={48} className="text-slate-400" strokeWidth={1.5} />
        </div>
      )}

      <div className="max-w-md space-y-2">
        <h3 className="text-lg font-semibold text-slate-900">{title}</h3>
        <p className="text-sm text-slate-600 leading-relaxed">{description}</p>
      </div>

      {actions && actions.length > 0 && (
        <div className="flex flex-wrap items-center justify-center gap-3">
          {actions.map((action, index) => {
            if (action.href) {
              return (
                <Button
                  key={index}
                  variant={action.variant || (index === 0 ? 'default' : 'secondary')}
                  size="lg"
                  asChild
                >
                  <a href={action.href}>{action.label}</a>
                </Button>
              )
            }

            return (
              <Button
                key={index}
                variant={action.variant || (index === 0 ? 'default' : 'secondary')}
                size="lg"
                onClick={action.onClick}
              >
                {action.label}
              </Button>
            )
          })}
        </div>
      )}
    </div>
  )
}
