import React from 'react'

interface GraphNotificationsProps {
  editError: string | null
  isEditMode: boolean
  hasGraphChanges: boolean
  graphNotice: string | null
}

/**
 * Displays graph-related notifications (errors, edit mode status, general notices).
 * Pure presentational component for status messages.
 */
const GraphNotifications: React.FC<GraphNotificationsProps> = ({
  editError,
  isEditMode,
  hasGraphChanges,
  graphNotice
}) => {
  return (
    <>
      {editError ? (
        <div className="graph-notice graph-notice--error" role="alert">
          <span className="graph-notice__bullet" aria-hidden="true">
            •
          </span>
          {editError}
        </div>
      ) : null}

      {isEditMode ? (
        <div className="graph-notice graph-notice--editing" role="status">
          <span className="graph-notice__bullet" aria-hidden="true">
            •
          </span>
          {hasGraphChanges
            ? 'Edit mode active. Unsaved changes will persist when you exit edit mode.'
            : 'Edit mode active. Drag nodes or connect stages; changes save when you exit edit mode.'}
        </div>
      ) : null}

      {!isEditMode && graphNotice ? (
        <div className="graph-notice" role="status">
          <span className="graph-notice__bullet" aria-hidden="true">
            •
          </span>
          {graphNotice}
        </div>
      ) : null}
    </>
  )
}

export default GraphNotifications
