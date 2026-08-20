import { useState, useEffect } from 'react';
import { RefreshCw, Plus } from 'lucide-react';
import type { InvestigationScript } from '../../../types';
import { LoadingSkeleton } from '../../../shared/components/LoadingSkeleton';
import { extractErrorMessage, protoFetch } from '../../../shared/api/apiFetch';
import { useToast } from '../../../shared/components/ToastProvider';
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
  const { showApiError } = useToast();
  const [scripts, setScripts] = useState<InvestigationScript[]>([]);
  const [loading, setLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  const visibleScripts = scripts.filter(script => script.enabled);

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
      setErrorMessage(extractErrorMessage(error, 'Unknown error'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void loadScripts();
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
      showApiError(error);
    }
  };

  const showMoreAlign = embedded ? 'center' : 'flex-end';

  const renderScriptsList = () => {
    if (loading) {
      return <LoadingSkeleton variant="list" count={3} />;
    }

    if (errorMessage) {
      return (
        <div data-sm-style="sm-style-4e86208a56">
          FAILED TO LOAD SCRIPTS
          <br />
          <span data-sm-style="sm-style-634e37ebe1">{errorMessage}</span>
          <br />
          <button type="button" className="btn btn-action"
            onClick={() => { void loadScripts(); }}
            data-sm-style="sm-style-9ed1054ccd">
            <RefreshCw size={14} /> RETRY
          </button>
        </div>
      );
    }

    if (visibleScripts.length === 0) {
      return (
        <div data-sm-style="sm-style-97ea871f93">
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
            onSelect={(script) => { void openScript(script); }}
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
              data-sm-style="sm-style-df8e8add6b"
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
      <div className="panel-header" data-sm-style="sm-style-88ffe06cf7">
        <h2 data-sm-style="sm-style-59e966dafb">
          INVESTIGATION SCRIPTS
        </h2>

        <div className="investigation-script-controls" data-sm-style="sm-style-6f5a4005c4">
          <button
            className="btn btn-action"
            onClick={showNewScriptDialog}
          >
            <Plus size={16} />
            NEW SCRIPT
          </button>
          <button
            className="btn btn-action"
            onClick={() => { void loadScripts(); }}
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
