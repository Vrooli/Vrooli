/**
 * DiscoverControls - Toggle for unified topic+skill discovery and complexity budgeting.
 *
 * Shown only on the Skills tab of the AI search modal.
 * Controls whether search uses the /discover endpoint (topic-aware)
 * and optionally sets a complexity level for content budgeting.
 * Budget tiers are loaded from the API and can be edited inline.
 */

import { useState } from 'react'
import { Settings } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { BudgetConfig } from '@/lib/schemas'

const DEFAULT_TIERS = [
  { value: 'minor', label: 'Minor' },
  { value: 'moderate', label: 'Moderate' },
  { value: 'major', label: 'Major' },
  { value: 'architectural', label: 'Architectural' },
] as const

function formatChars(n: number): string {
  return n >= 1000 ? `${(n / 1000).toFixed(1)}K` : `${n}`
}

function getBudgetForTier(tier: string, config: BudgetConfig | null): number {
  if (!config) return 0
  switch (tier) {
    case 'minor': return config.minor
    case 'moderate': return config.moderate
    case 'major': return config.major
    case 'architectural': return config.architectural
    default: return 0
  }
}

interface DiscoverControlsProps {
  useDiscover: boolean
  onToggleDiscover: (enabled: boolean) => void
  complexity: string | undefined
  onComplexityChange: (complexity: string | undefined) => void
  budgetChars?: number
  budgetStatus?: string
  totalContentChars?: number
  selectedContentChars?: number
  budgetConfig: BudgetConfig | null
  onBudgetConfigSave: (config: BudgetConfig) => void
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
  budgetConfig,
  onBudgetConfigSave,
}: DiscoverControlsProps) {
  const showBudget = useDiscover && complexity && budgetChars
  const [showEditor, setShowEditor] = useState(false)
  const [draft, setDraft] = useState<BudgetConfig | null>(null)

  const handleOpenEditor = () => {
    setDraft(budgetConfig ?? { minor: 4000, moderate: 8000, major: 12000, architectural: 18000 })
    setShowEditor(true)
  }

  const handleSave = () => {
    if (!draft) return
    // Validate ascending order and positive values
    if (draft.minor <= 0 || draft.moderate <= 0 || draft.major <= 0 || draft.architectural <= 0) return
    if (draft.minor >= draft.moderate || draft.moderate >= draft.major || draft.major >= draft.architectural) return
    onBudgetConfigSave(draft)
    setShowEditor(false)
  }

  const handleCancel = () => {
    setDraft(null)
    setShowEditor(false)
  }

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
            {DEFAULT_TIERS.map((tier) => {
              const tierBudget = getBudgetForTier(tier.value, budgetConfig)
              return (
                <button
                  key={tier.value}
                  type="button"
                  onClick={() => onComplexityChange(complexity === tier.value ? undefined : tier.value)}
                  className={cn(
                    'px-2 py-0.5 text-[10px] rounded transition-colors',
                    complexity === tier.value
                      ? 'bg-primary text-primary-foreground'
                      : 'bg-muted text-muted-foreground hover:bg-muted/80 hover:text-foreground'
                  )}
                  title={tierBudget > 0 ? `${tier.label} (${formatChars(tierBudget)} chars)` : tier.label}
                >
                  {tier.label}
                </button>
              )
            })}
          </div>
          <button
            type="button"
            onClick={handleOpenEditor}
            className="p-0.5 rounded text-muted-foreground hover:text-foreground transition-colors"
            title="Configure budget tiers"
          >
            <Settings className="h-3 w-3" />
          </button>
        </div>
      )}

      {/* Budget editor */}
      {useDiscover && showEditor && draft && (
        <BudgetEditor
          draft={draft}
          onChange={setDraft}
          onSave={handleSave}
          onCancel={handleCancel}
        />
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

function BudgetEditor({
  draft,
  onChange,
  onSave,
  onCancel,
}: {
  draft: BudgetConfig
  onChange: (config: BudgetConfig) => void
  onSave: () => void
  onCancel: () => void
}) {
  const isValid =
    draft.minor > 0 &&
    draft.moderate > draft.minor &&
    draft.major > draft.moderate &&
    draft.architectural > draft.major

  return (
    <div className="rounded border border-border bg-background p-2 flex flex-col gap-1.5">
      {DEFAULT_TIERS.map((tier) => (
        <div key={tier.value} className="flex items-center gap-2">
          <span className="text-[10px] text-muted-foreground w-20">{tier.label}</span>
          <input
            type="number"
            value={draft[tier.value as keyof BudgetConfig]}
            onChange={(e) => onChange({ ...draft, [tier.value]: Number(e.target.value) || 0 })}
            className="h-6 w-24 px-1.5 text-xs rounded border border-border bg-background text-foreground focus:outline-none focus:ring-1 focus:ring-primary"
            min={1}
            step={1000}
          />
          <span className="text-[10px] text-muted-foreground">chars</span>
        </div>
      ))}
      <div className="flex items-center gap-1.5 mt-1">
        <button
          type="button"
          onClick={onSave}
          disabled={!isValid}
          className={cn(
            'px-2 py-0.5 text-[10px] rounded transition-colors',
            isValid
              ? 'bg-primary text-primary-foreground hover:bg-primary/90'
              : 'bg-muted text-muted-foreground cursor-not-allowed'
          )}
        >
          Save
        </button>
        <button
          type="button"
          onClick={onCancel}
          className="px-2 py-0.5 text-[10px] rounded bg-muted text-muted-foreground hover:text-foreground transition-colors"
        >
          Cancel
        </button>
        {!isValid && (
          <span className="text-[10px] text-red-400">Values must be ascending</span>
        )}
      </div>
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
