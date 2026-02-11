/**
 * WorldControls - UI overlay for world controls.
 * Includes settings button (opening WorldSettingsPopup) and help button.
 */

import { useState, useCallback } from 'react'
import { Settings, HelpCircle, Users, Bot, FileText } from 'lucide-react'
import { Button } from '@/components/ui/button'
import type { CameraMode } from '@/stores/cameraStore'
import { HelpModal } from './HelpModal'
import { WorldSettingsPopup } from './WorldSettingsPopup'
import { selectors } from '@/constants/selectors'

interface WorldControlsProps {
  teamCount: number
  nodeCount: number
  selectionCount: number
  agentCount: number
  /** Callback when camera mode changes via settings popup */
  onCameraModeChange?: (mode: CameraMode, agentId?: string, position?: [number, number, number]) => void
}

export function WorldControls({
  teamCount,
  nodeCount,
  selectionCount,
  agentCount,
  onCameraModeChange,
}: WorldControlsProps) {
  const [isHelpOpen, setIsHelpOpen] = useState(false)
  const [isSettingsOpen, setIsSettingsOpen] = useState(false)

  const handleSettingsClose = useCallback(() => {
    setIsSettingsOpen(false)
  }, [])

  return (
    <>
      {/* Settings and Help buttons - top right */}
      <div className="absolute top-4 right-4 flex flex-col gap-1">
        <Button
          variant="outline"
          size="icon"
          onClick={() => setIsSettingsOpen(true)}
          className="h-8 w-8 bg-slate-800/80 border-slate-700 hover:bg-slate-700"
          title="World Settings"
          data-testid={selectors.settings.button}
        >
          <Settings className="h-4 w-4" />
        </Button>
        <Button
          variant="outline"
          size="icon"
          onClick={() => setIsHelpOpen(true)}
          className="h-8 w-8 bg-slate-800/80 border-slate-700 hover:bg-slate-700"
          title="Help"
        >
          <HelpCircle className="h-4 w-4" />
        </Button>
      </div>

      {/* Info - top left */}
      <div className="absolute top-4 left-4 flex items-center gap-3">
        <div
          className="px-3 py-1.5 bg-slate-800/80 border border-slate-700 rounded-md text-xs text-slate-300"
          data-testid={selectors.world.stats}
        >
          <span className="inline-flex items-center gap-1">{teamCount} <Users className="h-3.5 w-3.5" /></span>
          {' • '}
          <span className="inline-flex items-center gap-1">{agentCount} <Bot className="h-3.5 w-3.5" /></span>
          {' • '}
          <span className="inline-flex items-center gap-1">{nodeCount} <FileText className="h-3.5 w-3.5" /></span>
        </div>
        {selectionCount > 0 && (
          <div className="px-3 py-1.5 bg-amber-500/20 border border-amber-500/30 rounded-md text-xs text-amber-300">
            {selectionCount} selected
          </div>
        )}
      </div>

      {/* Help Modal */}
      <HelpModal isOpen={isHelpOpen} onClose={() => setIsHelpOpen(false)} />

      {/* Settings Popup */}
      <WorldSettingsPopup
        isOpen={isSettingsOpen}
        onClose={handleSettingsClose}
        onCameraModeChange={onCameraModeChange}
      />
    </>
  )
}
