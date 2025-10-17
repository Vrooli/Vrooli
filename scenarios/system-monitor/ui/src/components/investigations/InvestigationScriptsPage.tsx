import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { FileText, RefreshCw, Plus, Loader2, Eye, Edit, Play, Save } from 'lucide-react';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { tomorrow } from 'react-syntax-highlighter/dist/esm/styles/prism';
import type { InvestigationScript } from '../../types';
import { LoadingSkeleton } from '../common/LoadingSkeleton';
import { buildApiUrl } from '../../utils/apiBase';

interface InvestigationScriptsPageProps {
  onOpenScriptEditor: (script?: InvestigationScript, content?: string, mode?: 'create' | 'edit' | 'view') => void;
  onExecuteScript?: (scriptId: string, content: string) => Promise<void>;
  onSaveScript?: (script: InvestigationScript, content: string) => Promise<void>;
}

interface ScriptContentCache {
  [id: string]: string;
}

export const InvestigationScriptsPage = ({ onOpenScriptEditor, onExecuteScript, onSaveScript }: InvestigationScriptsPageProps) => {
  const [scripts, setScripts] = useState<InvestigationScript[]>([]);
  const [loading, setLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedScriptId, setSelectedScriptId] = useState<string | null>(null);
  const [isFetchingContent, setIsFetchingContent] = useState(false);
  const [selectedContent, setSelectedContent] = useState('');
  const contentCache = useRef<ScriptContentCache>({});
  const [isDesktop, setIsDesktop] = useState(() => window.innerWidth >= 1024);
  const [editorMode, setEditorMode] = useState<'view' | 'edit'>('view');
  const [scriptDraft, setScriptDraft] = useState<InvestigationScript | null>(null);
  const [isRunningScript, setIsRunningScript] = useState(false);
  const [isSavingScript, setIsSavingScript] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);

  const handleResize = useCallback(() => {
    setIsDesktop(window.innerWidth >= 1024);
  }, []);

  useEffect(() => {
    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, [handleResize]);

  const loadScripts = useCallback(async () => {
    setLoading(true);
    setErrorMessage(null);
    try {
      const response = await fetch(buildApiUrl('/api/investigations/scripts'));
      if (!response.ok) {
        throw new Error(`Request failed with status ${response.status}`);
      }
      const data = await response.json();
      const loaded: InvestigationScript[] = Array.isArray(data.scripts) ? data.scripts : [];
      setScripts(loaded);
      if (loaded.length > 0) {
        setSelectedScriptId(prev => prev ?? loaded[0].id);
      } else {
        setSelectedScriptId(null);
        setSelectedContent('');
        setScriptDraft(null);
        setEditorMode('view');
      }
    } catch (error) {
      console.error('Failed to load scripts:', error);
      setScripts([]);
      setErrorMessage(error instanceof Error ? error.message : 'Unknown error');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadScripts();
  }, [loadScripts]);

  const filteredScripts = useMemo(() => {
    if (!searchTerm) {
      return scripts;
    }
    const term = searchTerm.toLowerCase();
    return scripts.filter(script =>
      script.name.toLowerCase().includes(term) ||
      script.description.toLowerCase().includes(term) ||
      script.category.toLowerCase().includes(term) ||
      script.id.toLowerCase().includes(term)
    );
  }, [scripts, searchTerm]);

  const selectedScript = useMemo(() => {
    if (!selectedScriptId) {
      return null;
    }
    return scripts.find(script => script.id === selectedScriptId) ?? null;
  }, [scripts, selectedScriptId]);

  const fetchScriptContent = useCallback(async (script: InvestigationScript) => {
    if (contentCache.current[script.id]) {
      const cachedContent = contentCache.current[script.id];
      setSelectedContent(cachedContent);
      return;
    }
    setIsFetchingContent(true);
    try {
      const response = await fetch(buildApiUrl(`/api/investigations/scripts/${encodeURIComponent(script.id)}`));
      if (!response.ok) {
        throw new Error(`Request failed with status ${response.status}`);
      }
      const data = await response.json();
      const content = typeof data.content === 'string' ? data.content : '';
      contentCache.current[script.id] = content;
      setSelectedContent(content);
    } catch (error) {
      console.error('Failed to load script content:', error);
      setSelectedContent('');
    } finally {
      setIsFetchingContent(false);
    }
  }, []);

  const formatTimestamp = (value?: string) => {
    if (!value) {
      return 'Unknown';
    }
    const parsed = new Date(value);
    if (Number.isNaN(parsed.getTime())) {
      return 'Unknown';
    }
    return parsed.toLocaleString();
  };

  const handleScriptFieldChange = <K extends keyof InvestigationScript>(field: K, value: InvestigationScript[K]) => {
    setScriptDraft(prev => {
      if (!prev) {
        return prev;
      }
      return { ...prev, [field]: value };
    });
  };

  const handleToggleEnabled = () => {
    setScriptDraft(prev => {
      if (!prev) {
        return prev;
      }
      return { ...prev, enabled: !prev.enabled };
    });
  };

  const handleRunScript = useCallback(async () => {
    const scriptId = scriptDraft?.id ?? selectedScript?.id;
    if (!scriptId) {
      return;
    }

    if (!onExecuteScript) {
      onOpenScriptEditor(selectedScript ?? scriptDraft ?? undefined, selectedContent, editorMode === 'edit' ? 'edit' : 'view');
      return;
    }

    setIsRunningScript(true);
    try {
      await onExecuteScript(scriptId, selectedContent);
    } catch (error) {
      console.error('Failed to execute script:', error);
    } finally {
      setIsRunningScript(false);
    }
  }, [editorMode, onExecuteScript, onOpenScriptEditor, scriptDraft, selectedContent, selectedScript]);

  const handleSaveScript = useCallback(async () => {
    if (!scriptDraft) {
      return;
    }

    if (!onSaveScript) {
      onOpenScriptEditor(scriptDraft, selectedContent, 'edit');
      return;
    }

    setIsSavingScript(true);
    setSaveError(null);
    try {
      await onSaveScript(scriptDraft, selectedContent);
      setScripts(prev => prev.map(script => script.id === scriptDraft.id ? { ...scriptDraft } : script));
      contentCache.current[scriptDraft.id] = selectedContent;
      setEditorMode('view');
    } catch (error) {
      console.error('Failed to save script:', error);
      setSaveError(error instanceof Error ? error.message : 'Failed to save script');
    } finally {
      setIsSavingScript(false);
    }
  }, [onSaveScript, onOpenScriptEditor, scriptDraft, selectedContent]);

  const handleSelectScript = useCallback(async (script: InvestigationScript) => {
    setSelectedScriptId(script.id);
    setEditorMode('view');
    setSaveError(null);
    if (!isDesktop) {
      try {
        const response = await fetch(buildApiUrl(`/api/investigations/scripts/${encodeURIComponent(script.id)}`));
        if (!response.ok) {
          throw new Error(`Request failed with status ${response.status}`);
        }
        const data = await response.json();
        const content = typeof data.content === 'string' ? data.content : '';
        onOpenScriptEditor(data.script ?? script, content, 'view');
      } catch (error) {
        console.error('Failed to load script:', error);
        alert(`Failed to load script: ${error instanceof Error ? error.message : 'Unknown error'}`);
      }
      return;
    }

    if (contentCache.current[script.id]) {
      const cachedContent = contentCache.current[script.id];
      setSelectedContent(cachedContent);
      return;
    }
    await fetchScriptContent(script);
  }, [fetchScriptContent, isDesktop, onOpenScriptEditor]);

  useEffect(() => {
    if (!isDesktop || !selectedScript) {
      return;
    }
    if (contentCache.current[selectedScript.id]) {
      setSelectedContent(contentCache.current[selectedScript.id]);
      return;
    }
    void fetchScriptContent(selectedScript);
  }, [fetchScriptContent, isDesktop, selectedScript]);

  useEffect(() => {
    setEditorMode('view');
    setSaveError(null);
    if (selectedScript) {
      setScriptDraft({ ...selectedScript });
    } else {
      setScriptDraft(null);
      setSelectedContent('');
    }
  }, [selectedScript]);

  const handleCreateScript = () => {
    onOpenScriptEditor(undefined, '', 'create');
  };

  const currentScriptData = editorMode === 'edit' && scriptDraft ? scriptDraft : selectedScript;
  const isRunDisabled = isFetchingContent || isRunningScript || !currentScriptData;
  const isSaveDisabled = isSavingScript || !scriptDraft;

  const handleEnterEditMode = () => {
    if (!selectedScript) {
      return;
    }
    setScriptDraft(prev => prev ?? { ...selectedScript });
    setEditorMode('edit');
    setSaveError(null);
  };

  const handleExitEditMode = () => {
    setEditorMode('view');
    setSaveError(null);
  };

  return (
    <div style={{
      padding: '2rem',
      maxWidth: '1200px',
      margin: '0 auto',
      display: 'flex',
      flexDirection: 'column',
      gap: 'var(--spacing-lg)'
    }}>
      <div style={{
        display: 'flex',
        flexDirection: 'column',
        gap: 'var(--spacing-sm)'
      }}>
        <h2 style={{ margin: 0, color: 'var(--color-text-bright)' }}>
          Investigation Scripts Library
        </h2>
        <p style={{ margin: 0, color: 'var(--color-text-dim)', fontSize: 'var(--font-size-sm)' }}>
          Browse and inspect reusable investigative tools. Select a script to review its source, or open it in the editor for deeper analysis.
        </p>
      </div>

      <div style={{
        display: 'flex',
        flexWrap: 'wrap',
        gap: 'var(--spacing-sm)'
      }}>
        <div style={{ position: 'relative', flex: '1 1 260px', maxWidth: '360px' }}>
          <input
            type="text"
            placeholder="Search scripts by name, id, or category"
            value={searchTerm}
            onChange={(event) => setSearchTerm(event.target.value)}
            style={{
              width: '100%',
              padding: '0.6rem 0.8rem',
              background: 'rgba(0, 0, 0, 0.35)',
              border: '1px solid var(--color-accent)',
              borderRadius: 'var(--border-radius-sm)',
              color: 'var(--color-text)',
              fontSize: 'var(--font-size-sm)',
              fontFamily: 'var(--font-family-mono)'
            }}
          />
        </div>

        <div style={{ display: 'flex', gap: 'var(--spacing-sm)' }}>
          <button
            type="button"
            className="btn btn-action"
            onClick={handleCreateScript}
          >
            <Plus size={16} />
            NEW SCRIPT
          </button>
          <button
            type="button"
            className="btn btn-action"
            onClick={() => void loadScripts()}
          >
            <RefreshCw size={16} />
            REFRESH
          </button>
        </div>
      </div>

      {loading ? (
        <LoadingSkeleton variant="card" count={2} />
      ) : errorMessage ? (
        <div style={{
          background: 'rgba(255, 0, 0, 0.1)',
          border: '1px solid var(--color-error)',
          borderRadius: 'var(--border-radius-md)',
          padding: 'var(--spacing-lg)',
          color: 'var(--color-error)',
          textAlign: 'center'
        }}>
          Failed to load scripts: {errorMessage}
        </div>
      ) : filteredScripts.length === 0 ? (
        <div style={{
          textAlign: 'center',
          color: 'var(--color-text-dim)',
          padding: 'var(--spacing-xl)',
          border: '1px dashed var(--color-accent)',
          borderRadius: 'var(--border-radius-md)'
        }}>
          No scripts match the current search.
        </div>
      ) : (
        <div style={{
          display: 'grid',
          gridTemplateColumns: isDesktop ? 'minmax(280px, 1.3fr) minmax(360px, 2fr)' : '1fr',
          gap: 'var(--spacing-lg)'
        }}>
          <div style={{
            border: '1px solid var(--color-accent)',
            borderRadius: 'var(--border-radius-md)',
            overflow: 'hidden',
            background: 'rgba(0, 0, 0, 0.25)'
          }}>
            <div style={{
              padding: 'var(--spacing-sm) var(--spacing-md)',
              borderBottom: '1px solid var(--color-accent)',
              background: 'rgba(0, 255, 0, 0.1)',
              fontSize: 'var(--font-size-xs)',
              letterSpacing: '0.08em',
              textTransform: 'uppercase',
              color: 'var(--color-text-dim)'
            }}>
              {filteredScripts.length} Scripts
            </div>
            <div style={{ maxHeight: isDesktop ? '60vh' : 'auto', overflowY: 'auto' }}>
              {filteredScripts.map(script => {
                const isSelected = script.id === selectedScriptId;
                return (
                  <button
                    key={script.id}
                    type="button"
                    onClick={() => handleSelectScript(script)}
                    style={{
                      width: '100%',
                      textAlign: 'left',
                      padding: 'var(--spacing-md)',
                      border: 'none',
                      background: isSelected ? 'rgba(0, 255, 0, 0.12)' : 'transparent',
                      color: 'var(--color-text)',
                      display: 'flex',
                      flexDirection: 'column',
                      gap: 'var(--spacing-xs)',
                      cursor: 'pointer'
                    }}
                  >
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                      <span style={{
                        display: 'flex',
                        alignItems: 'center',
                        gap: 'var(--spacing-sm)',
                        fontSize: 'var(--font-size-sm)',
                        fontWeight: 600,
                        textTransform: 'uppercase'
                      }}>
                        <FileText size={16} style={{ color: 'var(--color-accent)' }} />
                        {script.name}
                      </span>
                      <span style={{
                        fontSize: 'var(--font-size-xs)',
                        color: script.enabled ? 'var(--color-success)' : 'var(--color-text-dim)',
                        letterSpacing: '0.08em'
                      }}>
                        {script.enabled ? 'ENABLED' : 'DISABLED'}
                      </span>
                    </div>
                    <div style={{
                      display: 'flex',
                      justifyContent: 'space-between',
                      fontSize: 'var(--font-size-xs)',
                      color: 'var(--color-text-dim)'
                    }}>
                      <span>{script.category}</span>
                      <span>{script.author}</span>
                    </div>
                    <p style={{
                      margin: 0,
                      fontSize: 'var(--font-size-xs)',
                      color: 'var(--color-text-dim)'
                    }}>
                      {script.description}
                    </p>
                  </button>
                );
              })}
            </div>
          </div>

          {isDesktop && currentScriptData && (
            <div
              className="script-viewer-pane"
              style={{
                border: '1px solid var(--color-accent)',
                borderRadius: 'var(--border-radius-md)',
                background: 'rgba(0, 0, 0, 0.3)',
                display: 'flex',
                flexDirection: 'column',
                minHeight: '520px',
                overflow: 'hidden'
              }}
            >
              <div
                style={{
                  padding: 'var(--spacing-md) var(--spacing-lg)',
                  borderBottom: '1px solid var(--color-accent)',
                  background: 'rgba(0, 255, 0, 0.05)'
                }}
              >
                <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--spacing-sm)' }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: 'var(--spacing-md)' }}>
                    <div>
                      <h3 style={{ margin: 0, color: 'var(--color-text-bright)', fontSize: 'var(--font-size-lg)' }}>
                        {currentScriptData.name}
                      </h3>
                      <div style={{
                        display: 'flex',
                        flexWrap: 'wrap',
                        gap: 'var(--spacing-xs)',
                        fontSize: 'var(--font-size-xs)',
                        color: 'var(--color-text-dim)',
                        textTransform: 'uppercase',
                        letterSpacing: '0.08em'
                      }}>
                        <span>{currentScriptData.category}</span>
                        <span>•</span>
                        <span>{currentScriptData.author}</span>
                        <span>•</span>
                        <span>{formatTimestamp(currentScriptData.updated_at)}</span>
                      </div>
                    </div>
                    <div style={{ display: 'flex', flexWrap: 'wrap', justifyContent: 'flex-end', gap: 'var(--spacing-sm)' }}>
                      {editorMode === 'view' ? (
                        <button
                          type="button"
                          className="btn btn-action"
                          onClick={handleEnterEditMode}
                          disabled={isFetchingContent}
                          style={{ display: 'flex', alignItems: 'center', gap: 'var(--spacing-xs)' }}
                        >
                          <Edit size={16} />
                          EDIT
                        </button>
                      ) : (
                        <button
                          type="button"
                          className="btn btn-secondary"
                          onClick={handleExitEditMode}
                          style={{ display: 'flex', alignItems: 'center', gap: 'var(--spacing-xs)' }}
                        >
                          <Eye size={16} />
                          VIEW
                        </button>
                      )}
                      <button
                        type="button"
                        className="btn btn-primary"
                        onClick={handleRunScript}
                        disabled={isRunDisabled}
                        style={{ display: 'flex', alignItems: 'center', gap: 'var(--spacing-xs)' }}
                      >
                        {isRunningScript ? <Loader2 size={16} className="animate-spin" /> : <Play size={16} />}
                        {isRunningScript ? 'RUNNING…' : 'RUN'}
                      </button>
                      {editorMode === 'edit' && (
                        <button
                          type="button"
                          className="btn btn-action"
                          onClick={handleSaveScript}
                          disabled={isSaveDisabled}
                          style={{ display: 'flex', alignItems: 'center', gap: 'var(--spacing-xs)' }}
                        >
                          {isSavingScript ? <Loader2 size={16} className="animate-spin" /> : <Save size={16} />}
                          {isSavingScript ? 'SAVING…' : 'SAVE'}
                        </button>
                      )}
                    </div>
                  </div>
                </div>
              </div>

              <div
                style={{
                  padding: 'var(--spacing-md) var(--spacing-lg)',
                  borderBottom: '1px solid rgba(0, 255, 0, 0.2)',
                  background: 'rgba(0, 0, 0, 0.35)',
                  display: 'flex',
                  flexDirection: 'column',
                  gap: 'var(--spacing-md)'
                }}
              >
                {editorMode === 'edit' ? (
                  <>
                    <div style={{
                      display: 'grid',
                      gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))',
                      gap: 'var(--spacing-md)'
                    }}>
                      <div>
                        <label style={{
                          display: 'block',
                          color: 'var(--color-text-dim)',
                          fontSize: 'var(--font-size-sm)',
                          marginBottom: 'var(--spacing-xs)'
                        }}>
                          Name
                        </label>
                        <input
                          type="text"
                          value={scriptDraft?.name ?? ''}
                          onChange={(event) => handleScriptFieldChange('name', event.target.value)}
                          placeholder="Human readable name"
                          style={{
                            width: '100%',
                            padding: 'var(--spacing-sm)',
                            background: 'rgba(0, 0, 0, 0.5)',
                            border: '1px solid var(--color-accent)',
                            borderRadius: 'var(--border-radius-sm)',
                            color: 'var(--color-text)',
                            fontFamily: 'var(--font-family-mono)',
                            fontSize: 'var(--font-size-sm)'
                          }}
                        />
                      </div>
                      <div>
                        <label style={{
                          display: 'block',
                          color: 'var(--color-text-dim)',
                          fontSize: 'var(--font-size-sm)',
                          marginBottom: 'var(--spacing-xs)'
                        }}>
                          Category
                        </label>
                        <select
                          value={scriptDraft?.category ?? 'performance'}
                          onChange={(event) => handleScriptFieldChange('category', event.target.value)}
                          style={{
                            width: '100%',
                            padding: 'var(--spacing-sm)',
                            background: 'rgba(0, 0, 0, 0.5)',
                            border: '1px solid var(--color-accent)',
                            borderRadius: 'var(--border-radius-sm)',
                            color: 'var(--color-text)',
                            fontFamily: 'var(--font-family-mono)',
                            fontSize: 'var(--font-size-sm)'
                          }}
                        >
                          <option value="performance">Performance</option>
                          <option value="process-analysis">Process Analysis</option>
                          <option value="resource-management">Resource Management</option>
                          <option value="network">Network</option>
                          <option value="storage">Storage</option>
                        </select>
                      </div>
                      <div>
                        <label style={{
                          display: 'block',
                          color: 'var(--color-text-dim)',
                          fontSize: 'var(--font-size-sm)',
                          marginBottom: 'var(--spacing-xs)'
                        }}>
                          Author
                        </label>
                        <input
                          type="text"
                          value={scriptDraft?.author ?? ''}
                          onChange={(event) => handleScriptFieldChange('author', event.target.value)}
                          placeholder="Script owner"
                          style={{
                            width: '100%',
                            padding: 'var(--spacing-sm)',
                            background: 'rgba(0, 0, 0, 0.5)',
                            border: '1px solid var(--color-accent)',
                            borderRadius: 'var(--border-radius-sm)',
                            color: 'var(--color-text)',
                            fontFamily: 'var(--font-family-mono)',
                            fontSize: 'var(--font-size-sm)'
                          }}
                        />
                      </div>
                      <div>
                        <label style={{
                          display: 'block',
                          color: 'var(--color-text-dim)',
                          fontSize: 'var(--font-size-sm)',
                          marginBottom: 'var(--spacing-xs)'
                        }}>
                          Script ID
                        </label>
                        <input
                          type="text"
                          value={scriptDraft?.id ?? ''}
                          readOnly
                          style={{
                            width: '100%',
                            padding: 'var(--spacing-sm)',
                            background: 'rgba(0, 0, 0, 0.2)',
                            border: '1px solid rgba(0, 255, 0, 0.15)',
                            borderRadius: 'var(--border-radius-sm)',
                            color: 'var(--color-text-dim)',
                            fontFamily: 'var(--font-family-mono)',
                            fontSize: 'var(--font-size-sm)'
                          }}
                        />
                      </div>
                    </div>

                    <div style={{
                      display: 'flex',
                      flexWrap: 'wrap',
                      gap: 'var(--spacing-md)',
                      alignItems: 'center'
                    }}>
                      <div style={{ flex: '1 1 280px' }}>
                        <label style={{
                          display: 'block',
                          color: 'var(--color-text-dim)',
                          fontSize: 'var(--font-size-sm)',
                          marginBottom: 'var(--spacing-xs)'
                        }}>
                          Description
                        </label>
                        <textarea
                          value={scriptDraft?.description ?? ''}
                          onChange={(event) => handleScriptFieldChange('description', event.target.value)}
                          placeholder="Brief description of what this script investigates"
                          rows={3}
                          style={{
                            width: '100%',
                            padding: 'var(--spacing-sm)',
                            background: 'rgba(0, 0, 0, 0.5)',
                            border: '1px solid var(--color-accent)',
                            borderRadius: 'var(--border-radius-sm)',
                            color: 'var(--color-text)',
                            fontFamily: 'var(--font-family-mono)',
                            fontSize: 'var(--font-size-sm)',
                            resize: 'vertical'
                          }}
                        />
                      </div>

                      <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--spacing-xs)' }}>
                        <span style={{
                          color: 'var(--color-text-dim)',
                          fontSize: 'var(--font-size-sm)',
                          textTransform: 'uppercase',
                          letterSpacing: '0.08em'
                        }}>
                          Status
                        </span>
                        <button
                          type="button"
                          className="btn btn-secondary"
                          onClick={handleToggleEnabled}
                          style={{
                            display: 'flex',
                            alignItems: 'center',
                            gap: 'var(--spacing-xs)',
                            color: scriptDraft?.enabled ? 'var(--color-success)' : 'var(--color-text-dim)'
                          }}
                        >
                          {scriptDraft?.enabled ? 'ENABLED' : 'DISABLED'}
                        </button>
                      </div>

                      <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--spacing-xs)' }}>
                        <span style={{
                          color: 'var(--color-text-dim)',
                          fontSize: 'var(--font-size-sm)',
                          textTransform: 'uppercase',
                          letterSpacing: '0.08em'
                        }}>
                          Created
                        </span>
                        <span style={{
                          fontFamily: 'var(--font-family-mono)',
                          fontSize: 'var(--font-size-xs)',
                          color: 'var(--color-text)'
                        }}>
                          {formatTimestamp(scriptDraft?.created_at)}
                        </span>
                      </div>

                      <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--spacing-xs)' }}>
                        <span style={{
                          color: 'var(--color-text-dim)',
                          fontSize: 'var(--font-size-sm)',
                          textTransform: 'uppercase',
                          letterSpacing: '0.08em'
                        }}>
                          Updated
                        </span>
                        <span style={{
                          fontFamily: 'var(--font-family-mono)',
                          fontSize: 'var(--font-size-xs)',
                          color: 'var(--color-text)'
                        }}>
                          {formatTimestamp(scriptDraft?.updated_at)}
                        </span>
                      </div>
                    </div>
                  </>
                ) : (
                  <>
                    <p style={{ margin: 0, color: 'var(--color-text-dim)', fontSize: 'var(--font-size-sm)' }}>
                      {currentScriptData.description}
                    </p>
                    <div style={{
                      display: 'grid',
                      gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))',
                      gap: 'var(--spacing-md)'
                    }}>
                      <div>
                        <div style={{
                          fontSize: 'var(--font-size-xs)',
                          color: 'var(--color-text-dim)',
                          textTransform: 'uppercase',
                          letterSpacing: '0.08em'
                        }}>
                          Script ID
                        </div>
                        <div style={{
                          fontFamily: 'var(--font-family-mono)',
                          fontSize: 'var(--font-size-xs)',
                          color: 'var(--color-text)'
                        }}>
                          {currentScriptData.id}
                        </div>
                      </div>
                      <div>
                        <div style={{
                          fontSize: 'var(--font-size-xs)',
                          color: 'var(--color-text-dim)',
                          textTransform: 'uppercase',
                          letterSpacing: '0.08em'
                        }}>
                          Created
                        </div>
                        <div style={{
                          fontFamily: 'var(--font-family-mono)',
                          fontSize: 'var(--font-size-xs)',
                          color: 'var(--color-text)'
                        }}>
                          {formatTimestamp(currentScriptData.created_at)}
                        </div>
                      </div>
                      <div>
                        <div style={{
                          fontSize: 'var(--font-size-xs)',
                          color: 'var(--color-text-dim)',
                          textTransform: 'uppercase',
                          letterSpacing: '0.08em'
                        }}>
                          Updated
                        </div>
                        <div style={{
                          fontFamily: 'var(--font-family-mono)',
                          fontSize: 'var(--font-size-xs)',
                          color: 'var(--color-text)'
                        }}>
                          {formatTimestamp(currentScriptData.updated_at)}
                        </div>
                      </div>
                      <div>
                        <div style={{
                          fontSize: 'var(--font-size-xs)',
                          color: 'var(--color-text-dim)',
                          textTransform: 'uppercase',
                          letterSpacing: '0.08em'
                        }}>
                          Status
                        </div>
                        <div style={{
                          fontFamily: 'var(--font-family-mono)',
                          fontSize: 'var(--font-size-xs)',
                          color: currentScriptData.enabled ? 'var(--color-success)' : 'var(--color-text-dim)'
                        }}>
                          {currentScriptData.enabled ? 'ENABLED' : 'DISABLED'}
                        </div>
                      </div>
                    </div>
                  </>
                )}

                {saveError && (
                  <div style={{
                    marginTop: 'var(--spacing-sm)',
                    color: 'var(--color-error)',
                    fontSize: 'var(--font-size-xs)'
                  }}>
                    Failed to save script: {saveError}
                  </div>
                )}
              </div>

              <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
                <div style={{
                  display: 'flex',
                  justifyContent: 'space-between',
                  alignItems: 'center',
                  padding: 'var(--spacing-sm) var(--spacing-lg)',
                  background: 'rgba(0, 255, 0, 0.05)',
                  borderBottom: '1px solid rgba(0, 255, 0, 0.2)',
                  fontSize: 'var(--font-size-sm)',
                  color: 'var(--color-text-bright)'
                }}>
                  <span>Script Code</span>
                  <div style={{ color: 'var(--color-text-dim)' }}>
                    {editorMode === 'view' ? 'Read Only' : 'Editable'} | {(selectedContent || '').length} chars
                  </div>
                </div>
                <div style={{ flex: 1, overflow: 'auto' }}>
                  {isFetchingContent ? (
                    <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--spacing-sm)', color: 'var(--color-text-dim)', padding: 'var(--spacing-md) var(--spacing-lg)' }}>
                      <Loader2 size={16} className="animate-spin" />
                      Loading script…
                    </div>
                  ) : editorMode === 'view' ? (
                    <SyntaxHighlighter
                      language="bash"
                      style={{
                        ...tomorrow,
                        'pre[class*="language-"]': {
                          ...tomorrow['pre[class*="language-"]'],
                          background: 'rgba(0, 0, 0, 0.8)',
                          margin: 0,
                          padding: 'var(--spacing-lg)',
                          fontSize: 'var(--font-size-sm)',
                          fontFamily: 'var(--font-family-mono)',
                          lineHeight: '1.5'
                        }
                      }}
                      customStyle={{
                        background: 'rgba(0, 0, 0, 0.8)',
                        margin: 0,
                        fontSize: 'var(--font-size-sm)',
                        fontFamily: 'var(--font-family-mono)'
                      }}
                    >
                      {selectedContent || '# No script content available'}
                    </SyntaxHighlighter>
                  ) : (
                    <textarea
                      value={selectedContent}
                      onChange={(event) => setSelectedContent(event.target.value)}
                      placeholder="#!/bin/bash\n# Your investigation script here..."
                      style={{
                        width: '100%',
                        height: '100%',
                        padding: 'var(--spacing-lg)',
                        background: 'rgba(0, 0, 0, 0.8)',
                        border: 'none',
                        color: 'var(--color-text)',
                        fontFamily: 'var(--font-family-mono)',
                        fontSize: 'var(--font-size-sm)',
                        lineHeight: '1.5',
                        resize: 'none',
                        outline: 'none',
                        whiteSpace: 'pre',
                        overflowWrap: 'normal',
                        tabSize: 2
                      }}
                    />
                  )}
                </div>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
};
