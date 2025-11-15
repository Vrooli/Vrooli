import React from 'react'
import { X } from 'lucide-react'
import type { ScenarioCatalogEntry } from '../../types/techTree'
import type { ScenarioEntryMap, StageLookupEntry } from '../../types/graph'
import StageDetail from './StageDetail'
import ScenarioDetail from './ScenarioDetail'

interface StageDialogProps {
  isOpen: boolean
  selectedStage: StageLookupEntry | null
  selectedScenario: ScenarioCatalogEntry | null
  scenarioEntryMap: ScenarioEntryMap
  onClose: () => void
  scenarioTitleId?: string
  stageTitleId?: string
  onToggleScenarioVisibility: (scenarioName: string, hidden: boolean) => void
  onLinkScenario?: (stage: StageLookupEntry) => void
}

const StageDialog: React.FC<StageDialogProps> = ({
  isOpen,
  selectedStage,
  selectedScenario,
  scenarioEntryMap,
  onClose,
  scenarioTitleId,
  stageTitleId,
  onToggleScenarioVisibility,
  onLinkScenario
}) => {
  if (!isOpen || (!selectedStage && !selectedScenario)) {
    return null
  }

  return (
    <div className="stage-dialog-overlay" role="dialog" aria-modal="true" aria-labelledby={selectedScenario ? scenarioTitleId : stageTitleId} onClick={onClose}>
      <div className="stage-dialog" role="document" onClick={(event) => event.stopPropagation()}>
        <button type="button" className="stage-dialog__close" onClick={onClose} aria-label="Close stage details">
          <X className="stage-dialog__close-icon" aria-hidden="true" />
        </button>
        {selectedScenario ? (
          <ScenarioDetail
            scenario={selectedScenario}
            titleId={scenarioTitleId}
            onToggleVisibility={onToggleScenarioVisibility}
          />
        ) : selectedStage ? (
          <StageDetail
            stage={selectedStage}
            titleId={stageTitleId}
            scenarioEntryMap={scenarioEntryMap}
            onToggleScenarioVisibility={onToggleScenarioVisibility}
            onLinkScenario={onLinkScenario}
          />
        ) : null}
      </div>
    </div>
  )
}

export default StageDialog
