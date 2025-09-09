import React, { useState } from 'react';
import { X, Cpu, Zap, Code } from 'lucide-react';
import axios from 'axios';
import './AddMCPModal.css';

function AddMCPModal({ scenario, onClose, onSuccess }) {
  const [template, setTemplate] = useState('auto-detect');
  const [autoDetect, setAutoDetect] = useState(true);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  const templates = [
    { id: 'auto-detect', name: 'Auto-Detect', icon: Zap, description: 'Analyze scenario and generate optimal MCP' },
    { id: 'basic-api', name: 'Basic API', icon: Code, description: 'Standard API exposure template' },
    { id: 'data-processor', name: 'Data Processor', icon: Cpu, description: 'For data processing scenarios' }
  ];

  const handleSubmit = async () => {
    setLoading(true);
    setError(null);

    try {
      const response = await axios.post('/api/v1/mcp/add', {
        scenario_name: scenario.name,
        agent_config: {
          template: template === 'auto-detect' ? null : template,
          auto_detect: autoDetect
        }
      });

      if (response.data.success) {
        onSuccess(scenario, response.data.agent_session_id);
      } else {
        setError('Failed to initiate MCP addition');
      }
    } catch (err) {
      setError(err.response?.data?.error || 'Failed to add MCP support');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-content" onClick={e => e.stopPropagation()}>
        <div className="modal-header">
          <h2>Add MCP Support</h2>
          <button className="close-btn" onClick={onClose}>
            <X />
          </button>
        </div>

        <div className="modal-body">
          <div className="scenario-info">
            <h3>{scenario.name}</h3>
            <p className="scenario-description">
              This will spawn a Claude-code agent to add Model Context Protocol support to this scenario.
            </p>
          </div>

          <div className="template-selection">
            <h4>Select Template</h4>
            <div className="template-grid">
              {templates.map(tpl => (
                <div
                  key={tpl.id}
                  className={`template-card ${template === tpl.id ? 'selected' : ''}`}
                  onClick={() => setTemplate(tpl.id)}
                >
                  <tpl.icon className="template-icon" />
                  <div className="template-name">{tpl.name}</div>
                  <div className="template-description">{tpl.description}</div>
                </div>
              ))}
            </div>
          </div>

          <div className="options">
            <label className="checkbox-label">
              <input
                type="checkbox"
                checked={autoDetect}
                onChange={e => setAutoDetect(e.target.checked)}
              />
              <span>Auto-detect scenario capabilities</span>
            </label>
          </div>

          {error && (
            <div className="error-box">
              {error}
            </div>
          )}

          <div className="modal-info">
            <p>The agent will:</p>
            <ul>
              <li>Analyze the scenario's API and CLI interfaces</li>
              <li>Generate MCP server implementation</li>
              <li>Create tool manifests for all capabilities</li>
              <li>Set up health monitoring and registry integration</li>
            </ul>
          </div>
        </div>

        <div className="modal-footer">
          <button className="btn-secondary" onClick={onClose} disabled={loading}>
            Cancel
          </button>
          <button className="btn-primary" onClick={handleSubmit} disabled={loading}>
            {loading ? 'Spawning Agent...' : 'Spawn Claude-code Agent'}
          </button>
        </div>
      </div>
    </div>
  );
}

export default AddMCPModal;