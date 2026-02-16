/**
 * NewFolderDialog - Modal for creating new folder path.
 *
 * Reuses existing CategoryPathEditor component
 * Confirm/Cancel buttons
 */

import { useState, useCallback } from 'react'
import { cn } from '@/lib/utils'
import { Dialog } from '@/components/shared/Dialog'
import { CategoryPathEditor } from '../editor/CategoryPathEditor'

interface NewFolderDialogProps {
  isOpen: boolean
  onClose: () => void
  onConfirm: (path: string[]) => void
  /** Function to get suggestions at each path level */
  getSuggestionsAtLevel: (level: number, parentPath: string[]) => string[]
  /** Initial path to start with */
  initialPath?: string[]
}

/**
 * New folder dialog component.
 */
export function NewFolderDialog({
  isOpen,
  onClose,
  onConfirm,
  getSuggestionsAtLevel,
  initialPath = [],
}: NewFolderDialogProps) {
  const [path, setPath] = useState<string[]>(initialPath)

  const handleConfirm = useCallback(() => {
    const cleanPath = path.filter(Boolean)
    if (cleanPath.length > 0) {
      onConfirm(cleanPath)
      setPath([])
    }
    onClose()
  }, [path, onConfirm, onClose])

  const handleClose = useCallback(() => {
    setPath([])
    onClose()
  }, [onClose])

  const isValid = path.filter(Boolean).length > 0

  return (
    <Dialog
      isOpen={isOpen}
      onClose={handleClose}
      title="Create New Folder"
      maxWidth="max-w-md"
      className="p-4 bg-card border-border rounded-lg"
      testId="new-folder-dialog"
    >
      {/* Content */}
      <div className="mb-4">
        <p className="text-sm text-muted-foreground mb-3">
          Enter a folder path for the skill. You can create nested folders using multiple levels.
        </p>
        <CategoryPathEditor
          value={path}
          onChange={setPath}
          getSuggestionsAtLevel={getSuggestionsAtLevel}
          label="Folder Path"
          placeholder="e.g., Writing, Development, etc."
        />
      </div>

      {/* Footer */}
      <div className="flex items-center justify-end gap-2">
        <button
          type="button"
          onClick={handleClose}
          className={cn(
            'px-4 py-2 text-sm rounded-lg transition-colors',
            'bg-muted hover:bg-muted/80 text-foreground'
          )}
        >
          Cancel
        </button>
        <button
          type="button"
          onClick={handleConfirm}
          disabled={!isValid}
          className={cn(
            'px-4 py-2 text-sm rounded-lg transition-colors',
            'bg-primary hover:bg-primary/90 text-primary-foreground',
            !isValid && 'opacity-50 cursor-not-allowed'
          )}
        >
          Move Here
        </button>
      </div>
    </Dialog>
  )
}
