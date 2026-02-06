/**
 * WorldEditorToolbar - Controls for world editing mode.
 * Provides toggle, palette access, and object manipulation tools.
 */

import { useCallback } from 'react'
import {
  Edit3,
  Eye,
  Plus,
  Trash2,
  Undo2,
  Redo2,
  Move,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useWorldEditorStore } from '@/stores/worldEditorStore'
import { useCameraStore } from '@/stores/cameraStore'

interface WorldEditorToolbarProps {
  className?: string
}

/**
 * Toolbar for world editor controls.
 */
export function WorldEditorToolbar({ className }: WorldEditorToolbarProps) {
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
  const actionHistory = useWorldEditorStore((state) => state.actionHistory)
  const redoStack = useWorldEditorStore((state) => state.redoStack)

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

  return (
    <div className={`flex items-center gap-1 p-1.5 bg-slate-800/90 backdrop-blur-sm border border-slate-700 rounded-lg shadow-lg ${className ?? ''}`}>
      {/* Edit Mode Toggle */}
      <Button
        variant="ghost"
        size="sm"
        onClick={handleToggleEditMode}
        className={`h-8 px-2 gap-1.5 ${
          isEditMode
            ? 'bg-indigo-500/30 text-indigo-300 hover:bg-indigo-500/40'
            : 'text-slate-400 hover:text-slate-200'
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
      {isEditMode && <div className="w-px h-5 bg-slate-600 mx-1" />}

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
                : 'text-slate-400 hover:text-slate-200'
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
                : 'text-slate-600'
            }`}
            title="Delete Selected"
          >
            <Trash2 className="h-4 w-4" />
          </Button>

          {/* Divider */}
          <div className="w-px h-5 bg-slate-600 mx-1" />

          {/* Undo */}
          <Button
            variant="ghost"
            size="sm"
            onClick={undo}
            disabled={actionHistory.length === 0}
            className={`h-8 w-8 p-0 ${
              actionHistory.length > 0
                ? 'text-slate-400 hover:text-slate-200'
                : 'text-slate-600'
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
                ? 'text-slate-400 hover:text-slate-200'
                : 'text-slate-600'
            }`}
            title="Redo"
          >
            <Redo2 className="h-4 w-4" />
          </Button>
        </>
      )}
    </div>
  )
}
