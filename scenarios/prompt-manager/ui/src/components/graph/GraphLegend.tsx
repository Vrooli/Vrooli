/**
 * GraphLegend - Shows node type shapes and health color meanings.
 *
 * Collapsible: click the header to toggle content visibility.
 */

import { useState } from 'react'
import { ChevronDown } from 'lucide-react'
import { cn } from '@/lib/utils'

interface LegendItem {
  label: string
  shape: string
}

const NODE_TYPES: LegendItem[] = [
  { label: 'Team', shape: 'rounded-sm' },
  { label: 'Agent', shape: 'rounded-full' },
  { label: 'Skill', shape: 'rotate-45 rounded-sm' },
  { label: 'CLI', shape: 'clip-hexagon' },
]

const HEALTH_LEVELS = [
  { label: 'Critical (<30%)', classes: 'bg-red-500/20 border-red-400/90' },
  { label: 'Warning (30-60%)', classes: 'bg-yellow-500/20 border-yellow-300/90' },
  { label: 'Healthy (>60%)', classes: 'bg-emerald-500/20 border-emerald-300/80' },
]

const EDGE_TYPES = [
  { label: 'Membership', classes: 'border-muted-foreground' },
  { label: 'CLI Read', classes: 'border-violet-500' },
  { label: 'Bold-listed Reference', classes: 'border-violet-500 border-dashed' },
  { label: 'Path Reference', classes: 'border-purple-500' },
  { label: 'Default Scope', classes: 'border-blue-500 border-dashed' },
  { label: 'Code Usage', classes: 'border-orange-500' },
]

interface GraphLegendProps {
  className?: string
}

export function GraphLegend({ className }: GraphLegendProps) {
  const [isOpen, setIsOpen] = useState(true)

  return (
    <div className={cn('p-2 bg-card border border-border rounded-lg text-xs', className)}>
      <button
        type="button"
        onClick={() => setIsOpen((v) => !v)}
        className="flex items-center gap-1.5 w-full font-medium text-muted-foreground hover:text-foreground transition-colors"
      >
        <ChevronDown className={cn('h-3 w-3 transition-transform', !isOpen && '-rotate-90')} />
        <span>Legend</span>
      </button>

      {isOpen && (
        <div className="mt-2 space-y-2">
          <div>
            <p className="font-medium text-muted-foreground mb-1">Node Types</p>
            <div className="space-y-1">
              {NODE_TYPES.map((item) => (
                <div key={item.label} className="flex items-center gap-2">
                  <div className={cn('w-3 h-3 border border-foreground/70 bg-muted/20', item.shape)} />
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
                  <div className={cn('w-3 h-3 rounded-sm border-2', item.classes)} />
                  <span>{item.label}</span>
                </div>
              ))}
            </div>
          </div>

          <div className="border-t border-border pt-2">
            <p className="font-medium text-muted-foreground mb-1">Edge Types</p>
            <div className="space-y-1">
              {EDGE_TYPES.map((item) => (
                <div key={item.label} className="flex items-center gap-2">
                  <div className={cn('w-5 border-t-2', item.classes)} />
                  <span>{item.label}</span>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
