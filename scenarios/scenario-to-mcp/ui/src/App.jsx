import React, { useState, useEffect, useCallback, useMemo, useRef } from 'react';
import ScenarioGrid from './components/ScenarioGrid';
import AddMCPModal from './components/AddMCPModal';
import DocumentationPanel from './components/DocumentationPanel';
import { RefreshCw, Activity, Server, GitBranch, BookOpen } from 'lucide-react';
import { api, buildApiUrl } from './utils/apiClient';
import './App.css';

function App() {
  const [scenarios, setScenarios] = useState([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState('all');
  const [selectedScenario, setSelectedScenario] = useState(null);
  const [showAddMCP, setShowAddMCP] = useState(false);
  const [stats, setStats] = useState({
    total: 0,
    withMCP: 0,
    active: 0,
    pending: 0
  });
  const [docsOpen, setDocsOpen] = useState(false);
  const [docsLoading, setDocsLoading] = useState(false);
  const [docsError, setDocsError] = useState('');
  const [docEntries, setDocEntries] = useState([]);
  const [activeDocId, setActiveDocId] = useState(null);
  const [activeDocMeta, setActiveDocMeta] = useState(null);
  const [docContent, setDocContent] = useState('');
  const [docContentLoading, setDocContentLoading] = useState(false);
  const [docContentError, setDocContentError] = useState('');
  const [docVersion, setDocVersion] = useState(0);
  const activeDocIdRef = useRef(null);

  const selectedDoc = useMemo(() => {
    if (activeDocMeta && activeDocMeta.id === activeDocId) {
      return activeDocMeta;
    }
    return docEntries.find(doc => doc.id === activeDocId) || activeDocMeta || null;
  }, [activeDocMeta, activeDocId, docEntries]);

  const loadScenarios = useCallback(async () => {
    try {
      setLoading(true);
      const response = await api.get('mcp/endpoints');
      const data = response.data || {};
      const scenarioList = Array.isArray(data.scenarios) ? data.scenarios : [];

      setScenarios(scenarioList);
      setStats({
        total: scenarioList.length,
        withMCP: scenarioList.filter(s => s.hasMCP).length,
        active: scenarioList.filter(s => s.status === 'active').length,
        pending: scenarioList.filter(s => s.status === 'pending').length
      });
    } catch (error) {
      console.error('Failed to load scenarios:', error);
    } finally {
      setLoading(false);
    }
  }, []);

  const loadDocumentationIndex = useCallback(async (preserveSelection = true) => {
    try {
      setDocsLoading(true);
      setDocsError('');
      setDocContentError('');
      const response = await api.get('docs');
      const entries = Array.isArray(response.data?.docs) ? response.data.docs : [];

      setDocEntries(entries);

      if (entries.length === 0) {
        setActiveDocId(null);
        setActiveDocMeta(null);
        setDocContent('');
        return;
      }

      const currentId = activeDocIdRef.current;

      if (preserveSelection && currentId) {
        const existing = entries.find(entry => entry.id === currentId);
        if (existing) {
          setActiveDocId(existing.id);
          setActiveDocMeta(existing);
          return;
        }
      }

      const firstEntry = entries[0];
      setActiveDocId(firstEntry.id);
      setActiveDocMeta(firstEntry);
    } catch (error) {
      console.error('Failed to load documentation index:', error);
      setDocsError('Unable to load documentation index.');
    } finally {
      setDocsLoading(false);
    }
  }, []);

  useEffect(() => {
    activeDocIdRef.current = activeDocId;
  }, [activeDocId]);

  useEffect(() => {
    loadScenarios();
    const interval = setInterval(loadScenarios, 30000);
    return () => clearInterval(interval);
  }, [loadScenarios]);

  useEffect(() => {
    loadDocumentationIndex(false);
  }, [loadDocumentationIndex]);

  useEffect(() => {
    if (!docsOpen) {
      return;
    }

    if (docEntries.length === 0 && !docsLoading) {
      loadDocumentationIndex(true);
    }
  }, [docsOpen, docEntries.length, docsLoading, loadDocumentationIndex]);

  useEffect(() => {
    if (!docsOpen || !activeDocId) {
      return;
    }

    let cancelled = false;

    const fetchContent = async () => {
      setDocContent('');
      setDocContentError('');
      setDocContentLoading(true);

      try {
        const response = await api.get('docs/content', { params: { id: activeDocId } });
        if (cancelled) {
          return;
        }

        const data = response.data || {};
        const { content = '', ...metadata } = data;

        setDocContent(content);

        if (metadata && metadata.id) {
          setActiveDocMeta(prev => {
            if (prev && prev.id === metadata.id) {
              return { ...prev, ...metadata };
            }
            return metadata;
          });
        }
      } catch (error) {
        if (cancelled) {
          return;
        }

        console.error('Failed to load document content:', error);
        setDocContent('');
        setDocContentError('Unable to load document content.');
      } finally {
        if (!cancelled) {
          setDocContentLoading(false);
        }
      }
    };

    fetchContent();

    return () => {
      cancelled = true;
    };
  }, [docsOpen, activeDocId, docVersion]);

  const handleAddMCP = async (scenario) => {
    setSelectedScenario(scenario);
    setShowAddMCP(true);
  };

  const handleMCPAdded = async (scenario, sessionId) => {
    setShowAddMCP(false);
    await loadScenarios();
    
    // Open Claude-code session in new tab if session ID provided
    if (sessionId) {
      const sessionUrl = buildApiUrl(`mcp/sessions/${sessionId}`);
      window.open(sessionUrl, '_blank', 'noopener,noreferrer');
    }
  };

  const handleDocsOpen = (event) => {
    if (event) {
      event.preventDefault();
    }
    setDocsOpen(true);
  };

  const handleDocsClose = () => {
    setDocsOpen(false);
    setDocContentLoading(false);
    setDocContentError('');
  };

  const handleDocSelect = (doc) => {
    if (!doc) {
      return;
    }
    setActiveDocId(doc.id);
    setActiveDocMeta(doc);
  };

  const handleDocsRefresh = async () => {
    await loadDocumentationIndex(true);
    setDocVersion(version => version + 1);
  };

  const filteredScenarios = scenarios.filter(scenario => {
    if (filter === 'all') return true;
    if (filter === 'with-mcp') return scenario.hasMCP;
    if (filter === 'without-mcp') return !scenario.hasMCP;
    if (filter === 'active') return scenario.status === 'active';
    return true;
  });

  return (
    <div className="app">
      <header className="header">
        <div className="header-left">
          <h1 className="title">
            <Server className="icon" />
            Scenario to MCP
          </h1>
          <div className="subtitle">Model Context Protocol Management</div>
        </div>
        
        <div className="header-right">
          <button 
            className="docs-btn"
            type="button"
            onClick={handleDocsOpen}
            aria-label="Open scenario documentation"
          >
            <BookOpen size={18} />
            <span>Docs</span>
            {docEntries.length > 0 && (
              <span className="docs-count" aria-hidden="true">{docEntries.length}</span>
            )}
          </button>
          <button 
            className="refresh-btn"
            type="button"
            onClick={loadScenarios}
            disabled={loading}
          >
            <RefreshCw className={loading ? 'spinning' : ''} />
            Refresh
          </button>
        </div>
      </header>

      <div className="stats-bar">
        <div className="stat">
          <div className="stat-value">{stats.total}</div>
          <div className="stat-label">Total Scenarios</div>
        </div>
        <div className="stat">
          <div className="stat-value accent">{stats.withMCP}</div>
          <div className="stat-label">With MCP</div>
        </div>
        <div className="stat">
          <div className="stat-value success">{stats.active}</div>
          <div className="stat-label">Active</div>
        </div>
        <div className="stat">
          <div className="stat-value warning">{stats.pending}</div>
          <div className="stat-label">Pending</div>
        </div>
      </div>

      <div className="controls">
        <div className="filter-tabs">
          <button 
            className={`tab ${filter === 'all' ? 'active' : ''}`}
            onClick={() => setFilter('all')}
          >
            All
          </button>
          <button 
            className={`tab ${filter === 'with-mcp' ? 'active' : ''}`}
            onClick={() => setFilter('with-mcp')}
          >
            With MCP
          </button>
          <button 
            className={`tab ${filter === 'without-mcp' ? 'active' : ''}`}
            onClick={() => setFilter('without-mcp')}
          >
            Without MCP
          </button>
          <button 
            className={`tab ${filter === 'active' ? 'active' : ''}`}
            onClick={() => setFilter('active')}
          >
            Active
          </button>
        </div>

        <div className="search-box">
          <input 
            type="text" 
            placeholder="Search scenarios..."
            className="search-input"
          />
        </div>
      </div>

      <main className="main-content">
        {loading && scenarios.length === 0 ? (
          <div className="loading">
            <Activity className="spinning" />
            <div>Loading scenarios...</div>
          </div>
        ) : (
          <ScenarioGrid 
            scenarios={filteredScenarios}
            onAddMCP={handleAddMCP}
            onRefresh={loadScenarios}
          />
        )}
      </main>

      {showAddMCP && (
        <AddMCPModal
          scenario={selectedScenario}
          onClose={() => setShowAddMCP(false)}
          onSuccess={handleMCPAdded}
        />
      )}

      <DocumentationPanel
        open={docsOpen}
        onClose={handleDocsClose}
        docs={docEntries}
        loadingDocs={docsLoading}
        docsError={docsError}
        selectedDoc={selectedDoc}
        selectedDocId={activeDocId}
        onSelectDoc={handleDocSelect}
        content={docContent}
        loadingContent={docContentLoading}
        contentError={docContentError}
        onRefreshDocs={handleDocsRefresh}
      />

      <footer className="footer">
        <div className="footer-left">
          <GitBranch className="icon-small" />
          <span>MCP Protocol v1.0</span>
        </div>
        <div className="footer-right">
          <a href={buildApiUrl('mcp/registry')} target="_blank" rel="noopener noreferrer">
            View Registry
          </a>
          <span className="separator">|</span>
          <a href="#" onClick={handleDocsOpen}>
            Documentation
          </a>
        </div>
      </footer>
    </div>
  );
}

export default App;
