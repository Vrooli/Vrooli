import React from 'react'
import { Maximize2, Minimize2, PenSquare } from 'lucide-react'

interface GraphControlsProps {
  isFullscreen: boolean
  canFullscreen: boolean
  isEditMode: boolean
  isPersisting: boolean
  hasGraphChanges: boolean
  onToggleFullscreen: () => void
  onToggleEditMode: () => void
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
  onToggleFullscreen,
  onToggleEditMode
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
