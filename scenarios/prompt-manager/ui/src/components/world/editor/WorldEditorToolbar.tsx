/**
 * WorldEditorToolbar - Controls for world editing mode.
 * Provides toggle, palette access, object manipulation tools, and scene reset.
 */

import { useCallback, useState, useRef, useEffect } from 'react'
import {
  Edit3,
  Eye,
  Plus,
  Trash2,
  Undo2,
  Redo2,
  Move,
  RotateCcw,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { ConfirmDialog } from '@/components/shared/ConfirmDialog'
import { useWorldEditorStore } from '@/stores/worldEditorStore'
import { useDecorationStore } from '@/stores/decorationStore'
import { useFurnitureStore } from '@/stores/furnitureStore'
import { useCameraStore } from '@/stores/cameraStore'
import { useAgentData } from '@/hooks/useAgentData'

interface WorldEditorToolbarProps {
  className?: string
}

/**
 * Toolbar for world editor controls.
 */
export function WorldEditorToolbar({ className }: WorldEditorToolbarProps) {
  const { agents } = useAgentData()
  const isEditMode = useWorldEditorStore((state) => state.isEditMode)
  const setEditMode = useWorldEditorStore((state) => state.setEditMode)
  const togglePalette = useWorldEditorStore((state) => state.togglePalette)
  const setTopDown = useCameraStore((state) => state.setTopDown)
  const setFreeform = useCameraStore((state) => state.setFreeform)
  const isPaletteOpen = useWorldEditorStore((state) => state.isPaletteOpen)
  const selectedObject = useWorldEditorStore((state) => state.selectedObject)
  const deleteSelected = useWorldEditorStore((state) => state.deleteSelected)
  const undo = useWorldEditorStore((state) => state.undo)
  const redo = useWorldEditorStore((state) => state.redo)
  const clearHistory = useWorldEditorStore((state) => state.clearHistory)
  const actionHistory = useWorldEditorStore((state) => state.actionHistory)
  const redoStack = useWorldEditorStore((state) => state.redoStack)

  // Reset dropdown state
  const [isResetOpen, setIsResetOpen] = useState(false)
  const resetRef = useRef<HTMLDivElement>(null)

  // Confirm dialog state
  const [confirmAction, setConfirmAction] = useState<'clear' | 'defaults' | null>(null)

  // Close dropdown on outside click
  useEffect(() => {
    if (!isResetOpen) return
    function handleClick(e: MouseEvent) {
      if (resetRef.current && !resetRef.current.contains(e.target as Node)) {
        setIsResetOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClick)
    return () => document.removeEventListener('mousedown', handleClick)
  }, [isResetOpen])

  const handleToggleEditMode = useCallback(() => {
    const entering = !isEditMode
    setEditMode(entering)
    if (entering) {
      setTopDown()
    } else {
      setFreeform()
    }
  }, [isEditMode, setEditMode, setTopDown, setFreeform])

  const handleDelete = useCallback(() => {
    if (selectedObject) {
      deleteSelected()
    }
  }, [selectedObject, deleteSelected])

  const handleConfirmReset = useCallback(() => {
    if (confirmAction === 'clear') {
      useDecorationStore.getState().reset()
      useFurnitureStore.getState().reset()
      clearHistory()
    } else if (confirmAction === 'defaults') {
      const ctx = { numAgents: agents.length }
      useDecorationStore.getState().resetToDefaults(undefined, ctx)
      useFurnitureStore.getState().resetToDefaults(undefined, ctx)
      clearHistory()
    }
    setConfirmAction(null)
  }, [confirmAction, clearHistory, agents.length])

  return (
    <>
      <div className={`flex items-center gap-1 p-1.5 bg-card/90 backdrop-blur-sm border border-border rounded-lg shadow-lg ${className ?? ''}`}>
        {/* Edit Mode Toggle */}
        <Button
          variant="ghost"
          size="sm"
          onClick={handleToggleEditMode}
          className={`h-8 px-2 gap-1.5 ${
            isEditMode
              ? 'bg-indigo-500/30 text-indigo-300 hover:bg-indigo-500/40'
              : 'text-muted-foreground hover:text-foreground'
          }`}
          title={isEditMode ? 'Exit Edit Mode' : 'Enter Edit Mode'}
        >
          {isEditMode ? (
            <>
              <Eye className="h-4 w-4" />
              <span className="text-xs">View</span>
            </>
          ) : (
            <>
              <Edit3 className="h-4 w-4" />
              <span className="text-xs">Edit</span>
            </>
          )}
        </Button>

        {/* Divider */}
        {isEditMode && <div className="w-px h-5 bg-border mx-1" />}

        {/* Edit Mode Tools */}
        {isEditMode && (
          <>
            {/* Add Objects */}
            <Button
              variant="ghost"
              size="sm"
              onClick={togglePalette}
              className={`h-8 w-8 p-0 ${
                isPaletteOpen
                  ? 'bg-green-500/30 text-green-300'
                  : 'text-muted-foreground hover:text-foreground'
              }`}
              title="Add Objects"
            >
              <Plus className="h-4 w-4" />
            </Button>

            {/* Move Tool (visual indicator - dragging is always enabled in edit mode) */}
            <Button
              variant="ghost"
              size="sm"
              className="h-8 w-8 p-0 text-blue-400"
              title="Drag to Move Objects"
              disabled
            >
              <Move className="h-4 w-4" />
            </Button>

            {/* Delete Selected */}
            <Button
              variant="ghost"
              size="sm"
              onClick={handleDelete}
              disabled={!selectedObject}
              className={`h-8 w-8 p-0 ${
                selectedObject
                  ? 'text-red-400 hover:text-red-300 hover:bg-red-500/20'
                  : 'text-muted-foreground/40'
              }`}
              title="Delete Selected"
            >
              <Trash2 className="h-4 w-4" />
            </Button>

            {/* Divider */}
            <div className="w-px h-5 bg-border mx-1" />

            {/* Undo */}
            <Button
              variant="ghost"
              size="sm"
              onClick={undo}
              disabled={actionHistory.length === 0}
              className={`h-8 w-8 p-0 ${
                actionHistory.length > 0
                  ? 'text-muted-foreground hover:text-foreground'
                  : 'text-muted-foreground/40'
              }`}
              title="Undo"
            >
              <Undo2 className="h-4 w-4" />
            </Button>

            {/* Redo */}
            <Button
              variant="ghost"
              size="sm"
              onClick={redo}
              disabled={redoStack.length === 0}
              className={`h-8 w-8 p-0 ${
                redoStack.length > 0
                  ? 'text-muted-foreground hover:text-foreground'
                  : 'text-muted-foreground/40'
              }`}
              title="Redo"
            >
              <Redo2 className="h-4 w-4" />
            </Button>

            {/* Divider */}
            <div className="w-px h-5 bg-border mx-1" />

            {/* Reset dropdown */}
            <div className="relative" ref={resetRef}>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setIsResetOpen((v) => !v)}
                className={`h-8 px-2 gap-1 ${
                  isResetOpen
                    ? 'bg-amber-500/30 text-amber-300'
                    : 'text-muted-foreground hover:text-foreground'
                }`}
                title="Reset Scene"
              >
                <RotateCcw className="h-4 w-4" />
                <span className="text-xs">Reset</span>
              </Button>

              {isResetOpen && (
                <div className="absolute top-full mt-1 right-0 w-44 bg-card border border-border rounded-md shadow-xl z-50 py-1">
                  <button
                    className="w-full text-left px-3 py-1.5 text-sm text-foreground hover:bg-muted"
                    onClick={() => {
                      setIsResetOpen(false)
                      setConfirmAction('clear')
                    }}
                  >
                    Clear Scene
                  </button>
                  <button
                    className="w-full text-left px-3 py-1.5 text-sm text-foreground hover:bg-muted"
                    onClick={() => {
                      setIsResetOpen(false)
                      setConfirmAction('defaults')
                    }}
                  >
                    Reset to Default
                  </button>
                </div>
              )}
            </div>
          </>
        )}
      </div>

      {/* Confirm dialog for reset actions */}
      <ConfirmDialog
        isOpen={confirmAction !== null}
        onClose={() => setConfirmAction(null)}
        onConfirm={handleConfirmReset}
        title={confirmAction === 'clear' ? 'Clear Scene' : 'Reset to Default'}
        message={
          confirmAction === 'clear'
            ? 'Remove all objects from the current scene? This cannot be undone.'
            : 'Replace all objects with the scene defaults? This cannot be undone.'
        }
        confirmLabel={confirmAction === 'clear' ? 'Clear' : 'Reset'}
        variant="warning"
      />
    </>
  )
}
