import React from 'react'
import { formatStageTypeLabel } from '../../utils/constants'
import type { ScenarioFormState } from '../../types/forms'
import type { ProgressionStage } from '../../types/techTree'

interface LinkScenarioModalProps {
  isOpen: boolean
  formState: ScenarioFormState
  targetStage: ProgressionStage | null
  onFieldChange: (field: keyof ScenarioFormState, value: string) => void
  onSubmit: (event: React.FormEvent<HTMLFormElement>) => void
  onClose: () => void
  isSaving: boolean
  errorMessage: string | null
}

const LinkScenarioModal: React.FC<LinkScenarioModalProps> = ({
  isOpen,
  formState,
  targetStage,
  onFieldChange,
  onSubmit,
  onClose,
  isSaving,
  errorMessage
}) => {
  if (!isOpen) {
    return null
  }

  const statusOptions = ['not_started', 'in_progress', 'completed', 'blocked']

  return (
    <div className="modal-overlay" role="dialog" aria-modal="true" aria-label="Link scenario">
      <div className="modal">
        <h3>Link Scenario</h3>
        {targetStage ? <p className="modal-subtitle">{targetStage.name}</p> : null}
        {errorMessage ? <p className="form-error">{errorMessage}</p> : null}
        <form onSubmit={onSubmit} className="modal-form">
          <div className="modal-field">
            <label htmlFor="scenario-name">Scenario Name</label>
            <input
              id="scenario-name"
              type="text"
              value={formState.scenarioName}
              onChange={(event) => onFieldChange('scenarioName', event.target.value)}
              required
            />
          </div>
          <div className="modal-grid">
            <div className="modal-field">
              <label htmlFor="scenario-status">Status</label>
              <select
                id="scenario-status"
                value={formState.status}
                onChange={(event) => onFieldChange('status', event.target.value)}
              >
                {statusOptions.map((status) => (
                  <option key={status} value={status}>
                    {formatStageTypeLabel(status)}
                  </option>
                ))}
              </select>
            </div>
            <div className="modal-field">
              <label htmlFor="scenario-weight">Contribution Weight</label>
              <input
                id="scenario-weight"
                type="number"
                min={0}
                max={1}
                step="0.1"
                value={formState.contributionWeight}
                onChange={(event) => onFieldChange('contributionWeight', event.target.value)}
              />
            </div>
            <div className="modal-field">
              <label htmlFor="scenario-priority">Priority</label>
              <input
                id="scenario-priority"
                type="number"
                min={1}
                max={5}
                value={formState.priority}
                onChange={(event) => onFieldChange('priority', event.target.value)}
              />
            </div>
            <div className="modal-field">
              <label htmlFor="scenario-impact">Estimated Impact</label>
              <input
                id="scenario-impact"
                type="number"
                min={0}
                max={10}
                value={formState.estimatedImpact}
                onChange={(event) => onFieldChange('estimatedImpact', event.target.value)}
              />
            </div>
          </div>
          <div className="modal-field">
            <label htmlFor="scenario-notes">Notes</label>
            <textarea
              id="scenario-notes"
              rows={3}
              value={formState.notes}
              onChange={(event) => onFieldChange('notes', event.target.value)}
            />
          </div>
          <div className="modal-actions">
            <button type="button" className="button button--ghost" onClick={onClose}>
              Cancel
            </button>
            <button type="submit" className="button" disabled={isSaving}>
              {isSaving ? 'Linking…' : 'Link Scenario'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

export default LinkScenarioModal
