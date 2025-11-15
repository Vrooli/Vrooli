import React from 'react'
import { BarChart3, Brain, Database, GitBranch, Network, Settings, Target, Zap } from 'lucide-react'
import { getSectorIcon } from '../utils/icons'
import { formatCurrency } from '../utils/formatters'
import { formatStageTypeLabel } from '../utils/constants'
import type { Sector, StrategicMilestone } from '../types/techTree'

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
  onRequestNewStage
}) => {
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
        <div className="insight-card">
          <div className="insight-label">Potential Strategic Value</div>
          <div className="insight-value">{formatCurrency(insightMetrics.totalPotentialValue)}</div>
          <div className="insight-subtext">Cumulative milestone value</div>
        </div>
      </section>

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
            <button
              type="button"
              className="panel-reset"
              onClick={() => onSelectSector(sectors[0] || null)}
              disabled={!sectors.length}
            >
              Reset focus
            </button>
          </div>

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
