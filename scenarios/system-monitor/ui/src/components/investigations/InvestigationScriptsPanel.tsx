import { useState, useEffect } from 'react';
import { FileText, RefreshCw, Plus } from 'lucide-react';
import type { InvestigationScript } from '../../types';
import { LoadingSkeleton } from '../common/LoadingSkeleton';

interface InvestigationScriptsPanelProps {
  onOpenScriptEditor: (script?: InvestigationScript, content?: string, mode?: 'create' | 'edit' | 'view') => void;
  embedded?: boolean;
  searchFilter?: string;
  maxVisible?: number;
  onShowAll?: () => void;
}

export const InvestigationScriptsPanel = ({
  onOpenScriptEditor,
  embedded = false,
  searchFilter = '',
  maxVisible,
  onShowAll
}: InvestigationScriptsPanelProps) => {
  const [scripts, setScripts] = useState<InvestigationScript[]>([]);
  const [loading, setLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  const visibleScripts = scripts.filter(script => script.enabled !== false);

  // Filter scripts based on search
  const filteredScripts = visibleScripts.filter(script => {
    if (!searchFilter) return true;
    const searchLower = searchFilter.toLowerCase();
    return script.name.toLowerCase().includes(searchLower) ||
           script.description.toLowerCase().includes(searchLower) ||
           script.category.toLowerCase().includes(searchLower) ||
           script.id.toLowerCase().includes(searchLower);
  });

  const scriptsToDisplay = typeof maxVisible === 'number' && maxVisible >= 0
    ? filteredScripts.slice(0, maxVisible)
    : filteredScripts;
  const hasMoreScripts = typeof maxVisible === 'number' && filteredScripts.length > maxVisible;

  const loadScripts = async () => {
    setLoading(true);
    setErrorMessage(null);
    try {
      const response = await fetch('/api/investigations/scripts');
      if (!response.ok) {
        throw new Error(`Request failed with status ${response.status}`);
      }

      const data = await response.json();
      const loadedScripts: InvestigationScript[] = Array.isArray(data.scripts) ? data.scripts : [];
      setScripts(loadedScripts);
    } catch (error) {
      console.error('Failed to load scripts:', error);
      setScripts([]);
      setErrorMessage(error instanceof Error ? error.message : 'Unknown error');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadScripts();
  }, []);

  const showNewScriptDialog = () => {
    onOpenScriptEditor(undefined, '', 'create');
  };

  const openScript = async (script: InvestigationScript) => {
    try {
      const response = await fetch(`/api/investigations/scripts/${encodeURIComponent(script.id)}`);
      if (!response.ok) {
        throw new Error(`Request failed with status ${response.status}`);
      }

      const data = await response.json();
      const scriptContent = typeof data.content === 'string' ? data.content : '';
      const scriptMetadata: InvestigationScript = data.script ?? script;

      onOpenScriptEditor(scriptMetadata, scriptContent, 'view');
    } catch (error) {
      console.error('Failed to load script:', error);
      alert(`Failed to load script: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  };

  // If embedded, render without header
  if (embedded) {
    return (
      <div className="investigation-scripts-list">
        {loading ? (
          <LoadingSkeleton variant="list" count={3} />
        ) : errorMessage ? (
          <div style={{
            textAlign: 'center',
            color: 'var(--color-warning)',
            padding: 'var(--spacing-lg)',
            fontSize: 'var(--font-size-sm)'
          }}>
            FAILED TO LOAD SCRIPTS
            <br />
            <span style={{ fontSize: 'var(--font-size-xs)', color: 'var(--color-text-dim)' }}>{errorMessage}</span>
          </div>
        ) : visibleScripts.length === 0 ? (
          <div style={{
            textAlign: 'center',
            color: 'var(--color-text-dim)',
            padding: 'var(--spacing-lg)',
            fontSize: 'var(--font-size-lg)'
          }}>
            NO SCRIPTS AVAILABLE
          </div>
        ) : (
          <div className="scripts-list">
            {scriptsToDisplay.map(script => (
              <div 
                key={script.id} 
                className="script-item" 
                onClick={() => openScript(script)}
                style={{
                  padding: 'var(--spacing-md)',
                  borderBottom: '1px solid var(--color-accent)',
                  background: 'rgba(0, 0, 0, 0.2)',
                  cursor: 'pointer',
                  transition: 'background 0.2s'
                }}
                onMouseEnter={(e) => e.currentTarget.style.background = 'rgba(0, 0, 0, 0.4)'}
                onMouseLeave={(e) => e.currentTarget.style.background = 'rgba(0, 0, 0, 0.2)'}
              >
                <div style={{
                  display: 'flex',
                  justifyContent: 'space-between',
                  alignItems: 'center',
                  marginBottom: 'var(--spacing-xs)'
                }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--spacing-sm)' }}>
                    <FileText size={16} style={{ color: 'var(--color-accent)' }} />
                    <span style={{ 
                      margin: 0, 
                      color: 'var(--color-text-bright)',
                      fontSize: 'var(--font-size-md)',
                      fontWeight: 'bold'
                    }}>
                      {script.name}
                    </span>
                  </div>
                  
                  <span style={{
                    color: script.enabled ? 'var(--color-success)' : 'var(--color-text-dim)',
                    fontSize: 'var(--font-size-sm)',
                    textTransform: 'uppercase',
                    fontWeight: 'bold'
                  }}>
                    {script.enabled ? 'ENABLED' : 'DISABLED'}
                  </span>
                </div>
                
                <div style={{
                  display: 'flex',
                  justifyContent: 'space-between',
                  marginBottom: 'var(--spacing-sm)'
                }}>
                  <span style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-sm)' }}>
                    Category: {script.category}
                  </span>
                  <span style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-sm)' }}>
                    By: {script.author}
                  </span>
                </div>
                
                <div style={{
                  color: 'var(--color-text)',
                  fontSize: 'var(--font-size-sm)',
                  lineHeight: 1.4
                }}>
                  {script.description}
                </div>
              </div>
            ))}
            {hasMoreScripts && onShowAll && (
              <div style={{
                padding: 'var(--spacing-md)',
                display: 'flex',
                justifyContent: 'center'
              }}>
                <button
                  type="button"
                  className="btn btn-secondary"
                  onClick={onShowAll}
                  style={{ textTransform: 'uppercase', letterSpacing: '0.08em' }}
                >
                  Show More Scripts
                </button>
              </div>
            )}
          </div>
        )}
      </div>
    );
  }

  return (
    <section className="investigation-scripts-panel card">
      <div className="panel-header" style={{
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        marginBottom: 'var(--spacing-md)'
      }}>
        <h2 style={{ margin: 0, color: 'var(--color-text-bright)' }}>
          INVESTIGATION SCRIPTS
        </h2>
        
        <div className="investigation-script-controls" style={{
          display: 'flex',
          gap: 'var(--spacing-sm)'
        }}>
          <button 
            className="btn btn-action"
            onClick={showNewScriptDialog}
          >
            <Plus size={16} />
            NEW SCRIPT
          </button>
          <button 
            className="btn btn-action"
            onClick={loadScripts}
          >
            <RefreshCw size={16} />
            REFRESH
          </button>
          {hasMoreScripts && onShowAll && (
            <button
              type="button"
              className="btn btn-action"
              onClick={onShowAll}
            >
              SHOW ALL
            </button>
          )}
        </div>
      </div>

      <div className="investigation-scripts-list">
        {loading ? (
          <LoadingSkeleton variant="list" count={3} />
        ) : errorMessage ? (
          <div style={{
            textAlign: 'center',
            color: 'var(--color-warning)',
            padding: 'var(--spacing-lg)',
            fontSize: 'var(--font-size-sm)'
          }}>
            FAILED TO LOAD SCRIPTS
            <br />
            <span style={{ fontSize: 'var(--font-size-xs)', color: 'var(--color-text-dim)' }}>{errorMessage}</span>
          </div>
        ) : visibleScripts.length === 0 ? (
          <div style={{
            textAlign: 'center',
            color: 'var(--color-text-dim)',
            padding: 'var(--spacing-lg)',
            fontSize: 'var(--font-size-lg)'
          }}>
            NO SCRIPTS AVAILABLE
          </div>
        ) : (
          <div className="scripts-list">
            {scriptsToDisplay.map(script => (
              <div 
                key={script.id} 
                className="script-item" 
                onClick={() => openScript(script)}
                style={{
                  padding: 'var(--spacing-md)',
                  borderBottom: '1px solid var(--color-accent)',
                  background: 'rgba(0, 0, 0, 0.2)',
                  cursor: 'pointer',
                  transition: 'background 0.2s'
                }}
                onMouseEnter={(e) => e.currentTarget.style.background = 'rgba(0, 0, 0, 0.4)'}
                onMouseLeave={(e) => e.currentTarget.style.background = 'rgba(0, 0, 0, 0.2)'}
              >
                <div style={{
                  display: 'flex',
                  justifyContent: 'space-between',
                  alignItems: 'center',
                  marginBottom: 'var(--spacing-xs)'
                }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--spacing-sm)' }}>
                    <FileText size={16} style={{ color: 'var(--color-accent)' }} />
                    <span style={{ 
                      margin: 0, 
                      color: 'var(--color-text-bright)',
                      fontSize: 'var(--font-size-md)',
                      fontWeight: 'bold'
                    }}>
                      {script.name}
                    </span>
                  </div>
                  
                  <span style={{
                    color: script.enabled ? 'var(--color-success)' : 'var(--color-text-dim)',
                    fontSize: 'var(--font-size-sm)',
                    textTransform: 'uppercase',
                    fontWeight: 'bold'
                  }}>
                    {script.enabled ? 'ENABLED' : 'DISABLED'}
                  </span>
                </div>
                
                <div style={{
                  display: 'flex',
                  justifyContent: 'space-between',
                  marginBottom: 'var(--spacing-sm)'
                }}>
                  <span style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-sm)' }}>
                    Category: {script.category}
                  </span>
                  <span style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-sm)' }}>
                    By: {script.author}
                  </span>
                </div>
                
                <div style={{
                  color: 'var(--color-text)',
                  fontSize: 'var(--font-size-sm)',
                  lineHeight: 1.4
                }}>
                  {script.description}
                </div>
              </div>
            ))}
            {hasMoreScripts && onShowAll && (
              <div style={{
                padding: 'var(--spacing-md)',
                display: 'flex',
                justifyContent: 'flex-end'
              }}>
                <button
                  type="button"
                  className="btn btn-secondary"
                  onClick={onShowAll}
                  style={{ textTransform: 'uppercase', letterSpacing: '0.08em' }}
                >
                  Show More Scripts
                </button>
              </div>
            )}
          </div>
        )}
      </div>
    </section>
  );
};
