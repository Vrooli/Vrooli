import React, { useState, useEffect, useCallback, useMemo, useRef } from 'react'
import { FixedSizeList as List } from 'react-window'
import CharacterItem from './components/CharacterItem'
import CharacterModal from './components/CharacterModal'
import SearchFilters from './components/SearchFilters'
import { useDebounce } from './hooks/useDebounce'
import { useSymbolSearch } from './hooks/useSymbolSearch'
import './index.css'

// Constants for performance optimization
const ITEM_HEIGHT = 80
const VISIBLE_ITEMS = 10
const CONTAINER_HEIGHT = ITEM_HEIGHT * VISIBLE_ITEMS

function App() {
  // State management
  const [searchQuery, setSearchQuery] = useState('')
  const [filters, setFilters] = useState({
    category: '',
    block: '',
    unicodeVersion: ''
  })
  const [selectedCharacter, setSelectedCharacter] = useState(null)
  const [viewMode, setViewMode] = useState('grid') // grid | list
  const [sidebarVisible, setSidebarVisible] = useState(true)

  const loadMoreTriggerRef = useRef(null)
  
  // Debounce search query for performance
  const debouncedQuery = useDebounce(searchQuery, 300)
  
  // Custom hook for API calls
  const {
    characters,
    loading,
    error,
    total,
    queryTime,
    categories,
    blocks,
    searchCharacters,
    loadMore,
    hasMore
  } = useSymbolSearch()

  // Perform search when query or filters change
  useEffect(() => {
    searchCharacters({
      query: debouncedQuery,
      ...filters,
      limit: 100,
      offset: 0
    })
  }, [debouncedQuery, filters, searchCharacters])

  // Handle search input change
  const handleSearchChange = useCallback((e) => {
    setSearchQuery(e.target.value)
  }, [])

  // Handle filter changes
  const handleFilterChange = useCallback((filterName, value) => {
    setFilters(prev => ({
      ...prev,
      [filterName]: value
    }))
  }, [])

  // Handle character selection
  const handleCharacterSelect = useCallback((character) => {
    setSelectedCharacter(character)
  }, [])

  // Handle character modal close
  const handleModalClose = useCallback(() => {
    setSelectedCharacter(null)
  }, [])

  // Load more characters for infinite scrolling
  const handleLoadMore = useCallback(() => {
    if (hasMore && !loading) {
      loadMore({
        query: debouncedQuery,
        ...filters,
        limit: 100,
        offset: characters.length
      })
    }
  }, [hasMore, loading, loadMore, debouncedQuery, filters, characters.length])

  // Trigger load more when grid sentinel reaches viewport
  useEffect(() => {
    if (viewMode !== 'grid') return

    const node = loadMoreTriggerRef.current
    if (!node) return

    const observer = new IntersectionObserver((entries) => {
      const [entry] = entries
      if (entry?.isIntersecting && hasMore && !loading) {
        handleLoadMore()
      }
    }, {
      rootMargin: '200px'
    })

    observer.observe(node)

    return () => {
      observer.disconnect()
    }
  }, [viewMode, hasMore, loading, handleLoadMore])

  // Virtualized list item renderer
  const ListItem = useCallback(({ index, style }) => {
    const character = characters[index]
    
    if (!character) {
      // Loading placeholder
      return (
        <div style={style}>
          <div className="loading-item">
            <div className="loading-spinner" />
            Loading...
          </div>
        </div>
      )
    }

    return (
      <div style={style}>
        <CharacterItem 
          character={character}
          onClick={() => handleCharacterSelect(character)}
          viewMode={viewMode}
        />
      </div>
    )
  }, [characters, viewMode, handleCharacterSelect])

  // Optimized search stats
  const searchStats = useMemo(() => ({
    total,
    loaded: characters.length,
    queryTime: queryTime ? `${queryTime.toFixed(1)}ms` : '0ms',
    hasFilters: !!(filters.category || filters.block || filters.unicodeVersion)
  }), [total, characters.length, queryTime, filters])

  // Clear all filters
  const clearFilters = useCallback(() => {
    setFilters({
      category: '',
      block: '',
      unicodeVersion: ''
    })
    setSearchQuery('')
  }, [])

  const toggleSidebarVisibility = useCallback(() => {
    setSidebarVisible(prev => !prev)
  }, [])

  return (
    <div className="app">
      {/* Header */}
      <header className="header">
        <div className="header-content">
          <div className="header-title">
            <h1>Symbol Search</h1>
            <span className="header-badge">Unicode & ASCII</span>
          </div>
          <div className="header-stats">
            <span>{searchStats.total.toLocaleString()} symbols available</span>
            {searchStats.queryTime && (
              <span>Query time: {searchStats.queryTime}</span>
            )}
          </div>
        </div>
      </header>

      {/* Main content */}
      <main className={`main${sidebarVisible ? '' : ' main--sidebar-hidden'}`}>
        {/* Sidebar Filters */}
        <aside
          className="sidebar"
          id="filters-panel"
          hidden={!sidebarVisible}
        >
          <div className="sidebar-content">
            <SearchFilters
              searchQuery={searchQuery}
              onSearchChange={handleSearchChange}
              filters={filters}
              onFilterChange={handleFilterChange}
              categories={categories}
              blocks={blocks}
              loading={loading}
              onClearFilters={clearFilters}
              hasFilters={searchStats.hasFilters}
            />
          </div>
        </aside>

        {/* Results */}
        <section className="results">
          <div className="results-header">
            <div className="results-info">
              <span className="results-count">
                {searchStats.loaded.toLocaleString()} of {searchStats.total.toLocaleString()}
              </span>
              {searchStats.queryTime && (
                <span className="query-time">({searchStats.queryTime})</span>
              )}
            </div>
            <div className="results-controls">
              <button
                className={`btn btn-sm ${viewMode === 'grid' ? 'btn-primary' : 'btn-secondary'}`}
                onClick={() => setViewMode('grid')}
                type="button"
              >
                Grid
              </button>
              <button
                className={`btn btn-sm ${viewMode === 'list' ? 'btn-primary' : 'btn-secondary'}`}
                onClick={() => setViewMode('list')}
                type="button"
              >
                List
              </button>
              <button
                className={`btn btn-sm ${sidebarVisible ? 'btn-secondary' : 'btn-primary'}`}
                onClick={toggleSidebarVisibility}
                type="button"
                aria-pressed={!sidebarVisible}
                aria-expanded={sidebarVisible}
                aria-controls="filters-panel"
              >
                {sidebarVisible ? 'Hide Filters' : 'Show Filters'}
              </button>
            </div>
          </div>

          <div className="results-content">
            {error && (
              <div className="empty-state">
                <h3>Error</h3>
                <p>{error}</p>
              </div>
            )}

            {!error && characters.length === 0 && !loading && (
              <div className="empty-state">
                <h3>No symbols found</h3>
                <p>Try adjusting your search query or filters.</p>
                {searchStats.hasFilters && (
                  <button 
                    className="btn btn-secondary"
                    onClick={clearFilters}
                    style={{ marginTop: '1rem' }}
                  >
                    Clear Filters
                  </button>
                )}
              </div>
            )}

            {!error && characters.length > 0 && viewMode === 'list' && (
              <div className="virtual-list">
                <List
                  height={CONTAINER_HEIGHT}
                  itemCount={hasMore ? characters.length + 1 : characters.length}
                  itemSize={ITEM_HEIGHT}
                  itemData={characters}
                  onItemsRendered={({ visibleStopIndex }) => {
                    // Load more when approaching the end
                    if (visibleStopIndex >= characters.length - 5) {
                      handleLoadMore()
                    }
                  }}
                  overscanCount={5}
                  className="character-list"
                  style={{ height: 'calc(100vh - 300px)' }}
                >
                  {ListItem}
                </List>
              </div>
            )}

            {!error && characters.length > 0 && viewMode === 'grid' && (
              <div className="grid-results">
                <div className="grid-results-inner">
                  {characters.map((character) => (
                    <CharacterItem
                      key={character.codepoint}
                      character={character}
                      onClick={() => handleCharacterSelect(character)}
                      viewMode="grid"
                    />
                  ))}
                </div>
                <div ref={loadMoreTriggerRef} className="grid-results-sentinel" aria-hidden="true" />
                {loading && hasMore && (
                  <div className="loading-item">Loading more symbols...</div>
                )}
              </div>
            )}

            {loading && characters.length === 0 && (
              <div className="loading-item">
                <div className="loading-spinner" />
                Searching symbols...
              </div>
            )}
          </div>
        </section>
      </main>

      {/* Character Detail Modal */}
      {selectedCharacter && (
        <CharacterModal
          character={selectedCharacter}
          onClose={handleModalClose}
        />
      )}
    </div>
  )
}

export default App
