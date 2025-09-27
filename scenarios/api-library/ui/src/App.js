import React, { useState, useEffect } from 'react';
import { Search, Database, Tag, DollarSign, AlertCircle, CheckCircle, XCircle, Plus, BookOpen, Settings, Globe, Key, Calendar, FileText } from 'lucide-react';
import axios from 'axios';
import './App.css';

const DEFAULT_API_BASE_URL = 'http://localhost:15100/api/v1';

const resolveApiBaseUrl = () => {
  const raw = process.env.REACT_APP_API_URL;
  if (!raw || typeof raw !== 'string') {
    return DEFAULT_API_BASE_URL;
  }

  const trimmed = raw.trim().replace(/\/+$/, '');
  if (/\/api\/v\d+(?:\/|$)/i.test(trimmed)) {
    return trimmed;
  }

  return `${trimmed}/api/v1`;
};

const API_BASE_URL = resolveApiBaseUrl();

const buildApiUrl = (path) => {
  const normalizedBase = API_BASE_URL.endsWith('/') ? API_BASE_URL.slice(0, -1) : API_BASE_URL;
  const normalizedPath = path.startsWith('/') ? path : `/${path}`;
  return `${normalizedBase}${normalizedPath}`;
};

const normalizeApiSummary = (api) => {
  const score = typeof api?.relevance_score === 'number' ? api.relevance_score : 1;
  const minPrice = typeof api?.min_price === 'number' ? api.min_price : null;
  const summary = (() => {
    if (typeof api?.pricing_summary === 'string' && api.pricing_summary.trim()) {
      return api.pricing_summary.trim();
    }
    if (minPrice === null) {
      return 'Pricing not available';
    }
    if (minPrice === 0) {
      return 'Free tier available';
    }
    const formatted = Number(minPrice).toLocaleString(undefined, {
      minimumFractionDigits: 0,
      maximumFractionDigits: minPrice < 1 ? 4 : 2,
    });
    return `Starts at $${formatted}`;
  })();
  return {
    ...api,
    relevance_score: score,
    configured: Boolean(api?.configured),
    pricing_summary: summary,
    min_price: minPrice,
  };
};

function App() {
  const [searchQuery, setSearchQuery] = useState('');
  const [searchResults, setSearchResults] = useState([]);
  const [selectedAPI, setSelectedAPI] = useState(null);
  const [configuredAPIs, setConfiguredAPIs] = useState([]);
  const [loading, setLoading] = useState(false);
  const [activeTab, setActiveTab] = useState('search');
  const [showAddNote, setShowAddNote] = useState(false);
  const [newNote, setNewNote] = useState({ content: '', type: 'tip' });
  const [researchCapability, setResearchCapability] = useState('');
  const [hasSearched, setHasSearched] = useState(false);
  const [lastSearchTerm, setLastSearchTerm] = useState('');
  const [configuredOnly, setConfiguredOnly] = useState(false);
  const [maxPrice, setMaxPrice] = useState('');
  const [categoryInput, setCategoryInput] = useState('');
  const [searchMethod, setSearchMethod] = useState('keyword');

  useEffect(() => {
    fetchConfiguredAPIs();
    fetchInitialAPIs();
  }, []);

  const fetchConfiguredAPIs = async () => {
    try {
      const response = await axios.get(buildApiUrl('/configured'));
      setConfiguredAPIs(response.data || []);
    } catch (error) {
      console.error('Failed to fetch configured APIs:', error);
    }
  };

  const fetchInitialAPIs = async () => {
    try {
      setLoading(true);
      const response = await axios.get(buildApiUrl('/apis'));
      const apis = Array.isArray(response.data) ? response.data : [];
      setSearchResults(apis.map(normalizeApiSummary));
      setSearchMethod('keyword');
    } catch (error) {
      console.error('Failed to load API catalog:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleSearch = async (e) => {
    e.preventDefault();
    const term = searchQuery.trim();
    if (!term) return;

    setLoading(true);
    setHasSearched(true);
    setLastSearchTerm(term);
    try {
      const filters = {};
      if (configuredOnly) {
        filters.configured = true;
      }

      if (maxPrice.trim()) {
        const parsedPrice = parseFloat(maxPrice.trim());
        if (Number.isNaN(parsedPrice) || parsedPrice < 0) {
          alert('Please enter a valid max price (non-negative number).');
          setLoading(false);
          return;
        }
        filters.max_price = parsedPrice;
      }

      const categories = categoryInput
        .split(',')
        .map((c) => c.trim())
        .filter((c) => c.length > 0);
      if (categories.length > 0) {
        filters.categories = categories;
      }

      const payload = {
        query: term,
        limit: 20,
      };

      if (Object.keys(filters).length > 0) {
        payload.filters = filters;
      }

      const response = await axios.post(buildApiUrl('/search'), payload);
      const results = Array.isArray(response.data?.results) ? response.data.results : [];
      setSearchResults(results.map(normalizeApiSummary));
      setActiveTab('search');
      setSearchMethod(response.data?.method === 'semantic' ? 'semantic' : 'keyword');
    } catch (error) {
      console.error('Search failed:', error);
      alert('Search failed. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  const fetchAPIDetails = async (apiId) => {
    setLoading(true);
    try {
      const response = await axios.get(buildApiUrl(`/apis/${apiId}`));
      setSelectedAPI(response.data);
    } catch (error) {
      console.error('Failed to fetch API details:', error);
      alert('Failed to fetch API details');
    } finally {
      setLoading(false);
    }
  };

  const handleAddNote = async () => {
    if (!selectedAPI || !newNote.content.trim()) return;

    try {
      await axios.post(buildApiUrl(`/apis/${selectedAPI.api.id}/notes`), {
        content: newNote.content,
        type: newNote.type
      });
      
      // Refresh API details to show new note
      await fetchAPIDetails(selectedAPI.api.id);
      setNewNote({ content: '', type: 'tip' });
      setShowAddNote(false);
      alert('Note added successfully!');
    } catch (error) {
      console.error('Failed to add note:', error);
      alert('Failed to add note');
    }
  };

  const handleRequestResearch = async (e) => {
    e.preventDefault();
    if (!researchCapability.trim()) return;

    setLoading(true);
    try {
      const response = await axios.post(buildApiUrl('/request-research'), {
        capability: researchCapability
      });
      alert(`Research requested! ID: ${response.data.research_id}\nEstimated time: ${response.data.estimated_time} seconds`);
      setResearchCapability('');
    } catch (error) {
      console.error('Failed to request research:', error);
      alert('Failed to request research');
    } finally {
      setLoading(false);
    }
  };

  const getNoteIcon = (type) => {
    switch (type) {
      case 'gotcha':
      case 'warning':
        return <AlertCircle className="note-icon warning" />;
      case 'success':
        return <CheckCircle className="note-icon success" />;
      case 'failure':
        return <XCircle className="note-icon failure" />;
      default:
        return <FileText className="note-icon info" />;
    }
  };

  const getStatusBadge = (status) => {
    const statusClasses = {
      active: 'badge-success',
      deprecated: 'badge-warning',
      sunset: 'badge-error',
      beta: 'badge-info'
    };
    return <span className={`badge ${statusClasses[status] || 'badge-default'}`}>{status}</span>;
  };

  return (
    <div className="app">
      <header className="header">
        <div className="header-content">
          <div className="logo">
            <Database />
            <h1>API Library</h1>
          </div>
          <div className="header-subtitle">
            Discover, track, and manage external APIs with institutional knowledge
          </div>
        </div>
      </header>

      <nav className="nav-tabs">
        <button 
          className={`nav-tab ${activeTab === 'search' ? 'active' : ''}`}
          onClick={() => setActiveTab('search')}
        >
          <Search size={16} />
          Search APIs
        </button>
        <button 
          className={`nav-tab ${activeTab === 'configured' ? 'active' : ''}`}
          onClick={() => setActiveTab('configured')}
        >
          <Key size={16} />
          Configured ({configuredAPIs.length})
        </button>
        <button 
          className={`nav-tab ${activeTab === 'research' ? 'active' : ''}`}
          onClick={() => setActiveTab('research')}
        >
          <Globe size={16} />
          Request Research
        </button>
      </nav>

      <main className="main-content">
        {activeTab === 'search' && (
          <div className="search-section">
            <form onSubmit={handleSearch} className="search-form">
              <div className="search-input-wrapper">
                <Search className="search-icon" />
                <input
                  type="text"
                  placeholder="Search for APIs by capability (e.g., 'send emails', 'process payments')"
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="search-input"
                />
                <button type="submit" className="search-button" disabled={loading}>
                  {loading ? 'Searching...' : 'Search'}
                </button>
              </div>
            </form>

            <div className="search-filters">
              <label className="filter-checkbox">
                <input
                  type="checkbox"
                  checked={configuredOnly}
                  onChange={(e) => setConfiguredOnly(e.target.checked)}
                />
                Show configured APIs only
              </label>

              <div className="filter-field">
                <label htmlFor="max-price-input">Max price per request</label>
                <input
                  id="max-price-input"
                  type="number"
                  min="0"
                  step="0.0001"
                  value={maxPrice}
                  onChange={(e) => setMaxPrice(e.target.value)}
                  placeholder="e.g. 0.01"
                />
              </div>

              <div className="filter-field">
                <label htmlFor="category-input">Categories</label>
                <input
                  id="category-input"
                  type="text"
                  value={categoryInput}
                  onChange={(e) => setCategoryInput(e.target.value)}
                  placeholder="Comma separated (e.g. email, sms)"
                />
              </div>
            </div>

            {searchResults.length > 0 && (
              <>
                <div className="results-meta">
                  <span>{`${searchResults.length} ${searchResults.length === 1 ? 'result' : 'results'}`}</span>
                  <span className={`method-badge ${searchMethod}`}>
                    {searchMethod === 'semantic' ? 'Semantic match' : 'Keyword match'}
                  </span>
                </div>
                <div className="results-grid">
                  {searchResults.map((api) => (
                    <div 
                      key={api.id} 
                      className="api-card"
                      onClick={() => fetchAPIDetails(api.id)}
                  >
                    <div className="api-card-header">
                      <h3>{api.name}</h3>
                      {api.configured && <CheckCircle className="configured-icon" />}
                    </div>
                    <p className="api-provider">{api.provider}</p>
                    <p className="api-description">{api.description}</p>
                    <div className="api-pricing">
                      <DollarSign size={12} />
                      <span>{api.pricing_summary || 'Pricing not available'}</span>
                    </div>
                    <div className="api-card-footer">
                      <span className="category-tag">
                        <Tag size={12} />
                        {api.category || 'uncategorized'}
                      </span>
                      <span className="relevance-score">
                        Match: {(api.relevance_score * 100).toFixed(0)}%
                      </span>
                    </div>
                  </div>
                  ))}
                </div>
              </>
            )}
            {hasSearched && !loading && searchResults.length === 0 && (
              <div className="empty-state">
                {lastSearchTerm
                  ? `No APIs matched "${lastSearchTerm}". Try different keywords or broaden the capability.`
                  : 'No APIs matched your search.'}
              </div>
            )}
          </div>
        )}

        {activeTab === 'configured' && (
          <div className="configured-section">
            <h2>Configured APIs</h2>
            <div className="configured-grid">
              {configuredAPIs.map((api) => (
                <div key={api.id} className="configured-card">
                  <div className="configured-header">
                    <h3>{api.name}</h3>
                    <CheckCircle className="configured-icon-large" />
                  </div>
                  <p className="api-provider">{api.provider}</p>
                  <p className="api-description">{api.description}</p>
                  <div className="configured-meta">
                    <span>
                      <Settings size={14} />
                      {api.environment}
                    </span>
                    <span>
                      <Calendar size={14} />
                      {new Date(api.configuration_date).toLocaleDateString()}
                    </span>
                  </div>
                </div>
              ))}
              {configuredAPIs.length === 0 && (
                <p className="empty-state">No APIs configured yet</p>
              )}
            </div>
          </div>
        )}

        {activeTab === 'research' && (
          <div className="research-section">
            <h2>Request API Research</h2>
            <p className="research-description">
              Can't find the API you need? Request automated research to discover new APIs.
            </p>
            <form onSubmit={handleRequestResearch} className="research-form">
              <textarea
                placeholder="Describe the capability you need (e.g., 'payment processing for European markets with strong PSD2 compliance')"
                value={researchCapability}
                onChange={(e) => setResearchCapability(e.target.value)}
                className="research-input"
                rows={4}
              />
              <button type="submit" className="research-button" disabled={loading}>
                <Globe size={16} />
                {loading ? 'Requesting...' : 'Request Research'}
              </button>
            </form>
          </div>
        )}
      </main>

      {selectedAPI && (
        <div className="modal-overlay" onClick={() => setSelectedAPI(null)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h2>{selectedAPI.api.name}</h2>
              {getStatusBadge(selectedAPI.api.status)}
              <button className="close-button" onClick={() => setSelectedAPI(null)}>×</button>
            </div>
            
            <div className="modal-body">
              <div className="api-details">
                <div className="detail-row">
                  <span className="detail-label">Provider:</span>
                  <span>{selectedAPI.api.provider}</span>
                </div>
                <div className="detail-row">
                  <span className="detail-label">Category:</span>
                  <span>{selectedAPI.api.category || 'N/A'}</span>
                </div>
                <div className="detail-row">
                  <span className="detail-label">Auth Type:</span>
                  <span>{selectedAPI.api.auth_type || 'N/A'}</span>
                </div>
                <div className="detail-row">
                  <span className="detail-label">Description:</span>
                  <p>{selectedAPI.api.description}</p>
                </div>
                
                {selectedAPI.api.base_url && (
                  <div className="detail-row">
                    <span className="detail-label">Base URL:</span>
                    <a href={selectedAPI.api.base_url} target="_blank" rel="noopener noreferrer">
                      {selectedAPI.api.base_url}
                    </a>
                  </div>
                )}
                
                {selectedAPI.api.documentation_url && (
                  <div className="detail-row">
                    <span className="detail-label">Documentation:</span>
                    <a href={selectedAPI.api.documentation_url} target="_blank" rel="noopener noreferrer">
                      <BookOpen size={14} /> View Docs
                    </a>
                  </div>
                )}
                
                {selectedAPI.api.pricing_url && (
                  <div className="detail-row">
                    <span className="detail-label">Pricing:</span>
                    <a href={selectedAPI.api.pricing_url} target="_blank" rel="noopener noreferrer">
                      <DollarSign size={14} /> View Pricing
                    </a>
                  </div>
                )}
                
                {selectedAPI.api.tags && selectedAPI.api.tags.length > 0 && (
                  <div className="detail-row">
                    <span className="detail-label">Tags:</span>
                    <div className="tags-list">
                      {selectedAPI.api.tags.map((tag, idx) => (
                        <span key={idx} className="tag">{tag}</span>
                      ))}
                    </div>
                  </div>
                )}
              </div>

              <div className="notes-section">
                <div className="notes-header">
                  <h3>Notes & Gotchas</h3>
                  <button 
                    className="add-note-button"
                    onClick={() => setShowAddNote(!showAddNote)}
                  >
                    <Plus size={16} /> Add Note
                  </button>
                </div>
                
                {showAddNote && (
                  <div className="add-note-form">
                    <select 
                      value={newNote.type}
                      onChange={(e) => setNewNote({...newNote, type: e.target.value})}
                      className="note-type-select"
                    >
                      <option value="tip">Tip</option>
                      <option value="gotcha">Gotcha</option>
                      <option value="warning">Warning</option>
                      <option value="example">Example</option>
                      <option value="success">Success Story</option>
                      <option value="failure">Failure/Issue</option>
                    </select>
                    <textarea
                      placeholder="Enter your note..."
                      value={newNote.content}
                      onChange={(e) => setNewNote({...newNote, content: e.target.value})}
                      className="note-input"
                      rows={3}
                    />
                    <div className="note-actions">
                      <button onClick={handleAddNote} className="save-note-button">Save</button>
                      <button onClick={() => setShowAddNote(false)} className="cancel-note-button">Cancel</button>
                    </div>
                  </div>
                )}
                
                <div className="notes-list">
                  {selectedAPI.notes && selectedAPI.notes.length > 0 ? (
                    selectedAPI.notes.map((note) => (
                      <div key={note.id} className={`note note-${note.type}`}>
                        <div className="note-header">
                          {getNoteIcon(note.type)}
                          <span className="note-type">{note.type.toUpperCase()}</span>
                          <span className="note-date">
                            {new Date(note.created_at).toLocaleDateString()}
                          </span>
                        </div>
                        <p className="note-content">{note.content}</p>
                        <span className="note-author">by {note.created_by}</span>
                      </div>
                    ))
                  ) : (
                    <p className="no-notes">No notes yet. Be the first to add one!</p>
                  )}
                </div>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default App;
