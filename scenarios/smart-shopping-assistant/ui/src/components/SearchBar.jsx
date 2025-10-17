import React, { useState } from 'react'
import './SearchBar.css'

function SearchBar({ onSearch, isLoading }) {
  const [query, setQuery] = useState('')
  const [budgetMax, setBudgetMax] = useState('')
  const [includeAlternatives, setIncludeAlternatives] = useState(true)

  const handleSubmit = (e) => {
    e.preventDefault()
    if (query.trim()) {
      onSearch(query, {
        budgetMax: budgetMax ? parseFloat(budgetMax) : null,
        includeAlternatives,
      })
    }
  }

  return (
    <div className="search-bar">
      <form onSubmit={handleSubmit}>
        <div className="search-main">
          <input
            type="text"
            placeholder="Search for products... (e.g., 'laptop under $1000', 'eco-friendly water bottle')"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            className="search-input"
            disabled={isLoading}
          />
          <button type="submit" className="search-button" disabled={isLoading || !query.trim()}>
            {isLoading ? 'Searching...' : '🔍 Search'}
          </button>
        </div>
        
        <div className="search-options">
          <div className="option-group">
            <label>
              Max Budget:
              <input
                type="number"
                placeholder="Optional"
                value={budgetMax}
                onChange={(e) => setBudgetMax(e.target.value)}
                className="budget-input"
                min="0"
                step="0.01"
              />
            </label>
          </div>
          
          <div className="option-group">
            <label className="checkbox-label">
              <input
                type="checkbox"
                checked={includeAlternatives}
                onChange={(e) => setIncludeAlternatives(e.target.checked)}
              />
              Include alternatives (used, generic, rental)
            </label>
          </div>
        </div>
      </form>
    </div>
  )
}

export default SearchBar