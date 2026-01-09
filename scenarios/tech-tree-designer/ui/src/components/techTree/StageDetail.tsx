import React from 'react'
import { Eye, EyeOff } from 'lucide-react'
import type { StageLookupEntry, ScenarioEntryMap } from '../../types/graph'
import { formatStageTypeLabel, getStageTypeColor, stageTypeLabel } from '../../utils/constants'

interface StageDetailProps {
  stage: StageLookupEntry
  titleId?: string
  scenarioEntryMap: ScenarioEntryMap
  onToggleScenarioVisibility: (scenarioName: string, hidden: boolean) => void
  onLinkScenario?: (stage: StageLookupEntry) => void
}

const StageDetail: React.FC<StageDetailProps> = ({
  stage,
  titleId,
  scenarioEntryMap,
  onToggleScenarioVisibility,
  onLinkScenario
}) => {
  const renderScenarioMappings = () => {
    const mappings = stage?.scenario_mappings || []
    if (!mappings.length) {
      return <li className="stage-detail__empty">No scenario mappings recorded yet.</li>
    }

    return mappings.slice(0, 6).map((mapping) => {
      const scenarioName = mapping.scenario_name
      if (!scenarioName) {
        return null
      }
      const entry =
        scenarioEntryMap.get(scenarioName) || scenarioEntryMap.get(scenarioName.toLowerCase())
      const isHidden = entry?.hidden
      return (
        <li key={mapping.id || scenarioName} className="stage-detail__mapping">
          <div>
            <span className="stage-detail__mapping-name">{entry?.display_name || scenarioName}</span>
            <span className="stage-detail__mapping-status">{mapping.completion_status}</span>
            {entry?.description ? (
              <p className="stage-detail__mapping-description">{entry.description}</p>
            ) : null}
          </div>
          <button
            type="button"
            className="button button--ghost"
            onClick={() => onToggleScenarioVisibility(scenarioName, !isHidden)}
          >
            {isHidden ? (
              <>
                <Eye className="w-4 h-4" aria-hidden="true" /> Unhide
              </>
            ) : (
              <>
                <EyeOff className="w-4 h-4" aria-hidden="true" /> Hide
              </>
            )}
          </button>
        </li>
      )
    })
  }

  const stageType = stage.stage_type || 'foundation'
  const badgeLabel = stageTypeLabel[stageType as keyof typeof stageTypeLabel] || formatStageTypeLabel(stageType)

  return (
    <div className="stage-detail">
      <header className="stage-detail__header">
        <h3 id={titleId}>{stage.name}</h3>
        <span className="stage-detail__badge" style={{ background: getStageTypeColor(stageType) }}>
          {badgeLabel}
        </span>
      </header>
      <p className="stage-detail__sector">{stageType}</p>
      <p className="stage-detail__description">{stage.description || 'No description available yet.'}</p>

      <div className="stage-detail__metrics">
        <div>
          <span className="stage-detail__metric-label">Progress</span>
          <span className="stage-detail__metric-value">{stage.progress_percentage ?? 0}%</span>
        </div>
        <div>
          <span className="stage-detail__metric-label">Order</span>
          <span className="stage-detail__metric-value">{stage.stage_order ?? '—'}</span>
        </div>
      </div>

      <div className="stage-detail__stack">
        <h4>Enabled Scenarios</h4>
        <ul>{renderScenarioMappings()}</ul>
      </div>

      {Array.isArray(stage.examples) && stage.examples.length ? (
        <div className="stage-detail__stack">
          <h4>Examples</h4>
          <ul>
            {stage.examples.map((example) => (
              <li key={example}>{example}</li>
            ))}
          </ul>
        </div>
      ) : null}

      {typeof onLinkScenario === 'function' ? (
        <div className="stage-detail__actions">
          <button type="button" className="button" onClick={() => onLinkScenario(stage)}>
            Link scenario
          </button>
        </div>
      ) : null}
    </div>
  )
}

export default StageDetail
