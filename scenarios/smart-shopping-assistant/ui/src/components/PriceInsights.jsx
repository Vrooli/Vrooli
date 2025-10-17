import React from 'react'
import './PriceInsights.css'

function PriceInsights({ insights }) {
  if (!insights) return null

  return (
    <div className="price-insights">
      <h3>📈 Price Analysis</h3>
      <div className="insights-grid">
        <div className="insight-card">
          <span className="insight-label">Current Trend</span>
          <span className={`insight-value trend-${insights.current_trend}`}>
            {insights.current_trend === 'rising' && '📈 Rising'}
            {insights.current_trend === 'falling' && '📉 Falling'}
            {insights.current_trend === 'stable' && '➡️ Stable'}
          </span>
        </div>
        
        <div className="insight-card">
          <span className="insight-label">Best Time to Buy</span>
          <span className={`insight-value ${insights.best_time_to_wait ? 'wait' : 'buy'}`}>
            {insights.best_time_to_wait ? '⏳ Wait' : '✅ Now'}
          </span>
        </div>
        
        <div className="insight-card">
          <span className="insight-label">Historical Low</span>
          <span className="insight-value">${insights.historical_low}</span>
        </div>
        
        <div className="insight-card">
          <span className="insight-label">Historical High</span>
          <span className="insight-value">${insights.historical_high}</span>
        </div>
        
        {insights.predicted_drop && (
          <div className="insight-card full-width">
            <span className="insight-label">Predicted Price Drop</span>
            <span className="insight-value highlight">-${insights.predicted_drop} expected</span>
          </div>
        )}
      </div>
    </div>
  )
}

export default PriceInsights