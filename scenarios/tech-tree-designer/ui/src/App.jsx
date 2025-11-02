import React, { useState, useEffect, useMemo, useCallback, useRef } from 'react'
import {
  Brain,
  GitBranch,
  Target,
  TrendingUp,
  Zap,
  Settings,
  BarChart3,
  Network,
  Cpu,
  Database,
  LayoutDashboard,
  PenSquare,
  ExternalLink,
  Maximize2,
  Minimize2,
  X
} from 'lucide-react'
import ReactFlow, { Background, Controls, MiniMap } from 'react-flow-renderer'
import { resolveApiBase, buildApiUrl } from '@vrooli/api-base'
import 'react-flow-renderer/dist/style.css'

const resolveDefaultApiPort = () => {
  const candidates = [
    import.meta.env.VITE_API_PORT,
    import.meta.env.VITE_PROXY_API_PORT,
    import.meta.env.API_PORT
  ]

  for (const candidate of candidates) {
    if (typeof candidate === 'string' && candidate.trim().length > 0) {
      return candidate.trim()
    }
  }

  return '8080'
}

const DEFAULT_API_PORT = resolveDefaultApiPort()

const API_BASE = resolveApiBase({
  explicitUrl: typeof import.meta.env.VITE_API_BASE_URL === 'string'
    ? import.meta.env.VITE_API_BASE_URL.trim()
    : undefined,
  defaultPort: DEFAULT_API_PORT,
  appendSuffix: true
})

const apiUrl = (path) => buildApiUrl(path, { baseUrl: API_BASE, appendSuffix: false })

const stageTypePalette = {
  foundation: '#10b981',
  operational: '#38bdf8',
  analytics: '#facc15',
  integration: '#a855f7',
  digital_twin: '#f97316'
}

const stageTypeLabel = {
  foundation: 'Foundation',
  operational: 'Operational',
  analytics: 'Analytics',
  integration: 'Integration',
  digital_twin: 'Digital Twin'
}

function App() {
  const [sectors, setSectors] = useState([])
  const [milestones, setMilestones] = useState([])
  const [dependencies, setDependencies] = useState([])
  const [loading, setLoading] = useState(true)
  const [selectedSector, setSelectedSector] = useState(null)
  const [viewMode, setViewMode] = useState('overview')
  const [graphNotice, setGraphNotice] = useState(null)
  const [canFullscreen, setCanFullscreen] = useState(true)
  const [isNativeFullscreen, setIsNativeFullscreen] = useState(false)
  const [isLayoutFullscreen, setIsLayoutFullscreen] = useState(false)
  const techTreeCanvasRef = useRef(null)
  const isFullscreen = isNativeFullscreen || isLayoutFullscreen

  useEffect(() => {
    fetchData()
  }, [])

  const safeFetch = useCallback(async (path, fallback) => {
    try {
      const response = await fetch(apiUrl(path))
      if (!response.ok) {
        throw new Error(`Request failed with status ${response.status}`)
      }
      return await response.json()
    } catch (error) {
      console.warn(`Falling back for ${path}:`, error)
      return typeof fallback === 'function' ? fallback() : fallback
    }
  }, [])

  const attachStageMetadata = useCallback((sectorList) => {
    if (!Array.isArray(sectorList)) {
      return []
    }

    return sectorList.map((sector, sectorIndex) => {
      const baseX = typeof sector.position_x === 'number' ? sector.position_x : 220 + sectorIndex * 240
      const baseY = typeof sector.position_y === 'number' ? sector.position_y : 80

      const stages = (sector.stages || []).map((stage, stageIndex) => ({
        ...stage,
        id: stage.id || `${sector.id}-stage-${stageIndex + 1}`,
        position_x: typeof stage.position_x === 'number' ? stage.position_x : baseX,
        position_y: typeof stage.position_y === 'number' ? stage.position_y : baseY + stageIndex * 110,
        stage_type: stage.stage_type || stage.stageType || 'foundation'
      }))

      return {
        ...sector,
        stages,
        position_x: baseX,
        position_y: baseY
      }
    })
  }, [])

  const generateLinearDependenciesFromSectors = useCallback((sectorList) => {
    const inferred = []

    sectorList.forEach((sector) => {
      const stages = Array.isArray(sector.stages) ? sector.stages : []
      for (let index = 1; index < stages.length; index += 1) {
        const current = stages[index]
        const previous = stages[index - 1]
        if (!current || !previous) {
          continue
        }

        const currentId = current.id || `${sector.id}-stage-${index + 1}`
        const previousId = previous.id || `${sector.id}-stage-${index}`

        inferred.push({
          dependency: {
            id: `${previousId}->${currentId}`,
            dependent_stage_id: currentId,
            prerequisite_stage_id: previousId,
            dependency_type: 'progression',
            dependency_strength: Math.min(1, 0.5 + index * 0.1),
            description: `${current.name} builds on ${previous.name}`
          },
          dependent_name: current.name,
          prerequisite_name: previous.name
        })
      }
    })

    return inferred
  }, [])

  const fetchData = async () => {
    try {
      const [sectorsData, milestonesData, dependenciesData] = await Promise.all([
        safeFetch('/tech-tree/sectors', { sectors: [] }),
        safeFetch('/milestones', { milestones: [] }),
        safeFetch('/dependencies', { dependencies: [] })
      ])

      const normalizedSectors = attachStageMetadata(sectorsData.sectors || [])
      const fallbackSectors = getDemoSectors(attachStageMetadata)
      const sectorPayload = normalizedSectors.length ? normalizedSectors : fallbackSectors

      setSectors(sectorPayload)
      setMilestones((milestonesData.milestones || []).length ? milestonesData.milestones : getDemoMilestones())

      if ((dependenciesData.dependencies || []).length) {
        setDependencies(dependenciesData.dependencies)
        setGraphNotice(null)
      } else {
        setDependencies(generateLinearDependenciesFromSectors(sectorPayload))
        setGraphNotice('Live dependency data unavailable. Showing inferred progression links so the designer remains usable.')
      }
    } catch (error) {
      console.error('Failed to fetch data:', error)
      const demoSectors = getDemoSectors(attachStageMetadata)
      setSectors(demoSectors)
      setMilestones(getDemoMilestones())
      setDependencies(generateLinearDependenciesFromSectors(demoSectors))
      setGraphNotice('Operating in demo mode due to API connectivity issues. Graph interactions use sample data.')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    if (!loading && sectors.length > 0 && !selectedSector) {
      setSelectedSector(sectors[0])
    }
  }, [loading, sectors, selectedSector])

  const insightMetrics = useMemo(() => {
    if (!sectors.length && !milestones.length) {
      return {
        averageSectorProgress: 0,
        activeMilestones: 0,
        totalPotentialValue: 0
      }
    }

    const averageSectorProgress = sectors.length
      ? Math.round(
          sectors.reduce((sum, sector) => sum + (sector.progress_percentage || 0), 0) /
            sectors.length
        )
      : 0

    const activeMilestones = milestones.filter(
      (milestone) => (milestone.completion_percentage || 0) >= 50
    ).length

    const totalPotentialValue = milestones.reduce(
      (sum, milestone) => sum + (milestone.business_value_estimate || 0),
      0
    )

    return { averageSectorProgress, activeMilestones, totalPotentialValue }
  }, [milestones, sectors])

  const getDemoSectors = (augmenter) => {
    const baseSectors = [
      {
        id: '1',
        name: 'Personal Productivity',
        category: 'individual',
        description: 'Individual tools for task management, note-taking, time tracking',
        progress_percentage: 75,
        color: '#10B981',
        stages: [
          { name: 'Task Management', progress_percentage: 85, stage_type: 'foundation' },
          { name: 'Personal Automation', progress_percentage: 70, stage_type: 'operational' },
          { name: 'Self-Analytics', progress_percentage: 60, stage_type: 'analytics' },
          { name: 'Life Integration', progress_percentage: 45, stage_type: 'integration' },
          { name: 'Personal Digital Twin', progress_percentage: 30, stage_type: 'digital_twin' }
        ]
      },
      {
        id: '2',
        name: 'Software Engineering',
        category: 'software',
        description: 'Development tools and platforms that accelerate all other domains',
        progress_percentage: 45,
        color: '#EC4899',
        stages: [
          { name: 'Development Tools', progress_percentage: 65, stage_type: 'foundation' },
          { name: 'DevOps', progress_percentage: 50, stage_type: 'operational' },
          { name: 'Code Intelligence', progress_percentage: 25, stage_type: 'analytics' },
          { name: 'Platform Integration', progress_percentage: 15, stage_type: 'integration' },
          { name: 'Engineering Digital Twin', progress_percentage: 5, stage_type: 'digital_twin' }
        ]
      },
      {
        id: '3',
        name: 'Manufacturing Systems',
        category: 'manufacturing',
        description: 'Complete manufacturing ecosystem from design to production',
        progress_percentage: 25,
        color: '#3B82F6',
        stages: [
          { name: 'Product Lifecycle Management', progress_percentage: 40, stage_type: 'foundation' },
          { name: 'Manufacturing Execution', progress_percentage: 20, stage_type: 'operational' },
          { name: 'Production Analytics', progress_percentage: 15, stage_type: 'analytics' },
          { name: 'Smart Factory Integration', progress_percentage: 10, stage_type: 'integration' },
          { name: 'Manufacturing Digital Twin', progress_percentage: 5, stage_type: 'digital_twin' }
        ]
      },
      {
        id: '4',
        name: 'Healthcare Systems',
        category: 'healthcare',
        description: 'Healthcare technology from records to population health twins',
        progress_percentage: 20,
        color: '#EF4444',
        stages: [
          { name: 'Electronic Health Records', progress_percentage: 35, stage_type: 'foundation' },
          { name: 'Clinical Operations', progress_percentage: 15, stage_type: 'operational' },
          { name: 'Clinical Decision Support', progress_percentage: 10, stage_type: 'analytics' },
          { name: 'Health Information Exchange', progress_percentage: 8, stage_type: 'integration' },
          { name: 'Population Health Twin', progress_percentage: 4, stage_type: 'digital_twin' }
        ]
      }
    ]

    return typeof augmenter === 'function' ? augmenter(baseSectors) : baseSectors
  }

  const getDemoMilestones = () => [
    {
      id: '1',
      name: 'Individual Productivity Mastery',
      description: 'Complete personal productivity and self-optimization capabilities',
      completion_percentage: 75,
      business_value_estimate: 10000000,
      milestone_type: 'sector_complete'
    },
    {
      id: '2',
      name: 'Core Sector Digital Twins',
      description: 'Manufacturing, Healthcare, Finance digital twins operational',
      completion_percentage: 25,
      business_value_estimate: 1000000000,
      milestone_type: 'cross_sector_integration'
    },
    {
      id: '3',
      name: 'Civilization Digital Twin',
      description: 'Complete society-scale simulation integrating all sectors',
      completion_percentage: 5,
      business_value_estimate: 100000000000,
      milestone_type: 'civilization_twin'
    }
  ]

  const getSectorIcon = (category) => {
    const icons = {
      individual: Target,
      software: Cpu,
      manufacturing: Settings,
      healthcare: Database,
      finance: TrendingUp,
      education: Brain
    }
    return icons[category] || Network
  }

  const formatCurrency = (value) => {
    if (value >= 1e12) return `$${(value / 1e12).toFixed(1)}T`
    if (value >= 1e9) return `$${(value / 1e9).toFixed(1)}B`
    if (value >= 1e6) return `$${(value / 1e6).toFixed(1)}M`
    return `$${value.toLocaleString()}`
  }

  const graphStudioUrl = useMemo(() => {
    if (typeof window === 'undefined') {
      return ''
    }

    try {
      const current = new URL(window.location.href)
      if (current.href.includes('tech-tree-designer')) {
        return current.href.replace('tech-tree-designer', 'graph-studio')
      }
      return `${current.origin}/apps/graph-studio/proxy/`
    } catch (error) {
      console.warn('Unable to resolve Graph Studio URL:', error)
      return '/apps/graph-studio/proxy/'
    }
  }, [])

  useEffect(() => {
    if (typeof document === 'undefined') {
      return undefined
    }

    const fullscreenSupported = Boolean(
      document.fullscreenEnabled ||
        document.webkitFullscreenEnabled ||
        document.mozFullScreenEnabled ||
        document.msFullscreenEnabled ||
        document.fullscreenEnabled === undefined
    )

    const fallbackAvailable = true
    setCanFullscreen(fullscreenSupported || fallbackAvailable)

    const handleFullscreenChange = () => {
      const fullscreenElement =
        document.fullscreenElement ||
        document.webkitFullscreenElement ||
        document.mozFullScreenElement ||
        document.msFullscreenElement

      const isCanvasFullscreen = Boolean(
        fullscreenElement &&
          techTreeCanvasRef.current &&
          techTreeCanvasRef.current.contains(fullscreenElement)
      )

      setIsNativeFullscreen(isCanvasFullscreen)

      if (
        fullscreenElement &&
        techTreeCanvasRef.current &&
        !techTreeCanvasRef.current.contains(fullscreenElement)
      ) {
        setIsLayoutFullscreen(false)
      }
    }

    const events = ['fullscreenchange', 'webkitfullscreenchange', 'mozfullscreenchange', 'MSFullscreenChange']
    events.forEach((event) => document.addEventListener(event, handleFullscreenChange))
    handleFullscreenChange()

    return () => {
      events.forEach((event) => document.removeEventListener(event, handleFullscreenChange))
    }
  }, [])

  const toggleFullscreen = useCallback(() => {
    if (typeof document === 'undefined' || !techTreeCanvasRef.current) {
      return
    }

    const element = techTreeCanvasRef.current
    const requestFullscreen =
      element.requestFullscreen ||
      element.webkitRequestFullscreen ||
      element.mozRequestFullScreen ||
      element.msRequestFullscreen
    const exitFullscreen =
      document.exitFullscreen ||
      document.webkitExitFullscreen ||
      document.mozCancelFullScreen ||
      document.msExitFullscreen

    if (isNativeFullscreen) {
      if (typeof exitFullscreen === 'function') {
        exitFullscreen.call(document).catch((error) => {
          console.warn('Failed to exit fullscreen mode:', error)
        })
      } else {
        console.warn('Fullscreen exit API unavailable; clearing immersive layout state')
        setIsLayoutFullscreen(false)
      }
      return
    }

    if (isLayoutFullscreen) {
      setIsLayoutFullscreen(false)
      return
    }

    const enterNativeFullscreen = async () => {
      if (typeof requestFullscreen !== 'function') {
        throw new Error('Fullscreen API unavailable on element')
      }

      const existingFullscreenElement =
        document.fullscreenElement ||
        document.webkitFullscreenElement ||
        document.mozFullScreenElement ||
        document.msFullscreenElement

      if (
        existingFullscreenElement &&
        existingFullscreenElement !== element &&
        typeof exitFullscreen === 'function'
      ) {
        await exitFullscreen.call(document)
      }

      await requestFullscreen.call(element)
    }

    enterNativeFullscreen().catch((error) => {
      console.warn('Native fullscreen unavailable; enabling immersive fallback:', error)
      setIsLayoutFullscreen(true)
    })
  }, [isNativeFullscreen, isLayoutFullscreen])

  useEffect(() => {
    if (typeof document === 'undefined') {
      return undefined
    }

    const immersiveClass = 'tech-tree-immersive-active'
    const { body } = document

    if (isLayoutFullscreen) {
      body.classList.add(immersiveClass)
    } else {
      body.classList.remove(immersiveClass)
    }

    return () => {
      body.classList.remove(immersiveClass)
    }
  }, [isLayoutFullscreen])

  useEffect(() => {
    if (!isLayoutFullscreen || typeof window === 'undefined') {
      return undefined
    }

    const handleKeydown = (event) => {
      if (event.key === 'Escape' || event.key === 'Esc') {
        setIsLayoutFullscreen(false)
      }
    }

    window.addEventListener('keydown', handleKeydown)

    return () => {
      window.removeEventListener('keydown', handleKeydown)
    }
  }, [isLayoutFullscreen])

  if (loading) {
    return (
      <div className="w-full h-full flex items-center justify-center">
        <div className="text-center">
          <div className="loading-spinner mb-4"></div>
          <h3>Loading Strategic Intelligence...</h3>
          <p className="text-slate-400">Initializing civilization roadmap</p>
        </div>
      </div>
    )
  }

  return (
    <main className="app-shell">
      <header className="app-header">
        <div className="app-heading">
          <h1 className="app-title">Tech Tree Designer</h1>
          <p className="app-subtitle">
            Civilizational technology intelligence that maps sector momentum to strategic milestones.
          </p>
        </div>
        <div className="app-meta" role="status" aria-label="Scenario status">
          <span className="status-pill status-pill--active">Live Scenario</span>
          <span className="status-pill status-pill--ghost">Scenario • tech-tree-designer</span>
        </div>
      </header>

      <section className="view-toggle" role="tablist" aria-label="Primary views">
        <button
          type="button"
          role="tab"
          aria-selected={viewMode === 'overview'}
          className={`view-toggle__button ${viewMode === 'overview' ? 'is-active' : ''}`}
          onClick={() => setViewMode('overview')}
        >
          <LayoutDashboard className="view-toggle__icon" aria-hidden="true" />
          Overview Dashboard
        </button>
        <button
          type="button"
          role="tab"
          aria-selected={viewMode === 'designer'}
          className={`view-toggle__button ${viewMode === 'designer' ? 'is-active' : ''}`}
          onClick={() => setViewMode('designer')}
        >
          <PenSquare className="view-toggle__icon" aria-hidden="true" />
          Tech Tree Designer
        </button>
      </section>

      {viewMode === 'overview' ? (
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
            {/* Left Sidebar - Sector Overview */}
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
                  onClick={() => setSelectedSector(sectors[0] || null)}
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
                        onClick={() => setSelectedSector(sector)}
                        style={{ borderLeftColor: sector.color }}
                        tabIndex={0}
                        onKeyDown={(event) => {
                          if (event.key === 'Enter' || event.key === ' ') {
                            event.preventDefault()
                            setSelectedSector(sector)
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

            {/* Main Content - Tech Tree Visualization */}
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
                    <h3 className="stage-title">Progression Stages</h3>
                    <div className="stage-grid">
                      {selectedSector.stages?.map((stage, index) => {
                        const stageIcons = {
                          foundation: Database,
                          operational: Settings,
                          analytics: BarChart3,
                          integration: Network,
                          digital_twin: Brain
                        }
                        const StageIcon = stageIcons[stage.stage_type] || Target

                        return (
                          <article key={`${stage.id || stage.name}-${index}`} className="tech-node stage-card">
                            <div className="stage-head">
                              <StageIcon className="stage-icon" aria-hidden="true" />
                              <div>
                                <span className="stage-name">{stage.name}</span>
                                <span className="stage-type">{stage.stage_type.replace('_', ' ')}</span>
                              </div>
                              <div
                                className={`status-indicator ${
                                  stage.progress_percentage >= 80
                                    ? 'status-success'
                                    : stage.progress_percentage >= 50
                                      ? 'status-warning'
                                      : 'status-info'
                                }`}
                              >
                                {stage.progress_percentage >= 80
                                  ? 'Advanced'
                                  : stage.progress_percentage >= 50
                                    ? 'Active'
                                    : 'Foundation'}
                              </div>
                            </div>

                            <div className="progress-container">
                              <div className="progress-bar" style={{ width: `${stage.progress_percentage}%` }} />
                            </div>

                            <p className="stage-footnote">{stage.progress_percentage}% complete</p>
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

            {/* Right Sidebar - Strategic Milestones */}
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
                  milestones.map((milestone) => (
                    <article key={milestone.id} role="listitem" className="tech-node milestone-card">
                      <div className="milestone-head">
                        <div
                          className={`milestone-status-dot ${
                            milestone.completion_percentage >= 80
                              ? 'milestone-status-success'
                              : milestone.completion_percentage >= 50
                                ? 'milestone-status-progress'
                                : milestone.completion_percentage >= 25
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
                        <div className="progress-bar" style={{ width: `${milestone.completion_percentage}%` }} />
                      </div>

                      <div className="milestone-metrics">
                        <span>{milestone.completion_percentage}% complete</span>
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
                  ))
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
      ) : (
        <TechTreeCanvas
          sectors={sectors}
          dependencies={dependencies}
          graphStudioUrl={graphStudioUrl}
          graphNotice={graphNotice}
          techTreeCanvasRef={techTreeCanvasRef}
          isFullscreen={isFullscreen}
          isLayoutFullscreen={isLayoutFullscreen}
          toggleFullscreen={toggleFullscreen}
          canFullscreen={canFullscreen}
        />
      )}
    </main>
  )
}

function TechTreeCanvas({
  sectors,
  dependencies,
  graphStudioUrl,
  graphNotice,
  techTreeCanvasRef,
  isFullscreen,
  isLayoutFullscreen,
  toggleFullscreen,
  canFullscreen
}) {
  const [selectedStageId, setSelectedStageId] = useState(null)
  const [isCompactLayout, setIsCompactLayout] = useState(false)

  const { nodes, edges, stageLookup } = useMemo(() => {
    const stageMap = new Map()
    const stageNodes = []

    sectors.forEach((sector, sectorIndex) => {
      const baseX = typeof sector.position_x === 'number' ? sector.position_x : sectorIndex * 260
      const baseY = typeof sector.position_y === 'number' ? sector.position_y : 120 + sectorIndex * 45

      ;(sector.stages || []).forEach((stage, stageIndex) => {
        const stageId = stage.id || `${sector.id}-stage-${stageIndex + 1}`
        const positionX = typeof stage.position_x === 'number' ? stage.position_x : baseX + stageIndex * 48
        const positionY = typeof stage.position_y === 'number' ? stage.position_y : baseY + stageIndex * 90
        const progress = typeof stage.progress_percentage === 'number' ? stage.progress_percentage : 0
        const stageType = stage.stage_type || 'foundation'

        stageMap.set(stageId, {
          ...stage,
          sector,
          id: stageId
        })

        stageNodes.push({
          id: stageId,
          position: { x: positionX, y: positionY },
          data: {
            label: stage.name,
            progress,
            type: stageType,
            sectorName: sector.name,
            sectorColor: sector.color
          },
          type: 'default',
          style: {
            background: 'rgba(15, 23, 42, 0.92)',
            color: '#f8fafc',
            border: `1px solid ${sector.color || 'rgba(59,130,246,0.6)'}`,
            borderRadius: 12,
            padding: 12,
            fontSize: 12,
            boxShadow: '0 12px 24px rgba(2, 6, 23, 0.45)',
            minWidth: 200
          }
        })
      })
    })

    const edgeList = (dependencies || []).reduce((accumulator, item) => {
      const dep = item?.dependency || item
      const source = dep?.prerequisite_stage_id || dep?.prerequisiteStageId
      const target = dep?.dependent_stage_id || dep?.dependentStageId

      if (!stageMap.has(source) || !stageMap.has(target)) {
        return accumulator
      }

      accumulator.push({
        id: dep?.id || `${source}->${target}`,
        source,
        target,
        label: dep?.dependency_type ? dep.dependency_type.replace('_', ' ') : 'dependency',
        animated: (dep?.dependency_strength || 0) >= 0.85,
        style: {
          stroke: 'rgba(14, 165, 233, 0.45)',
          strokeWidth: 2
        },
        labelBgPadding: [4, 2],
        labelStyle: {
          fill: '#bae6fd',
          fontSize: 10,
          textTransform: 'uppercase',
          letterSpacing: 0.6
        }
      })

      return accumulator
    }, [])

    return { nodes: stageNodes, edges: edgeList, stageLookup: stageMap }
  }, [sectors, dependencies])

  const selectedStage = selectedStageId ? stageLookup.get(selectedStageId) : null
  const stageDetailTitleId = selectedStage
    ? (() => {
        const rawId = String(selectedStage.id || selectedStage.name || 'stage')
          .toLowerCase()
          .replace(/[^a-z0-9-]+/g, '-')
          .replace(/^-+|-+$/g, '')
        const normalizedId = rawId.length ? rawId : 'selected-stage'
        return `stage-detail-${normalizedId}`
      })()
    : undefined
  const shouldShowDialog = Boolean(isFullscreen || isLayoutFullscreen || isCompactLayout)

  const handleNodeClick = useCallback((_, node) => {
    setSelectedStageId(node?.id || null)
  }, [])

  const handleCloseStageDetail = useCallback(() => {
    setSelectedStageId(null)
  }, [])

  useEffect(() => {
    if (typeof window === 'undefined') {
      return undefined
    }

    const mediaQuery = window.matchMedia('(max-width: 1120px)')
    const handleChange = (event) => {
      setIsCompactLayout(event.matches)
    }

    setIsCompactLayout(mediaQuery.matches)

    if (typeof mediaQuery.addEventListener === 'function') {
      mediaQuery.addEventListener('change', handleChange)
      return () => mediaQuery.removeEventListener('change', handleChange)
    }

    mediaQuery.addListener(handleChange)
    return () => mediaQuery.removeListener(handleChange)
  }, [])

  useEffect(() => {
    if (!shouldShowDialog || !selectedStageId || typeof document === 'undefined') {
      return undefined
    }

    const handleKeyDown = (event) => {
      if (event.key === 'Escape') {
        handleCloseStageDetail()
      }
    }

    document.addEventListener('keydown', handleKeyDown)
    return () => {
      document.removeEventListener('keydown', handleKeyDown)
    }
  }, [handleCloseStageDetail, selectedStageId, shouldShowDialog])

  const renderScenarioMappings = (stage) => {
    const mappings = stage?.scenario_mappings || stage?.scenarioMappings || []
    if (!mappings.length) {
      return <li className="stage-detail__empty">No scenario mappings recorded yet.</li>
    }

    return mappings.slice(0, 6).map((mapping) => (
      <li key={mapping.id || mapping.scenario_name} className="stage-detail__mapping">
        <span className="stage-detail__mapping-name">{mapping.scenario_name}</span>
        <span className="stage-detail__mapping-status">{mapping.completion_status}</span>
      </li>
    ))
  }

  const renderStageDetailContent = (stage, titleId) => (
    <div className="stage-detail">
      <header className="stage-detail__header">
        <h3 id={titleId}>{stage.name}</h3>
        <span
          className="stage-detail__badge"
          style={{ background: stageTypePalette[stage.stage_type || 'foundation'] }}
        >
          {stageTypeLabel[stage.stage_type || 'foundation']}
        </span>
      </header>
      <p className="stage-detail__sector">{stage.sector?.name}</p>
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
        <ul>{renderScenarioMappings(stage)}</ul>
      </div>

      {Array.isArray(stage.examples) ? (
        <div className="stage-detail__stack">
          <h4>Examples</h4>
          <ul>
            {stage.examples.map((example) => (
              <li key={example}>{example}</li>
            ))}
          </ul>
        </div>
      ) : null}
    </div>
  )

  const sidebarClassName = ['tech-tree-sidebar', shouldShowDialog ? 'tech-tree-sidebar--hidden' : null]
    .filter(Boolean)
    .join(' ')

  return (
    <section className="tech-tree-designer" aria-label="Interactive tech tree canvas">
      <div
        ref={techTreeCanvasRef}
        className={[
          'tech-tree-canvas',
          isFullscreen && 'tech-tree-canvas--fullscreen',
          isLayoutFullscreen && 'tech-tree-canvas--immersive'
        ]
          .filter(Boolean)
          .join(' ')}
        role="application"
        aria-label="Tech tree graph"
      >
        <div className="tech-tree-canvas__header">
          <div>
            <h2>Tech Tree Graph</h2>
            <p>Navigate nodes, inspect dependencies, and hand off to Graph Studio for structural edits.</p>
          </div>
          <div className="tech-tree-actions">
            <button
              type="button"
              className={`canvas-fullscreen-button${isFullscreen ? ' is-active' : ''}`}
              onClick={toggleFullscreen}
              aria-pressed={isFullscreen}
              aria-label={isFullscreen ? 'Exit full screen for tech tree graph' : 'Enter full screen for tech tree graph'}
              disabled={!canFullscreen}
            >
              {isFullscreen ? (
                <Minimize2 className="canvas-fullscreen-button__icon" aria-hidden="true" />
              ) : (
                <Maximize2 className="canvas-fullscreen-button__icon" aria-hidden="true" />
              )}
              <span>{isFullscreen ? 'Exit full screen' : 'Full screen'}</span>
            </button>
            <a
              className="graph-studio-link"
              href={graphStudioUrl || '#'}
              target="_blank"
              rel="noreferrer noopener"
            >
              <ExternalLink className="graph-studio-link__icon" aria-hidden="true" />
              Open in Graph Studio
            </a>
          </div>
        </div>

        {graphNotice ? (
          <div className="graph-notice" role="status">
            <span className="graph-notice__bullet" aria-hidden="true">•</span>
            {graphNotice}
          </div>
        ) : null}

        <div className="tech-tree-flow">
          <ReactFlow
            nodes={nodes}
            edges={edges}
            onNodeClick={handleNodeClick}
            onPaneClick={handleCloseStageDetail}
            fitView
            fitViewOptions={{ padding: 0.2, minZoom: 0.4, maxZoom: 1.4 }}
          >
            <MiniMap
              className="tech-tree-minimap"
              nodeColor={(node) => node?.data?.sectorColor || '#1e293b'}
              maskColor="rgba(15, 23, 42, 0.65)"
              pannable
              zoomable
            />
            <Controls showInteractive={false} />
            <Background gap={18} color="rgba(30, 41, 59, 0.35)" />
          </ReactFlow>
        </div>

        {shouldShowDialog && selectedStage ? (
          <div
            className="stage-dialog-overlay"
            role="dialog"
            aria-modal="true"
            aria-labelledby={stageDetailTitleId}
            onClick={handleCloseStageDetail}
          >
            <div
              className="stage-dialog"
              role="document"
              onClick={(event) => event.stopPropagation()}
            >
              <button
                type="button"
                className="stage-dialog__close"
                onClick={handleCloseStageDetail}
                aria-label="Close stage details"
              >
                <X className="stage-dialog__close-icon" aria-hidden="true" />
              </button>
              {renderStageDetailContent(selectedStage, stageDetailTitleId)}
            </div>
          </div>
        ) : null}
      </div>

      <aside
        className={sidebarClassName}
        aria-label="Stage details"
        aria-hidden={shouldShowDialog}
      >
        {selectedStage ? (
          renderStageDetailContent(selectedStage, stageDetailTitleId)
        ) : (
          <div className="stage-placeholder">
            <p>Select a node in the graph to inspect sector context and linked scenarios.</p>
            <p className="stage-placeholder__hint">Tip: Use the minimap or scroll wheel to navigate expansive trees.</p>
          </div>
        )}
      </aside>
    </section>
  )
}

export default App
