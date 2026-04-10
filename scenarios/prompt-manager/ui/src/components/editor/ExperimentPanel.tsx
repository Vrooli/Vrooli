/**
 * ExperimentPanel - Manage A/B testing experiments for a skill.
 *
 * Displays experiments grouped by status, with controls to start, conclude,
 * and create new experiments. Shows per-variant weight distribution and
 * outcome counts.
 */

import { useState } from 'react'
import { Plus, Play, CheckCircle, FlaskConical, Trophy } from 'lucide-react'
import { cn } from '@/lib/utils'
import {
  useExperimentsBySkill,
  useCreateExperiment,
  useStartExperiment,
  useConcludeExperiment,
} from '@/hooks/useExperiments'
import { useVariantList } from '@/hooks/useVariants'
import { ConfirmDialog } from '@/components/shared/ConfirmDialog'
import { Dialog } from '@/components/shared/Dialog'
import { LoadingSpinner } from '@/components/ui/loading-spinner'
import type { Experiment, ExperimentArmInput } from '@/lib/schemas'

interface ExperimentPanelProps {
  skillId: string
  className?: string
}

/** Status badge colors */
const STATUS_STYLES: Record<string, string> = {
  draft: 'bg-slate-500/20 text-slate-300',
  running: 'bg-green-500/20 text-green-300',
  concluded: 'bg-blue-500/20 text-blue-300',
}

/** Format a timestamp for display. */
function formatTimestamp(ts: string | null | undefined): string {
  if (!ts) return ''
  try {
    return new Date(ts).toLocaleDateString(undefined, {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    })
  } catch {
    return ts
  }
}

export function ExperimentPanel({ skillId, className }: ExperimentPanelProps) {
  const { data: experiments = [], isLoading, isError, error } = useExperimentsBySkill(skillId)
  const { data: variants = [] } = useVariantList(skillId)
  const createMutation = useCreateExperiment()
  const startMutation = useStartExperiment()
  const concludeMutation = useConcludeExperiment()

  const [showCreateDialog, setShowCreateDialog] = useState(false)
  const [startTarget, setStartTarget] = useState<Experiment | null>(null)
  const [concludeTarget, setConcludeTarget] = useState<Experiment | null>(null)
  const [concludeWinner, setConcludeWinner] = useState('')
  const [concludeNotes, setConcludeNotes] = useState('')

  // Create form state
  const [newName, setNewName] = useState('')
  const [newHypothesis, setNewHypothesis] = useState('')
  const [armWeights, setArmWeights] = useState<Record<string, number>>({})

  // Available variant IDs for arms: control + custom variants
  const availableVariants = [
    { id: 'control', name: 'Control (Original)' },
    ...variants.map((v) => ({ id: v.id, name: v.name })),
  ]

  const resetCreateForm = () => {
    setNewName('')
    setNewHypothesis('')
    setArmWeights({})
  }

  const handleOpenCreate = () => {
    // Pre-populate with equal weights for control + all variants
    const initial: Record<string, number> = {}
    const count = availableVariants.length
    for (const v of availableVariants) {
      initial[v.id] = Math.round((1 / count) * 100) / 100
    }
    setArmWeights(initial)
    setShowCreateDialog(true)
  }

  const handleCreate = () => {
    if (!newName.trim()) return
    const arms: ExperimentArmInput[] = Object.entries(armWeights)
      .filter(([, w]) => w > 0)
      .map(([variantId, weight]) => ({ variantId, weight }))

    if (arms.length < 2) return

    createMutation.mutate(
      {
        skillId,
        name: newName.trim(),
        hypothesis: newHypothesis.trim() || undefined,
        arms,
      },
      {
        onSuccess: () => {
          setShowCreateDialog(false)
          resetCreateForm()
        },
      }
    )
  }

  const handleStart = () => {
    if (!startTarget) return
    startMutation.mutate(
      { experimentId: startTarget.id, skillId },
      { onSettled: () => setStartTarget(null) }
    )
  }

  const handleConclude = () => {
    if (!concludeTarget || !concludeWinner) return
    concludeMutation.mutate(
      {
        experimentId: concludeTarget.id,
        skillId,
        req: {
          winnerVariantId: concludeWinner,
          notes: concludeNotes.trim() || undefined,
        },
      },
      {
        onSettled: () => {
          setConcludeTarget(null)
          setConcludeWinner('')
          setConcludeNotes('')
        },
      }
    )
  }

  const weightSum = Object.values(armWeights).reduce((a, b) => a + b, 0)
  const isWeightValid = Math.abs(weightSum - 1) < 0.02

  if (isLoading) {
    return (
      <div className={cn('flex items-center justify-center py-12', className)}>
        <LoadingSpinner size="md" />
      </div>
    )
  }

  if (isError) {
    return (
      <div className={cn('p-4 text-sm text-destructive', className)}>
        Failed to load experiments: {error.message}
      </div>
    )
  }

  return (
    <div className={cn('flex flex-col gap-1 p-2', className)}>
      {/* Header */}
      <div className="flex items-center justify-between px-2 py-1">
        <h3 className="text-sm font-medium text-muted-foreground">
          Experiments ({experiments.length})
        </h3>
        <button
          type="button"
          onClick={handleOpenCreate}
          disabled={availableVariants.length < 2}
          className={cn(
            'flex items-center gap-1 px-2 py-1 text-xs rounded-md',
            'text-primary hover:bg-primary/10 transition-colors',
            availableVariants.length < 2 && 'opacity-50 cursor-not-allowed'
          )}
          title={availableVariants.length < 2 ? 'Create at least one variant first' : 'Create experiment'}
        >
          <Plus className="h-3 w-3" />
          New Experiment
        </button>
      </div>

      {/* Experiment list */}
      {experiments.map((exp) => (
        <ExperimentCard
          key={exp.id}
          experiment={exp}
          onStart={() => setStartTarget(exp)}
          onConclude={() => {
            setConcludeTarget(exp)
            setConcludeWinner('')
            setConcludeNotes('')
          }}
        />
      ))}

      {experiments.length === 0 && (
        <div className="flex flex-col items-center py-6 text-muted-foreground">
          <FlaskConical className="h-6 w-6 mb-2 opacity-50" />
          <p className="text-xs">No experiments yet</p>
          <p className="text-xs mt-1">Create an experiment to compare variants</p>
        </div>
      )}

      {/* Create experiment dialog */}
      <Dialog
        isOpen={showCreateDialog}
        onClose={() => { setShowCreateDialog(false); resetCreateForm() }}
        title="Create Experiment"
        maxWidth="max-w-md"
        isLoading={createMutation.isPending}
      >
        <div className="flex flex-col gap-4">
          <div>
            <label htmlFor="exp-name" className="block text-sm font-medium text-slate-300 mb-1">
              Name
            </label>
            <input
              id="exp-name"
              type="text"
              value={newName}
              onChange={(e) => setNewName(e.target.value)}
              placeholder="e.g., Conciseness test"
              className={cn(
                'w-full px-3 py-2 text-sm rounded-lg',
                'bg-slate-800 border border-white/10 text-white',
                'placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-primary/50'
              )}
              autoFocus
            />
          </div>

          <div>
            <label htmlFor="exp-hypothesis" className="block text-sm font-medium text-slate-300 mb-1">
              Hypothesis (optional)
            </label>
            <input
              id="exp-hypothesis"
              type="text"
              value={newHypothesis}
              onChange={(e) => setNewHypothesis(e.target.value)}
              placeholder="What do you expect to happen?"
              className={cn(
                'w-full px-3 py-2 text-sm rounded-lg',
                'bg-slate-800 border border-white/10 text-white',
                'placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-primary/50'
              )}
            />
          </div>

          {/* Arm weights */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <label className="block text-sm font-medium text-slate-300">
                Variant Weights
              </label>
              <span className={cn(
                'text-xs',
                isWeightValid ? 'text-green-400' : 'text-amber-400'
              )}>
                Sum: {weightSum.toFixed(2)}
              </span>
            </div>
            <div className="flex flex-col gap-2">
              {availableVariants.map((v) => (
                <div key={v.id} className="flex items-center gap-2">
                  <span className="text-xs text-slate-400 w-32 truncate" title={v.name}>
                    {v.name}
                  </span>
                  <input
                    type="number"
                    min={0}
                    max={1}
                    step={0.05}
                    value={armWeights[v.id] ?? 0}
                    onChange={(e) =>
                      setArmWeights((prev) => ({
                        ...prev,
                        [v.id]: parseFloat(e.target.value) || 0,
                      }))
                    }
                    className={cn(
                      'w-20 px-2 py-1 text-sm rounded',
                      'bg-slate-800 border border-white/10 text-white',
                      'focus:outline-none focus:ring-1 focus:ring-primary/50'
                    )}
                  />
                  {/* Visual weight bar */}
                  <div className="flex-1 h-2 rounded-full bg-slate-700 overflow-hidden">
                    <div
                      className="h-full bg-primary rounded-full transition-all"
                      style={{ width: `${((armWeights[v.id] ?? 0) * 100).toFixed(0)}%` }}
                    />
                  </div>
                </div>
              ))}
            </div>
            {!isWeightValid && (
              <p className="text-xs text-amber-400 mt-1">Weights must sum to 1.0</p>
            )}
          </div>

          <div className="flex gap-3 pt-2">
            <button
              type="button"
              onClick={() => { setShowCreateDialog(false); resetCreateForm() }}
              disabled={createMutation.isPending}
              className={cn(
                'flex-1 px-4 py-2 text-sm font-medium rounded-lg',
                'bg-slate-800 text-slate-300 hover:bg-slate-700',
                'border border-white/10 transition-colors'
              )}
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={handleCreate}
              disabled={!newName.trim() || !isWeightValid || createMutation.isPending}
              className={cn(
                'flex-1 px-4 py-2 text-sm font-medium rounded-lg',
                'bg-primary text-primary-foreground hover:bg-primary/90',
                'transition-colors',
                (!newName.trim() || !isWeightValid || createMutation.isPending) &&
                  'opacity-50 cursor-not-allowed'
              )}
            >
              {createMutation.isPending ? 'Creating...' : 'Create'}
            </button>
          </div>
        </div>
      </Dialog>

      {/* Start confirmation */}
      <ConfirmDialog
        isOpen={startTarget !== null}
        onClose={() => setStartTarget(null)}
        onConfirm={handleStart}
        title="Start experiment?"
        message={`This will activate "${startTarget?.name ?? ''}" and begin routing traffic to variants based on configured weights.`}
        confirmLabel="Start"
        variant="warning"
        isLoading={startMutation.isPending}
      />

      {/* Conclude dialog */}
      <Dialog
        isOpen={concludeTarget !== null}
        onClose={() => { setConcludeTarget(null); setConcludeWinner(''); setConcludeNotes('') }}
        title="Conclude Experiment"
        maxWidth="max-w-md"
        isLoading={concludeMutation.isPending}
      >
        <div className="flex flex-col gap-4">
          <p className="text-sm text-slate-400">
            Select the winning variant. Its content will be promoted to the main SKILL.md.
          </p>
          <div>
            <label htmlFor="conclude-winner" className="block text-sm font-medium text-slate-300 mb-1">
              Winner
            </label>
            <select
              id="conclude-winner"
              value={concludeWinner}
              onChange={(e) => setConcludeWinner(e.target.value)}
              className={cn(
                'w-full px-3 py-2 text-sm rounded-lg',
                'bg-slate-800 border border-white/10 text-white',
                'focus:outline-none focus:ring-2 focus:ring-primary/50'
              )}
            >
              <option value="">Select a winner...</option>
              {concludeTarget?.arms.map((arm) => (
                <option key={arm.variantId} value={arm.variantId}>
                  {arm.variantName || arm.variantId}
                  {' '}({concludeTarget.outcomeCounts[arm.variantId] ?? 0} outcomes)
                </option>
              ))}
            </select>
          </div>
          <div>
            <label htmlFor="conclude-notes" className="block text-sm font-medium text-slate-300 mb-1">
              Notes (optional)
            </label>
            <textarea
              id="conclude-notes"
              value={concludeNotes}
              onChange={(e) => setConcludeNotes(e.target.value)}
              placeholder="Why was this variant chosen?"
              rows={3}
              className={cn(
                'w-full px-3 py-2 text-sm rounded-lg',
                'bg-slate-800 border border-white/10 text-white',
                'placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-primary/50'
              )}
            />
          </div>
          <div className="flex gap-3 pt-2">
            <button
              type="button"
              onClick={() => { setConcludeTarget(null); setConcludeWinner(''); setConcludeNotes('') }}
              disabled={concludeMutation.isPending}
              className={cn(
                'flex-1 px-4 py-2 text-sm font-medium rounded-lg',
                'bg-slate-800 text-slate-300 hover:bg-slate-700',
                'border border-white/10 transition-colors'
              )}
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={handleConclude}
              disabled={!concludeWinner || concludeMutation.isPending}
              className={cn(
                'flex-1 px-4 py-2 text-sm font-medium rounded-lg',
                'bg-primary text-primary-foreground hover:bg-primary/90',
                'transition-colors',
                (!concludeWinner || concludeMutation.isPending) && 'opacity-50 cursor-not-allowed'
              )}
            >
              {concludeMutation.isPending ? 'Concluding...' : 'Conclude'}
            </button>
          </div>
        </div>
      </Dialog>
    </div>
  )
}

/** Individual experiment card */
function ExperimentCard({
  experiment,
  onStart,
  onConclude,
}: {
  experiment: Experiment
  onStart: () => void
  onConclude: () => void
}) {
  const totalOutcomes = Object.values(experiment.outcomeCounts).reduce((a, b) => a + b, 0)

  return (
    <div className="px-3 py-2.5 rounded-lg border border-border/50 hover:border-border transition-colors">
      {/* Name + status */}
      <div className="flex items-center justify-between gap-2 mb-1.5">
        <span className="text-sm font-medium truncate">{experiment.name}</span>
        <span className={cn(
          'text-xs px-1.5 py-0.5 rounded font-medium flex-shrink-0',
          STATUS_STYLES[experiment.status] ?? STATUS_STYLES.draft
        )}>
          {experiment.status}
        </span>
      </div>

      {/* Hypothesis */}
      {experiment.hypothesis && (
        <p className="text-xs text-muted-foreground mb-2 line-clamp-2">
          {experiment.hypothesis}
        </p>
      )}

      {/* Arms with weight bars */}
      <div className="flex flex-col gap-1 mb-2">
        {experiment.arms.map((arm) => {
          const count = experiment.outcomeCounts[arm.variantId] ?? 0
          const isWinner = experiment.winnerVariantId === arm.variantId
          return (
            <div key={arm.variantId} className="flex items-center gap-2 text-xs">
              <span className={cn(
                'w-24 truncate',
                isWinner ? 'text-primary font-medium' : 'text-muted-foreground'
              )} title={arm.variantName || arm.variantId}>
                {isWinner && <Trophy className="h-3 w-3 inline mr-1" />}
                {arm.variantName || arm.variantId}
              </span>
              <div className="flex-1 h-1.5 rounded-full bg-muted overflow-hidden">
                <div
                  className={cn(
                    'h-full rounded-full transition-all',
                    isWinner ? 'bg-primary' : 'bg-muted-foreground/40'
                  )}
                  style={{ width: `${(arm.weight * 100).toFixed(0)}%` }}
                />
              </div>
              <span className="text-muted-foreground/70 w-8 text-right">
                {(arm.weight * 100).toFixed(0)}%
              </span>
              {totalOutcomes > 0 && (
                <span className="text-muted-foreground/70 w-6 text-right">{count}</span>
              )}
            </div>
          )
        })}
      </div>

      {/* Timestamps + actions */}
      <div className="flex items-center justify-between text-xs text-muted-foreground/70">
        <span>
          {experiment.status === 'running' && experiment.startedAt
            ? `Started ${formatTimestamp(experiment.startedAt)}`
            : experiment.status === 'concluded' && experiment.concludedAt
              ? `Concluded ${formatTimestamp(experiment.concludedAt)}`
              : `Created ${formatTimestamp(experiment.createdAt)}`}
          {totalOutcomes > 0 && ` | ${totalOutcomes} outcomes`}
        </span>
        <div className="flex items-center gap-1">
          {experiment.status === 'draft' && (
            <button
              type="button"
              onClick={onStart}
              className={cn(
                'flex items-center gap-1 px-2 py-0.5 rounded text-xs',
                'text-green-400 hover:bg-green-500/10 transition-colors'
              )}
            >
              <Play className="h-3 w-3" />
              Start
            </button>
          )}
          {experiment.status === 'running' && (
            <button
              type="button"
              onClick={onConclude}
              className={cn(
                'flex items-center gap-1 px-2 py-0.5 rounded text-xs',
                'text-blue-400 hover:bg-blue-500/10 transition-colors'
              )}
            >
              <CheckCircle className="h-3 w-3" />
              Conclude
            </button>
          )}
        </div>
      </div>

      {/* Winner notes for concluded experiments */}
      {experiment.status === 'concluded' && experiment.notes && (
        <p className="text-xs text-muted-foreground/70 mt-1.5 italic border-t border-border/30 pt-1.5">
          {experiment.notes}
        </p>
      )}
    </div>
  )
}
