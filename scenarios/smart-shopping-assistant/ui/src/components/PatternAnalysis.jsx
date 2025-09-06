import React from 'react'
import './PatternAnalysis.css'

function PatternAnalysis({ patterns }) {
  if (!patterns) return null

  return (
    <div className="pattern-analysis">
      <h2>📊 Your Shopping Patterns</h2>
      
      {patterns.patterns && patterns.patterns.length > 0 && (
        <div className="patterns-section">
          <h3>Detected Patterns</h3>
          <div className="patterns-grid">
            {patterns.patterns.map((pattern, index) => (
              <div key={index} className="pattern-card">
                <div className="pattern-header">
                  <span className="pattern-category">{pattern.category}</span>
                  <span className="pattern-type">{pattern.pattern_type}</span>
                </div>
                <div className="pattern-details">
                  <div className="detail">
                    <span className="label">Frequency:</span>
                    <span className="value">Every {pattern.frequency_days} days</span>
                  </div>
                  <div className="detail">
                    <span className="label">Average Spend:</span>
                    <span className="value">${pattern.average_spend}</span>
                  </div>
                  <div className="detail">
                    <span className="label">Confidence:</span>
                    <span className="value">{Math.round(pattern.confidence * 100)}%</span>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
      
      {patterns.predictions && patterns.predictions.length > 0 && (
        <div className="predictions-section">
          <h3>🔮 Restock Predictions</h3>
          <div className="predictions-list">
            {patterns.predictions.map((prediction, index) => (
              <div key={index} className="prediction-item">
                <div className="prediction-main">
                  <span className="prediction-category">{prediction.product_category}</span>
                  <span className="prediction-date">
                    Needed by {new Date(prediction.predicted_date).toLocaleDateString()}
                  </span>
                </div>
                {prediction.suggested_items && prediction.suggested_items.length > 0 && (
                  <div className="suggested-items">
                    Suggested: {prediction.suggested_items.join(', ')}
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      )}
      
      {patterns.savings_opportunities && patterns.savings_opportunities.length > 0 && (
        <div className="savings-section">
          <h3>💡 Savings Opportunities</h3>
          <div className="savings-list">
            {patterns.savings_opportunities.map((opportunity, index) => (
              <div key={index} className="savings-card">
                <div className="savings-header">
                  <span className="savings-amount">${opportunity.potential_savings}</span>
                  <span className="savings-label">potential savings</span>
                </div>
                <p className="savings-description">{opportunity.description}</p>
                <p className="savings-action">{opportunity.action_required}</p>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

export default PatternAnalysis