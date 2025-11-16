import React from 'react'
import { Sparkles, Copy, Plus, Circle } from 'lucide-react'
import type { TechTreeSummary } from '../../types/techTree'

interface TreeSelectorProps {
  techTrees: TechTreeSummary[]
  selectedTreeId: string | null
  disabled: boolean
  badgeClassName: string
  badgeLabel: string
  statsSummary: string | null
  onChange: (value: string | null) => void
  onCreateSector: () => void
  onCreateTree?: () => void
  onCloneTree?: () => void
}

const TreeSelector: React.FC<TreeSelectorProps> = ({
  techTrees,
  selectedTreeId,
  disabled,
  badgeClassName,
  badgeLabel,
  statsSummary,
  onChange,
  onCreateSector,
  onCreateTree,
  onCloneTree
}) => {
  const handleChange = (event: React.ChangeEvent<HTMLSelectElement>) => {
    const value = event.target.value.trim()
    onChange(value.length ? value : null)
  }

  const getTreeTypeIcon = (treeType: string) => {
    const iconProps = { size: 12, style: { display: 'inline', verticalAlign: 'middle', marginRight: '4px' } }
    switch (treeType) {
      case 'official': return <Circle {...iconProps} fill="#3b82f6" stroke="#3b82f6" />
      case 'draft': return <Circle {...iconProps} fill="#eab308" stroke="#eab308" />
      case 'experimental': return <Circle {...iconProps} fill="#a855f7" stroke="#a855f7" />
      default: return <Circle {...iconProps} fill="#94a3b8" stroke="#94a3b8" />
    }
  }

  return (
    <div className="tree-selector" aria-live="polite">
      <label htmlFor="tech-tree-select">Active Tech Tree</label>
      <div className="tree-selector__control">
        <select
          id="tech-tree-select"
          className="tree-selector__select"
          value={selectedTreeId || ''}
          onChange={handleChange}
          disabled={disabled}
        >
          {techTrees.length === 0 ? (
            <option value="">No trees available</option>
          ) : (
            techTrees.map((entry) => (
              <option key={entry.tree.id} value={entry.tree.id}>
                {getTreeTypeIcon(entry.tree.tree_type)} {entry.tree.name}
                {entry.tree.tree_type === 'official'
                  ? ' • Official'
                  : entry.tree.tree_type === 'draft'
                    ? ' • Draft'
                    : ' • Experimental'}
              </option>
            ))
          )}
        </select>
        <span className={`tree-badge ${badgeClassName}`}>
          {badgeClassName.includes('official') && <Circle size={10} fill="#3b82f6" stroke="#3b82f6" style={{ display: 'inline', marginRight: '4px' }} />}
          {badgeClassName.includes('draft') && <Circle size={10} fill="#eab308" stroke="#eab308" style={{ display: 'inline', marginRight: '4px' }} />}
          {badgeClassName.includes('experimental') && <Circle size={10} fill="#a855f7" stroke="#a855f7" style={{ display: 'inline', marginRight: '4px' }} />}
          {badgeLabel}
        </span>
      </div>
      <p className="tree-selector__meta">{statsSummary || 'Define a tree to begin mapping the capability graph.'}</p>
      <div className="tree-selector__actions">
        {onCreateTree && (
          <button
            type="button"
            className="button button--ghost"
            onClick={onCreateTree}
            title="Create a new tech tree"
          >
            <Sparkles size={14} style={{ display: 'inline', marginRight: '6px' }} />
            New Tree
          </button>
        )}
        {onCloneTree && selectedTreeId && (
          <button
            type="button"
            className="button button--ghost"
            onClick={onCloneTree}
            title="Clone the current tech tree for experimentation"
          >
            <Copy size={14} style={{ display: 'inline', marginRight: '6px' }} />
            Clone Tree
          </button>
        )}
        <button type="button" className="button button--ghost" onClick={onCreateSector}>
          <Plus size={14} style={{ display: 'inline', marginRight: '6px' }} />
          New Sector
        </button>
      </div>
    </div>
  )
}

export default TreeSelector
