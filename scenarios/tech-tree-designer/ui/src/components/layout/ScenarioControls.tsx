import React from 'react'
import { RefreshCcw } from 'lucide-react'

interface ScenarioControlsProps {
  showLiveScenarios: boolean
  scenarioOnlyMode: boolean
  showHiddenScenarios: boolean
  onToggleLive: (value: boolean) => void
  onToggleScenarioOnly: (value: boolean) => void
  onToggleHidden: (value: boolean) => void
  onSync: () => void
  isSyncing: boolean
  lastSyncedLabel: string
}

const ScenarioControls: React.FC<ScenarioControlsProps> = ({
  showLiveScenarios,
  scenarioOnlyMode,
  showHiddenScenarios,
  onToggleLive,
  onToggleScenarioOnly,
  onToggleHidden,
  onSync,
  isSyncing,
  lastSyncedLabel
}) => (
  <div className="scenario-controls" aria-live="polite">
    <div className="scenario-toggle-group">
      <label className="toggle">
        <input
          type="checkbox"
          checked={showLiveScenarios}
          onChange={(event) => onToggleLive(event.target.checked)}
          disabled={scenarioOnlyMode}
        />
        <span>Show live scenarios</span>
      </label>
      <label className="toggle">
        <input
          type="checkbox"
          checked={scenarioOnlyMode}
          onChange={(event) => onToggleScenarioOnly(event.target.checked)}
        />
        <span>Scenario-only graph</span>
      </label>
      <label className="toggle">
        <input
          type="checkbox"
          checked={showHiddenScenarios}
          onChange={(event) => onToggleHidden(event.target.checked)}
        />
        <span>Show hidden</span>
      </label>
    </div>
    <div className="scenario-sync">
      <button type="button" className="button" onClick={onSync} disabled={isSyncing}>
        <RefreshCcw className={isSyncing ? 'spin' : ''} aria-hidden="true" />
        {isSyncing ? 'Syncing…' : 'Sync scenarios'}
      </button>
      <p className="scenario-sync__meta">Last sync: {lastSyncedLabel}</p>
    </div>
  </div>
)

export default ScenarioControls
