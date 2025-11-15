import React from 'react'
import { Handle, Position } from 'react-flow-renderer'
import { ChevronDown, Layers } from 'lucide-react'

interface SectorGroupNodeData {
  sectorId: string
  sectorName: string
  sectorColor: string
  stageCount: number
  avgProgress: number
  collapsed: boolean
}

interface SectorGroupNodeProps {
  data: SectorGroupNodeData
  id: string
}

/**
 * Custom React Flow node for collapsed sectors.
 * Shows sector summary with expand button to reveal child stages.
 */
const SectorGroupNode: React.FC<SectorGroupNodeProps> = ({ data, id }) => {
  const { sectorName, sectorColor, stageCount, avgProgress } = data

  // Extract sector ID from node ID (format: "sector-group::actual-id")
  const sectorId = id.replace('sector-group::', '')

  const handleExpand = (event: React.MouseEvent) => {
    event.stopPropagation()
    // Dispatch custom event for parent to handle
    const expandEvent = new CustomEvent('expand-sector', {
      detail: { sectorId },
      bubbles: true
    })
    event.currentTarget.dispatchEvent(expandEvent)
  }

  return (
    <div className="sector-group-node" role="button" tabIndex={0}>
      <Handle type="target" position={Position.Left} style={{ opacity: 0 }} />
      <Handle type="source" position={Position.Right} style={{ opacity: 0 }} />

      <div className="sector-group-node__header">
        <div className="sector-group-node__icon-wrapper">
          <Layers
            className="sector-group-node__icon"
            style={{ color: sectorColor || '#3B82F6' }}
            size={24}
          />
        </div>
        <h3 className="sector-group-node__title" title={sectorName}>
          {sectorName}
        </h3>
      </div>

      <div className="sector-group-node__stats">
        <div className="sector-group-node__stat">
          <span className="sector-group-node__stat-label">Stages:</span>
          <span className="sector-group-node__stat-value">{stageCount}</span>
        </div>
        <div className="sector-group-node__stat">
          <span className="sector-group-node__stat-label">Progress:</span>
          <span className="sector-group-node__stat-value">{Math.round(avgProgress)}%</span>
        </div>
      </div>

      <div className="sector-group-node__progress-bar">
        <div
          className="sector-group-node__progress-fill"
          style={{
            width: `${avgProgress}%`,
            backgroundColor: sectorColor || '#3B82F6'
          }}
        />
      </div>

      <button
        type="button"
        className="sector-group-node__expand-button"
        onClick={handleExpand}
        aria-label={`Expand ${sectorName} to show stages`}
      >
        <ChevronDown size={16} />
        <span>Expand to show {stageCount} stages</span>
      </button>
    </div>
  )
}

export default SectorGroupNode
