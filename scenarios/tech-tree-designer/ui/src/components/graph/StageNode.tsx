import React, { useCallback, useMemo } from 'react'
import { Handle, Position, type NodeProps } from 'react-flow-renderer'
import { ChevronRight, ChevronDown, Loader2, Link, FolderTree } from 'lucide-react'
import type { StageNodeData } from '../../types/graph'

export interface ExtendedStageNodeData extends StageNodeData {
  stageId: string
  hasChildren?: boolean
  childrenLoaded?: boolean
  isExpanded?: boolean
  isLoading?: boolean
  scenarioCount?: number
  onToggleExpand?: (stageId: string, hasChildren: boolean, childrenLoaded: boolean) => void
}

/**
 * Custom node component for rendering progression stages with hierarchical expansion.
 *
 * Features:
 * - Visual progress indicator
 * - Expand/collapse button for nodes with children
 * - Loading spinner while fetching children
 * - Stage type badge
 * - Scenario count indicator
 * - Connection handles for dependencies
 */
const StageNode: React.FC<NodeProps<ExtendedStageNodeData>> = ({ data, selected }) => {
  const {
    label,
    progress,
    type,
    sectorColor,
    stageId,
    hasChildren,
    childrenLoaded,
    isExpanded,
    isLoading,
    scenarioCount,
    onToggleExpand
  } = data

  const handleExpandClick = useCallback(
    (e: React.MouseEvent) => {
      e.stopPropagation()
      if (onToggleExpand && hasChildren) {
        onToggleExpand(stageId, hasChildren, childrenLoaded || false)
      }
    },
    [onToggleExpand, stageId, hasChildren, childrenLoaded]
  )

  const borderColor = useMemo(() => {
    if (selected) return sectorColor || '#3B82F6'
    return `${sectorColor || 'rgba(59,130,246,0.6)'}`
  }, [selected, sectorColor])

  const progressBarWidth = `${Math.min(100, Math.max(0, progress))}%`

  return (
    <div
      className="stage-node"
      style={{
        background: 'rgba(15, 23, 42, 0.92)',
        color: '#f8fafc',
        border: `${selected ? '2px' : '1px'} solid ${borderColor}`,
        borderRadius: '12px',
        padding: '12px',
        fontSize: '12px',
        boxShadow: selected
          ? `0 0 0 3px ${sectorColor}33, 0 12px 24px rgba(2, 6, 23, 0.45)`
          : '0 12px 24px rgba(2, 6, 23, 0.45)',
        minWidth: '200px',
        position: 'relative',
        transition: 'all 0.2s ease'
      }}
    >
      {/* Connection Handles */}
      <Handle
        type="target"
        position={Position.Left}
        style={{ background: sectorColor || '#3B82F6', width: 8, height: 8 }}
      />
      <Handle
        type="source"
        position={Position.Right}
        style={{ background: sectorColor || '#3B82F6', width: 8, height: 8 }}
      />

      {/* Header with expand button, stage type, and scenario indicator */}
      <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '8px' }}>
        {/* Expand/Collapse Button */}
        {hasChildren && (
          <button
            onClick={handleExpandClick}
            disabled={isLoading}
            style={{
              background: isExpanded ? `${sectorColor || '#3B82F6'}22` : 'transparent',
              border: `2px solid ${sectorColor || '#3B82F6'}`,
              borderRadius: '6px',
              color: sectorColor || '#3B82F6',
              cursor: isLoading ? 'wait' : 'pointer',
              padding: '4px 8px',
              fontSize: '10px',
              fontWeight: 700,
              transition: 'all 0.2s',
              opacity: isLoading ? 0.5 : 1,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              gap: '4px',
              boxShadow: isExpanded ? `0 0 8px ${sectorColor || '#3B82F6'}44` : 'none'
            }}
            aria-label={isExpanded ? 'Collapse children' : 'Expand children'}
            title={
              isLoading
                ? 'Loading children...'
                : isExpanded
                  ? 'Click to collapse child stages'
                  : childrenLoaded
                    ? 'Click to show child stages'
                    : 'Click to load and show child stages'
            }
          >
            {isLoading ? (
              <Loader2 size={16} style={{ animation: 'spin 1s linear infinite' }} />
            ) : isExpanded ? (
              <ChevronDown size={16} />
            ) : (
              <ChevronRight size={16} />
            )}
            {!isLoading && <span style={{ fontSize: '9px' }}>{isExpanded ? 'Hide' : 'Show'}</span>}
          </button>
        )}

        {/* Stage Type Badge */}
        <span
          style={{
            background: `${sectorColor || '#3B82F6'}22`,
            color: sectorColor || '#3B82F6',
            padding: '2px 8px',
            borderRadius: '6px',
            fontSize: '10px',
            fontWeight: 600,
            textTransform: 'capitalize',
            flex: 1,
            display: 'flex',
            alignItems: 'center',
            gap: '4px'
          }}
        >
          {type.replace('_', ' ')}
          {hasChildren && !isExpanded && (
            <span title="Has child stages (click expand button to load)">
              <FolderTree
                size={10}
                style={{ opacity: 0.7 }}
              />
            </span>
          )}
        </span>

        {/* Scenario Count Indicator */}
        {scenarioCount !== undefined && scenarioCount > 0 && (
          <span
            style={{
              background: '#10b981',
              color: '#fff',
              padding: '2px 6px',
              borderRadius: '4px',
              fontSize: '9px',
              fontWeight: 700,
              display: 'flex',
              alignItems: 'center',
              gap: '3px'
            }}
            title={`${scenarioCount} linked scenario${scenarioCount > 1 ? 's' : ''}`}
          >
            <Link size={10} /> {scenarioCount}
          </span>
        )}
      </div>

      {/* Stage Name */}
      <div
        style={{
          fontWeight: 600,
          fontSize: '13px',
          marginBottom: '8px',
          lineHeight: '1.4',
          wordBreak: 'break-word'
        }}
      >
        {label}
      </div>

      {/* Progress Bar */}
      <div
        style={{
          width: '100%',
          height: '4px',
          background: 'rgba(30, 41, 59, 0.6)',
          borderRadius: '2px',
          overflow: 'hidden',
          marginTop: '8px'
        }}
      >
        <div
          style={{
            width: progressBarWidth,
            height: '100%',
            background: `linear-gradient(90deg, ${sectorColor || '#3B82F6'}, ${sectorColor || '#3B82F6'}dd)`,
            borderRadius: '2px',
            transition: 'width 0.3s ease'
          }}
        />
      </div>

      {/* Progress Percentage */}
      <div
        style={{
          fontSize: '10px',
          color: '#94a3b8',
          marginTop: '4px',
          textAlign: 'right'
        }}
      >
        {progress.toFixed(0)}% complete
      </div>
    </div>
  )
}

export default StageNode
