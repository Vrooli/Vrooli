import React, { useState } from 'react'
import { Maximize2, Minimize2, PenSquare, Network, Download, Copy, Check } from 'lucide-react'
import { useGraphExport, type ExportFormat } from '../../hooks/useGraphExport'
import type { Sector, StageDependency } from '../../types/techTree'

interface GraphControlsProps {
  isFullscreen: boolean
  canFullscreen: boolean
  isEditMode: boolean
  isPersisting: boolean
  hasGraphChanges: boolean
  autoLayoutEnabled: boolean
  treeId?: string
  sectors?: Sector[]
  dependencies?: StageDependency[]
  onToggleFullscreen: () => void
  onToggleEditMode: () => void
  onToggleAutoLayout: () => void
}

/**
 * Graph toolbar controls for fullscreen, edit mode, and export.
 */
const GraphControls: React.FC<GraphControlsProps> = ({
  isFullscreen,
  canFullscreen,
  isEditMode,
  isPersisting,
  hasGraphChanges,
  autoLayoutEnabled,
  treeId,
  sectors,
  dependencies,
  onToggleFullscreen,
  onToggleEditMode,
  onToggleAutoLayout
}) => {
  const { exportGraph, copyGraphAsText, isExporting } = useGraphExport()
  const [showExportMenu, setShowExportMenu] = useState(false)
  const [copied, setCopied] = useState(false)

  const handleExport = async (format: ExportFormat) => {
    setShowExportMenu(false)
    await exportGraph({ format, treeId, sectors, dependencies })
  }

  const handleCopyText = async () => {
    setShowExportMenu(false)
    if (sectors && dependencies) {
      const success = await copyGraphAsText(sectors, dependencies)
      if (success) {
        setCopied(true)
        setTimeout(() => setCopied(false), 2000)
      }
    }
  }

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
      <div style={{ position: 'relative', display: 'inline-block' }}>
        <button
          type="button"
          className="graph-edit-toggle"
          onClick={() => setShowExportMenu(!showExportMenu)}
          aria-label="Export graph"
          title="Export tech tree graph in various formats"
          disabled={isExporting}
        >
          {copied ? (
            <Check className="graph-edit-toggle__icon" aria-hidden="true" />
          ) : (
            <Download className="graph-edit-toggle__icon" aria-hidden="true" />
          )}
          <span>{isExporting ? 'Exporting...' : copied ? 'Copied!' : 'Export'}</span>
        </button>
        {showExportMenu && (
          <div
            style={{
              position: 'absolute',
              top: '100%',
              right: 0,
              marginTop: '4px',
              backgroundColor: '#1e293b',
              border: '1px solid #334155',
              borderRadius: '4px',
              boxShadow: '0 4px 6px rgba(0, 0, 0, 0.3)',
              zIndex: 1000,
              minWidth: '180px'
            }}
          >
            <button
              type="button"
              onClick={() => handleExport('dot')}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: '8px',
                width: '100%',
                padding: '8px 12px',
                border: 'none',
                background: 'transparent',
                color: 'white',
                cursor: 'pointer',
                fontSize: '14px',
                textAlign: 'left'
              }}
              onMouseEnter={(e) => (e.currentTarget.style.backgroundColor = '#334155')}
              onMouseLeave={(e) => (e.currentTarget.style.backgroundColor = 'transparent')}
            >
              <Download size={16} />
              <span>Download DOT (Graphviz)</span>
            </button>
            <button
              type="button"
              onClick={() => handleExport('json')}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: '8px',
                width: '100%',
                padding: '8px 12px',
                border: 'none',
                background: 'transparent',
                color: 'white',
                cursor: 'pointer',
                fontSize: '14px',
                textAlign: 'left'
              }}
              onMouseEnter={(e) => (e.currentTarget.style.backgroundColor = '#334155')}
              onMouseLeave={(e) => (e.currentTarget.style.backgroundColor = 'transparent')}
            >
              <Download size={16} />
              <span>Download JSON</span>
            </button>
            <button
              type="button"
              onClick={() => handleExport('text')}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: '8px',
                width: '100%',
                padding: '8px 12px',
                border: 'none',
                background: 'transparent',
                color: 'white',
                cursor: 'pointer',
                fontSize: '14px',
                textAlign: 'left'
              }}
              onMouseEnter={(e) => (e.currentTarget.style.backgroundColor = '#334155')}
              onMouseLeave={(e) => (e.currentTarget.style.backgroundColor = 'transparent')}
            >
              <Download size={16} />
              <span>Download Text</span>
            </button>
            <button
              type="button"
              onClick={handleCopyText}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: '8px',
                width: '100%',
                padding: '8px 12px',
                border: 'none',
                borderTop: '1px solid #334155',
                background: 'transparent',
                color: 'white',
                cursor: 'pointer',
                fontSize: '14px',
                textAlign: 'left'
              }}
              onMouseEnter={(e) => (e.currentTarget.style.backgroundColor = '#334155')}
              onMouseLeave={(e) => (e.currentTarget.style.backgroundColor = 'transparent')}
            >
              <Copy size={16} />
              <span>Copy as Text</span>
            </button>
          </div>
        )}
      </div>
    </div>
  )
}

export default GraphControls
