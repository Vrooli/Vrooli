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
        <div className="icon-text" data-sm-style="sm-style-8322eb66c1">
          <div data-sm-style="sm-style-634a28bea4">
            <h3 data-sm-style="sm-style-bd3930e88e">
              {script?.name || 'New Investigation Script'}
            </h3>
            <p data-sm-style="sm-style-5c239e09c9">
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

      <div className="modal-body" data-sm-style="sm-style-500d25478f">

        {/* Script Metadata (when editing) */}
        {(currentMode === 'edit' || mode === 'create') && (
          <div className="script-metadata" data-sm-style="sm-style-36227d717f">
            <div data-sm-style="sm-style-be85f48eed">
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

            <div data-sm-style="sm-style-323fdcc1e0">
              <label className="input-label">Description:</label>
              <textarea
                className="input-field"
                value={scriptData.description}
                onChange={(e) => { setScriptData({...scriptData, description: e.target.value}); }}
                placeholder="Brief description of what this script investigates"
                rows={2}
                data-sm-style="sm-style-d378f0446f"
              />
            </div>
          </div>
        )}

        {/* Code Editor/Viewer */}
        <div className="code-section" data-sm-style="sm-style-79f4859d20">
          <div className="code-header" data-sm-style="sm-style-f6a9ea2964">
            <span>Script Code</span>
            <div data-sm-style="sm-style-a6b497e153">
              {currentMode === 'view' ? 'Read Only' : 'Editable'} | {scriptContent.length} chars
            </div>
          </div>

          <div data-sm-style="sm-style-7cd8982c82">
            {currentMode === 'view' ? (
              <ScriptHighlighter content={scriptContent || '# No script content available'} />
            ) : (
              <textarea
                value={scriptContent}
                onChange={(e) => { setScriptContent(e.target.value); }}
                placeholder="#!/bin/bash&#10;# Your investigation script here..."
                data-sm-style="sm-style-799c3e1860"
              />
            )}
          </div>
        </div>
      </div>
    </Modal>
  );
};
