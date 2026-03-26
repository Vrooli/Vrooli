/**
 * DiscoverControls - Toggle for unified topic+skill discovery and complexity budgeting.
 *
 * Shown only on the Skills tab of the AI search modal.
 * Controls whether search uses the /discover endpoint (topic-aware)
 * and optionally sets a complexity level for content budgeting.
 */

import { cn } from '@/lib/utils'

const COMPLEXITY_OPTIONS = [
  { value: 'minor', label: 'Minor', budget: '4K' },
  { value: 'moderate', label: 'Moderate', budget: '8K' },
  { value: 'major', label: 'Major', budget: '12K' },
  { value: 'architectural', label: 'Architectural', budget: '18K' },
] as const

interface DiscoverControlsProps {
  useDiscover: boolean
  onToggleDiscover: (enabled: boolean) => void
  complexity: string | undefined
  onComplexityChange: (complexity: string | undefined) => void
  budgetChars?: number
  budgetStatus?: string
  totalContentChars?: number
  selectedContentChars?: number
}

export function DiscoverControls({
  useDiscover,
  onToggleDiscover,
  complexity,
  onComplexityChange,
  budgetChars,
  budgetStatus,
  totalContentChars,
  selectedContentChars,
}: DiscoverControlsProps) {
  const showBudget = useDiscover && complexity && budgetChars

  return (
    <div className="flex flex-col gap-2">
      {/* Topic context toggle */}
      <div className="flex items-center justify-between">
        <label
          htmlFor="discover-toggle"
          className="text-xs text-muted-foreground cursor-pointer"
        >
          Include topic context
        </label>
        <button
          id="discover-toggle"
          type="button"
          role="switch"
          aria-checked={useDiscover}
          onClick={() => onToggleDiscover(!useDiscover)}
          className={cn(
            'relative inline-flex h-5 w-9 items-center rounded-full transition-colors',
            useDiscover ? 'bg-primary' : 'bg-muted'
          )}
        >
          <span
            className={cn(
              'inline-block h-3.5 w-3.5 rounded-full bg-white transition-transform',
              useDiscover ? 'translate-x-[18px]' : 'translate-x-[3px]'
            )}
          />
        </button>
      </div>

      {/* Complexity selector */}
      {useDiscover && (
        <div className="flex items-center gap-1.5">
          <span className="text-xs text-muted-foreground whitespace-nowrap">Budget:</span>
          <div className="flex items-center gap-1">
            {COMPLEXITY_OPTIONS.map((opt) => (
              <button
                key={opt.value}
                type="button"
                onClick={() => onComplexityChange(complexity === opt.value ? undefined : opt.value)}
                className={cn(
                  'px-2 py-0.5 text-[10px] rounded transition-colors',
                  complexity === opt.value
                    ? 'bg-primary text-primary-foreground'
                    : 'bg-muted text-muted-foreground hover:bg-muted/80 hover:text-foreground'
                )}
                title={`${opt.label} (${opt.budget} chars)`}
              >
                {opt.label}
              </button>
            ))}
          </div>
        </div>
      )}

      {/* Budget gauge */}
      {showBudget && (
        <BudgetGauge
          budgetChars={budgetChars}
          budgetStatus={budgetStatus}
          totalContentChars={totalContentChars}
          selectedContentChars={selectedContentChars}
        />
      )}
    </div>
  )
}

function BudgetGauge({
  budgetChars,
  budgetStatus,
  totalContentChars,
  selectedContentChars,
}: {
  budgetChars: number
  budgetStatus?: string
  totalContentChars?: number
  selectedContentChars?: number
}) {
  const displayChars = selectedContentChars ?? totalContentChars ?? 0
  const ratio = budgetChars > 0 ? Math.min(displayChars / budgetChars, 1.5) : 0
  const percentage = Math.min(ratio * 100, 100)

  const colorClass =
    budgetStatus === 'over' || ratio > 1
      ? 'bg-red-500'
      : ratio > 0.8
        ? 'bg-yellow-500'
        : 'bg-green-500'

  const formatChars = (n: number) => n >= 1000 ? `${(n / 1000).toFixed(1)}K` : `${n}`

  return (
    <div className="flex flex-col gap-1">
      <div className="flex items-center justify-between text-[10px] text-muted-foreground">
        <span>{formatChars(displayChars)} / {formatChars(budgetChars)} chars</span>
        {budgetStatus === 'over' && (
          <span className="text-red-400">over budget</span>
        )}
      </div>
      <div className="h-1.5 w-full bg-muted rounded-full overflow-hidden">
        <div
          className={cn('h-full rounded-full transition-all', colorClass)}
          style={{ width: `${percentage}%` }}
        />
      </div>
    </div>
  )
}
