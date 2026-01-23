/**
 * WorldControls - UI overlay for world controls.
 * Includes camera mode toggle and help button.
 */

import { useState } from 'react'
import { Camera, HelpCircle, Eye, Map, User } from 'lucide-react'
import { Button } from '@/components/ui/button'
import type { CameraMode } from '@/stores/cameraStore'
import { HelpModal } from './HelpModal'

interface WorldControlsProps {
  cameraMode: CameraMode
  onCycleCameraMode: () => void
  nodeCount: number
  selectionCount: number
  memberCount: number
}

const CAMERA_MODE_ICONS: Record<CameraMode, React.ReactNode> = {
  'zoomed-member': <User className="h-4 w-4" />,
  freeform: <Eye className="h-4 w-4" />,
  'top-down': <Map className="h-4 w-4" />,
}

const CAMERA_MODE_LABELS: Record<CameraMode, string> = {
  'zoomed-member': 'Focus on Member',
  freeform: 'Default View',
  'top-down': 'Aerial View',
}

export function WorldControls({
  cameraMode,
  onCycleCameraMode,
  nodeCount,
  selectionCount,
  memberCount,
}: WorldControlsProps) {
  const [isHelpOpen, setIsHelpOpen] = useState(false)

  return (
    <>
      {/* Camera controls - top right */}
      <div className="absolute top-4 right-4 flex flex-col gap-1">
        <Button
          variant="outline"
          size="icon"
          onClick={onCycleCameraMode}
          className="h-8 w-8 bg-slate-800/80 border-slate-700 hover:bg-slate-700"
          title={`Camera: ${CAMERA_MODE_LABELS[cameraMode]} (click to cycle)`}
        >
          {CAMERA_MODE_ICONS[cameraMode]}
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
        <div className="px-3 py-1.5 bg-slate-800/80 border border-slate-700 rounded-md text-xs text-slate-300">
          {memberCount} member{memberCount !== 1 ? 's' : ''} • {nodeCount} skills
        </div>
        {selectionCount > 0 && (
          <div className="px-3 py-1.5 bg-amber-500/20 border border-amber-500/30 rounded-md text-xs text-amber-300">
            {selectionCount} selected
          </div>
        )}
      </div>

      {/* Camera mode indicator - bottom right */}
      <div className="absolute bottom-4 right-4 flex items-center gap-2 px-3 py-2 bg-slate-800/80 border border-slate-700 rounded-md">
        <Camera className="h-3.5 w-3.5 text-slate-500" />
        <span className="text-xs text-slate-400">
          {CAMERA_MODE_LABELS[cameraMode]}
        </span>
      </div>

      {/* Help Modal */}
      <HelpModal isOpen={isHelpOpen} onClose={() => setIsHelpOpen(false)} />
    </>
  )
}
