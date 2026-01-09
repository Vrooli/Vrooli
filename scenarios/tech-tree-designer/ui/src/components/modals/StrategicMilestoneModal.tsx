import React from 'react'
import type { FormEvent } from 'react'
import type { MilestoneFormState } from '../../types/forms'
import type { Sector } from '../../types/techTree'

interface StrategicMilestoneModalProps {
  isOpen: boolean
  formState: MilestoneFormState
  sectors: Sector[]
  isSaving: boolean
  errorMessage: string | null
  editMode: boolean
  onClose: () => void
  onFieldChange: (field: keyof MilestoneFormState, value: string | string[]) => void
  onSubmit: (event: FormEvent<HTMLFormElement>) => Promise<void>
}

const milestoneTypeOptions = [
  { value: 'sector_complete', label: 'Sector Complete' },
  { value: 'cross_sector_integration', label: 'Cross-Sector Integration' },
  { value: 'civilization_twin', label: 'Civilization Twin' },
  { value: 'meta_simulation', label: 'Meta Simulation' }
]

const StrategicMilestoneModal: React.FC<StrategicMilestoneModalProps> = ({
  isOpen,
  formState,
  sectors,
  isSaving,
  errorMessage,
  editMode,
  onClose,
  onFieldChange,
  onSubmit
}) => {
  if (!isOpen) {
    return null
  }

  const handleSectorSelection = (event: React.ChangeEvent<HTMLSelectElement>) => {
    const values = Array.from(event.target.selectedOptions).map((option) => option.value)
    onFieldChange('targetSectors', values)
  }

  const handleStageSelection = (event: React.ChangeEvent<HTMLSelectElement>) => {
    const values = Array.from(event.target.selectedOptions).map((option) => option.value)
    onFieldChange('targetStages', values)
  }

  const stageOptions = sectors.flatMap((sector) =>
    (sector.stages || []).map((stage) => ({
      id: stage.id,
      label: `${sector.name} — ${stage.name}`
    }))
  )

  const modalTitle = editMode ? 'Edit Strategic Milestone' : 'Add Strategic Milestone'
  const submitLabel = editMode ? 'Update Milestone' : 'Create Milestone'

  return (
    <div className="modal-overlay" role="dialog" aria-modal="true" aria-label={modalTitle}>
      <div className="modal">
        <h3>{modalTitle}</h3>
        <p className="modal-subtitle">
          Capture milestone targets and valuation so the dashboard reflects the real strategy.
        </p>
        {errorMessage && <p className="form-error">{errorMessage}</p>}
        <form className="modal-form" onSubmit={onSubmit}>
          <div className="modal-field">
            <label htmlFor="milestone-name">Name</label>
            <input
              id="milestone-name"
              type="text"
              value={formState.name}
              onChange={(event) => onFieldChange('name', event.target.value)}
              required
            />
          </div>

          <div className="modal-field">
            <label htmlFor="milestone-description">Description</label>
            <textarea
              id="milestone-description"
              rows={3}
              value={formState.description}
              onChange={(event) => onFieldChange('description', event.target.value)}
            />
          </div>

          <div className="modal-field">
            <label htmlFor="milestone-type">Milestone Type</label>
            <select
              id="milestone-type"
              value={formState.milestoneType}
              onChange={(event) => onFieldChange('milestoneType', event.target.value)}
            >
              {milestoneTypeOptions.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </select>
          </div>

          <div className="modal-grid">
            <div className="modal-field">
              <label htmlFor="milestone-completion">Completion %</label>
              <input
                id="milestone-completion"
                type="number"
                min="0"
                max="100"
                value={formState.completion}
                onChange={(event) => onFieldChange('completion', event.target.value)}
              />
            </div>
            <div className="modal-field">
              <label htmlFor="milestone-confidence">Confidence (0-1)</label>
              <input
                id="milestone-confidence"
                type="number"
                min="0"
                max="1"
                step="0.01"
                value={formState.confidence}
                onChange={(event) => onFieldChange('confidence', event.target.value)}
              />
            </div>
          </div>

          <div className="modal-grid">
            <div className="modal-field">
              <label htmlFor="milestone-value">Business Value ($)</label>
              <input
                id="milestone-value"
                type="number"
                min="0"
                value={formState.businessValue}
                onChange={(event) => onFieldChange('businessValue', event.target.value)}
              />
            </div>
            <div className="modal-field">
              <label htmlFor="milestone-date">Estimated Completion</label>
              <input
                id="milestone-date"
                type="date"
                value={formState.estimatedDate}
                onChange={(event) => onFieldChange('estimatedDate', event.target.value)}
              />
            </div>
          </div>

          <div className="modal-field">
            <label htmlFor="milestone-sectors">Target Sectors</label>
            <select id="milestone-sectors" multiple value={formState.targetSectors} onChange={handleSectorSelection}>
              {sectors.map((sector) => (
                <option key={sector.id} value={sector.id}>
                  {sector.name}
                </option>
              ))}
            </select>
          </div>

          <div className="modal-field">
            <label htmlFor="milestone-stages">Target Stages</label>
            <select id="milestone-stages" multiple value={formState.targetStages} onChange={handleStageSelection}>
              {stageOptions.map((option) => (
                <option key={option.id} value={option.id}>
                  {option.label}
                </option>
              ))}
            </select>
          </div>

          <div className="modal-actions">
            <button type="button" className="button button--ghost" onClick={onClose}>
              Cancel
            </button>
            <button type="submit" className="button" disabled={isSaving}>
              {isSaving ? 'Saving…' : submitLabel}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

export default StrategicMilestoneModal
