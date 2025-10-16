import React, { useState, useEffect } from 'react';
import { Search, Database, Tag, DollarSign, AlertCircle, CheckCircle, XCircle, Plus, BookOpen, Settings, Globe, Key, Calendar, FileText, Sparkles } from 'lucide-react';
import axios from 'axios';
import './App.css';
import { resolveApiBase, buildApiUrl as composeApiUrl } from '@vrooli/api-base';

const DEFAULT_API_PORT = process.env.REACT_APP_API_PORT || '15100';

const API_BASE_URL = resolveApiBase({
  explicitUrl: process.env.REACT_APP_API_URL,
  defaultPort: DEFAULT_API_PORT,
  appendSuffix: true,
});

const buildApiUrl = (path) => composeApiUrl(path, { baseUrl: API_BASE_URL });

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

  const getAvatarInitials = (name = '') => {
    if (!name || typeof name !== 'string') {
      return 'API';
    }

    const trimmed = name.trim();
    if (!trimmed) {
      return 'API';
    }

    const words = trimmed.split(/\s+/).filter(Boolean);
    if (words.length === 1) {
      const cleaned = words[0].replace(/[^a-zA-Z0-9]/g, '');
      return (cleaned.slice(0, 2) || words[0].slice(0, 2)).toUpperCase();
    }

    const first = words[0][0] || '';
    const second = words[1][0] || '';
    const initials = `${first}${second}`.trim();
    return (initials || trimmed.slice(0, 2)).toUpperCase();
  };

  const getHostname = (url) => {
    if (!url || typeof url !== 'string') {
      return null;
    }

    try {
      const hostname = new URL(url).hostname;
      return hostname.replace(/^www\./i, '');
    } catch (err) {
      const sanitized = url.replace(/^https?:\/\//i, '').replace(/\/.*$/, '');
      return sanitized ? sanitized : null;
    }
  };

  const normalizeStatusClass = (status) => {
    if (!status || typeof status !== 'string') {
      return 'status-default';
    }

    return `status-${status.toLowerCase().replace(/[^a-z0-9]+/g, '-')}`;
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
                      <div className="api-card-topline">
                        <span className="category-chip">
                          <Tag size={12} />
                          {api.category || 'uncategorized'}
                        </span>
                        {api.configured && (
                          <span className="configured-pill">
                            <CheckCircle size={14} />
                            Configured
                          </span>
                        )}
                      </div>
                      <div className="api-card-header">
                        <div className="api-avatar" aria-hidden="true">
                          {getAvatarInitials(api.name || api.provider)}
                        </div>
                        <div className="api-card-title">
                          <h3>{api.name}</h3>
                          <p className="api-provider">{api.provider}</p>
                        </div>
                        {api.status && (
                          <div className={`api-status-chip ${normalizeStatusClass(api.status)}`}>
                            <span className="status-dot" aria-hidden="true" />
                            <span className="status-label">{api.status}</span>
                          </div>
                        )}
                      </div>
                      <div className="api-card-quick-meta">
                        <div className="quick-meta-item">
                          <Key size={14} />
                          <span>{api.auth_type || 'Auth TBD'}</span>
                        </div>
                        {getHostname(api.base_url) && (
                          <div className="quick-meta-item">
                            <Globe size={14} />
                            <span>{getHostname(api.base_url)}</span>
                          </div>
                        )}
                      </div>
                      <p className="api-description">{api.description}</p>
                      <div className="api-card-metrics">
                        <div className="api-metric">
                          <DollarSign size={14} />
                          <span>{api.pricing_summary || 'Pricing not available'}</span>
                        </div>
                        <div className="api-metric match">
                          <Sparkles size={14} />
                          <span>{`Match ${Math.round(api.relevance_score * 100)}%`}</span>
                        </div>
                      </div>
                      {Array.isArray(api.tags) && api.tags.length > 0 && (
                        <div className="api-card-tags">
                          {api.tags.slice(0, 3).map((tag, idx) => (
                            <span key={`${api.id}-tag-${idx}`} className="api-tag">{tag}</span>
                          ))}
                          {api.tags.length > 3 && (
                            <span className="api-tag more">+{api.tags.length - 3}</span>
                          )}
                        </div>
                      )}
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
