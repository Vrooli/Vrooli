import React, { useState, useEffect } from 'react';
import ScenarioGrid from './components/ScenarioGrid';
import MCPStatus from './components/MCPStatus';
import AddMCPModal from './components/AddMCPModal';
import { RefreshCw, Activity, Server, GitBranch } from 'lucide-react';
import axios from 'axios';
import './App.css';

function App() {
  const [scenarios, setScenarios] = useState([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState('all');
  const [selectedScenario, setSelectedScenario] = useState(null);
  const [showAddMCP, setShowAddMCP] = useState(false);
  const [stats, setStats] = useState({
    total: 0,
    withMCP: 0,
    active: 0,
    pending: 0
  });

  useEffect(() => {
    loadScenarios();
    const interval = setInterval(loadScenarios, 30000);
    return () => clearInterval(interval);
  }, []);

  const loadScenarios = async () => {
    try {
      setLoading(true);
      const response = await axios.get('/api/v1/mcp/endpoints');
      const data = response.data;
      
      setScenarios(data.scenarios || []);
      setStats({
        total: data.scenarios.length,
        withMCP: data.scenarios.filter(s => s.hasMCP).length,
        active: data.scenarios.filter(s => s.status === 'active').length,
        pending: data.scenarios.filter(s => s.status === 'pending').length
      });
    } catch (error) {
      console.error('Failed to load scenarios:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleAddMCP = async (scenario) => {
    setSelectedScenario(scenario);
    setShowAddMCP(true);
  };

  const handleMCPAdded = async (scenario, sessionId) => {
    setShowAddMCP(false);
    await loadScenarios();
    
    // Open Claude-code session in new tab if session ID provided
    if (sessionId) {
      window.open(`/api/v1/mcp/sessions/${sessionId}`, '_blank');
    }
  };

  const filteredScenarios = scenarios.filter(scenario => {
    if (filter === 'all') return true;
    if (filter === 'with-mcp') return scenario.hasMCP;
    if (filter === 'without-mcp') return !scenario.hasMCP;
    if (filter === 'active') return scenario.status === 'active';
    return true;
  });

  return (
    <div className="app">
      <header className="header">
        <div className="header-left">
          <h1 className="title">
            <Server className="icon" />
            Scenario to MCP
          </h1>
          <div className="subtitle">Model Context Protocol Management</div>
        </div>
        
        <div className="header-right">
          <button 
            className="refresh-btn"
            onClick={loadScenarios}
            disabled={loading}
          >
            <RefreshCw className={loading ? 'spinning' : ''} />
            Refresh
          </button>
        </div>
      </header>

      <div className="stats-bar">
        <div className="stat">
          <div className="stat-value">{stats.total}</div>
          <div className="stat-label">Total Scenarios</div>
        </div>
        <div className="stat">
          <div className="stat-value accent">{stats.withMCP}</div>
          <div className="stat-label">With MCP</div>
        </div>
        <div className="stat">
          <div className="stat-value success">{stats.active}</div>
          <div className="stat-label">Active</div>
        </div>
        <div className="stat">
          <div className="stat-value warning">{stats.pending}</div>
          <div className="stat-label">Pending</div>
        </div>
      </div>

      <div className="controls">
        <div className="filter-tabs">
          <button 
            className={`tab ${filter === 'all' ? 'active' : ''}`}
            onClick={() => setFilter('all')}
          >
            All
          </button>
          <button 
            className={`tab ${filter === 'with-mcp' ? 'active' : ''}`}
            onClick={() => setFilter('with-mcp')}
          >
            With MCP
          </button>
          <button 
            className={`tab ${filter === 'without-mcp' ? 'active' : ''}`}
            onClick={() => setFilter('without-mcp')}
          >
            Without MCP
          </button>
          <button 
            className={`tab ${filter === 'active' ? 'active' : ''}`}
            onClick={() => setFilter('active')}
          >
            Active
          </button>
        </div>

        <div className="search-box">
          <input 
            type="text" 
            placeholder="Search scenarios..."
            className="search-input"
          />
        </div>
      </div>

      <main className="main-content">
        {loading && scenarios.length === 0 ? (
          <div className="loading">
            <Activity className="spinning" />
            <div>Loading scenarios...</div>
          </div>
        ) : (
          <ScenarioGrid 
            scenarios={filteredScenarios}
            onAddMCP={handleAddMCP}
            onRefresh={loadScenarios}
          />
        )}
      </main>

      {showAddMCP && (
        <AddMCPModal
          scenario={selectedScenario}
          onClose={() => setShowAddMCP(false)}
          onSuccess={handleMCPAdded}
        />
      )}

      <footer className="footer">
        <div className="footer-left">
          <GitBranch className="icon-small" />
          <span>MCP Protocol v1.0</span>
        </div>
        <div className="footer-right">
          <a href="/api/v1/mcp/registry" target="_blank">
            View Registry
          </a>
          <span className="separator">|</span>
          <a href="#" onClick={() => window.open('scenario-to-mcp://help')}>
            Documentation
          </a>
        </div>
      </footer>
    </div>
  );
}

export default App;