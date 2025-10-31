import React from 'react';
import MCPStatus from './MCPStatus';
import { Plus, Terminal, Eye, Settings, AlertCircle } from 'lucide-react';
import { buildApiUrl } from '../utils/apiClient';
import './ScenarioCard.css';

function ScenarioCard({ scenario, onAddMCP }) {
  const handleAddMCP = (e) => {
    e.stopPropagation();
    onAddMCP(scenario);
  };

  const handleViewDetails = () => {
    const detailsUrl = buildApiUrl(`mcp/scenarios/${scenario.name}`);
    window.open(detailsUrl, '_blank', 'noopener,noreferrer');
  };

  return (
    <div className={`scenario-card ${scenario.hasMCP ? 'has-mcp' : ''}`}>
      <div className="card-header">
        <h3 className="scenario-name">{scenario.name}</h3>
        <MCPStatus status={scenario.status} />
      </div>

      <div className="card-body">
        {scenario.hasMCP ? (
          <>
            <div className="mcp-info">
              <div className="info-row">
                <span className="label">Port:</span>
                <span className="value">{scenario.port || 'Not assigned'}</span>
              </div>
              <div className="info-row">
                <span className="label">Tools:</span>
                <span className="value">{scenario.tools?.length || 0}</span>
              </div>
              <div className="info-row">
                <span className="label">Confidence:</span>
                <span className={`value confidence-${scenario.confidence}`}>
                  {scenario.confidence || 'unknown'}
                </span>
              </div>
            </div>

            {scenario.tools && scenario.tools.length > 0 && (
              <div className="tools-preview">
                <div className="tools-label">Available Tools:</div>
                <div className="tools-list">
                  {scenario.tools.slice(0, 3).map((tool, index) => (
                    <span key={index} className="tool-badge">
                      {tool.name || tool}
                    </span>
                  ))}
                  {scenario.tools.length > 3 && (
                    <span className="tool-badge more">
                      +{scenario.tools.length - 3} more
                    </span>
                  )}
                </div>
              </div>
            )}
          </>
        ) : (
          <div className="no-mcp">
            <AlertCircle className="warning-icon" />
            <p>No MCP implementation detected</p>
            <p className="suggestion">Click below to add MCP support</p>
          </div>
        )}

        {scenario.error && (
          <div className="error-message">
            <AlertCircle className="error-icon" />
            {scenario.error}
          </div>
        )}
      </div>

      <div className="card-footer">
        {scenario.hasMCP ? (
          <>
            <button className="action-btn" onClick={handleViewDetails}>
              <Eye className="btn-icon" />
              View
            </button>
            <button className="action-btn">
              <Terminal className="btn-icon" />
              Test
            </button>
            <button className="action-btn">
              <Settings className="btn-icon" />
              Config
            </button>
          </>
        ) : (
          <button className="action-btn primary" onClick={handleAddMCP}>
            <Plus className="btn-icon" />
            Add MCP Support
          </button>
        )}
      </div>
    </div>
  );
}

export default ScenarioCard;
