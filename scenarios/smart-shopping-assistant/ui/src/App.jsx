import React, { useState, useEffect } from 'react'
import axios from 'axios'
import './App.css'
import SearchBar from './components/SearchBar'
import ProductCard from './components/ProductCard'
import AlternativesList from './components/AlternativesList'
import PriceInsights from './components/PriceInsights'
import PatternAnalysis from './components/PatternAnalysis'

function App() {
  const [activeTab, setActiveTab] = useState('search')
  const [searchResults, setSearchResults] = useState(null)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState(null)
  const [trackedProducts, setTrackedProducts] = useState([])
  const [patterns, setPatterns] = useState(null)
  const [profileId] = useState('11111111-1111-1111-1111-111111111111')

  useEffect(() => {
    fetchTrackedProducts()
  }, [profileId])

  const fetchTrackedProducts = async () => {
    try {
      const response = await axios.get(`/api/v1/shopping/tracking/${profileId}`)
      setTrackedProducts(response.data.tracked_products || [])
    } catch (err) {
      console.error('Failed to fetch tracked products:', err)
    }
  }

  const handleSearch = async (query, options = {}) => {
    setIsLoading(true)
    setError(null)
    
    try {
      const response = await axios.post('/api/v1/shopping/research', {
        profile_id: profileId,
        query: query,
        include_alternatives: options.includeAlternatives || true,
        budget_max: options.budgetMax || null,
      })
      
      setSearchResults(response.data)
      setActiveTab('search')
    } catch (err) {
      setError('Failed to search products. Please try again.')
      console.error('Search error:', err)
    } finally {
      setIsLoading(false)
    }
  }

  const handleTrackProduct = async (productId, targetPrice) => {
    try {
      await axios.post('/api/v1/shopping/tracking', {
        profile_id: profileId,
        product_id: productId,
        target_price: targetPrice,
      })
      
      fetchTrackedProducts()
      alert('Product tracking activated!')
    } catch (err) {
      alert('Failed to track product')
      console.error('Tracking error:', err)
    }
  }

  const analyzePatterns = async () => {
    setIsLoading(true)
    try {
      const response = await axios.post('/api/v1/shopping/pattern-analysis', {
        profile_id: profileId,
        timeframe: '30d',
      })
      
      setPatterns(response.data)
      setActiveTab('patterns')
    } catch (err) {
      setError('Failed to analyze patterns')
      console.error('Pattern analysis error:', err)
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="app">
      <header className="app-header">
        <div className="header-content">
          <h1>🛒 Smart Shopping Assistant</h1>
          <p className="tagline">Save money with intelligent shopping research & price tracking</p>
        </div>
      </header>

      <nav className="tab-nav">
        <button 
          className={`tab ${activeTab === 'search' ? 'active' : ''}`}
          onClick={() => setActiveTab('search')}
        >
          Search Products
        </button>
        <button 
          className={`tab ${activeTab === 'tracking' ? 'active' : ''}`}
          onClick={() => setActiveTab('tracking')}
        >
          Price Tracking ({trackedProducts.length})
        </button>
        <button 
          className={`tab ${activeTab === 'patterns' ? 'active' : ''}`}
          onClick={() => {
            setActiveTab('patterns')
            if (!patterns) analyzePatterns()
          }}
        >
          Purchase Patterns
        </button>
      </nav>

      <main className="main-content">
        {activeTab === 'search' && (
          <div className="search-section">
            <SearchBar onSearch={handleSearch} isLoading={isLoading} />
            
            {error && (
              <div className="error-message">{error}</div>
            )}
            
            {searchResults && (
              <div className="results-container">
                <div className="products-grid">
                  <h2>Products Found</h2>
                  <div className="products-list">
                    {searchResults.products.map((product, index) => (
                      <ProductCard 
                        key={product.id || index}
                        product={product}
                        onTrack={handleTrackProduct}
                        showAffiliate={true}
                      />
                    ))}
                  </div>
                </div>
                
                {searchResults.alternatives && searchResults.alternatives.length > 0 && (
                  <AlternativesList alternatives={searchResults.alternatives} />
                )}
                
                {searchResults.price_analysis && (
                  <PriceInsights insights={searchResults.price_analysis} />
                )}
                
                {searchResults.recommendations && searchResults.recommendations.length > 0 && (
                  <div className="recommendations">
                    <h3>💡 Smart Recommendations</h3>
                    <ul>
                      {searchResults.recommendations.map((rec, index) => (
                        <li key={index}>{rec}</li>
                      ))}
                    </ul>
                  </div>
                )}
              </div>
            )}
          </div>
        )}

        {activeTab === 'tracking' && (
          <div className="tracking-section">
            <h2>Tracked Products</h2>
            {trackedProducts.length === 0 ? (
              <p className="empty-state">No products being tracked. Search for products and click "Track Price" to start monitoring.</p>
            ) : (
              <div className="products-list">
                {trackedProducts.map((product) => (
                  <ProductCard 
                    key={product.id}
                    product={product}
                    showTracking={true}
                  />
                ))}
              </div>
            )}
          </div>
        )}

        {activeTab === 'patterns' && (
          <div className="patterns-section">
            {isLoading ? (
              <div className="loading">Analyzing your purchase patterns...</div>
            ) : patterns ? (
              <PatternAnalysis patterns={patterns} />
            ) : (
              <div className="empty-state">
                <p>No pattern analysis available yet.</p>
                <button onClick={analyzePatterns} className="btn-primary">
                  Analyze My Patterns
                </button>
              </div>
            )}
          </div>
        )}
      </main>

      <footer className="app-footer">
        <p>Smart Shopping Assistant v1.0.0 | Part of the Vrooli Ecosystem</p>
        <p className="affiliate-notice">
          Some links generate affiliate revenue to support development
        </p>
      </footer>
    </div>
  )
}

export default App