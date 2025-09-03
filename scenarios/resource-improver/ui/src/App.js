import React, { useState, useEffect } from 'react';
import axios from 'axios';
import { 
  FlaskConical, 
  Play, 
  RefreshCw, 
  Eye, 
  Trash2, 
  CheckCircle, 
  Clock, 
  XCircle, 
  AlertCircle,
  Settings,
  Database,
  Cpu,
  Zap,
  X
} from 'lucide-react';

const API_BASE = process.env.NODE_ENV === 'production' ? '' : 'http://localhost:8092';

function App() {
  const [experiments, setExperiments] = useState([]);
  const [scenarios, setScenarios] = useState([]);
  const [templates, setTemplates] = useState([]);
  const [loading, setLoading] = useState(true);
  const [creating, setCreating] = useState(false);
  const [filter, setFilter] = useState('all');
  const [error, setError] = useState(null);
  const [success, setSuccess] = useState(null);
  const [selectedExperiment, setSelectedExperiment] = useState(null);
  const [apiStatus, setApiStatus] = useState('unknown');
  
  // Form state
  const [formData, setFormData] = useState({
    name: '',
    description: '',
    prompt: '',
    target_scenario: '',
    new_resource: ''
  });

  // Load initial data
  useEffect(() => {
    loadInitialData();
    checkApiStatus();
    // Refresh data every 30 seconds
    const interval = setInterval(loadExperiments, 30000);
    return () => clearInterval(interval);
  }, []);

  const checkApiStatus = async () => {
    try {
      await axios.get(`${API_BASE}/health`);
      setApiStatus('healthy');
    } catch (error) {
      setApiStatus('error');
    }
  };

  const loadInitialData = async () => {
    setLoading(true);
    try {
      await Promise.all([
        loadExperiments(),
        loadScenarios(),
        loadTemplates()
      ]);
    } catch (error) {
      setError('Failed to load initial data');
    } finally {
      setLoading(false);
    }
  };

  const loadExperiments = async () => {
    try {
      const response = await axios.get(`${API_BASE}/api/experiments`);
      setExperiments(response.data || []);
    } catch (error) {
      console.error('Failed to load experiments:', error);
      setError('Failed to load experiments');
    }
  };

  const loadScenarios = async () => {
    try {
      const response = await axios.get(`${API_BASE}/api/scenarios`);
      setScenarios(response.data || []);
    } catch (error) {
      console.error('Failed to load scenarios:', error);
    }
  };

  const loadTemplates = async () => {
    try {
      const response = await axios.get(`${API_BASE}/api/templates`);
      setTemplates(response.data || []);
    } catch (error) {
      console.error('Failed to load templates:', error);
    }
  };

  const handleInputChange = (e) => {
    const { name, value } = e.target;
    setFormData(prev => ({
      ...prev,
      [name]: value
    }));

    // Auto-generate experiment name and description based on prompt
    if (name === 'prompt') {
      const match = value.match(/add\s+([a-zA-Z0-9-]+)\s+to\s+([a-zA-Z0-9-]+)/i);
      if (match) {
        const resource = match[1];
        const scenario = match[2];
        setFormData(prev => ({
          ...prev,
          name: `Add ${resource} to ${scenario}`,
          target_scenario: scenario,
          new_resource: resource,
          description: `Integrate ${resource} into the ${scenario} scenario to enhance its capabilities`
        }));
      }
    }
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!formData.prompt.trim()) return;

    setCreating(true);
    setError(null);

    try {
      const response = await axios.post(`${API_BASE}/api/experiments`, formData);
      setSuccess(`Experiment created: ${response.data.name}`);
      setFormData({
        name: '',
        description: '',
        prompt: '',
        target_scenario: '',
        new_resource: ''
      });
      loadExperiments();
    } catch (error) {
      setError(error.response?.data?.message || 'Failed to create experiment');
    } finally {
      setCreating(false);
    }
  };

  const handleDelete = async (id) => {
    if (!window.confirm('Are you sure you want to delete this experiment?')) return;

    try {
      await axios.delete(`${API_BASE}/api/experiments/${id}`);
      setSuccess('Experiment deleted');
      loadExperiments();
    } catch (error) {
      setError('Failed to delete experiment');
    }
  };

  const viewExperiment = async (id) => {
    try {
      const response = await axios.get(`${API_BASE}/api/experiments/${id}`);
      setSelectedExperiment(response.data);
    } catch (error) {
      setError('Failed to load experiment details');
    }
  };

  const getStatusIcon = (status) => {
    switch (status) {
      case 'completed': return <CheckCircle size={16} />;
      case 'running': return <Clock size={16} />;
      case 'failed': return <XCircle size={16} />;
      default: return <AlertCircle size={16} />;
    }
  };

  const getResourceIcon = (resource) => {
    const resourceTypes = {
      postgres: Database,
      redis: Database,
      minio: Database,
      qdrant: Database,
      ollama: Cpu,
      comfyui: Cpu,
      whisper: Cpu,
      n8n: Zap,
      windmill: Zap,
      default: Settings
    };
    const Icon = resourceTypes[resource] || resourceTypes.default;
    return <Icon size={14} />;
  };

  const filteredExperiments = experiments.filter(exp => {
    if (filter === 'all') return true;
    return exp.status === filter;
  });

  if (loading) {
    return (
      <div className="loading">
        <div className="spinner"></div>
        Loading Resource Experimenter...
      </div>
    );
  }

  return (
    <div className="App">
      {/* Header */}
      <header className="header">
        <div className="header-content">
          <div className="logo">
            <FlaskConical size={24} color="#667eea" />
            <h1>Resource Experimenter</h1>
          </div>
          <div className={`status-badge ${apiStatus === 'healthy' ? 'status-healthy' : 'status-error'}`}>
            API: {apiStatus === 'healthy' ? 'Connected' : 'Disconnected'}
          </div>
        </div>
      </header>

      <div className="container">
        {/* Messages */}
        {error && (
          <div className="error-message">
            <XCircle size={16} />
            {error}
            <button onClick={() => setError(null)} style={{ marginLeft: 'auto', background: 'none', border: 'none', color: 'inherit' }}>
              <X size={16} />
            </button>
          </div>
        )}

        {success && (
          <div className="success-message">
            <CheckCircle size={16} />
            {success}
            <button onClick={() => setSuccess(null)} style={{ marginLeft: 'auto', background: 'none', border: 'none', color: 'inherit' }}>
              <X size={16} />
            </button>
          </div>
        )}

        {/* New Experiment Form */}
        <div className="experiment-form">
          <h2 className="form-title">Start New Experiment</h2>
          <form onSubmit={handleSubmit}>
            <div className="form-group">
              <label className="form-label">Experiment Description</label>
              <input
                type="text"
                name="prompt"
                className="form-input"
                placeholder="e.g., Add redis to analytics-dashboard"
                value={formData.prompt}
                onChange={handleInputChange}
                required
              />
              <small style={{ color: '#6b7280', fontSize: '0.75rem' }}>
                Use format: "Add [resource] to [scenario]" for automatic parsing
              </small>
            </div>

            {formData.target_scenario && (
              <div style={{ display: 'flex', gap: '1rem' }}>
                <div className="form-group" style={{ flex: 1 }}>
                  <label className="form-label">Target Scenario</label>
                  <select
                    name="target_scenario"
                    className="form-select"
                    value={formData.target_scenario}
                    onChange={handleInputChange}
                  >
                    <option value="">Select scenario</option>
                    {scenarios.map(scenario => (
                      <option key={scenario.id} value={scenario.name}>
                        {scenario.display_name || scenario.name}
                      </option>
                    ))}
                  </select>
                </div>

                <div className="form-group" style={{ flex: 1 }}>
                  <label className="form-label">New Resource</label>
                  <input
                    type="text"
                    name="new_resource"
                    className="form-input"
                    value={formData.new_resource}
                    onChange={handleInputChange}
                    placeholder="e.g., redis, qdrant, ollama"
                  />
                </div>
              </div>
            )}

            <button
              type="submit"
              className="btn-primary"
              disabled={creating || !formData.prompt.trim()}
            >
              {creating ? (
                <>
                  <div className="spinner" style={{ width: '16px', height: '16px' }}></div>
                  Creating...
                </>
              ) : (
                <>
                  <Play size={16} />
                  Start Experiment
                </>
              )}
            </button>
          </form>
        </div>

        {/* Experiments List */}
        <div className="experiments-section">
          <div className="section-header">
            <div className="section-title">
              Experiments ({filteredExperiments.length})
            </div>
            <div className="filters">
              <select
                className="filter-select"
                value={filter}
                onChange={(e) => setFilter(e.target.value)}
              >
                <option value="all">All Experiments</option>
                <option value="requested">Requested</option>
                <option value="running">Running</option>
                <option value="completed">Completed</option>
                <option value="failed">Failed</option>
              </select>
              
              <button
                onClick={loadExperiments}
                className="btn-secondary"
                style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}
              >
                <RefreshCw size={14} />
                Refresh
              </button>
            </div>
          </div>

          {filteredExperiments.length === 0 ? (
            <div className="empty-state">
              <FlaskConical size={48} color="#d1d5db" />
              <h3>No experiments found</h3>
              <p>Start your first experiment above to begin exploring resource integrations.</p>
            </div>
          ) : (
            <div className="experiments-grid">
              {filteredExperiments.map(experiment => (
                <div key={experiment.id} className="experiment-card">
                  <div className={`status status-${experiment.status}`}>
                    {experiment.status}
                  </div>

                  <div className="experiment-name">
                    {experiment.name}
                  </div>

                  <div className="experiment-meta">
                    <div className="meta-item">
                      <Database size={12} />
                      {experiment.target_scenario}
                    </div>
                    <div className="meta-item">
                      {getResourceIcon(experiment.new_resource)}
                      {experiment.new_resource}
                    </div>
                    <div className="meta-item">
                      <Clock size={12} />
                      {new Date(experiment.created_at).toLocaleDateString()}
                    </div>
                  </div>

                  <div className="experiment-description">
                    {experiment.description}
                  </div>

                  <div className="experiment-actions">
                    <button
                      onClick={() => viewExperiment(experiment.id)}
                      className="btn-secondary"
                    >
                      <Eye size={14} />
                      View
                    </button>
                    <button
                      onClick={() => handleDelete(experiment.id)}
                      className="btn-secondary btn-danger"
                    >
                      <Trash2 size={14} />
                      Delete
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Experiment Detail Modal */}
      {selectedExperiment && (
        <div className="modal-overlay" onClick={() => setSelectedExperiment(null)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h2 className="modal-title">{selectedExperiment.name}</h2>
              <button
                onClick={() => setSelectedExperiment(null)}
                className="close-button"
              >
                <X size={18} />
              </button>
            </div>

            <div>
              <div style={{ marginBottom: '1.5rem' }}>
                <div style={{ display: 'flex', gap: '1rem', marginBottom: '1rem' }}>
                  <div className={`status status-${selectedExperiment.status}`}>
                    {getStatusIcon(selectedExperiment.status)}
                    {selectedExperiment.status}
                  </div>
                </div>

                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '1rem', marginBottom: '1rem' }}>
                  <div>
                    <strong>Target Scenario:</strong>
                    <div className="detail-badge">{selectedExperiment.target_scenario}</div>
                  </div>
                  <div>
                    <strong>New Resource:</strong>
                    <div className="detail-badge">
                      {getResourceIcon(selectedExperiment.new_resource)}
                      {selectedExperiment.new_resource}
                    </div>
                  </div>
                </div>

                <div>
                  <strong>Description:</strong>
                  <p style={{ color: '#6b7280', marginTop: '0.5rem' }}>
                    {selectedExperiment.description}
                  </p>
                </div>
              </div>

              {selectedExperiment.claude_prompt && (
                <div style={{ marginBottom: '1.5rem' }}>
                  <strong>Claude Prompt:</strong>
                  <pre style={{ 
                    background: '#f9fafb', 
                    padding: '1rem', 
                    borderRadius: '8px', 
                    fontSize: '0.75rem', 
                    overflow: 'auto',
                    maxHeight: '200px',
                    marginTop: '0.5rem'
                  }}>
                    {selectedExperiment.claude_prompt.substring(0, 1000)}
                    {selectedExperiment.claude_prompt.length > 1000 && '...\n[truncated]'}
                  </pre>
                </div>
              )}

              {selectedExperiment.claude_response && (
                <div style={{ marginBottom: '1.5rem' }}>
                  <strong>Claude Response:</strong>
                  <pre style={{ 
                    background: '#f0f9ff', 
                    padding: '1rem', 
                    borderRadius: '8px', 
                    fontSize: '0.75rem', 
                    overflow: 'auto',
                    maxHeight: '300px',
                    marginTop: '0.5rem'
                  }}>
                    {selectedExperiment.claude_response.substring(0, 2000)}
                    {selectedExperiment.claude_response.length > 2000 && '...\n[truncated]'}
                  </pre>
                </div>
              )}

              {selectedExperiment.generation_error && (
                <div style={{ marginBottom: '1.5rem' }}>
                  <strong>Error:</strong>
                  <div style={{ 
                    background: '#fecaca', 
                    color: '#991b1b', 
                    padding: '1rem', 
                    borderRadius: '8px',
                    marginTop: '0.5rem'
                  }}>
                    {selectedExperiment.generation_error}
                  </div>
                </div>
              )}

              <div style={{ fontSize: '0.75rem', color: '#6b7280' }}>
                <div>Created: {new Date(selectedExperiment.created_at).toLocaleString()}</div>
                {selectedExperiment.completed_at && (
                  <div>Completed: {new Date(selectedExperiment.completed_at).toLocaleString()}</div>
                )}
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default App;