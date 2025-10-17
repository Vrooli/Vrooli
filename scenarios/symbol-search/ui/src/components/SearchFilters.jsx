import React, { memo, useMemo } from 'react'

/**
 * Search filters component for the sidebar
 * Handles search input and filter controls
 */
const SearchFilters = memo(function SearchFilters({
  searchQuery,
  onSearchChange,
  filters,
  onFilterChange,
  categories,
  blocks,
  loading,
  onClearFilters,
  hasFilters
}) {
  // Sort categories alphabetically
  const sortedCategories = useMemo(() => {
    return [...categories].sort((a, b) => a.name.localeCompare(b.name))
  }, [categories])

  // Sort blocks alphabetically
  const sortedBlocks = useMemo(() => {
    return [...blocks].sort((a, b) => a.name.localeCompare(b.name))
  }, [blocks])

  // Get unique Unicode versions from blocks
  const unicodeVersions = useMemo(() => {
    const versions = new Set()
    blocks.forEach(block => {
      if (block.unicode_version) {
        versions.add(block.unicode_version)
      }
    })
    
    // Add some common Unicode versions if not present
    const commonVersions = ['1.1', '2.0', '3.0', '4.0', '5.0', '6.0', '7.0', '8.0', '9.0', '10.0', '11.0', '12.0', '13.0', '14.0', '15.0']
    commonVersions.forEach(v => versions.add(v))
    
    return Array.from(versions)
      .sort((a, b) => parseFloat(b) - parseFloat(a)) // Sort descending
  }, [blocks])

  return (
    <>
      {/* Search Input */}
      <div className="sidebar-section">
        <label htmlFor="search" className="form-label">Search</label>
        <input
          id="search"
          type="text"
          className="form-input search-input"
          placeholder="Search by name or codepoint..."
          value={searchQuery}
          onChange={onSearchChange}
          disabled={loading}
          aria-describedby="search-help"
        />
        <div id="search-help" style={{ 
          fontSize: '12px', 
          color: 'var(--color-gray-500)', 
          marginTop: '0.25rem' 
        }}>
          Try "heart", "arrow", "U+1F600", or "math"
        </div>
      </div>

      {/* Category Filter */}
      <div className="sidebar-section">
        <h3>Category</h3>
        <div className="form-group">
          <label htmlFor="category" className="form-label">Unicode Category</label>
          <select
            id="category"
            className="form-select"
            value={filters.category}
            onChange={(e) => onFilterChange('category', e.target.value)}
            disabled={loading}
          >
            <option value="">All Categories</option>
            {sortedCategories.map(category => (
              <option key={category.code} value={category.code}>
                {category.code} - {category.name} ({category.character_count?.toLocaleString() || 0})
              </option>
            ))}
          </select>
        </div>
      </div>

      {/* Unicode Block Filter */}
      <div className="sidebar-section">
        <h3>Unicode Block</h3>
        <div className="form-group">
          <label htmlFor="block" className="form-label">Character Block</label>
          <select
            id="block"
            className="form-select"
            value={filters.block}
            onChange={(e) => onFilterChange('block', e.target.value)}
            disabled={loading}
          >
            <option value="">All Blocks</option>
            {sortedBlocks.map(block => (
              <option key={block.name} value={block.name}>
                {block.name} ({block.character_count?.toLocaleString() || 0})
              </option>
            ))}
          </select>
        </div>
      </div>

      {/* Unicode Version Filter */}
      <div className="sidebar-section">
        <h3>Unicode Version</h3>
        <div className="form-group">
          <label htmlFor="unicode-version" className="form-label">Version Added</label>
          <select
            id="unicode-version"
            className="form-select"
            value={filters.unicodeVersion}
            onChange={(e) => onFilterChange('unicodeVersion', e.target.value)}
            disabled={loading}
          >
            <option value="">All Versions</option>
            {unicodeVersions.map(version => (
              <option key={version} value={version}>
                Unicode {version}
              </option>
            ))}
          </select>
        </div>
      </div>

      {/* Quick Filter Buttons */}
      <div className="sidebar-section">
        <h3>Quick Filters</h3>
        <div style={{ display: 'grid', gap: '0.5rem' }}>
          <button
            className="btn btn-secondary btn-sm"
            onClick={() => onFilterChange('category', 'So')}
            disabled={loading}
          >
            Symbols (So)
          </button>
          <button
            className="btn btn-secondary btn-sm"
            onClick={() => onFilterChange('category', 'Sm')}
            disabled={loading}
          >
            Math (Sm)
          </button>
          <button
            className="btn btn-secondary btn-sm"
            onClick={() => onFilterChange('block', 'Emoticons')}
            disabled={loading}
          >
            Emoji
          </button>
          <button
            className="btn btn-secondary btn-sm"
            onClick={() => onFilterChange('block', 'Arrows')}
            disabled={loading}
          >
            Arrows
          </button>
          <button
            className="btn btn-secondary btn-sm"
            onClick={() => onFilterChange('block', 'Currency Symbols')}
            disabled={loading}
          >
            Currency
          </button>
        </div>
      </div>

      {/* Clear Filters */}
      {hasFilters && (
        <div className="sidebar-section">
          <button
            className="btn btn-secondary"
            onClick={onClearFilters}
            disabled={loading}
            style={{ width: '100%' }}
          >
            Clear All Filters
          </button>
        </div>
      )}

      {/* Filter Summary */}
      {hasFilters && (
        <div className="sidebar-section">
          <h3>Active Filters</h3>
          <div style={{ fontSize: '12px', color: 'var(--color-gray-600)' }}>
            {filters.category && (
              <div>Category: {filters.category}</div>
            )}
            {filters.block && (
              <div>Block: {filters.block}</div>
            )}
            {filters.unicodeVersion && (
              <div>Version: {filters.unicodeVersion}</div>
            )}
          </div>
        </div>
      )}

      {/* Loading indicator */}
      {loading && (
        <div style={{
          position: 'absolute',
          top: '1rem',
          right: '1rem',
          display: 'flex',
          alignItems: 'center',
          gap: '0.5rem',
          fontSize: '12px',
          color: 'var(--color-gray-500)'
        }}>
          <div className="loading-spinner" style={{ width: '12px', height: '12px' }} />
          Searching...
        </div>
      )}

      {/* Help Text */}
      <div className="sidebar-section" style={{ 
        fontSize: '12px', 
        color: 'var(--color-gray-500)',
        borderTop: '1px solid var(--color-gray-200)',
        paddingTop: 'var(--spacing-md)'
      }}>
        <p>
          <strong>Search Tips:</strong>
        </p>
        <ul style={{ marginLeft: '1rem', lineHeight: '1.4' }}>
          <li>Use Unicode names: "grinning face"</li>
          <li>Search by codepoint: "U+1F600" or "128512"</li>
          <li>Find similar symbols: "heart", "star", "arrow"</li>
          <li>Combine filters for precise results</li>
        </ul>
      </div>
    </>
  )
})

export default SearchFilters