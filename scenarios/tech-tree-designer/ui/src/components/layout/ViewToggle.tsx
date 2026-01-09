import React from 'react'
import { LayoutDashboard, PenSquare } from 'lucide-react'
import type { TreeViewMode } from '../../types/techTree'

interface ViewToggleProps {
  viewMode: TreeViewMode
  onChange: (mode: TreeViewMode) => void
}

const ViewToggle: React.FC<ViewToggleProps> = ({ viewMode, onChange }) => (
  <section className="view-toggle" role="tablist" aria-label="Primary views">
    <button
      type="button"
      role="tab"
      aria-selected={viewMode === 'overview'}
      className={`view-toggle__button ${viewMode === 'overview' ? 'is-active' : ''}`}
      onClick={() => onChange('overview')}
    >
      <LayoutDashboard className="view-toggle__icon" aria-hidden="true" />
      Overview Dashboard
    </button>
    <button
      type="button"
      role="tab"
      aria-selected={viewMode === 'designer'}
      className={`view-toggle__button ${viewMode === 'designer' ? 'is-active' : ''}`}
      onClick={() => onChange('designer')}
    >
      <PenSquare className="view-toggle__icon" aria-hidden="true" />
      Tech Tree Designer
    </button>
  </section>
)

export default ViewToggle
