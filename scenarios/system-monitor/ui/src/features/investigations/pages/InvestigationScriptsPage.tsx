import { useCallback, useEffect, useMemo, useReducer, useRef, useState } from 'react';
import { RefreshCw, Plus, Loader2, Eye, Edit, Play, Save } from 'lucide-react';
import type { InvestigationScript } from '../../../types';
import { LoadingSkeleton } from '../../../shared/components/LoadingSkeleton';
import { ScriptHighlighter } from '../../../shared/components/LazyScriptHighlighter';
import { protoFetch } from '../../../shared/api/apiFetch';
import { formatTimestampDisplay } from '../../../shared/utils/formatters';
import { parseListScriptsResponse, parseGetScriptResponse } from '../../../shared/api/proto-contracts';
import { ScriptListItem } from '../components/ScriptListItem';

interface InvestigationScriptsPageProps {
  onOpenScriptEditor: (script?: InvestigationScript, content?: string, mode?: 'create' | 'edit' | 'view') => void;
  onExecuteScript?: (scriptId: string, content: string) => Promise<void>;
  onSaveScript?: (script: InvestigationScript, content: string) => Promise<void>;
}

interface ScriptContentCache {
  [id: string]: string;
}

// ── Editor state reducer ──

interface EditorState {
  editorMode: 'view' | 'edit';
  scriptDraft: InvestigationScript | null;
  selectedContent: string;
  isFetchingContent: boolean;
  isRunningScript: boolean;
  isSavingScript: boolean;
  saveError: string | null;
}

type EditorAction =
  | { type: 'SET_EDITOR_MODE'; mode: 'view' | 'edit' }
  | { type: 'SET_SCRIPT_DRAFT'; draft: InvestigationScript | null }
  | { type: 'SET_CONTENT'; content: string }
  | { type: 'SET_FETCHING'; fetching: boolean }
  | { type: 'SET_RUNNING'; running: boolean }
  | { type: 'SET_SAVING'; saving: boolean }
  | { type: 'SET_SAVE_ERROR'; error: string | null }
  | { type: 'RESET_EDITOR' }
  | { type: 'SELECT_SCRIPT'; script: InvestigationScript | null };

const initialEditorState: EditorState = {
  editorMode: 'view',
  scriptDraft: null,
  selectedContent: '',
  isFetchingContent: false,
  isRunningScript: false,
  isSavingScript: false,
  saveError: null,
};

function editorReducer(state: EditorState, action: EditorAction): EditorState {
  switch (action.type) {
    case 'SET_EDITOR_MODE':
      return { ...state, editorMode: action.mode, saveError: null };
    case 'SET_SCRIPT_DRAFT':
      return { ...state, scriptDraft: action.draft };
    case 'SET_CONTENT':
      return { ...state, selectedContent: action.content };
    case 'SET_FETCHING':
      return { ...state, isFetchingContent: action.fetching };
    case 'SET_RUNNING':
      return { ...state, isRunningScript: action.running };
    case 'SET_SAVING':
      return { ...state, isSavingScript: action.saving };
    case 'SET_SAVE_ERROR':
      return { ...state, saveError: action.error };
    case 'RESET_EDITOR':
      return { ...initialEditorState };
    case 'SELECT_SCRIPT':
      return {
        ...state,
        editorMode: 'view',
        saveError: null,
        scriptDraft: action.script ? { ...action.script } : null,
        selectedContent: action.script ? state.selectedContent : '',
      };
    default:
      return state;
  }
}

// ── Component ──

export const InvestigationScriptsPage = ({ onOpenScriptEditor, onExecuteScript, onSaveScript }: InvestigationScriptsPageProps) => {
  const [scripts, setScripts] = useState<InvestigationScript[]>([]);
  const [loading, setLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedScriptId, setSelectedScriptId] = useState<string | null>(null);
  const contentCache = useRef<ScriptContentCache>({});
  const [isDesktop, setIsDesktop] = useState(() => window.innerWidth >= 1024);

  const [editor, dispatch] = useReducer(editorReducer, initialEditorState);
  const { editorMode, scriptDraft, selectedContent, isFetchingContent, isRunningScript, isSavingScript, saveError } = editor;

  const handleResize = useCallback(() => {
    setIsDesktop(window.innerWidth >= 1024);
  }, []);

  useEffect(() => {
    window.addEventListener('resize', handleResize);
    return () => { window.removeEventListener('resize', handleResize); };
  }, [handleResize]);

  const loadScripts = useCallback(async () => {
    setLoading(true);
    setErrorMessage(null);
    try {
      const data = await protoFetch('/investigations/scripts', parseListScriptsResponse);
      const loaded: InvestigationScript[] = Array.isArray(data.scripts) ? [...data.scripts] : [];
      setScripts(loaded);
      if (loaded.length > 0) {
        const firstScript = loaded[0];
        setSelectedScriptId(prev => prev ?? firstScript?.id ?? null);
      } else {
        setSelectedScriptId(null);
        dispatch({ type: 'RESET_EDITOR' });
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
    const cached = contentCache.current[script.id];
    if (cached !== undefined) {
      dispatch({ type: 'SET_CONTENT', content: cached });
      return;
    }
    dispatch({ type: 'SET_FETCHING', fetching: true });
    try {
      const data = await protoFetch(`/investigations/scripts/${encodeURIComponent(script.id)}`, parseGetScriptResponse);
      const content = data.content ?? '';
      contentCache.current[script.id] = content;
      dispatch({ type: 'SET_CONTENT', content });
    } catch (error) {
      console.error('Failed to load script content:', error);
      dispatch({ type: 'SET_CONTENT', content: '' });
    } finally {
      dispatch({ type: 'SET_FETCHING', fetching: false });
    }
  }, []);

  const handleScriptFieldChange = <K extends keyof InvestigationScript>(field: K, value: InvestigationScript[K]) => {
    dispatch({
      type: 'SET_SCRIPT_DRAFT',
      draft: scriptDraft ? { ...scriptDraft, [field]: value } : null,
    });
  };

  const handleToggleEnabled = () => {
    if (scriptDraft) {
      dispatch({
        type: 'SET_SCRIPT_DRAFT',
        draft: { ...scriptDraft, enabled: !scriptDraft.enabled },
      });
    }
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

    dispatch({ type: 'SET_RUNNING', running: true });
    try {
      await onExecuteScript(scriptId, selectedContent);
    } catch (error) {
      console.error('Failed to execute script:', error);
    } finally {
      dispatch({ type: 'SET_RUNNING', running: false });
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

    dispatch({ type: 'SET_SAVING', saving: true });
    dispatch({ type: 'SET_SAVE_ERROR', error: null });
    try {
      await onSaveScript(scriptDraft, selectedContent);
      setScripts(prev => prev.map(script => script.id === scriptDraft.id ? { ...scriptDraft } : script));
      contentCache.current[scriptDraft.id] = selectedContent;
      dispatch({ type: 'SET_EDITOR_MODE', mode: 'view' });
    } catch (error) {
      console.error('Failed to save script:', error);
      dispatch({ type: 'SET_SAVE_ERROR', error: error instanceof Error ? error.message : 'Failed to save script' });
    } finally {
      dispatch({ type: 'SET_SAVING', saving: false });
    }
  }, [onSaveScript, onOpenScriptEditor, scriptDraft, selectedContent]);

  const handleSelectScript = useCallback(async (script: InvestigationScript) => {
    setSelectedScriptId(script.id);
    dispatch({ type: 'SET_EDITOR_MODE', mode: 'view' });
    dispatch({ type: 'SET_SAVE_ERROR', error: null });
    if (!isDesktop) {
      try {
        const data = await protoFetch(`/investigations/scripts/${encodeURIComponent(script.id)}`, parseGetScriptResponse);
        const content = data.content ?? '';
        onOpenScriptEditor(data.script ?? script, content, 'view');
      } catch (error) {
        console.error('Failed to load script:', error);
        alert(`Failed to load script: ${error instanceof Error ? error.message : 'Unknown error'}`);
      }
      return;
    }

    const cachedContent = contentCache.current[script.id];
    if (cachedContent !== undefined) {
      dispatch({ type: 'SET_CONTENT', content: cachedContent });
      return;
    }
    await fetchScriptContent(script);
  }, [fetchScriptContent, isDesktop, onOpenScriptEditor]);

  useEffect(() => {
    if (!isDesktop || !selectedScript) {
      return;
    }
    const cachedSelected = contentCache.current[selectedScript.id];
    if (cachedSelected !== undefined) {
      dispatch({ type: 'SET_CONTENT', content: cachedSelected });
      return;
    }
    void fetchScriptContent(selectedScript);
  }, [fetchScriptContent, isDesktop, selectedScript]);

  useEffect(() => {
    dispatch({ type: 'SELECT_SCRIPT', script: selectedScript });
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
    if (!scriptDraft) {
      dispatch({ type: 'SET_SCRIPT_DRAFT', draft: { ...selectedScript } });
    }
    dispatch({ type: 'SET_EDITOR_MODE', mode: 'edit' });
  };

  const handleExitEditMode = () => {
    dispatch({ type: 'SET_EDITOR_MODE', mode: 'view' });
  };

  return (
    <div className="page-container">
      <div data-sm-style="sm-style-90cbf1ff5e">
        <h2 data-sm-style="sm-style-59e966dafb">
          Investigation Scripts Library
        </h2>
        <p data-sm-style="sm-style-dc296a3b4c">
          Browse and inspect reusable investigative tools. Select a script to review its source, or open it in the editor for deeper analysis.
        </p>
      </div>

      <div data-sm-style="sm-style-e88a5a04fd">
        <div data-sm-style="sm-style-ac4ff15a26">
          <input
            type="text"
            className="input-field"
            placeholder="Search scripts by name, id, or category"
            value={searchTerm}
            onChange={(event) => { setSearchTerm(event.target.value); }}
          />
        </div>

        <div data-sm-style="sm-style-2f94a4de27">
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
        <div className="error-banner" data-sm-style="sm-style-116ddd511d">
          Failed to load scripts: {errorMessage}
        </div>
      ) : filteredScripts.length === 0 ? (
        <div className="text-muted" data-sm-style="sm-style-6e40010e0e">
          No scripts match the current search.
        </div>
      ) : (
        <div style={{
          display: 'grid',
          gridTemplateColumns: isDesktop ? 'minmax(280px, 1.3fr) minmax(360px, 2fr)' : '1fr',
          gap: 'var(--spacing-lg)'
        }}>
          <div data-sm-style="sm-style-b293ccbee1">
            <div className="detail-row-label" data-sm-style="sm-style-593a9e7e18">
              {filteredScripts.length} Scripts
            </div>
            <div style={{ maxHeight: isDesktop ? '60vh' : 'auto', overflowY: 'auto' }}>
              {filteredScripts.map(script => (
                <ScriptListItem
                  key={script.id}
                  script={script}
                  isSelected={script.id === selectedScriptId}
                  onSelect={(script) => { void handleSelectScript(script); }}
                />
              ))}
            </div>
          </div>

          {isDesktop && currentScriptData && (
            <div
              className="script-viewer-pane"
              data-sm-style="sm-style-15502ee84e"
            >
              <div
                data-sm-style="sm-style-640bf0cb84"
              >
                <div data-sm-style="sm-style-9f304072b4">
                  <div data-sm-style="sm-style-748cbc8aad">
                    <div>
                      <h3 data-sm-style="sm-style-d339ab43c7">
                        {currentScriptData.name}
                      </h3>
                      <div className="detail-row-label" data-sm-style="sm-style-2edfc291a6">
                        <span>{currentScriptData.category}</span>
                        <span>•</span>
                        <span>{currentScriptData.author}</span>
                        <span>•</span>
                        <span>{formatTimestampDisplay(currentScriptData.updatedAt)}</span>
                      </div>
                    </div>
                    <div data-sm-style="sm-style-364b891340">
                      {editorMode === 'view' ? (
                        <button
                          type="button"
                          className="btn btn-action"
                          onClick={handleEnterEditMode}
                          disabled={isFetchingContent}
                        >
                          <Edit size={16} />
                          EDIT
                        </button>
                      ) : (
                        <button
                          type="button"
                          className="btn btn-secondary"
                          onClick={handleExitEditMode}
                        >
                          <Eye size={16} />
                          VIEW
                        </button>
                      )}
                      <button
                        type="button"
                        className="btn btn-primary"
                        onClick={() => { void handleRunScript(); }}
                        disabled={isRunDisabled}
                      >
                        {isRunningScript ? <Loader2 size={16} className="animate-spin" /> : <Play size={16} />}
                        {isRunningScript ? 'RUNNING\u2026' : 'RUN'}
                      </button>
                      {editorMode === 'edit' && (
                        <button
                          type="button"
                          className="btn btn-action"
                          onClick={() => { void handleSaveScript(); }}
                          disabled={isSaveDisabled}
                        >
                          {isSavingScript ? <Loader2 size={16} className="animate-spin" /> : <Save size={16} />}
                          {isSavingScript ? 'SAVING\u2026' : 'SAVE'}
                        </button>
                      )}
                    </div>
                  </div>
                </div>
              </div>

              <div
                data-sm-style="sm-style-d93416d25f"
              >
                {editorMode === 'edit' ? (
                  <>
                    <div data-sm-style="sm-style-2babf03b98">
                      <div>
                        <label className="input-label">Name</label>
                        <input
                          type="text"
                          className="input-field"
                          value={scriptDraft?.name ?? ''}
                          onChange={(event) => { handleScriptFieldChange('name', event.target.value); }}
                          placeholder="Human readable name"
                        />
                      </div>
                      <div>
                        <label className="input-label">Category</label>
                        <select
                          className="input-field"
                          value={scriptDraft?.category ?? 'performance'}
                          onChange={(event) => { handleScriptFieldChange('category', event.target.value); }}
                        >
                          <option value="performance">Performance</option>
                          <option value="process-analysis">Process Analysis</option>
                          <option value="resource-management">Resource Management</option>
                          <option value="network">Network</option>
                          <option value="storage">Storage</option>
                        </select>
                      </div>
                      <div>
                        <label className="input-label">Author</label>
                        <input
                          type="text"
                          className="input-field"
                          value={scriptDraft?.author ?? ''}
                          onChange={(event) => { handleScriptFieldChange('author', event.target.value); }}
                          placeholder="Script owner"
                        />
                      </div>
                      <div>
                        <label className="input-label">Script ID</label>
                        <input
                          type="text"
                          className="input-field"
                          value={scriptDraft?.id ?? ''}
                          readOnly
                        />
                      </div>
                    </div>

                    <div data-sm-style="sm-style-14eef746cd">
                      <div data-sm-style="sm-style-ecb84d9e1d">
                        <label className="input-label">Description</label>
                        <textarea
                          className="input-field"
                          value={scriptDraft?.description ?? ''}
                          onChange={(event) => { handleScriptFieldChange('description', event.target.value); }}
                          placeholder="Brief description of what this script investigates"
                          rows={3}
                          data-sm-style="sm-style-d378f0446f"
                        />
                      </div>

                      <div className="detail-row">
                        <span className="detail-row-label">Status</span>
                        <button
                          type="button"
                          className="btn btn-secondary"
                          onClick={handleToggleEnabled}
                          style={{
                            color: scriptDraft?.enabled ? 'var(--color-success)' : 'var(--color-text-secondary)'
                          }}
                        >
                          {scriptDraft?.enabled ? 'ENABLED' : 'DISABLED'}
                        </button>
                      </div>

                      <div className="detail-row">
                        <span className="detail-row-label">Created</span>
                        <span className="detail-row-value-sm">
                          {formatTimestampDisplay(scriptDraft?.createdAt)}
                        </span>
                      </div>

                      <div className="detail-row">
                        <span className="detail-row-label">Updated</span>
                        <span className="detail-row-value-sm">
                          {formatTimestampDisplay(scriptDraft?.updatedAt)}
                        </span>
                      </div>
                    </div>
                  </>
                ) : (
                  <>
                    <p data-sm-style="sm-style-dc296a3b4c">
                      {currentScriptData.description}
                    </p>
                    <div data-sm-style="sm-style-80e86a94fb">
                      <div className="detail-row">
                        <div className="detail-row-label">Script ID</div>
                        <div className="detail-row-value-sm">{currentScriptData.id}</div>
                      </div>
                      <div className="detail-row">
                        <div className="detail-row-label">Created</div>
                        <div className="detail-row-value-sm">{formatTimestampDisplay(currentScriptData.createdAt)}</div>
                      </div>
                      <div className="detail-row">
                        <div className="detail-row-label">Updated</div>
                        <div className="detail-row-value-sm">{formatTimestampDisplay(currentScriptData.updatedAt)}</div>
                      </div>
                      <div className="detail-row">
                        <div className="detail-row-label">Status</div>
                        <div className="detail-row-value-sm" style={{
                          color: currentScriptData.enabled ? 'var(--color-success)' : 'var(--color-text-secondary)'
                        }}>
                          {currentScriptData.enabled ? 'ENABLED' : 'DISABLED'}
                        </div>
                      </div>
                    </div>
                  </>
                )}

                {saveError && (
                  <div className="error-banner" data-sm-style="sm-style-d75c27402a">
                    Failed to save script: {saveError}
                  </div>
                )}
              </div>

              <div data-sm-style="sm-style-6748d27aca">
                <div data-sm-style="sm-style-63de2fe25f">
                  <span>Script Code</span>
                  <div data-sm-style="sm-style-a6b497e153">
                    {editorMode === 'view' ? 'Read Only' : 'Editable'} | {(selectedContent || '').length} chars
                  </div>
                </div>
                <div data-sm-style="sm-style-7cd8982c82">
                  {isFetchingContent ? (
                    <div className="icon-text" data-sm-style="sm-style-80c3287350">
                      <Loader2 size={16} className="animate-spin" />
                      Loading script&hellip;
                    </div>
                  ) : editorMode === 'view' ? (
                    <ScriptHighlighter
                      content={selectedContent || '# No script content available'}
                      padding="var(--spacing-lg)"
                    />
                  ) : (
                    <textarea
                      value={selectedContent}
                      onChange={(event) => { dispatch({ type: 'SET_CONTENT', content: event.target.value }); }}
                      placeholder="#!/bin/bash\n# Your investigation script here..."
                      data-sm-style="sm-style-40528c32d1"
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
