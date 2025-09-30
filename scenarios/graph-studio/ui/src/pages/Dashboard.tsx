import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { api } from '../api/client'
import LoadingSpinner from '../components/LoadingSpinner'
import './Dashboard.css'

function Dashboard() {
  const { data: stats, isLoading: statsLoading } = useQuery({
    queryKey: ['stats'],
    queryFn: () => api.getStats(),
  })
  
  const { data: recentGraphs, isLoading: graphsLoading } = useQuery({
    queryKey: ['recentGraphs'],
    queryFn: () => api.getGraphs({ limit: 6 }),
  })
  
  const { data: plugins } = useQuery({
    queryKey: ['plugins'],
    queryFn: () => api.getPlugins(),
  })
  
  if (statsLoading || graphsLoading) {
    return <LoadingSpinner message="Loading dashboard..." />
  }
  
  return (
    <div className="dashboard">
      <h1 className="dashboard-title">Welcome to Graph Studio</h1>
      <p className="dashboard-subtitle">
        Create, validate, and convert all forms of graph-based visualizations
      </p>
      
      <div className="stats-grid">
        <div className="stat-card">
          <div className="stat-icon">📊</div>
          <div className="stat-content">
            <div className="stat-value">{stats?.totalGraphs || 0}</div>
            <div className="stat-label">Total Graphs</div>
          </div>
        </div>
        
        <div className="stat-card">
          <div className="stat-icon">🔌</div>
          <div className="stat-content">
            <div className="stat-value">{(plugins as any)?.data?.length || (plugins as any)?.total || 0}</div>
            <div className="stat-label">Active Plugins</div>
          </div>
        </div>
        
        <div className="stat-card">
          <div className="stat-icon">🔄</div>
          <div className="stat-content">
            <div className="stat-value">{stats?.conversionsToday || 0}</div>
            <div className="stat-label">Conversions Today</div>
          </div>
        </div>
        
        <div className="stat-card">
          <div className="stat-icon">👥</div>
          <div className="stat-content">
            <div className="stat-value">{stats?.activeUsers || 1}</div>
            <div className="stat-label">Active Users</div>
          </div>
        </div>
      </div>
      
      <div className="dashboard-sections">
        <section className="dashboard-section">
          <div className="section-header">
            <h2>Recent Graphs</h2>
            <Link to="/graphs" className="btn btn-outline">
              View All
            </Link>
          </div>
          
          <div className="graphs-grid">
            {recentGraphs?.data?.length === 0 ? (
              <div className="empty-state">
                <p>No graphs yet. Create your first graph!</p>
                <button className="btn btn-primary">Create Graph</button>
              </div>
            ) : (
              recentGraphs?.data?.map((graph: any) => (
                <Link key={graph.id} to={`/graphs/${graph.id}`} className="graph-card">
                  <div className="graph-preview">
                    <svg viewBox="0 0 200 150" fill="none">
                      <rect x="20" y="20" width="60" height="40" rx="4" fill="var(--primary-color)" opacity="0.2" />
                      <rect x="120" y="20" width="60" height="40" rx="4" fill="var(--secondary-color)" opacity="0.2" />
                      <rect x="70" y="90" width="60" height="40" rx="4" fill="var(--accent-color)" opacity="0.2" />
                      <path d="M80 40 L120 40 M50 60 L100 90 M150 60 L100 90" stroke="var(--text-secondary)" strokeWidth="2" opacity="0.5" />
                    </svg>
                  </div>
                  <div className="graph-info">
                    <h3>{graph.name}</h3>
                    <p>{graph.type}</p>
                    <span className="graph-date">
                      {new Date(graph.updated_at).toLocaleDateString()}
                    </span>
                  </div>
                </Link>
              ))
            )}
          </div>
        </section>
        
        <section className="dashboard-section">
          <div className="section-header">
            <h2>Quick Actions</h2>
          </div>
          
          <div className="actions-grid">
            <button className="action-card">
              <div className="action-icon">🧠</div>
              <h3>Create Mind Map</h3>
              <p>Organize ideas hierarchically</p>
            </button>
            
            <button className="action-card">
              <div className="action-icon">🔄</div>
              <h3>Create BPMN Diagram</h3>
              <p>Model business processes</p>
            </button>
            
            <button className="action-card">
              <div className="action-icon">🕸️</div>
              <h3>Create Network Graph</h3>
              <p>Visualize relationships</p>
            </button>
            
            <button className="action-card">
              <div className="action-icon">📝</div>
              <h3>Import from File</h3>
              <p>Load existing diagrams</p>
            </button>
          </div>
        </section>
      </div>
    </div>
  )
}

export default Dashboard