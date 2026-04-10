/**
 * VariantPanel - Manage prompt variants for a skill.
 *
 * Displays the control (original) variant at top, followed by custom variants.
 * Provides create and delete operations. Control variant cannot be deleted.
 */

import { useState } from 'react'
import { Plus, Trash2, GitBranch, Shield } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useVariantList, useCreateVariant, useDeleteVariant } from '@/hooks/useVariants'
import { ConfirmDialog } from '@/components/shared/ConfirmDialog'
import { Dialog } from '@/components/shared/Dialog'
import { LoadingSpinner } from '@/components/ui/loading-spinner'

interface VariantPanelProps {
  skillId: string
  /** Current SKILL.md content to pre-fill new variants */
  currentContent?: string
  className?: string
}

export function VariantPanel({ skillId, currentContent, className }: VariantPanelProps) {
  const { data: variants = [], isLoading, isError, error } = useVariantList(skillId)
  const createMutation = useCreateVariant()
  const deleteMutation = useDeleteVariant()

  const [showCreateDialog, setShowCreateDialog] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<{ id: string; name: string } | null>(null)
  const [newName, setNewName] = useState('')
  const [newDescription, setNewDescription] = useState('')

  const handleCreate = () => {
    if (!newName.trim()) return
    createMutation.mutate(
      {
        skillId,
        req: {
          name: newName.trim(),
          description: newDescription.trim() || undefined,
          content: currentContent ?? '',
        },
      },
      {
        onSuccess: () => {
          setShowCreateDialog(false)
          setNewName('')
          setNewDescription('')
        },
      }
    )
  }

  const handleDelete = () => {
    if (!deleteTarget) return
    deleteMutation.mutate(
      { skillId, variantId: deleteTarget.id },
      { onSettled: () => setDeleteTarget(null) }
    )
  }

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
        Failed to load variants: {error.message}
      </div>
    )
  }

  return (
    <div className={cn('flex flex-col gap-1 p-2', className)}>
      {/* Header */}
      <div className="flex items-center justify-between px-2 py-1">
        <h3 className="text-sm font-medium text-muted-foreground">
          Variants ({variants.length + 1})
        </h3>
        <button
          type="button"
          onClick={() => setShowCreateDialog(true)}
          className={cn(
            'flex items-center gap-1 px-2 py-1 text-xs rounded-md',
            'text-primary hover:bg-primary/10 transition-colors'
          )}
        >
          <Plus className="h-3 w-3" />
          New Variant
        </button>
      </div>

      {/* Control (original) - always shown first */}
      <div className="flex items-center gap-3 px-3 py-2 rounded-lg bg-primary/5 border border-primary/10">
        <Shield className="h-4 w-4 text-primary flex-shrink-0" />
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium">Control (Original)</span>
            <span className="text-xs px-1.5 py-0.5 rounded bg-primary/20 text-primary font-medium">
              control
            </span>
          </div>
          <p className="text-xs text-muted-foreground mt-0.5">
            The current SKILL.md content
          </p>
        </div>
      </div>

      {/* Custom variants */}
      {variants.map((v) => (
        <div
          key={v.id}
          className="flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-muted/50 transition-colors"
        >
          <GitBranch className="h-4 w-4 text-muted-foreground flex-shrink-0" />
          <div className="flex-1 min-w-0">
            <div className="text-sm font-medium truncate">{v.name}</div>
            {v.description && (
              <p className="text-xs text-muted-foreground mt-0.5 truncate">
                {v.description}
              </p>
            )}
            <p className="text-xs text-muted-foreground/70 mt-0.5">
              Updated {formatTimestamp(v.updatedAt)}
            </p>
          </div>
          <button
            type="button"
            onClick={() => setDeleteTarget({ id: v.id, name: v.name })}
            disabled={deleteMutation.isPending}
            className={cn(
              'p-1 rounded text-muted-foreground hover:text-destructive hover:bg-destructive/10',
              'transition-colors flex-shrink-0',
              deleteMutation.isPending && 'opacity-50 cursor-not-allowed'
            )}
            title={`Delete variant "${v.name}"`}
          >
            <Trash2 className="h-3.5 w-3.5" />
          </button>
        </div>
      ))}

      {variants.length === 0 && (
        <div className="flex flex-col items-center py-6 text-muted-foreground">
          <GitBranch className="h-6 w-6 mb-2 opacity-50" />
          <p className="text-xs">No custom variants yet</p>
          <p className="text-xs mt-1">Create a variant to start A/B testing</p>
        </div>
      )}

      {/* Create variant dialog */}
      <Dialog
        isOpen={showCreateDialog}
        onClose={() => setShowCreateDialog(false)}
        title="Create Variant"
        maxWidth="max-w-md"
        isLoading={createMutation.isPending}
      >
        <div className="flex flex-col gap-4">
          <div>
            <label htmlFor="variant-name" className="block text-sm font-medium text-slate-300 mb-1">
              Name
            </label>
            <input
              id="variant-name"
              type="text"
              value={newName}
              onChange={(e) => setNewName(e.target.value)}
              placeholder="e.g., Concise v2"
              className={cn(
                'w-full px-3 py-2 text-sm rounded-lg',
                'bg-slate-800 border border-white/10 text-white',
                'placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-primary/50'
              )}
              autoFocus
            />
          </div>
          <div>
            <label htmlFor="variant-desc" className="block text-sm font-medium text-slate-300 mb-1">
              Description (optional)
            </label>
            <input
              id="variant-desc"
              type="text"
              value={newDescription}
              onChange={(e) => setNewDescription(e.target.value)}
              placeholder="What makes this variant different?"
              className={cn(
                'w-full px-3 py-2 text-sm rounded-lg',
                'bg-slate-800 border border-white/10 text-white',
                'placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-primary/50'
              )}
            />
          </div>
          <p className="text-xs text-slate-500">
            The variant will be pre-filled with the current skill content. You can edit it after creation.
          </p>
          <div className="flex gap-3 pt-2">
            <button
              type="button"
              onClick={() => setShowCreateDialog(false)}
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
              disabled={!newName.trim() || createMutation.isPending}
              className={cn(
                'flex-1 px-4 py-2 text-sm font-medium rounded-lg',
                'bg-primary text-primary-foreground hover:bg-primary/90',
                'transition-colors',
                (!newName.trim() || createMutation.isPending) && 'opacity-50 cursor-not-allowed'
              )}
            >
              {createMutation.isPending ? 'Creating...' : 'Create'}
            </button>
          </div>
        </div>
      </Dialog>

      {/* Delete confirmation */}
      <ConfirmDialog
        isOpen={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        onConfirm={handleDelete}
        title="Delete variant?"
        message={`This will permanently delete the variant "${deleteTarget?.name ?? ''}". This action cannot be undone.`}
        confirmLabel="Delete"
        variant="danger"
        isLoading={deleteMutation.isPending}
      />
    </div>
  )
}

function formatTimestamp(ts: string): string {
  try {
    const date = new Date(ts)
    return date.toLocaleDateString(undefined, {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    })
  } catch {
    return ts
  }
}
