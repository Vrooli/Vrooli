/**
 * SkillTreeControls - UI overlay for skill tree controls.
 * Includes zoom controls, reset, and help text.
 */

import { ZoomIn, ZoomOut, Maximize2, HelpCircle } from 'lucide-react'
import { Button } from '@/components/ui/button'

interface SkillTreeControlsProps {
  onZoomIn: () => void
  onZoomOut: () => void
  onReset: () => void
  nodeCount: number
  selectionCount: number
}

export function SkillTreeControls({
  onZoomIn,
  onZoomOut,
  onReset,
  nodeCount,
  selectionCount,
}: SkillTreeControlsProps) {
  return (
    <>
      {/* Zoom controls - top right */}
      <div className="absolute top-4 right-4 flex flex-col gap-1">
        <Button
          variant="outline"
          size="icon"
          onClick={onZoomIn}
          className="h-8 w-8 bg-slate-800/80 border-slate-700 hover:bg-slate-700"
          title="Zoom in"
        >
          <ZoomIn className="h-4 w-4" />
        </Button>
        <Button
          variant="outline"
          size="icon"
          onClick={onZoomOut}
          className="h-8 w-8 bg-slate-800/80 border-slate-700 hover:bg-slate-700"
          title="Zoom out"
        >
          <ZoomOut className="h-4 w-4" />
        </Button>
        <Button
          variant="outline"
          size="icon"
          onClick={onReset}
          className="h-8 w-8 bg-slate-800/80 border-slate-700 hover:bg-slate-700"
          title="Reset view"
        >
          <Maximize2 className="h-4 w-4" />
        </Button>
      </div>

      {/* Info - top left */}
      <div className="absolute top-4 left-4 flex items-center gap-3">
        <div className="px-3 py-1.5 bg-slate-800/80 border border-slate-700 rounded-md text-xs text-slate-300">
          {nodeCount} prompts
        </div>
        {selectionCount > 0 && (
          <div className="px-3 py-1.5 bg-amber-500/20 border border-amber-500/30 rounded-md text-xs text-amber-300">
            {selectionCount} selected
          </div>
        )}
      </div>

      {/* Help text - bottom right */}
      <div className="absolute bottom-4 right-4 flex items-center gap-2 px-3 py-2 bg-slate-800/80 border border-slate-700 rounded-md">
        <HelpCircle className="h-3.5 w-3.5 text-slate-500" />
        <span className="text-xs text-slate-400">
          Click to select • Cmd/Ctrl+Click for multi-select • Drag to orbit
        </span>
      </div>
    </>
  )
}
