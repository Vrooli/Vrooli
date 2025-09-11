import { useState, useEffect } from 'react'
import DiffPanel from './components/DiffPanel'
import './styles/globals.css'
import './styles/App.css'

type Tool = 'diff' | 'search' | 'transform' | 'extract' | 'analyze' | 'pipeline'

interface ConnectionStatus {
  connected: boolean
  loading: boolean
}

function App() {
  const [activeTool, setActiveTool] = useState<Tool>('diff')
  const [connectionStatus, setConnectionStatus] = useState<ConnectionStatus>({
    connected: false,
    loading: true
  })

  useEffect(() => {
    checkApiConnection()
    const interval = setInterval(checkApiConnection, 30000) // Check every 30 seconds
    return () => clearInterval(interval)
  }, [])

  const checkApiConnection = async () => {
    try {
      const response = await fetch('/api/health', { 
        method: 'GET',
        signal: AbortSignal.timeout(5000) // 5 second timeout
      })
      setConnectionStatus({
        connected: response.ok,
        loading: false
      })
    } catch (error) {
      setConnectionStatus({
        connected: false,
        loading: false
      })
    }
  }

  const handleApiDocs = () => {
    window.open('/api/docs', '_blank')
  }

  const renderTool = () => {
    switch (activeTool) {
      case 'diff':
        return <DiffPanel />
      case 'search':
        return <div className="tool-panel">Search functionality coming soon...</div>
      case 'transform':
        return <div className="tool-panel">Transform functionality coming soon...</div>
      case 'extract':
        return <div className="tool-panel">Extract functionality coming soon...</div>
      case 'analyze':
        return <div className="tool-panel">Analyze functionality coming soon...</div>
      case 'pipeline':
        return <div className="tool-panel">Pipeline functionality coming soon...</div>
      default:
        return <DiffPanel />
    }
  }

  const getStatusDotClass = () => {
    if (connectionStatus.loading) return 'status-dot'
    return connectionStatus.connected ? 'status-dot connected' : 'status-dot disconnected'
  }

  const getStatusText = () => {
    if (connectionStatus.loading) return 'Connecting...'
    return connectionStatus.connected ? 'Connected' : 'Disconnected'
  }

  return (
    <div className="app-container">
      {/* Header */}
      <header className="app-header">
        <div className="logo">
          <span className="logo-icon">📝</span>
          <h1>Text Tools</h1>
        </div>
        
        <nav className="nav-tabs">
          {(['diff', 'search', 'transform', 'extract', 'analyze', 'pipeline'] as Tool[]).map((tool) => (
            <button
              key={tool}
              className={`nav-tab ${activeTool === tool ? 'active' : ''}`}
              onClick={() => setActiveTool(tool)}
            >
              {tool.charAt(0).toUpperCase() + tool.slice(1)}
            </button>
          ))}
        </nav>
        
        <div className="header-actions">
          <button className="btn-secondary" onClick={handleApiDocs}>
            API Docs
          </button>
          <span className="status-indicator">
            <span className={getStatusDotClass()}></span>
            <span className="status-text">{getStatusText()}</span>
          </span>
        </div>
      </header>

      {/* Main Content */}
      <main className="app-main">
        {renderTool()}
      </main>

      {/* Footer */}
      <footer className="app-footer">
        <div className="footer-content">
          <span className="footer-text">Text Tools v1.0.0</span>
          <span className="footer-separator">|</span>
          <span className="keyboard-shortcuts">
            Press <kbd>Ctrl</kbd>+<kbd>K</kbd> for quick search
          </span>
        </div>
      </footer>
    </div>
  )
}

export default App