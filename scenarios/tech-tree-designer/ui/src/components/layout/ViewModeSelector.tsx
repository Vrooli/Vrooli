import React from 'react'
import type { GraphViewMode } from '../../types/techTree'

interface ViewModeSelectorProps {
  viewMode: GraphViewMode
  onChange: (mode: GraphViewMode) => void
  showHiddenScenarios: boolean
  showIsolatedScenarios: boolean
  onToggleHidden: (value: boolean) => void
  onToggleIsolated: (value: boolean) => void
}

const ViewModeSelector: React.FC<ViewModeSelectorProps> = ({
  viewMode,
  onChange,
  showHiddenScenarios,
  showIsolatedScenarios,
  onToggleHidden,
  onToggleIsolated
}) => {
  const showScenarioFilters = viewMode === 'hybrid' || viewMode === 'scenarios'
  const showIsolatedFilter = viewMode === 'scenarios'

  return (
    <div className="view-mode-selector" role="radiogroup" aria-label="Graph view mode">
      <div className="view-mode-options">
        <label className="view-mode-option">
          <input
            type="radio"
            name="viewMode"
            value="tech-tree"
            checked={viewMode === 'tech-tree'}
            onChange={() => onChange('tech-tree')}
          />
          <div className="view-mode-label">
            <span className="view-mode-title">Tech Tree</span>
            <span className="view-mode-desc">Sectors & stages only</span>
          </div>
        </label>

        <label className="view-mode-option">
          <input
            type="radio"
            name="viewMode"
            value="hybrid"
            checked={viewMode === 'hybrid'}
            onChange={() => onChange('hybrid')}
          />
          <div className="view-mode-label">
            <span className="view-mode-title">Hybrid View</span>
            <span className="view-mode-desc">Tech tree + scenario overlays</span>
          </div>
        </label>

        <label className="view-mode-option">
          <input
            type="radio"
            name="viewMode"
            value="scenarios"
            checked={viewMode === 'scenarios'}
            onChange={() => onChange('scenarios')}
          />
          <div className="view-mode-label">
            <span className="view-mode-title">Scenario Graph</span>
            <span className="view-mode-desc">Scenario dependencies only</span>
          </div>
        </label>
      </div>

      {showScenarioFilters && (
        <div className="scenario-filters" role="group" aria-label="Scenario filters">
          <label className="filter-toggle">
            <input
              type="checkbox"
              checked={showHiddenScenarios}
              onChange={(e) => onToggleHidden(e.target.checked)}
            />
            <span>Show hidden scenarios</span>
          </label>

          {showIsolatedFilter && (
            <label className="filter-toggle">
              <input
                type="checkbox"
                checked={showIsolatedScenarios}
                onChange={(e) => onToggleIsolated(e.target.checked)}
                title="Show scenarios with no dependencies"
              />
              <span>Show isolated scenarios</span>
            </label>
          )}
        </div>
      )}
    </div>
  )
}

export default ViewModeSelector
