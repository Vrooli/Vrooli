import React from 'react'
import { Maximize2, Minimize2, PenSquare, Network } from 'lucide-react'

interface GraphControlsProps {
  isFullscreen: boolean
  canFullscreen: boolean
  isEditMode: boolean
  isPersisting: boolean
  hasGraphChanges: boolean
  autoLayoutEnabled: boolean
  onToggleFullscreen: () => void
  onToggleEditMode: () => void
  onToggleAutoLayout: () => void
}

/**
 * Graph toolbar controls for fullscreen and edit mode.
 * Pure presentational component with no internal state.
 */
const GraphControls: React.FC<GraphControlsProps> = ({
  isFullscreen,
  canFullscreen,
  isEditMode,
  isPersisting,
  hasGraphChanges,
  autoLayoutEnabled,
  onToggleFullscreen,
  onToggleEditMode,
  onToggleAutoLayout
}) => {
  return (
    <div className="tech-tree-actions">
      <button
        type="button"
        className={`canvas-fullscreen-button${isFullscreen ? ' is-active' : ''}`}
        onClick={onToggleFullscreen}
        aria-pressed={isFullscreen}
        aria-label={
          isFullscreen ? 'Exit full screen for tech tree graph' : 'Enter full screen for tech tree graph'
        }
        disabled={!canFullscreen}
      >
        {isFullscreen ? (
          <Minimize2 className="canvas-fullscreen-button__icon" aria-hidden="true" />
        ) : (
          <Maximize2 className="canvas-fullscreen-button__icon" aria-hidden="true" />
        )}
        <span>{isFullscreen ? 'Exit full screen' : 'Full screen'}</span>
      </button>
      <button
        type="button"
        className={`graph-edit-toggle${autoLayoutEnabled ? ' is-active' : ''}`}
        onClick={onToggleAutoLayout}
        aria-pressed={autoLayoutEnabled}
        aria-label={autoLayoutEnabled ? 'Disable automatic graph layout' : 'Enable automatic graph layout'}
        title={
          autoLayoutEnabled
            ? 'Auto-layout is ON. Click to use manual positions from database.'
            : 'Auto-layout is OFF. Click to automatically arrange nodes based on dependencies.'
        }
        disabled={isEditMode}
      >
        <Network className="graph-edit-toggle__icon" aria-hidden="true" />
        <span>{autoLayoutEnabled ? 'Auto-Layout: ON' : 'Auto-Layout: OFF'}</span>
      </button>
      <button
        type="button"
        className={`graph-edit-toggle${isEditMode ? ' is-active' : ''}`}
        onClick={onToggleEditMode}
        aria-pressed={isEditMode}
        aria-label={isEditMode ? 'Save changes and exit edit mode' : 'Enter graph edit mode'}
        title="Edit mode lets you drag stages or connect them. Exit edit mode to save."
        disabled={isPersisting}
      >
        <PenSquare className="graph-edit-toggle__icon" aria-hidden="true" />
        <span>
          {isPersisting
            ? 'Saving graph...'
            : isEditMode
              ? hasGraphChanges
                ? 'Save & exit edit mode'
                : 'Exit edit mode'
              : 'Enter edit mode'}
        </span>
      </button>
    </div>
  )
}

export default GraphControls
