import React, { useEffect, useMemo, useRef, useState } from 'react'
import {
  ArrowUpDown,
  BarChart3,
  Brain,
  Check,
  Database,
  GitBranch,
  Info,
  Network,
  Settings,
  SlidersHorizontal,
  Target,
  Trash2,
  X,
  Zap
} from 'lucide-react'
import { getSectorIcon } from '../utils/icons'
import { formatCurrency } from '../utils/formatters'
import { formatStageTypeLabel } from '../utils/constants'
import type {
  Sector,
  SectorSortOption,
  StrategicMilestone,
  StrategicValuePreset,
  StrategicValueBreakdown,
  StrategicValueSettings
} from '../types/techTree'
import {
  DEFAULT_STRATEGIC_VALUE_SETTINGS,
  DEFAULT_STRATEGIC_VALUE_PRESET_ID
} from '../utils/strategicValue'

interface InsightMetrics {
  averageSectorProgress: number
  activeMilestones: number
  totalPotentialValue: number
}

interface OverviewDashboardProps {
  sectors: Sector[]
  selectedSector: Sector | null
  onSelectSector: (sector: Sector | null) => void
  milestones: StrategicMilestone[]
  insightMetrics: InsightMetrics
  onRequestNewStage: (options?: { sectorId?: string; stageType?: string }) => void
  sectorSort: SectorSortOption
  onSectorSortChange: (value: SectorSortOption) => void
  strategicValueBreakdown: StrategicValueBreakdown
  strategicValueSettings: StrategicValueSettings
  onStrategicValueSettingsChange: (settings: StrategicValueSettings) => void
  isStrategicValuePanelOpen: boolean
  onStrategicValuePanelToggle: (open?: boolean) => void
  valuePresets: StrategicValuePreset[]
  customValuePresets: StrategicValuePreset[]
  activeValuePresetId: string | null
  onApplyValuePreset: (presetId: string) => void
  onCreateValuePreset: (payload: { name: string; description?: string }) => void
  onDeleteValuePreset: (presetId: string) => void
}

const stageIcons = {
  foundation: Database,
  operational: Settings,
  analytics: BarChart3,
  integration: Network,
  digital_twin: Brain
}

const OverviewDashboard: React.FC<OverviewDashboardProps> = ({
  sectors,
  selectedSector,
  onSelectSector,
  milestones,
  insightMetrics,
  onRequestNewStage,
  sectorSort,
  onSectorSortChange,
  strategicValueBreakdown,
  strategicValueSettings,
  onStrategicValueSettingsChange,
  isStrategicValuePanelOpen,
  onStrategicValuePanelToggle,
  valuePresets,
  customValuePresets,
  activeValuePresetId,
  onApplyValuePreset,
  onCreateValuePreset,
  onDeleteValuePreset
}) => {
  const [sortOpen, setSortOpen] = useState(false)
  const sortMenuRef = useRef<HTMLDivElement | null>(null)
  const [presetFormOpen, setPresetFormOpen] = useState(false)
  const [presetName, setPresetName] = useState('')
  const [presetDescription, setPresetDescription] = useState('')

  const sortOptions: { value: SectorSortOption; label: string }[] = useMemo(
    () => [
      { value: 'most-progress', label: 'Most progress' },
      { value: 'least-progress', label: 'Least progress' },
      { value: 'most-strategic', label: 'Most strategic value' },
      { value: 'least-strategic', label: 'Least strategic value' },
      { value: 'alpha-asc', label: 'Name A-Z' },
      { value: 'alpha-desc', label: 'Name Z-A' }
    ],
    []
  )

  const weightControls: Array<{
    key: 'completionWeight' | 'readinessWeight' | 'influenceWeight'
    label: string
    description: string
  }> = useMemo(
    () => [
      {
        key: 'completionWeight',
        label: 'Completion influence',
        description: 'How much milestone completion drives the projection'
      },
      {
        key: 'readinessWeight',
        label: 'Readiness influence',
        description: 'Accounts for sector maturity and stage progress'
      },
      {
        key: 'influenceWeight',
        label: 'Connectivity influence',
        description: 'Rewards milestones that unlock cross-sector leverage'
      }
    ],
    []
  )

  const topValueContributions = useMemo(() => {
    return [...strategicValueBreakdown.contributions]
      .sort((a, b) => b.adjustedValue - a.adjustedValue)
      .slice(0, 4)
  }, [strategicValueBreakdown.contributions])

  const sectorHighlights = useMemo(() => {
    return [...strategicValueBreakdown.sectorSummaries]
      .sort(
        (a, b) =>
          b.readinessScore + b.influenceScore + b.milestoneCount * 0.1 -
          (a.readinessScore + a.influenceScore + a.milestoneCount * 0.1)
      )
      .slice(0, 3)
  }, [strategicValueBreakdown.sectorSummaries])

  const penaltyPercent = Math.round((strategicValueSettings.dependencyPenalty || 0) * 100)

  const handlePresetFormSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const trimmed = presetName.trim()
    if (!trimmed) {
      return
    }
    onCreateValuePreset({ name: trimmed, description: presetDescription.trim() })
    setPresetFormOpen(false)
    setPresetName('')
    setPresetDescription('')
  }

  const handleWeightInputChange = (key: keyof StrategicValueSettings, rawValue: string) => {
    const numericValue = Number(rawValue)
    if (Number.isNaN(numericValue)) {
      return
    }
    onStrategicValueSettingsChange({
      ...strategicValueSettings,
      [key]: numericValue
    })
  }

  const handlePenaltyInputChange = (rawValue: string) => {
    const numericValue = Number(rawValue)
    if (Number.isNaN(numericValue)) {
      return
    }
    onStrategicValueSettingsChange({
      ...strategicValueSettings,
      dependencyPenalty: Math.max(0, Math.min(1, numericValue / 100))
    })
  }

  const handleResetValueModel = () => {
    onApplyValuePreset(DEFAULT_STRATEGIC_VALUE_PRESET_ID)
    setPresetFormOpen(false)
    setPresetName('')
    setPresetDescription('')
  }

  const activeSortLabel = useMemo(() => {
    const active = sortOptions.find((option) => option.value === sectorSort)
    return active?.label || sortOptions[0].label
  }, [sectorSort, sortOptions])

  useEffect(() => {
    if (!sortOpen) {
      return
    }
    const handleClickOutside = (event: MouseEvent) => {
      if (sortMenuRef.current && !sortMenuRef.current.contains(event.target as Node)) {
        setSortOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [sortOpen])

  return (
    <>
      <section className="insight-grid" aria-label="Strategic indicators">
        <div className="insight-card">
          <div className="insight-label">Average Sector Progress</div>
          <div className="insight-value">{insightMetrics.averageSectorProgress}%</div>
          <div className="insight-subtext">{sectors.length} sectors monitored</div>
        </div>
        <div className="insight-card">
          <div className="insight-label">Active Milestones</div>
          <div className="insight-value">{insightMetrics.activeMilestones}</div>
          <div className="insight-subtext">50%+ completion momentum</div>
        </div>
        <div
          className={`insight-card insight-card--interactive ${
            isStrategicValuePanelOpen ? 'is-active' : ''
          }`}
        >
          <button
            type="button"
            className="insight-card-button"
            onClick={() => onStrategicValuePanelToggle()}
            aria-pressed={isStrategicValuePanelOpen}
            title="Inspect the strategic value model"
          >
            <div className="insight-label">Potential Strategic Value</div>
            <div className="insight-value">{formatCurrency(insightMetrics.totalPotentialValue)}</div>
            <div className="insight-subtext">
              <span>Adjusted: {formatCurrency(strategicValueBreakdown.adjustedValue)}</span>
              <span className="insight-link">{isStrategicValuePanelOpen ? 'Hide breakdown' : 'View breakdown'}</span>
            </div>
          </button>
        </div>
      </section>

      {isStrategicValuePanelOpen && (
        <section className="strategic-panel strategic-panel--value" aria-label="Strategic value breakdown">
          <div className="panel-header">
            <div className="panel-header-heading">
              <BarChart3 className="panel-icon" aria-hidden="true" />
              <div>
                <h2 className="panel-title">Potential Strategic Value</h2>
                <p className="panel-subtitle">Total upside if the current tree reaches 100% completion.</p>
              </div>
            </div>
            <div className="panel-header-actions">
              <button
                type="button"
                className="button button--ghost"
                onClick={handleResetValueModel}
              >
                Reset model
              </button>
              <button
                type="button"
                className="panel-icon-button"
                aria-label="Close strategic value breakdown"
                onClick={() => onStrategicValuePanelToggle(false)}
              >
                <X aria-hidden="true" />
              </button>
            </div>
          </div>

          <section className="value-presets" aria-label="Valuation presets">
            <div className="value-presets-head">
              <div>
                <h3>Valuation presets</h3>
                <p>Snap between saved strategies instead of rebalancing sliders.</p>
              </div>
              <button
                type="button"
                className="button button--ghost"
                onClick={() => setPresetFormOpen((open) => !open)}
              >
                {presetFormOpen ? 'Cancel' : 'Save current preset'}
              </button>
            </div>
            <div className="value-preset-chip-group">
              {valuePresets.map((preset) => (
                <button
                  key={preset.id}
                  type="button"
                  className={`value-preset-chip ${
                    activeValuePresetId === preset.id ? 'is-active' : ''
                  }`}
                  onClick={() => onApplyValuePreset(preset.id)}
                  title={preset.description}
                >
                  <span>{preset.name}</span>
                  {preset.builtIn && <span className="value-preset-chip-label">Default</span>}
                </button>
              ))}
            </div>

            {presetFormOpen && (
              <form className="value-preset-form" onSubmit={handlePresetFormSubmit}>
                <label>
                  <span>Preset name</span>
                  <input
                    type="text"
                    value={presetName}
                    onChange={(event) => setPresetName(event.target.value)}
                    placeholder="e.g. Efficiency push"
                    required
                  />
                </label>
                <label>
                  <span>Description (optional)</span>
                  <textarea
                    value={presetDescription}
                    onChange={(event) => setPresetDescription(event.target.value)}
                    placeholder="What strategy does this reflect?"
                    rows={2}
                  />
                </label>
                <div className="value-preset-form-actions">
                  <button type="submit" className="button" disabled={!presetName.trim()}>
                    Save preset
                  </button>
                  <button
                    type="button"
                    className="button button--ghost"
                    onClick={() => {
                      setPresetFormOpen(false)
                      setPresetName('')
                      setPresetDescription('')
                    }}
                  >
                    Dismiss
                  </button>
                </div>
              </form>
            )}

            {customValuePresets.length > 0 && (
              <ul className="value-preset-list">
                {customValuePresets.map((preset) => (
                  <li key={preset.id}>
                    <div>
                      <strong>{preset.name}</strong>
                      {preset.description && <p>{preset.description}</p>}
                    </div>
                    <button
                      type="button"
                      className="panel-icon-button"
                      aria-label={`Delete ${preset.name} preset`}
                      onClick={() => onDeleteValuePreset(preset.id)}
                    >
                      <Trash2 aria-hidden="true" />
                    </button>
                  </li>
                ))}
              </ul>
            )}
          </section>

          <div className="value-summary-grid" role="presentation">
            <article className="value-summary-card">
              <div className="value-summary-label">Full potential</div>
              <div className="value-summary-value">
                {formatCurrency(strategicValueBreakdown.fullPotentialValue)}
              </div>
              <p className="value-summary-copy">Model assumes milestones hit 100% completion.</p>
            </article>
            <article className="value-summary-card">
              <div className="value-summary-label">Counted today</div>
              <div className="value-summary-value">
                {formatCurrency(strategicValueBreakdown.adjustedValue)}
              </div>
              <p className="value-summary-copy">Applies completion, readiness, and influence weights.</p>
            </article>
            <article className="value-summary-card">
              <div className="value-summary-label">Value locked</div>
              <div className="value-summary-value">
                {formatCurrency(strategicValueBreakdown.lockedValue)}
              </div>
              <p className="value-summary-copy">Gap between total potential and discounted view.</p>
            </article>
          </div>

          <div className="value-breakdown-grid">
            <section className="value-controls" aria-label="Strategic value weighting controls">
              <header className="value-controls-head">
                <SlidersHorizontal aria-hidden="true" />
                <div>
                  <h3>Model Weights</h3>
                  <p>Adjust how each factor influences the active valuation.</p>
                </div>
              </header>
              {weightControls.map((control) => (
                <label key={control.key} className="value-control">
                  <div className="value-control-heading">
                    <span>{control.label}</span>
                    <span>{strategicValueSettings[control.key]}%</span>
                  </div>
                  <input
                    type="range"
                    min={0}
                    max={100}
                    value={strategicValueSettings[control.key]}
                    onChange={(event) => handleWeightInputChange(control.key, event.target.value)}
                    aria-label={control.label}
                  />
                  <p className="value-control-description">{control.description}</p>
                </label>
              ))}

              <label className="value-control">
                <div className="value-control-heading">
                  <span>Dependency penalties</span>
                  <span>{penaltyPercent}%</span>
                </div>
                <input
                  type="range"
                  min={0}
                  max={60}
                  value={penaltyPercent}
                  step={5}
                  onChange={(event) => handlePenaltyInputChange(event.target.value)}
                  aria-label="Dependency penalty"
                />
                <p className="value-control-description">
                  Reduces credit for milestones with low completion despite critical dependencies.
                </p>
              </label>
            </section>

            <section className="value-contributions" aria-label="Top strategic value contributors">
              <header className="value-contributions-head">
                <Info aria-hidden="true" />
                <div>
                  <h3>Top Drivers</h3>
                  <p>Highest contributors after applying the current model.</p>
                </div>
              </header>
              {topValueContributions.length === 0 ? (
                <div className="empty-state compact">
                  <Zap className="empty-icon" aria-hidden="true" />
                  <p>No milestones have valuation data yet.</p>
                </div>
              ) : (
                topValueContributions.map((contribution) => (
                  <article key={contribution.id} className="value-contribution-card">
                    <header>
                      <h4>{contribution.name}</h4>
                      <span>{Math.round(contribution.completionScore * 100)}% complete</span>
                    </header>
                    <div className="value-contribution-amounts">
                      <div>
                        <span className="value-contribution-label">Potential</span>
                        <strong>{formatCurrency(contribution.baseValue)}</strong>
                      </div>
                      <div>
                        <span className="value-contribution-label">Counted</span>
                        <strong>{formatCurrency(contribution.adjustedValue)}</strong>
                      </div>
                    </div>
                    {contribution.linkedStages.length > 0 && (
                      <div className="value-contribution-stages">
                        {contribution.linkedStages.slice(0, 3).map((stage) => (
                          <span key={stage.stageId}>
                            {stage.stageName}
                            {stage.sectorName ? ` (${stage.sectorName})` : ''}
                          </span>
                        ))}
                        {contribution.linkedStages.length > 3 && (
                          <span className="value-contribution-stage-more">
                            +{contribution.linkedStages.length - 3} more nodes
                          </span>
                        )}
                      </div>
                    )}
                    <footer>
                      <span>Readiness {Math.round(contribution.readinessScore * 100)}%</span>
                      <span>Influence {Math.round(contribution.influenceScore * 100)}%</span>
                      {contribution.linkedSectors.length > 0 && (
                        <span className="value-contribution-sectors">
                          Targets {contribution.linkedSectors.length} sector
                          {contribution.linkedSectors.length > 1 ? 's' : ''}
                        </span>
                      )}
                    </footer>
                  </article>
                ))
              )}
            </section>
          </div>

          <section className="value-sector-highlights" aria-label="Sector readiness signals">
            <header>
              <Brain aria-hidden="true" />
              <div>
                <h3>Sector Signals</h3>
                <p>Readiness and connectivity scores feeding the model.</p>
              </div>
            </header>
            <div className="sector-highlight-grid">
              {sectorHighlights.map((summary) => (
                <article key={summary.sectorId} className="sector-highlight-card">
                  <div className="sector-highlight-head">
                    <span className="sector-highlight-dot" style={{ backgroundColor: summary.color }} />
                    <div>
                      <h4>{summary.name}</h4>
                      <p>{summary.milestoneCount} milestone{summary.milestoneCount === 1 ? '' : 's'} linked</p>
                    </div>
                    <span className="sector-highlight-progress">{summary.progressPercentage}%</span>
                  </div>
                  <div className="sector-highlight-metrics">
                    <div>
                      <span className="value-contribution-label">Readiness</span>
                      <strong>{Math.round(summary.readinessScore * 100)}%</strong>
                    </div>
                    <div>
                      <span className="value-contribution-label">Influence</span>
                      <strong>{Math.round(summary.influenceScore * 100)}%</strong>
                    </div>
                    <div>
                      <span className="value-contribution-label">Scenario links</span>
                      <strong>{summary.scenarioLinks}</strong>
                    </div>
                  </div>
                </article>
              ))}
              {!sectorHighlights.length && (
                <div className="empty-state compact">
                  <Network className="empty-icon" aria-hidden="true" />
                  <p>No sector readiness signals available.</p>
                </div>
              )}
            </div>
          </section>
        </section>
      )}

      <div className="strategic-grid three-col w-full" aria-label="Strategic intelligence layout">
        <section className="strategic-panel strategic-panel--sectors">
          <div className="panel-header">
            <div className="panel-header-heading">
              <GitBranch className="panel-icon" aria-hidden="true" />
              <div>
                <h2 className="panel-title">Technology Sectors</h2>
                <p className="panel-subtitle">Track momentum and emerging strengths.</p>
              </div>
            </div>
            <div className="panel-header-actions" ref={sortMenuRef}>
              <button
                type="button"
                className="panel-icon-button"
                aria-haspopup="menu"
                aria-expanded={sortOpen}
                aria-label="Sort technology sectors"
                onClick={() => setSortOpen((open) => !open)}
                title={`Sorted by ${activeSortLabel}`}
                disabled={!sectors.length}
              >
                <ArrowUpDown aria-hidden="true" />
                <span className="sr-only">Sort technology sectors</span>
              </button>
              {sortOpen && (
                <div className="panel-sort-menu" role="menu">
                  {sortOptions.map((option) => (
                    <button
                      key={option.value}
                      type="button"
                      role="menuitemradio"
                      aria-checked={sectorSort === option.value}
                      className={`panel-sort-option ${
                        sectorSort === option.value ? 'is-active' : ''
                      }`}
                      onClick={() => {
                        onSectorSortChange(option.value)
                        setSortOpen(false)
                      }}
                    >
                      <span>{option.label}</span>
                      {sectorSort === option.value && <Check aria-hidden="true" />}
                    </button>
                  ))}
                </div>
              )}
            </div>
          </div>
          <p className="panel-sort-label" aria-live="polite">
            Sorted by <strong>{activeSortLabel}</strong>
          </p>

          <div className="scrollable sector-list" role="list">
            {sectors.length === 0 ? (
              <div className="empty-state">
                <Zap className="empty-icon" aria-hidden="true" />
                <p>No sectors available yet. Populate the API to begin tracking.</p>
              </div>
            ) : (
              sectors.map((sector) => {
                const IconComponent = getSectorIcon(sector.category)
                const isSelected = selectedSector?.id === sector.id

                return (
                  <article
                    key={sector.id}
                    role="listitem"
                    className={`tech-node selectable transition-all ${isSelected ? 'active' : ''}`}
                    onClick={() => onSelectSector(sector)}
                    style={{ borderLeftColor: sector.color }}
                    tabIndex={0}
                    onKeyDown={(event) => {
                      if (event.key === 'Enter' || event.key === ' ') {
                        event.preventDefault()
                        onSelectSector(sector)
                      }
                    }}
                  >
                    <div className="sector-card-head">
                      <span className="sector-icon" style={{ color: sector.color }}>
                        <IconComponent className="w-5 h-5" aria-hidden="true" />
                      </span>
                      <div>
                        <span className="sector-name">{sector.name}</span>
                        <span className="sector-category capitalize">{sector.category}</span>
                      </div>
                      <span className="sector-progress">{sector.progress_percentage}%</span>
                    </div>

                    <div className="progress-container">
                      <div className="progress-bar" style={{ width: `${sector.progress_percentage}%` }} />
                    </div>

                    <p className="sector-description">{sector.description}</p>
                  </article>
                )
              })
            )}
          </div>

          <div className="panel-footer">
            <div className="civilization-progress">
              <Zap className="w-6 h-6" aria-hidden="true" />
              <div>
                <h4 className="civilization-title">Civilization Progress</h4>
                <p className="civilization-subtitle">Path to superintelligence alignment</p>
              </div>
            </div>
          </div>
        </section>

        <section className="strategic-panel strategic-panel--main">
          <div className="panel-header">
            <div className="panel-header-heading">
              <Network className="panel-icon" aria-hidden="true" />
              <div>
                <h2 className="panel-title">Strategic Intelligence Console</h2>
                <p className="panel-subtitle">Inspect engineering stages and acceleration levers.</p>
              </div>
            </div>
            <div className="superintelligence-indicator">Strategic Intelligence System</div>
          </div>

          {selectedSector ? (
            <div className="sector-detail scrollable">
              <header className="sector-detail-header">
                {React.createElement(getSectorIcon(selectedSector.category), {
                  className: 'sector-detail-icon',
                  style: { color: selectedSector.color },
                  'aria-hidden': true
                })}
                <div>
                  <h2 className="sector-detail-title">{selectedSector.name}</h2>
                  <p className="sector-detail-description">{selectedSector.description}</p>
                </div>
                <div className="sector-detail-metric">
                  <span className="metric-label">Overall progress</span>
                  <span className="metric-value">{selectedSector.progress_percentage}%</span>
                </div>
              </header>

              <div className="progress-container large">
                <div className="progress-bar" style={{ width: `${selectedSector.progress_percentage}%` }} />
              </div>

              <section className="stage-section">
                <div className="stage-title-row">
                  <h3 className="stage-title">Progression Stages</h3>
                  <button
                    type="button"
                    className="button button--ghost"
                    onClick={() => onRequestNewStage({ sectorId: selectedSector.id })}
                  >
                    New Stage
                  </button>
                </div>
                <div className="stage-grid">
                  {selectedSector.stages?.map((stage, index) => {
                    const StageIcon =
                      stageIcons[stage.stage_type as keyof typeof stageIcons] || Target
                    const progress = stage.progress_percentage ?? 0

                    return (
                      <article key={`${stage.id || stage.name}-${index}`} className="tech-node stage-card">
                        <div className="stage-head">
                          <StageIcon className="stage-icon" aria-hidden="true" />
                          <div>
                            <span className="stage-name">{stage.name}</span>
                            <span className="stage-type">{formatStageTypeLabel(stage.stage_type)}</span>
                          </div>
                          <div
                            className={`status-indicator ${
                              progress >= 80
                                ? 'status-success'
                                : progress >= 50
                                  ? 'status-warning'
                                  : 'status-info'
                            }`}
                          >
                            {progress >= 80
                              ? 'Advanced'
                              : progress >= 50
                                ? 'Active'
                                : 'Foundation'}
                          </div>
                        </div>

                        <div className="progress-container">
                          <div className="progress-bar" style={{ width: `${progress}%` }} />
                        </div>

                        <p className="stage-footnote">{progress}% complete</p>
                      </article>
                    )
                  })}
                </div>
              </section>
            </div>
          ) : (
            <div className="empty-state">
              <Network className="empty-icon" aria-hidden="true" />
              <h3>No sector selected</h3>
              <p>Select a technology sector to unlock its strategic intelligence report.</p>
            </div>
          )}
        </section>

        <section className="strategic-panel strategic-panel--milestones">
          <div className="panel-header">
            <div className="panel-header-heading">
              <Target className="panel-icon" aria-hidden="true" />
              <div>
                <h2 className="panel-title">Strategic Milestones</h2>
                <p className="panel-subtitle">Value checkpoints across the portfolio.</p>
              </div>
            </div>
          </div>

          <div className="scrollable milestone-list" role="list">
            {milestones.length === 0 ? (
              <div className="empty-state">
                <Zap className="empty-icon" aria-hidden="true" />
                <p>No milestones defined. Add them to showcase transformation goals.</p>
              </div>
            ) : (
              milestones.map((milestone) => {
                const completion = milestone.completion_percentage ?? 0
                return (
                  <article key={milestone.id} role="listitem" className="tech-node milestone-card">
                    <div className="milestone-head">
                      <div
                        className={`milestone-status-dot ${
                          completion >= 80
                            ? 'milestone-status-success'
                            : completion >= 50
                              ? 'milestone-status-progress'
                              : completion >= 25
                                ? 'milestone-status-emerging'
                                : 'milestone-status-pending'
                        }`}
                      />
                      <div>
                        <h4 className="milestone-name">{milestone.name}</h4>
                        <p className="milestone-description">{milestone.description}</p>
                      </div>
                    </div>

                    <div className="progress-container">
                      <div className="progress-bar" style={{ width: `${completion}%` }} />
                    </div>

                    <div className="milestone-metrics">
                      <span>{completion}% complete</span>
                      <span className="milestone-value">{formatCurrency(milestone.business_value_estimate)}</span>
                    </div>

                  <div
                    className={`status-indicator milestone-tag ${
                      milestone.milestone_type === 'sector_complete'
                        ? 'status-info'
                        : milestone.milestone_type === 'cross_sector_integration'
                          ? 'status-warning'
                          : milestone.milestone_type === 'civilization_twin'
                            ? 'status-success'
                            : 'status-info'
                    }`}
                  >
                    {milestone.milestone_type.replace('_', ' ')}
                  </div>
                  </article>
                )
              })
            )}
          </div>

          <div className="panel-footer">
            <div className="ultimate-goal">
              <Brain className="ultimate-icon" aria-hidden="true" />
              <div>
                <h4 className="ultimate-title">Ultimate Goal</h4>
                <p className="ultimate-copy">
                  Meta-civilization simulation to optimise governance, economics, and society at planetary scale.
                </p>
              </div>
            </div>
          </div>
        </section>
      </div>
    </>
  )
}

export default OverviewDashboard
