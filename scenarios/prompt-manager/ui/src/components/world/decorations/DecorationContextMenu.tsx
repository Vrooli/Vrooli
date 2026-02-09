/**
 * DecorationContextMenu - Context menu for decoration interactions.
 * Shows options to move or delete decorations, and light mode controls
 * for light-emitting decorations.
 */

import { useCallback } from 'react'
import { Move, Trash2, X, Sun, Lightbulb, LightbulbOff } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useDecorationStore } from '@/stores/decorationStore'
import { useWorldEditorStore } from '@/stores/worldEditorStore'
import { DECORATION_CONFIGS, type DecorationInstance, type LightMode } from '@/types/decoration'

interface DecorationContextMenuProps {
  decoration: DecorationInstance | null
  onClose: () => void
  className?: string
}

const LIGHT_MODES: Array<{ mode: LightMode; label: string; Icon: typeof Sun }> = [
  { mode: 'auto', label: 'Auto', Icon: Sun },
  { mode: 'on', label: 'On', Icon: Lightbulb },
  { mode: 'off', label: 'Off', Icon: LightbulbOff },
]

/**
 * Context menu for decoration interactions.
 */
export function DecorationContextMenu({
  decoration,
  onClose,
  className,
}: DecorationContextMenuProps) {
  const removeDecoration = useDecorationStore((state) => state.removeDecoration)
  const setLightMode = useDecorationStore((state) => state.setLightMode)
  const setEditMode = useWorldEditorStore((state) => state.setEditMode)
  const selectObject = useWorldEditorStore((state) => state.selectObject)

  // Read the live decoration from the store so light mode changes are reflected immediately
  const liveDecoration = useDecorationStore(
    (state) => state.decorations.find((d) => d.id === decoration?.id) ?? null
  )
  const current = liveDecoration ?? decoration
  const config = current ? DECORATION_CONFIGS[current.type] : null

  const handleMove = useCallback(() => {
    if (!decoration) return
    setEditMode(true)
    selectObject({ id: decoration.id, type: 'decoration' })
    onClose()
  }, [decoration, setEditMode, selectObject, onClose])

  const handleDelete = useCallback(() => {
    if (!decoration) return
    removeDecoration(decoration.id)
    onClose()
  }, [decoration, removeDecoration, onClose])

  const handleSetLightMode = useCallback(
    (mode: LightMode) => {
      if (!decoration) return
      setLightMode(decoration.id, mode)
    },
    [decoration, setLightMode]
  )

  if (!current) {
    return null
  }

  const currentMode = current.lightMode ?? 'auto'

  return (
    <div
      className={`
        w-64 p-3
        bg-slate-800/95 backdrop-blur-sm
        border border-slate-700 rounded-lg
        shadow-xl
        ${className ?? ''}
      `}
    >
      {/* Header */}
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-sm font-medium text-slate-200">
          {config?.displayName ?? 'Decoration'}
        </h3>
        <Button
          variant="ghost"
          size="sm"
          onClick={onClose}
          className="h-6 w-6 p-0 text-slate-400 hover:text-slate-200"
        >
          <X className="h-4 w-4" />
        </Button>
      </div>

      {/* Light Mode Controls - only for light-emitting decorations */}
      {config?.emitsLight && (
        <div className="mb-3">
          <div className="text-xs text-slate-400 mb-1.5">Light Mode</div>
          <div className="flex gap-1">
            {LIGHT_MODES.map(({ mode, label, Icon }) => (
              <button
                key={mode}
                onClick={() => handleSetLightMode(mode)}
                className={`
                  flex-1 flex items-center justify-center gap-1 py-1.5 rounded text-xs
                  transition-colors
                  ${currentMode === mode
                    ? 'bg-amber-500/20 text-amber-300 border border-amber-500/40'
                    : 'bg-slate-700/30 text-slate-400 hover:bg-slate-600/50 hover:text-slate-300 border border-transparent'
                  }
                `}
              >
                <Icon className="h-3 w-3" />
                {label}
              </button>
            ))}
          </div>
          <div className="text-[10px] text-slate-500 mt-1">
            {currentMode === 'auto' && 'Follows day/night cycle'}
            {currentMode === 'on' && 'Always on'}
            {currentMode === 'off' && 'Always off'}
          </div>
        </div>
      )}

      {/* Actions */}
      <div className="flex gap-2">
        <Button
          variant="ghost"
          size="sm"
          onClick={handleMove}
          className="flex-1 h-8 gap-1.5 text-blue-400 hover:text-blue-300 hover:bg-blue-500/20"
        >
          <Move className="h-3.5 w-3.5" />
          <span className="text-xs">Move</span>
        </Button>
        <Button
          variant="ghost"
          size="sm"
          onClick={handleDelete}
          className="flex-1 h-8 gap-1.5 text-red-400 hover:text-red-300 hover:bg-red-500/20"
        >
          <Trash2 className="h-3.5 w-3.5" />
          <span className="text-xs">Delete</span>
        </Button>
      </div>
    </div>
  )
}
