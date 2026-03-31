/**
 * DiscoverControls - Toggle for unified topic+skill discovery and complexity budgeting.
 *
 * Shown only on the Skills tab of the AI search modal.
 * Controls whether search uses the /discover endpoint (topic-aware)
 * and optionally sets a complexity level for content budgeting.
 * Budget tiers and discovery filters are edited in a dialog opened from the gear button.
 */

import { useState } from 'react'
import { Settings } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Dialog } from '@/components/shared/Dialog'
import type { BudgetConfig, DiscoverFilterConfig } from '@/lib/schemas'

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
  totalContentChars?: number
  selectedContentChars?: number
  budgetConfig: BudgetConfig | null
  onBudgetConfigSave: (config: BudgetConfig) => void
  filterConfig: DiscoverFilterConfig | null
  onFilterConfigSave: (config: DiscoverFilterConfig) => void
  availableModes: string[]
  availableTags: string[]
}

export function DiscoverControls({
  useDiscover,
  onToggleDiscover,
  complexity,
  onComplexityChange,
  budgetChars,
  totalContentChars,
  selectedContentChars,
  budgetConfig,
  onBudgetConfigSave,
  filterConfig,
  onFilterConfigSave,
  availableModes,
  availableTags,
}: DiscoverControlsProps) {
  const showBudget = useDiscover && complexity && budgetChars
  const [showSettings, setShowSettings] = useState(false)
  const [budgetDraft, setBudgetDraft] = useState<BudgetConfig | null>(null)
  const [filterDraft, setFilterDraft] = useState<DiscoverFilterConfig | null>(null)

  const handleOpenSettings = () => {
    setBudgetDraft(budgetConfig ?? { minor: 4000, moderate: 8000, major: 12000, architectural: 18000 })
    setFilterDraft(filterConfig ?? { includeDrafts: false, excludeModes: ['scope'], excludeIds: [], excludeTags: [] })
    setShowSettings(true)
  }

  const handleSave = () => {
    if (budgetDraft) {
      if (budgetDraft.minor > 0 && budgetDraft.moderate > budgetDraft.minor &&
          budgetDraft.major > budgetDraft.moderate && budgetDraft.architectural > budgetDraft.major) {
        onBudgetConfigSave(budgetDraft)
      }
    }
    if (filterDraft) {
      onFilterConfigSave(filterDraft)
    }
    setShowSettings(false)
  }

  const handleClose = () => {
    setBudgetDraft(null)
    setFilterDraft(null)
    setShowSettings(false)
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
            onClick={handleOpenSettings}
            className="p-0.5 rounded text-muted-foreground hover:text-foreground transition-colors"
            title="Configure discovery settings"
          >
            <Settings className="h-3 w-3" />
          </button>
        </div>
      )}

      {/* Discovery settings dialog */}
      {budgetDraft && filterDraft && (
        <DiscoverSettingsDialog
          isOpen={showSettings}
          onClose={handleClose}
          budgetDraft={budgetDraft}
          onBudgetChange={setBudgetDraft}
          filterDraft={filterDraft}
          onFilterChange={setFilterDraft}
          availableModes={availableModes}
          availableTags={availableTags}
          onSave={handleSave}
        />
      )}

      {/* Budget gauge */}
      {showBudget && (
        <BudgetGauge
          budgetChars={budgetChars}
          totalContentChars={totalContentChars}
          selectedContentChars={selectedContentChars}
        />
      )}
    </div>
  )
}

function DiscoverSettingsDialog({
  isOpen,
  onClose,
  budgetDraft,
  onBudgetChange,
  filterDraft,
  onFilterChange,
  availableModes,
  availableTags,
  onSave,
}: {
  isOpen: boolean
  onClose: () => void
  budgetDraft: BudgetConfig
  onBudgetChange: (config: BudgetConfig) => void
  filterDraft: DiscoverFilterConfig
  onFilterChange: (config: DiscoverFilterConfig) => void
  availableModes: string[]
  availableTags: string[]
  onSave: () => void
}) {
  const isBudgetValid =
    budgetDraft.minor > 0 &&
    budgetDraft.moderate > budgetDraft.minor &&
    budgetDraft.major > budgetDraft.moderate &&
    budgetDraft.architectural > budgetDraft.major

  const toggleMode = (mode: string) => {
    const modes = filterDraft.excludeModes
    onFilterChange({
      ...filterDraft,
      excludeModes: modes.includes(mode) ? modes.filter((m) => m !== mode) : [...modes, mode],
    })
  }

  const toggleTag = (tag: string) => {
    const tags = filterDraft.excludeTags
    onFilterChange({
      ...filterDraft,
      excludeTags: tags.includes(tag) ? tags.filter((t) => t !== tag) : [...tags, tag],
    })
  }

  return (
    <Dialog isOpen={isOpen} onClose={onClose} title="Discovery Settings" maxWidth="max-w-md">
      <div className="flex flex-col gap-5">
        {/* Budget Tiers */}
        <div className="flex flex-col gap-2">
          <span className="text-xs font-medium text-slate-400 uppercase tracking-wider">Budget Tiers</span>
          {DEFAULT_TIERS.map((tier) => (
            <div key={tier.value} className="flex items-center gap-3">
              <span className="text-xs text-slate-400 w-24">{tier.label}</span>
              <input
                type="number"
                value={budgetDraft[tier.value]}
                onChange={(e) => onBudgetChange({ ...budgetDraft, [tier.value]: Number(e.target.value) || 0 })}
                className="h-7 w-28 px-2 text-xs rounded border border-white/10 bg-slate-800 text-white focus:outline-none focus:ring-1 focus:ring-primary"
                min={1}
                step={1000}
              />
              <span className="text-xs text-slate-500">chars</span>
            </div>
          ))}
          {!isBudgetValid && (
            <span className="text-xs text-red-400">Values must be ascending</span>
          )}
        </div>

        {/* Discovery Filters */}
        <div className="flex flex-col gap-3">
          <span className="text-xs font-medium text-slate-400 uppercase tracking-wider">Discovery Filters</span>

          {/* Include drafts toggle */}
          <div className="flex items-center justify-between">
            <span className="text-xs text-slate-300">Include drafts</span>
            <button
              type="button"
              role="switch"
              aria-checked={filterDraft.includeDrafts}
              onClick={() => onFilterChange({ ...filterDraft, includeDrafts: !filterDraft.includeDrafts })}
              className={cn(
                'relative inline-flex h-5 w-9 items-center rounded-full transition-colors',
                filterDraft.includeDrafts ? 'bg-primary' : 'bg-slate-600'
              )}
            >
              <span
                className={cn(
                  'inline-block h-3.5 w-3.5 rounded-full bg-white transition-transform',
                  filterDraft.includeDrafts ? 'translate-x-[18px]' : 'translate-x-[3px]'
                )}
              />
            </button>
          </div>

          {/* Exclude modes */}
          {availableModes.length > 0 && (
            <div className="flex flex-col gap-1.5">
              <span className="text-xs text-slate-400">Exclude modes</span>
              <div className="flex flex-wrap gap-1.5">
                {availableModes.map((mode) => {
                  const isExcluded = filterDraft.excludeModes.includes(mode)
                  return (
                    <button
                      key={mode}
                      type="button"
                      onClick={() => toggleMode(mode)}
                      className={cn(
                        'px-2 py-1 text-xs rounded transition-colors',
                        isExcluded
                          ? 'bg-red-500/20 text-red-400 border border-red-500/30'
                          : 'bg-slate-700 text-slate-300 hover:bg-slate-600'
                      )}
                    >
                      {mode}
                    </button>
                  )
                })}
              </div>
            </div>
          )}

          {/* Exclude tags */}
          {availableTags.length > 0 && (
            <div className="flex flex-col gap-1.5">
              <span className="text-xs text-slate-400">Exclude tags</span>
              <div className="flex flex-wrap gap-1.5">
                {availableTags.map((tag) => {
                  const isExcluded = filterDraft.excludeTags.includes(tag)
                  return (
                    <button
                      key={tag}
                      type="button"
                      onClick={() => toggleTag(tag)}
                      className={cn(
                        'px-2 py-1 text-xs rounded transition-colors',
                        isExcluded
                          ? 'bg-red-500/20 text-red-400 border border-red-500/30'
                          : 'bg-slate-700 text-slate-300 hover:bg-slate-600'
                      )}
                    >
                      {tag}
                    </button>
                  )
                })}
              </div>
            </div>
          )}

          {/* Exclude IDs */}
          <div className="flex flex-col gap-1.5">
            <span className="text-xs text-slate-400">Exclude skill IDs (comma-separated)</span>
            <input
              type="text"
              value={filterDraft.excludeIds.join(', ')}
              onChange={(e) => {
                const ids = e.target.value
                  .split(',')
                  .map((s) => s.trim())
                  .filter(Boolean)
                onFilterChange({ ...filterDraft, excludeIds: ids })
              }}
              className="h-7 px-2 text-xs rounded border border-white/10 bg-slate-800 text-white focus:outline-none focus:ring-1 focus:ring-primary"
              placeholder="skill-id-1, skill-id-2"
            />
          </div>
        </div>

        {/* Actions */}
        <div className="flex items-center gap-2 pt-2 border-t border-white/10">
          <button
            type="button"
            onClick={onSave}
            className="px-3 py-1.5 text-xs rounded bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
          >
            Save
          </button>
          <button
            type="button"
            onClick={onClose}
            className="px-3 py-1.5 text-xs rounded bg-slate-700 text-slate-300 hover:text-white hover:bg-slate-600 transition-colors"
          >
            Cancel
          </button>
        </div>
      </div>
    </Dialog>
  )
}

function BudgetGauge({
  budgetChars,
  totalContentChars,
  selectedContentChars,
}: {
  budgetChars: number
  totalContentChars?: number
  selectedContentChars?: number
}) {
  const displayChars = selectedContentChars ?? totalContentChars ?? 0
  const ratio = budgetChars > 0 ? Math.min(displayChars / budgetChars, 1.5) : 0
  const percentage = Math.min(ratio * 100, 100)
  const isOver = ratio > 1

  const colorClass = isOver
    ? 'bg-red-500'
    : ratio > 0.8
      ? 'bg-yellow-500'
      : 'bg-green-500'

  return (
    <div className="flex flex-col gap-1">
      <div className="flex items-center justify-between text-[10px] text-muted-foreground">
        <span>{formatChars(displayChars)} / {formatChars(budgetChars)} chars</span>
        {isOver && (
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
