import { useState, useEffect } from 'react';
import { RefreshCw, Plus } from 'lucide-react';
import type { InvestigationScript } from '../../../types';
import { LoadingSkeleton } from '../../../shared/components/LoadingSkeleton';
import { protoFetch } from '../../../shared/api/apiFetch';
import { parseListScriptsResponse, parseGetScriptResponse } from '../../../shared/api/proto-contracts';
import { ScriptListItem } from './ScriptListItem';

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
      const data = await protoFetch('/investigations/scripts', parseListScriptsResponse);
      const loadedScripts: InvestigationScript[] = Array.isArray(data.scripts) ? [...data.scripts] : [];
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
      const data = await protoFetch(
        `/investigations/scripts/${encodeURIComponent(script.id)}`,
        parseGetScriptResponse,
      );
      const scriptContent = data.content ?? '';
      const scriptMetadata: InvestigationScript = data.script ?? script;

      onOpenScriptEditor(scriptMetadata, scriptContent, 'view');
    } catch (error) {
      console.error('Failed to load script:', error);
      alert(`Failed to load script: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  };

  const showMoreAlign = embedded ? 'center' : 'flex-end';

  const renderScriptsList = () => {
    if (loading) {
      return <LoadingSkeleton variant="list" count={3} />;
    }

    if (errorMessage) {
      return (
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
      );
    }

    if (visibleScripts.length === 0) {
      return (
        <div style={{
          textAlign: 'center',
          color: 'var(--color-text-dim)',
          padding: 'var(--spacing-lg)',
          fontSize: 'var(--font-size-lg)'
        }}>
          NO SCRIPTS AVAILABLE
        </div>
      );
    }

    return (
      <div className="scripts-list">
        {scriptsToDisplay.map(script => (
          <ScriptListItem
            key={script.id}
            script={script}
            isSelected={false}
            onSelect={openScript}
          />
        ))}
        {hasMoreScripts && onShowAll && (
          <div style={{
            padding: 'var(--spacing-md)',
            display: 'flex',
            justifyContent: showMoreAlign
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
    );
  };

  if (embedded) {
    return (
      <div className="investigation-scripts-list">
        {renderScriptsList()}
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
        {renderScriptsList()}
      </div>
    </section>
  );
};
