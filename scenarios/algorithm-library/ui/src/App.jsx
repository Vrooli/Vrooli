import React, { useState, useEffect } from 'react'
import axios from 'axios'
import { Light as SyntaxHighlighter } from 'react-syntax-highlighter'
import { atomOneDark } from 'react-syntax-highlighter/dist/esm/styles/hljs'
import AlgorithmVisualizer from './AlgorithmVisualizer'
import './App.css'

const DEFAULT_API_PORT = (import.meta.env.VITE_API_PORT || '').trim() || '16796'
const API_SUFFIX = '/api/v1'

const normalizeBase = (value = '') => value.replace(/\/+$/, '')

const ensureSuffix = (base) => {
  const normalized = normalizeBase(base)
  return normalized.endsWith(API_SUFFIX) ? normalized : `${normalized}${API_SUFFIX}`
}

const isLocalHost = (hostname = '') => /^(localhost|127\.0\.0\.1|0\.0\.0\.0|\[?::1\]?)/i.test(hostname)

const isProxyPath = (pathname = '') => pathname.includes('/apps/') && pathname.includes('/proxy/')

const computeApiBaseUrl = () => {
  const explicit = (import.meta.env.VITE_API_BASE_URL || '').trim()
  if (explicit) {
    return ensureSuffix(explicit)
  }

  if (typeof window === 'undefined') {
    return ensureSuffix(`http://127.0.0.1:${DEFAULT_API_PORT}`)
  }

  const { origin, hostname, pathname } = window.location
  const hasProxyBootstrap = typeof window.__APP_MONITOR_PROXY_INFO__ !== 'undefined'
  const proxied = hasProxyBootstrap || isProxyPath(pathname)

  if (proxied) {
    const proxyRoot = pathname.split('/proxy')[0] || ''
    const proxyBase = `${origin}${proxyRoot}/proxy`
    return ensureSuffix(proxyBase)
  }

  if (!hostname || isLocalHost(hostname)) {
    return ensureSuffix(`http://127.0.0.1:${DEFAULT_API_PORT}`)
  }

  return ensureSuffix(origin)
}

const API_BASE_URL = computeApiBaseUrl()

const api = axios.create({
  baseURL: API_BASE_URL,
})

const App = () => {
  const [algorithms, setAlgorithms] = useState([])
  const [selectedAlgorithm, setSelectedAlgorithm] = useState(null)
  const [implementations, setImplementations] = useState([])
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedCategory, setSelectedCategory] = useState('')
  const [selectedLanguage, setSelectedLanguage] = useState('python')
  const [loading, setLoading] = useState(false)
  const [stats, setStats] = useState(null)
  const [categories, setCategories] = useState([])
  const [showVisualizer, setShowVisualizer] = useState(false)

  useEffect(() => {
    fetchStats()
    fetchCategories()
    searchAlgorithms()
  }, [])

  const fetchStats = async () => {
    try {
      const response = await api.get('/algorithms/stats')
      setStats(response.data)
    } catch (error) {
      console.error('Failed to fetch stats:', error)
    }
  }

  const fetchCategories = async () => {
    try {
      const response = await api.get('/algorithms/categories')
      setCategories(response.data)
    } catch (error) {
      console.error('Failed to fetch categories:', error)
    }
  }

  const searchAlgorithms = async () => {
    setLoading(true)
    try {
      const params = {}
      if (searchQuery) params.query = searchQuery
      if (selectedCategory) params.category = selectedCategory

      const response = await api.get('/algorithms/search', { params })
      setAlgorithms(response.data.algorithms || [])
    } catch (error) {
      console.error('Search failed:', error)
      setAlgorithms([])
    }
    setLoading(false)
  }

  const selectAlgorithm = async (algo) => {
    setSelectedAlgorithm(algo)
    try {
      const response = await api.get(`/algorithms/${algo.id}/implementations`)
      const impls = response.data.implementations || []
      setImplementations(impls)
      setSelectedLanguage(impls[0]?.language || 'python')
    } catch (error) {
      console.error('Failed to fetch implementations:', error)
      setImplementations([])
    }
  }

  const getComplexityColor = (complexity) => {
    if (complexity.includes('1')) return '#00ff00'
    if (complexity.includes('log')) return '#ffff00'
    if (complexity.includes('n²') || complexity.includes('n^2')) return '#ff9900'
    return '#ff0000'
  }

  return (
    <div className="app">
      <header className="header">
        <div className="header-content">
          <h1 className="title">
            <span className="prompt">$</span> ALGORITHM_LIBRARY
            <span className="cursor">_</span>
          </h1>
          {stats && (
            <div className="stats">
              <span>{stats.statistics.total_algorithms} algorithms</span>
              <span className="separator">|</span>
              <span>{stats.statistics.total_implementations} implementations</span>
              <span className="separator">|</span>
              <span>{stats.statistics.validated_implementations} validated</span>
            </div>
          )}
        </div>
      </header>

      <div className="main-container">
        <aside className="sidebar">
          <div className="search-section">
            <div className="search-box">
              <input
                type="text"
                placeholder="search algorithms..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                onKeyPress={(e) => e.key === 'Enter' && searchAlgorithms()}
                className="search-input"
              />
              <button onClick={searchAlgorithms} className="search-btn">
                SEARCH
              </button>
            </div>
            
            <select 
              value={selectedCategory} 
              onChange={(e) => setSelectedCategory(e.target.value)}
              className="category-select"
            >
              <option value="">All Categories</option>
              {categories.map(cat => (
                <option key={cat.name} value={cat.name}>
                  {cat.name} ({cat.count})
                </option>
              ))}
            </select>
          </div>

          <div className="algorithm-list">
            {loading ? (
              <div className="loading">SCANNING...</div>
            ) : (
              algorithms.map(algo => (
                <div
                  key={algo.id}
                  className={`algorithm-item ${selectedAlgorithm?.id === algo.id ? 'selected' : ''}`}
                  onClick={() => selectAlgorithm(algo)}
                >
                  <div className="algo-name">{algo.display_name}</div>
                  <div className="algo-meta">
                    <span className="complexity" style={{ color: getComplexityColor(algo.complexity_time) }}>
                      {algo.complexity_time}
                    </span>
                    <span className="difficulty">{algo.difficulty}</span>
                    {algo.has_validated_impl && <span className="validated">✓</span>}
                  </div>
                </div>
              ))
            )}
          </div>
        </aside>

        <main className="content">
          {selectedAlgorithm ? (
            <div className="algorithm-detail">
              <div className="detail-header">
                <h2>{selectedAlgorithm.display_name}</h2>
                <div className="detail-meta">
                  <span>Category: {selectedAlgorithm.category}</span>
                  <span>Time: {selectedAlgorithm.complexity_time}</span>
                  <span>Space: {selectedAlgorithm.complexity_space}</span>
                </div>
                {(selectedAlgorithm.category === 'sorting' || 
                  selectedAlgorithm.name?.includes('sort')) && (
                  <button 
                    className="visualize-btn"
                    onClick={() => setShowVisualizer(true)}
                  >
                    ▶ VISUALIZE
                  </button>
                )}
              </div>

              <div className="description">
                <h3>&gt; DESCRIPTION</h3>
                <p>{selectedAlgorithm.description}</p>
              </div>

              {selectedAlgorithm.tags && selectedAlgorithm.tags.length > 0 && (
                <div className="tags">
                  {selectedAlgorithm.tags.map(tag => (
                    <span key={tag} className="tag">#{tag}</span>
                  ))}
                </div>
              )}

              <div className="implementations">
                <h3>&gt; IMPLEMENTATIONS</h3>
                <div className="language-tabs">
                  {implementations.map(impl => (
                    <button
                      key={impl.id}
                      className={`language-tab ${selectedLanguage === impl.language ? 'active' : ''}`}
                      onClick={() => setSelectedLanguage(impl.language)}
                    >
                      {impl.language}
                      {impl.is_primary && <span className="primary">★</span>}
                      {impl.validated && <span className="validated">✓</span>}
                    </button>
                  ))}
                </div>

                {implementations
                  .filter(impl => impl.language === selectedLanguage)
                  .map(impl => (
                    <div key={impl.id} className="code-section">
                      <div className="code-header">
                        <span>{impl.language} v{impl.version}</span>
                        {impl.performance_score && (
                          <span className="performance">
                            Performance: {impl.performance_score}/100
                          </span>
                        )}
                      </div>
                      <SyntaxHighlighter
                        language={impl.language}
                        style={atomOneDark}
                        customStyle={{
                          background: '#0a0a0a',
                          border: '1px solid #00ff00',
                          fontSize: '14px'
                        }}
                      >
                        {impl.code}
                      </SyntaxHighlighter>
                    </div>
                  ))}
              </div>
            </div>
          ) : (
            <div className="welcome">
              <pre className="ascii-art">
{`    ___    __                _ __  __            
   /   |  / /___ _____  ____(_) /_/ /_  ____ ___ 
  / /| | / / __ \`/ __ \\/ ___/ / __/ __ \\/ __ \`__ \\
 / ___ |/ / /_/ / /_/ / /  / / /_/ / / / / / / / /
/_/  |_/_/\\__, /\\____/_/  /_/\\__/_/ /_/_/ /_/ /_/ 
         /____/                                    
         
    L I B R A R Y   v1.0.0
    
    > Ground truth for algorithm implementations
    > Multi-language support
    > Judge0 validated
    > Performance benchmarked
    
    SELECT AN ALGORITHM TO BEGIN...`}
              </pre>
            </div>
          )}
        </main>
      </div>

      {showVisualizer && selectedAlgorithm && (
        <AlgorithmVisualizer
          algorithm={selectedAlgorithm}
          onClose={() => setShowVisualizer(false)}
        />
      )}
    </div>
  )
}

export default App
