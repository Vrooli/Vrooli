import React from 'react'
import './ProductCard.css'

function ProductCard({ product, onTrack, showAffiliate, showTracking }) {
  const discount = product.original_price 
    ? Math.round(((product.original_price - product.current_price) / product.original_price) * 100)
    : 0

  return (
    <div className="product-card">
      <div className="product-header">
        <h3 className="product-name">{product.name}</h3>
        <span className="product-category">{product.category}</span>
      </div>
      
      <p className="product-description">{product.description}</p>
      
      <div className="price-section">
        <div className="price-display">
          <span className="current-price">${product.current_price}</span>
          {product.original_price && product.original_price > product.current_price && (
            <>
              <span className="original-price">${product.original_price}</span>
              <span className="discount-badge">-{discount}%</span>
            </>
          )}
        </div>
      </div>
      
      {product.reviews_summary && (
        <div className="reviews-section">
          <div className="rating">
            <span className="stars">{'★'.repeat(Math.round(product.reviews_summary.average_rating))}</span>
            <span className="rating-value">{product.reviews_summary.average_rating}</span>
            <span className="review-count">({product.reviews_summary.total_reviews} reviews)</span>
          </div>
          {product.reviews_summary.pros && product.reviews_summary.pros.length > 0 && (
            <div className="pros-cons">
              <span className="pro">✓ {product.reviews_summary.pros[0]}</span>
            </div>
          )}
        </div>
      )}
      
      <div className="product-actions">
        {showAffiliate && product.affiliate_link && (
          <a 
            href={product.affiliate_link} 
            target="_blank" 
            rel="noopener noreferrer"
            className="btn-buy"
          >
            View Deal 🔗
          </a>
        )}
        
        {onTrack && (
          <button 
            onClick={() => {
              const targetPrice = prompt('Set target price for alerts (optional):')
              onTrack(product.id, targetPrice ? parseFloat(targetPrice) : null)
            }}
            className="btn-track"
          >
            Track Price 📊
          </button>
        )}
        
        {showTracking && (
          <div className="tracking-info">
            <span className="tracking-badge">🔔 Tracking Active</span>
          </div>
        )}
      </div>
    </div>
  )
}

export default ProductCard