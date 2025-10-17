import React from 'react'
import './AlternativesList.css'

function AlternativesList({ alternatives }) {
  if (!alternatives || alternatives.length === 0) {
    return null
  }

  return (
    <div className="alternatives-list">
      <h3>💰 Money-Saving Alternatives</h3>
      <div className="alternatives-grid">
        {alternatives.map((alt, index) => (
          <div key={index} className="alternative-card">
            <div className="alternative-type">
              {alt.alternative_type === 'generic' && '🏷️ Generic'}
              {alt.alternative_type === 'used' && '♻️ Used/Refurbished'}
              {alt.alternative_type === 'rental' && '📅 Rental'}
              {alt.alternative_type === 'similar' && '🔄 Similar Product'}
            </div>
            <h4>{alt.product.name}</h4>
            <div className="alternative-price">
              <span className="price">${alt.product.current_price}</span>
              <span className="savings">Save ${alt.savings_amount}</span>
            </div>
            <p className="alternative-reason">{alt.reason}</p>
          </div>
        ))}
      </div>
    </div>
  )
}

export default AlternativesList