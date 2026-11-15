/**
 * NewFolderDialog - Modal for creating new folder path.
 *
 * Reuses existing CategoryPathEditor component
 * Confirm/Cancel buttons
 */

import { useState, useCallback } from 'react'
import { OverlayActionBar, type OverlayActionList } from '@vrooli/react-component-library/OverlayActionBar/1.0.1'
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
  const actions = [
    { id: 'create-folder', label: 'Move Here', onSelect: handleConfirm, disabled: !isValid, primary: true },
    { id: 'cancel', label: 'Cancel', onSelect: handleClose },
  ] satisfies OverlayActionList

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
      <OverlayActionBar actions={actions} className="prompt-overlay-actions--secondary-left" />
    </Dialog>
  )
}
