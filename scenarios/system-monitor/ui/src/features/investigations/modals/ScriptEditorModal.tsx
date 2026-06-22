import { useState, useEffect } from 'react';
import { Play, Save, Eye, Edit, Loader } from 'lucide-react';
import { ScriptHighlighter } from '../../../shared/components/LazyScriptHighlighter';
import { Modal, ModalHeader } from '../../../shared/components/Modal';
import type { InvestigationScript } from '../../../types';
import { timestampFromDate } from '@bufbuild/protobuf/wkt';

interface ScriptEditorModalProps {
  isOpen: boolean;
  script?: InvestigationScript;
  scriptContent?: string;
  mode: 'create' | 'edit' | 'view';
  onClose: () => void;
  onExecute: (scriptId: string, content: string) => Promise<void>;
  onSave?: (script: InvestigationScript, content: string) => Promise<void>;
}

export const ScriptEditorModal = ({
  isOpen,
  script,
  scriptContent: initialContent,
  mode,
  onClose,
  onExecute,
  onSave
}: ScriptEditorModalProps) => {
  const [currentMode, setCurrentMode] = useState<'view' | 'edit'>(mode === 'create' ? 'edit' : 'view');
  const [scriptContent, setScriptContent] = useState(initialContent || '');
  const [isExecuting, setIsExecuting] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [scriptData, setScriptData] = useState<InvestigationScript>({
    id: script?.id || '',
    name: script?.name || '',
    description: script?.description || '',
    category: script?.category || 'performance',
    createdAt: script?.createdAt ?? timestampFromDate(new Date()),
    updatedAt: script?.updatedAt ?? timestampFromDate(new Date()),
    author: script?.author || 'user',
    enabled: script?.enabled ?? true
  } as InvestigationScript);

  useEffect(() => {
    if (script) {
      setScriptData(script);
    }
    if (initialContent) {
      setScriptContent(initialContent);
    }
  }, [script, initialContent]);

  const handleExecute = async () => {
    if (!script?.id || !scriptContent) return;

    setIsExecuting(true);
    try {
      await onExecute(script.id, scriptContent);
    } catch (error) {
      console.error('Failed to execute script:', error);
    } finally {
      setIsExecuting(false);
    }
  };

  const handleSave = async () => {
    if (!onSave) return;

    setIsSaving(true);
    try {
      await onSave(scriptData, scriptContent);
    } catch (error) {
      console.error('Failed to save script:', error);
    } finally {
      setIsSaving(false);
    }
  };

  const toggleMode = () => {
    setCurrentMode(currentMode === 'view' ? 'edit' : 'view');
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} className="modal-lg" ariaLabel="Script editor">
      <ModalHeader onClose={onClose}>
        <div className="icon-text" style={{ gap: 'var(--spacing-md)' }}>
          <div style={{ flex: 1 }}>
            <h3 style={{
              margin: '0 0 var(--spacing-xs) 0',
              color: 'var(--color-text-heading)',
              fontSize: 'var(--text-xl)'
            }}>
              {script?.name || 'New Investigation Script'}
            </h3>
            <p style={{
              margin: 0,
              color: 'var(--color-text-secondary)',
              fontSize: 'var(--text-sm)'
            }}>
              {script?.description || 'Enter script description...'}
            </p>
          </div>

          <div className="icon-text">
            {mode !== 'create' && (
              <button
                className="btn btn-action"
                onClick={toggleMode}
                title={currentMode === 'view' ? 'Edit Script' : 'View Script'}
              >
                {currentMode === 'view' ? <Edit size={16} /> : <Eye size={16} />}
                {currentMode === 'view' ? 'EDIT' : 'VIEW'}
              </button>
            )}

            {scriptContent && (
              <button
                className="btn btn-primary"
                onClick={() => { void handleExecute(); }}
                disabled={isExecuting}
                title="Execute Script"
              >
                {isExecuting ? <Loader size={16} className="animate-spin" /> : <Play size={16} />}
                {isExecuting ? 'RUNNING...' : 'RUN'}
              </button>
            )}

            {(currentMode === 'edit' || mode === 'create') && onSave && (
              <button
                className="btn btn-action"
                onClick={() => { void handleSave(); }}
                disabled={isSaving}
                title="Save Script"
              >
                {isSaving ? <Loader size={16} className="animate-spin" /> : <Save size={16} />}
                {isSaving ? 'SAVING...' : 'SAVE'}
              </button>
            )}
          </div>
        </div>
      </ModalHeader>

      <div className="modal-body" style={{
        flex: 1,
        display: 'flex',
        flexDirection: 'column',
        overflow: 'hidden',
        padding: 0
      }}>

        {/* Script Metadata (when editing) */}
        {(currentMode === 'edit' || mode === 'create') && (
          <div className="script-metadata" style={{
            padding: 'var(--spacing-md)',
            borderBottom: '1px solid var(--color-primary-muted)',
            background: 'var(--overlay-medium)'
          }}>
            <div style={{
              display: 'grid',
              gridTemplateColumns: 'repeat(auto-fit, minmax(250px, 1fr))',
              gap: 'var(--spacing-md)'
            }}>
              <div>
                <label className="input-label">Script ID:</label>
                <input
                  type="text"
                  className="input-field"
                  value={scriptData.id}
                  onChange={(e) => { setScriptData({...scriptData, id: e.target.value}); }}
                  placeholder="script-name"
                />
              </div>

              <div>
                <label className="input-label">Name:</label>
                <input
                  type="text"
                  className="input-field"
                  value={scriptData.name}
                  onChange={(e) => { setScriptData({...scriptData, name: e.target.value}); }}
                  placeholder="Human readable name"
                />
              </div>

              <div>
                <label className="input-label">Category:</label>
                <select
                  className="input-field"
                  value={scriptData.category}
                  onChange={(e) => { setScriptData({...scriptData, category: e.target.value}); }}
                >
                  <option value="performance">Performance</option>
                  <option value="process-analysis">Process Analysis</option>
                  <option value="resource-management">Resource Management</option>
                  <option value="network">Network</option>
                  <option value="storage">Storage</option>
                </select>
              </div>
            </div>

            <div style={{ marginTop: 'var(--spacing-md)' }}>
              <label className="input-label">Description:</label>
              <textarea
                className="input-field"
                value={scriptData.description}
                onChange={(e) => { setScriptData({...scriptData, description: e.target.value}); }}
                placeholder="Brief description of what this script investigates"
                rows={2}
                style={{ resize: 'vertical' }}
              />
            </div>
          </div>
        )}

        {/* Code Editor/Viewer */}
        <div className="code-section" style={{
          flex: 1,
          display: 'flex',
          flexDirection: 'column',
          overflow: 'hidden'
        }}>
          <div className="code-header" style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            padding: 'var(--spacing-sm) var(--spacing-md)',
            background: 'var(--color-primary-muted)',
            borderBottom: '1px solid var(--color-primary-muted)',
            fontSize: 'var(--text-sm)',
            color: 'var(--color-text-heading)'
          }}>
            <span>Script Code</span>
            <div style={{ color: 'var(--color-text-secondary)' }}>
              {currentMode === 'view' ? 'Read Only' : 'Editable'} | {scriptContent.length} chars
            </div>
          </div>

          <div style={{ flex: 1, overflow: 'auto' }}>
            {currentMode === 'view' ? (
              <ScriptHighlighter content={scriptContent || '# No script content available'} />
            ) : (
              <textarea
                value={scriptContent}
                onChange={(e) => { setScriptContent(e.target.value); }}
                placeholder="#!/bin/bash&#10;# Your investigation script here..."
                style={{
                  width: '100%',
                  height: '100%',
                  padding: 'var(--spacing-md)',
                  background: 'var(--overlay-backdrop)',
                  border: 'none',
                  color: 'var(--color-text)',
                  fontFamily: 'var(--font-mono)',
                  fontSize: 'var(--text-sm)',
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
    </Modal>
  );
};
