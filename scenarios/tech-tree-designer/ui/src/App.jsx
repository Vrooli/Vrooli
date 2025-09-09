import React, { useState, useEffect } from 'react'
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
  Database
} from 'lucide-react'

// API service
const API_BASE = `http://localhost:${import.meta.env.VITE_API_PORT || '8080'}/api/v1`

function App() {
  const [sectors, setSectors] = useState([])
  const [milestones, setMilestones] = useState([])
  const [loading, setLoading] = useState(true)
  const [selectedSector, setSelectedSector] = useState(null)
  const [analysisData, setAnalysisData] = useState(null)

  useEffect(() => {
    fetchData()
  }, [])

  const fetchData = async () => {
    try {
      const [sectorsRes, milestonesRes] = await Promise.all([
        fetch(`${API_BASE}/tech-tree/sectors`).catch(() => ({ json: () => ({ sectors: [] }) })),
        fetch(`${API_BASE}/milestones`).catch(() => ({ json: () => ({ milestones: [] }) }))
      ])

      const sectorsData = await sectorsRes.json()
      const milestonesData = await milestonesRes.json()

      setSectors(sectorsData.sectors || [])
      setMilestones(milestonesData.milestones || [])
    } catch (error) {
      console.error('Failed to fetch data:', error)
      // Use fallback demo data
      setSectors(getDemoSectors())
      setMilestones(getDemoMilestones())
    } finally {
      setLoading(false)
    }
  }

  const getDemoSectors = () => [
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
        { name: 'Self-Analytics', progress_percentage: 60, stage_type: 'analytics' }
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
        { name: 'Code Intelligence', progress_percentage: 25, stage_type: 'analytics' }
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
        { name: 'Production Analytics', progress_percentage: 15, stage_type: 'analytics' }
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
        { name: 'Clinical Decision Support', progress_percentage: 10, stage_type: 'analytics' }
      ]
    }
  ]

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
    <div className="strategic-grid three-col w-full h-full">
      {/* Left Sidebar - Sector Overview */}
      <div className="strategic-panel h-full flex flex-col">
        <div className="flex items-center gap-2 mb-6">
          <GitBranch className="w-6 h-6 text-blue-400" />
          <h2 className="text-xl font-bold">Technology Sectors</h2>
        </div>
        
        <div className="scrollable flex-1 space-y-3">
          {sectors.map((sector) => {
            const IconComponent = getSectorIcon(sector.category)
            const isSelected = selectedSector?.id === sector.id
            
            return (
              <div
                key={sector.id}
                className={`tech-node cursor-pointer transition-all ${isSelected ? 'active' : ''}`}
                onClick={() => setSelectedSector(sector)}
                style={{ borderLeftColor: sector.color }}
              >
                <div className="flex items-center gap-3 mb-2">
                  <IconComponent className="w-5 h-5" style={{ color: sector.color }} />
                  <span className="font-medium text-sm">{sector.name}</span>
                </div>
                
                <div className="progress-container mb-2">
                  <div 
                    className="progress-bar" 
                    style={{ width: `${sector.progress_percentage}%` }}
                  />
                </div>
                
                <div className="flex justify-between text-xs text-slate-400">
                  <span>{sector.progress_percentage}% Complete</span>
                  <span className="capitalize">{sector.category}</span>
                </div>
              </div>
            )
          })}
        </div>

        <div className="mt-6 pt-4 border-t border-slate-700">
          <div className="civilization-progress">
            <Zap className="w-6 h-6 mx-auto mb-2 text-blue-400" />
            <h4 className="font-bold mb-1">Civilization Progress</h4>
            <div className="superintelligence-indicator">
              Path to Superintelligence
            </div>
          </div>
        </div>
      </div>

      {/* Main Content - Tech Tree Visualization */}
      <div className="strategic-panel h-full flex flex-col">
        <div className="flex items-center justify-between mb-6">
          <div className="flex items-center gap-2">
            <Network className="w-6 h-6 text-cyan-400" />
            <h1 className="text-2xl font-bold">Tech Tree Designer</h1>
          </div>
          <div className="superintelligence-indicator">
            Strategic Intelligence System
          </div>
        </div>

        {selectedSector ? (
          <div className="flex-1 scrollable">
            <div className="mb-6">
              <div className="flex items-center gap-3 mb-3">
                {React.createElement(getSectorIcon(selectedSector.category), {
                  className: "w-8 h-8",
                  style: { color: selectedSector.color }
                })}
                <div>
                  <h2 className="text-xl font-bold">{selectedSector.name}</h2>
                  <p className="text-slate-400 text-sm">{selectedSector.description}</p>
                </div>
              </div>
              
              <div className="progress-container mb-2">
                <div 
                  className="progress-bar" 
                  style={{ width: `${selectedSector.progress_percentage}%` }}
                />
              </div>
              <div className="text-sm text-slate-400">
                Overall Progress: {selectedSector.progress_percentage}%
              </div>
            </div>

            <div className="space-y-4">
              <h3 className="text-lg font-semibold mb-4">Progression Stages</h3>
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
                  <div key={index} className="tech-node">
                    <div className="flex items-center gap-3 mb-2">
                      <StageIcon className="w-5 h-5 text-slate-400" />
                      <span className="font-medium">{stage.name}</span>
                      <div className={`status-indicator ml-auto ${
                        stage.progress_percentage >= 80 ? 'status-success' :
                        stage.progress_percentage >= 50 ? 'status-warning' :
                        'status-info'
                      }`}>
                        {stage.progress_percentage >= 80 ? 'Advanced' :
                         stage.progress_percentage >= 50 ? 'Active' : 'Foundation'}
                      </div>
                    </div>
                    
                    <div className="progress-container mb-2">
                      <div 
                        className="progress-bar" 
                        style={{ width: `${stage.progress_percentage}%` }}
                      />
                    </div>
                    
                    <div className="text-xs text-slate-400 capitalize">
                      {stage.stage_type.replace('_', ' ')} Stage - {stage.progress_percentage}% Complete
                    </div>
                  </div>
                )
              })}
            </div>
          </div>
        ) : (
          <div className="flex-1 flex items-center justify-center text-center">
            <div>
              <Network className="w-16 h-16 mx-auto mb-4 text-slate-600" />
              <h3 className="text-lg font-semibold mb-2">Select a Technology Sector</h3>
              <p className="text-slate-400 max-w-md">
                Choose a sector from the left panel to explore its progression stages 
                and contribution to the path toward superintelligence.
              </p>
            </div>
          </div>
        )}
      </div>

      {/* Right Sidebar - Strategic Milestones */}
      <div className="strategic-panel h-full flex flex-col">
        <div className="flex items-center gap-2 mb-6">
          <Target className="w-6 h-6 text-green-400" />
          <h2 className="text-xl font-bold">Strategic Milestones</h2>
        </div>
        
        <div className="scrollable flex-1 space-y-4">
          {milestones.map((milestone) => (
            <div key={milestone.id} className="tech-node">
              <div className="flex items-start gap-2 mb-3">
                <div className={`w-3 h-3 rounded-full mt-1 ${
                  milestone.completion_percentage >= 80 ? 'bg-green-400' :
                  milestone.completion_percentage >= 50 ? 'bg-yellow-400' :
                  milestone.completion_percentage >= 25 ? 'bg-blue-400' : 'bg-slate-600'
                }`} />
                <div className="flex-1">
                  <h4 className="font-medium mb-1">{milestone.name}</h4>
                  <p className="text-xs text-slate-400 mb-2">{milestone.description}</p>
                </div>
              </div>
              
              <div className="progress-container mb-2">
                <div 
                  className="progress-bar" 
                  style={{ width: `${milestone.completion_percentage}%` }}
                />
              </div>
              
              <div className="flex justify-between text-xs">
                <span className="text-slate-400">
                  {milestone.completion_percentage}% Complete
                </span>
                <span className="text-green-400 font-mono">
                  {formatCurrency(milestone.business_value_estimate)}
                </span>
              </div>
              
              <div className={`status-indicator mt-2 ${
                milestone.milestone_type === 'sector_complete' ? 'status-info' :
                milestone.milestone_type === 'cross_sector_integration' ? 'status-warning' :
                milestone.milestone_type === 'civilization_twin' ? 'status-success' :
                'status-info'
              }`}>
                {milestone.milestone_type.replace('_', ' ')}
              </div>
            </div>
          ))}
        </div>

        <div className="mt-6 pt-4 border-t border-slate-700">
          <div className="text-center">
            <Brain className="w-8 h-8 mx-auto mb-2 text-purple-400" />
            <h4 className="font-bold mb-1">Ultimate Goal</h4>
            <p className="text-xs text-slate-400">
              Meta-Civilization Simulation capable of optimizing governance, 
              economics, and society at planetary scale.
            </p>
          </div>
        </div>
      </div>
    </div>
  )
}

export default App