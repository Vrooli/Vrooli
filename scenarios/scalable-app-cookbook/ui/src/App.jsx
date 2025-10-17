import React, { useState, useEffect } from 'react'
import { BrowserRouter as Router, Routes, Route, Link, useParams, useNavigate } from 'react-router-dom'
import axios from 'axios'
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter'
import { vscDarkPlus } from 'react-syntax-highlighter/dist/esm/styles/prism'
import { Search, Book, Code, Filter, Download, Zap } from 'lucide-react'
import './App.css'

const API_URL = 'http://localhost:3300'

function App() {
  return (
    <Router>
      <div className="app">
        <Routes>
          <Route path="/" element={<CookbookLayout />}>
            <Route index element={<HomePage />} />
            <Route path="patterns/:patternId" element={<PatternPage />} />
            <Route path="chapters" element={<ChaptersPage />} />
            <Route path="search" element={<SearchPage />} />
          </Route>
        </Routes>
      </div>
    </Router>
  )
}

function CookbookLayout({ children }) {
  const [chapters, setChapters] = useState([])
  const [stats, setStats] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  useEffect(() => {
    async function fetchData() {
      try {
        const [chaptersRes, statsRes] = await Promise.all([
          axios.get(`${API_URL}/api/v1/patterns/chapters`),
          axios.get(`${API_URL}/api/v1/patterns/stats`)
        ])
        setChapters(chaptersRes.data)
        setStats(statsRes.data)
      } catch (err) {
        setError(err.message)
      } finally {
        setLoading(false)
      }
    }
    fetchData()
  }, [])

  if (loading) {
    return <div className="loading">Loading Scalable App Cookbook</div>
  }

  if (error) {
    return (
      <div className="error">
        <h1>Service Unavailable</h1>
        <p>Cannot connect to Scalable App Cookbook API</p>
        <p className="error-detail">{error}</p>
        <button onClick={() => window.location.reload()}>Retry</button>
      </div>
    )
  }

  return (
    <div className="app">
      <Sidebar chapters={chapters} stats={stats} />
      <MainContent />
    </div>
  )
}

function Sidebar({ chapters, stats }) {
  const [searchQuery, setSearchQuery] = useState('')

  return (
    <div className="sidebar">
      <div className="sidebar-header">
        <Link to="/" className="sidebar-title">
          <Book className="sidebar-icon" />
          Scalable App Cookbook
        </Link>
        <div className="sidebar-subtitle">
          Architecture Patterns Library
        </div>
      </div>

      <div className="search-container">
        <div className="search-box">
          <Search className="search-icon" />
          <input
            type="text"
            placeholder="Search patterns..."
            className="search-input"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            onKeyPress={(e) => {
              if (e.key === 'Enter' && searchQuery.trim()) {
                window.location.href = `/search?q=${encodeURIComponent(searchQuery)}`
              }
            }}
          />
        </div>
      </div>

      <div className="sidebar-nav">
        <div className="nav-section">
          <div className="nav-section-title">Quick Access</div>
          <Link to="/" className="nav-item">
            <Zap className="nav-icon" />
            Overview
          </Link>
          <Link to="/chapters" className="nav-item">
            <Book className="nav-icon" />
            All Chapters
          </Link>
          <Link to="/search" className="nav-item">
            <Search className="nav-icon" />
            Advanced Search
          </Link>
        </div>

        <div className="nav-section">
          <div className="nav-section-title">Chapters</div>
          {chapters.slice(0, 10).map((chapter, index) => (
            <Link
              key={index}
              to={`/search?chapter=${encodeURIComponent(chapter.name)}`}
              className="nav-item"
            >
              <div>
                <div className="nav-item-title">
                  {chapter.name.replace(/^Part [A-Z] - /, '')}
                </div>
                <div className="nav-item-meta">
                  {chapter.pattern_count} patterns
                </div>
              </div>
            </Link>
          ))}
        </div>

        {stats && (
          <div className="sidebar-stats">
            <div className="stats-title">Library Stats</div>
            <div className="stats-grid">
              <div className="stat-item">
                <div className="stat-number">{stats.statistics.total_patterns}</div>
                <div className="stat-label">Patterns</div>
              </div>
              <div className="stat-item">
                <div className="stat-number">{stats.statistics.total_recipes}</div>
                <div className="stat-label">Recipes</div>
              </div>
              <div className="stat-item">
                <div className="stat-number">{stats.statistics.total_implementations}</div>
                <div className="stat-label">Examples</div>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

function MainContent() {
  return (
    <div className="main-content">
      <Routes>
        <Route index element={<HomePage />} />
        <Route path="patterns/:patternId" element={<PatternPage />} />
        <Route path="chapters" element={<ChaptersPage />} />
        <Route path="search" element={<SearchPage />} />
      </Routes>
    </div>
  )
}

function HomePage() {
  return (
    <div className="content">
      <div className="content-header">
        <h1 className="content-title">
          Scalable App Cookbook
        </h1>
        <div className="content-subtitle">
          A comprehensive, machine-actionable cookbook for building maintainable, scalable applications.
          Each pattern includes step-by-step recipes, reference implementations, and production guidance.
        </div>
      </div>

      <div className="home-sections">
        <div className="home-section">
          <h2>🎯 What's Inside</h2>
          <div className="feature-grid">
            <div className="feature-card">
              <h3>38 Comprehensive Chapters</h3>
              <p>From architectural foundations to AI/ML patterns, covering every aspect of scalable application design.</p>
            </div>
            <div className="feature-card">
              <h3>Machine-Actionable Recipes</h3>
              <p>JSON-structured recipes that agents can execute automatically, with validation and rollback instructions.</p>
            </div>
            <div className="feature-card">
              <h3>Maturity Progression</h3>
              <p>L0 (Prototype) → L4 (Enterprise) - patterns that grow with your application's needs.</p>
            </div>
            <div className="feature-card">
              <h3>Multi-Language Support</h3>
              <p>Reference implementations in Go, JavaScript/TypeScript, Python, Java, and more.</p>
            </div>
          </div>
        </div>

        <div className="home-section">
          <h2>🚀 Getting Started</h2>
          <div className="getting-started">
            <div className="step">
              <div className="step-number">1</div>
              <div className="step-content">
                <h3>Explore Patterns</h3>
                <p>Browse by chapter or search for specific architectural needs</p>
                <Link to="/chapters" className="btn primary">Browse Chapters</Link>
              </div>
            </div>
            <div className="step">
              <div className="step-number">2</div>
              <div className="step-content">
                <h3>Choose Recipe</h3>
                <p>Select greenfield, brownfield, or migration recipe for your use case</p>
              </div>
            </div>
            <div className="step">
              <div className="step-number">3</div>
              <div className="step-content">
                <h3>Generate Code</h3>
                <p>Use CLI or API to generate production-ready code templates</p>
                <code className="code-snippet">scalable-app-cookbook generate jwt-auth go</code>
              </div>
            </div>
          </div>
        </div>

        <div className="home-section">
          <h2>📚 Featured Patterns</h2>
          <div className="featured-patterns">
            <div className="pattern-preview">
              <h3>JWT Authentication & Refresh</h3>
              <div className="pattern-tags">
                <span className="tag">Security</span>
                <span className="tag">Authentication</span>
                <span className="tag">L1-L3</span>
              </div>
              <p>Complete implementation of secure JWT authentication with refresh token rotation and security best practices.</p>
            </div>
            <div className="pattern-preview">
              <h3>Dependency Injection</h3>
              <div className="pattern-tags">
                <span className="tag">Architecture</span>
                <span className="tag">Clean Code</span>
                <span className="tag">L0-L2</span>
              </div>
              <p>Modern DI patterns for testable, maintainable code with container management and lifecycle handling.</p>
            </div>
            <div className="pattern-preview">
              <h3>Event-Driven Architecture</h3>
              <div className="pattern-tags">
                <span className="tag">Microservices</span>
                <span className="tag">Scalability</span>
                <span className="tag">L2-L4</span>
              </div>
              <p>Event sourcing, CQRS, and saga patterns for distributed systems at scale.</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

function PatternPage() {
  const { patternId } = useParams()
  const [pattern, setPattern] = useState(null)
  const [recipes, setRecipes] = useState([])
  const [selectedRecipe, setSelectedRecipe] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  useEffect(() => {
    async function fetchPattern() {
      try {
        const [patternRes, recipesRes] = await Promise.all([
          axios.get(`${API_URL}/api/v1/patterns/${patternId}`),
          axios.get(`${API_URL}/api/v1/patterns/${patternId}/recipes`)
        ])
        setPattern(patternRes.data)
        setRecipes(recipesRes.data.recipes || [])
      } catch (err) {
        setError(err.message)
      } finally {
        setLoading(false)
      }
    }
    fetchPattern()
  }, [patternId])

  if (loading) return <div className="loading">Loading pattern...</div>
  if (error) return <div className="error">Error: {error}</div>

  return (
    <div className="content">
      <div className="content-header">
        <h1 className="content-title">{pattern.title}</h1>
        <div className="content-meta">
          <span className="badge">{pattern.chapter}</span>
          <span className="badge maturity">{pattern.maturity_level}</span>
          <div className="tags">
            {pattern.tags?.map(tag => (
              <span key={tag} className="tag">{tag}</span>
            ))}
          </div>
        </div>
      </div>

      <div className="pattern-content">
        <div className="section">
          <h2 className="section-title">What & Why</h2>
          <div className="section-content">
            {pattern.what_and_why}
          </div>
        </div>

        <div className="section">
          <h2 className="section-title">When to Use</h2>
          <div className="section-content">
            {pattern.when_to_use}
          </div>
        </div>

        <div className="section">
          <h2 className="section-title">Trade-offs</h2>
          <div className="section-content">
            {pattern.tradeoffs}
          </div>
        </div>

        {pattern.failure_modes && (
          <div className="section">
            <h2 className="section-title">Failure Modes & Runbooks</h2>
            <div className="section-content">
              {pattern.failure_modes}
            </div>
          </div>
        )}

        <div className="section">
          <h2 className="section-title">Implementation Recipes</h2>
          <div className="recipes-grid">
            {recipes.map(recipe => (
              <div 
                key={recipe.id} 
                className="recipe-card"
                onClick={() => setSelectedRecipe(recipe)}
              >
                <div className="recipe-card-header">
                  <h3 className="recipe-title">{recipe.title}</h3>
                  <span className={`recipe-type ${recipe.type}`}>
                    {recipe.type}
                  </span>
                </div>
                <div className="recipe-meta">
                  {recipe.steps?.length || 0} steps • {recipe.timeout_sec}s timeout
                </div>
                <div className="recipe-artifacts">
                  {recipe.artifacts?.slice(0, 3).map(artifact => (
                    <span key={artifact} className="tag">{artifact}</span>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </div>

        {selectedRecipe && (
          <RecipeModal 
            recipe={selectedRecipe} 
            onClose={() => setSelectedRecipe(null)} 
          />
        )}
      </div>
    </div>
  )
}

function RecipeModal({ recipe, onClose }) {
  const [implementations, setImplementations] = useState([])
  const [selectedLanguage, setSelectedLanguage] = useState('go')

  useEffect(() => {
    async function fetchImplementations() {
      try {
        const response = await axios.get(`${API_URL}/api/v1/implementations?recipe_id=${recipe.id}`)
        setImplementations(response.data || [])
        if (response.data?.length > 0) {
          setSelectedLanguage(response.data[0].language)
        }
      } catch (err) {
        console.error('Failed to fetch implementations:', err)
      }
    }
    fetchImplementations()
  }, [recipe.id])

  const currentImpl = implementations.find(impl => impl.language === selectedLanguage)

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" onClick={e => e.stopPropagation()}>
        <div className="modal-header">
          <h2>{recipe.title}</h2>
          <button className="modal-close" onClick={onClose}>×</button>
        </div>
        
        <div className="modal-content">
          <div className="recipe-details">
            <div className="recipe-steps">
              <h3>Implementation Steps</h3>
              {recipe.steps?.map((step, index) => (
                <div key={step.id || index} className="step-item">
                  <div className="step-number">{index + 1}</div>
                  <div className="step-content">
                    <h4>{step.desc}</h4>
                    {step.cmds && (
                      <div className="step-commands">
                        {step.cmds.map((cmd, i) => (
                          <code key={i} className="command">{cmd}</code>
                        ))}
                      </div>
                    )}
                  </div>
                </div>
              ))}
            </div>

            {implementations.length > 0 && (
              <div className="implementations">
                <h3>Reference Implementations</h3>
                <div className="language-tabs">
                  {implementations.map(impl => (
                    <button
                      key={impl.language}
                      className={`language-tab ${selectedLanguage === impl.language ? 'active' : ''}`}
                      onClick={() => setSelectedLanguage(impl.language)}
                    >
                      {impl.language}
                    </button>
                  ))}
                </div>
                
                {currentImpl && (
                  <div className="implementation-content">
                    <div className="impl-header">
                      <h4>{currentImpl.description}</h4>
                      <div className="impl-meta">
                        File: <code>{currentImpl.file_path}</code>
                      </div>
                    </div>
                    
                    <SyntaxHighlighter
                      language={getLanguageForHighlighting(currentImpl.language)}
                      style={vscDarkPlus}
                      className="code-block"
                    >
                      {currentImpl.code}
                    </SyntaxHighlighter>
                    
                    {currentImpl.dependencies?.length > 0 && (
                      <div className="dependencies">
                        <h5>Dependencies</h5>
                        <ul>
                          {currentImpl.dependencies.map(dep => (
                            <li key={dep}><code>{dep}</code></li>
                          ))}
                        </ul>
                      </div>
                    )}
                  </div>
                )}
              </div>
            )}
          </div>
        </div>
        
        <div className="modal-actions">
          <button className="btn" onClick={onClose}>Close</button>
          <button className="btn primary">
            <Download className="btn-icon" />
            Generate Code
          </button>
        </div>
      </div>
    </div>
  )
}

function SearchPage() {
  const [patterns, setPatterns] = useState([])
  const [loading, setLoading] = useState(false)
  const [filters, setFilters] = useState({
    query: '',
    chapter: '',
    maturity_level: '',
    tags: ''
  })

  const navigate = useNavigate()

  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    const query = params.get('q') || ''
    const chapter = params.get('chapter') || ''
    
    setFilters(prev => ({ ...prev, query, chapter }))
    if (query || chapter) {
      searchPatterns({ query, chapter })
    }
  }, [])

  const searchPatterns = async (searchFilters = filters) => {
    setLoading(true)
    try {
      const params = new URLSearchParams()
      Object.entries(searchFilters).forEach(([key, value]) => {
        if (value) params.set(key, value)
      })
      
      const response = await axios.get(`${API_URL}/api/v1/patterns/search?${params}`)
      setPatterns(response.data.patterns || [])
    } catch (err) {
      console.error('Search failed:', err)
    } finally {
      setLoading(false)
    }
  }

  const handleSearch = (e) => {
    e.preventDefault()
    searchPatterns()
  }

  return (
    <div className="content">
      <div className="content-header">
        <h1 className="content-title">Search Patterns</h1>
      </div>

      <form onSubmit={handleSearch} className="search-form">
        <div className="search-fields">
          <input
            type="text"
            placeholder="Search patterns, descriptions..."
            className="search-input large"
            value={filters.query}
            onChange={(e) => setFilters(prev => ({ ...prev, query: e.target.value }))}
          />
          <button type="submit" className="btn primary">
            <Search className="btn-icon" />
            Search
          </button>
        </div>
        
        <div className="filter-row">
          <select
            className="filter-select"
            value={filters.chapter}
            onChange={(e) => setFilters(prev => ({ ...prev, chapter: e.target.value }))}
          >
            <option value="">All Chapters</option>
            <option value="Part A - Architectural Foundations">Part A - Architectural Foundations</option>
            <option value="Part B - Resiliency & Scale">Part B - Resiliency & Scale</option>
            <option value="Part C - Platform & Operations">Part C - Platform & Operations</option>
            <option value="Part D - Security, Privacy, Compliance">Part D - Security, Privacy, Compliance</option>
          </select>
          
          <select
            className="filter-select"
            value={filters.maturity_level}
            onChange={(e) => setFilters(prev => ({ ...prev, maturity_level: e.target.value }))}
          >
            <option value="">All Levels</option>
            <option value="L0">L0 - Prototype</option>
            <option value="L1">L1 - Single-node</option>
            <option value="L2">L2 - HA/Multi-AZ</option>
            <option value="L3">L3 - Global</option>
            <option value="L4">L4 - Enterprise</option>
          </select>
        </div>
      </form>

      {loading && <div className="loading-inline">Searching patterns...</div>}

      <div className="search-results">
        {patterns.map(pattern => (
          <div 
            key={pattern.id} 
            className="pattern-result"
            onClick={() => navigate(`/patterns/${pattern.id}`)}
          >
            <div className="pattern-result-header">
              <h3 className="pattern-result-title">{pattern.title}</h3>
              <span className={`badge maturity`}>{pattern.maturity_level}</span>
            </div>
            <div className="pattern-result-meta">
              <span className="pattern-chapter">{pattern.section}</span>
              <span className="pattern-recipes">{pattern.recipe_count} recipes</span>
              <span className="pattern-impls">{pattern.implementation_count} examples</span>
            </div>
            <div className="pattern-result-tags">
              {pattern.tags?.slice(0, 5).map(tag => (
                <span key={tag} className="tag">{tag}</span>
              ))}
            </div>
          </div>
        ))}
      </div>

      {patterns.length === 0 && !loading && (
        <div className="no-results">
          <h3>No patterns found</h3>
          <p>Try adjusting your search terms or filters</p>
        </div>
      )}
    </div>
  )
}

function ChaptersPage() {
  const [chapters, setChapters] = useState([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    async function fetchChapters() {
      try {
        const response = await axios.get(`${API_URL}/api/v1/patterns/chapters`)
        setChapters(response.data || [])
      } catch (err) {
        console.error('Failed to fetch chapters:', err)
      } finally {
        setLoading(false)
      }
    }
    fetchChapters()
  }, [])

  if (loading) return <div className="loading">Loading chapters...</div>

  return (
    <div className="content">
      <div className="content-header">
        <h1 className="content-title">Cookbook Chapters</h1>
        <p className="content-subtitle">
          Comprehensive architectural patterns organized by domain and complexity
        </p>
      </div>

      <div className="chapters-grid">
        {chapters.map((chapter, index) => (
          <Link 
            key={index}
            to={`/search?chapter=${encodeURIComponent(chapter.name)}`}
            className="chapter-card"
          >
            <div className="chapter-header">
              <h3 className="chapter-title">
                {chapter.name.replace(/^Part [A-Z] - /, '')}
              </h3>
              <span className="chapter-part">
                {chapter.name.match(/^Part [A-Z]/)?.[0] || ''}
              </span>
            </div>
            <div className="chapter-stats">
              <span className="chapter-pattern-count">
                {chapter.pattern_count} patterns
              </span>
            </div>
          </Link>
        ))}
      </div>
    </div>
  )
}

function getLanguageForHighlighting(lang) {
  const langMap = {
    'javascript': 'javascript',
    'typescript': 'typescript',
    'go': 'go',
    'python': 'python',
    'java': 'java',
    'rust': 'rust',
    'csharp': 'csharp'
  }
  return langMap[lang] || 'text'
}

export default App