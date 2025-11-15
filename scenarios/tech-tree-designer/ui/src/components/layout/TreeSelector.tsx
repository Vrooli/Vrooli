import React from 'react'
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
                {entry.tree.name}
                {entry.tree.tree_type === 'official'
                  ? ' • Official'
                  : entry.tree.tree_type === 'draft'
                    ? ' • Draft'
                    : ' • Experimental'}
              </option>
            ))
          )}
        </select>
        <span className={`tree-badge ${badgeClassName}`}>{badgeLabel}</span>
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
            + New Tree
          </button>
        )}
        {onCloneTree && selectedTreeId && (
          <button
            type="button"
            className="button button--ghost"
            onClick={onCloneTree}
            title="Clone the current tech tree"
          >
            Clone Tree
          </button>
        )}
        <button type="button" className="button button--ghost" onClick={onCreateSector}>
          New Sector
        </button>
      </div>
    </div>
  )
}

export default TreeSelector
