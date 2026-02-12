/**
 * GraphLegend - Shows node type shapes and health color meanings.
 */

import { cn } from '@/lib/utils'

interface LegendItem {
  label: string
  shape: string
  color: string
}

const NODE_TYPES: LegendItem[] = [
  { label: 'Team', shape: 'rounded-sm', color: 'bg-blue-500/40' },
  { label: 'Agent', shape: 'rounded-full', color: 'bg-emerald-500/40' },
  { label: 'Skill', shape: 'rotate-45 rounded-sm', color: 'bg-violet-500/40' },
  { label: 'CLI', shape: 'rounded-sm', color: 'bg-orange-500/40' },
]

const HEALTH_LEVELS = [
  { label: 'Critical (<30%)', color: 'bg-red-500/60' },
  { label: 'Warning (30-60%)', color: 'bg-yellow-500/60' },
  { label: 'Healthy (>60%)', color: 'bg-green-500/60' },
]

interface GraphLegendProps {
  className?: string
}

export function GraphLegend({ className }: GraphLegendProps) {
  return (
    <div className={cn('p-2 bg-card border border-border rounded-lg text-xs space-y-2', className)}>
      <div>
        <p className="font-medium text-muted-foreground mb-1">Node Types</p>
        <div className="space-y-1">
          {NODE_TYPES.map((item) => (
            <div key={item.label} className="flex items-center gap-2">
              <div className={cn('w-3 h-3 border border-border', item.shape, item.color)} />
              <span>{item.label}</span>
            </div>
          ))}
        </div>
      </div>

      <div className="border-t border-border pt-2">
        <p className="font-medium text-muted-foreground mb-1">Health</p>
        <div className="space-y-1">
          {HEALTH_LEVELS.map((item) => (
            <div key={item.label} className="flex items-center gap-2">
              <div className={cn('w-3 h-3 rounded-sm', item.color)} />
              <span>{item.label}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
