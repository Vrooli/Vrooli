import React from 'react';
import ScenarioCard from './ScenarioCard';
import './ScenarioGrid.css';

function ScenarioGrid({ scenarios, onAddMCP, onRefresh }) {
  if (scenarios.length === 0) {
    return (
      <div className="empty-state">
        <div className="empty-message">No scenarios found</div>
        <button className="refresh-link" onClick={onRefresh}>
          Refresh list
        </button>
      </div>
    );
  }

  return (
    <div className="scenario-grid">
      {scenarios.map(scenario => (
        <ScenarioCard
          key={scenario.name}
          scenario={scenario}
          onAddMCP={onAddMCP}
        />
      ))}
    </div>
  );
}

export default ScenarioGrid;