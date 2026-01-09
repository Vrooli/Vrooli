import React from 'react'
import type { ScenarioCatalogEntry } from '../../types/techTree'

interface ScenarioDetailProps {
  scenario: ScenarioCatalogEntry
  titleId?: string
  onToggleVisibility: (scenarioName: string, hidden: boolean) => void
}

const ScenarioDetail: React.FC<ScenarioDetailProps> = ({ scenario, titleId, onToggleVisibility }) => (
  <div className="stage-detail">
    <header className="stage-detail__header">
      <h3 id={titleId}>{scenario.display_name || scenario.name}</h3>
      <span className="stage-detail__badge" style={{ background: 'rgba(14,165,233,0.35)' }}>
        Live scenario
      </span>
    </header>
    <p className="stage-detail__sector">{scenario.relative_path || 'Repo scenario'}</p>
    <p className="stage-detail__description">{scenario.description || 'No description provided.'}</p>

    <div className="stage-detail__metrics">
      <div>
        <span className="stage-detail__metric-label">Hidden</span>
        <span className="stage-detail__metric-value">{scenario.hidden ? 'Yes' : 'No'}</span>
      </div>
      <div>
        <span className="stage-detail__metric-label">Last updated</span>
        <span className="stage-detail__metric-value">
          {scenario.last_modified ? new Date(scenario.last_modified).toLocaleDateString() : '—'}
        </span>
      </div>
    </div>

    <div className="stage-detail__stack">
      <h4>Scenario Tags</h4>
      {scenario.tags?.length ? (
        <ul className="scenario-tag-list">
          {scenario.tags.map((tag) => (
            <li key={tag}>{tag}</li>
          ))}
        </ul>
      ) : (
        <p className="stage-detail__empty">No tags detected.</p>
      )}
    </div>

    <div className="stage-detail__stack">
      <h4>Scenario Dependencies</h4>
      {scenario.dependencies?.length ? (
        <ul>
          {scenario.dependencies.map((dependency) => (
            <li key={dependency}>{dependency}</li>
          ))}
        </ul>
      ) : (
        <p className="stage-detail__empty">No linked scenarios yet.</p>
      )}
    </div>

    <div className="stage-detail__actions">
      <button type="button" className="button" onClick={() => onToggleVisibility(scenario.name, !scenario.hidden)}>
        {scenario.hidden ? 'Unhide scenario' : 'Hide scenario'}
      </button>
    </div>
  </div>
)

export default ScenarioDetail
