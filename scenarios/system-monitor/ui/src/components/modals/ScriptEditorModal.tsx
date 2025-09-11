import { useState, useEffect } from 'react';
import { X, Play, Save, Eye, Edit, Loader } from 'lucide-react';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { tomorrow } from 'react-syntax-highlighter/dist/esm/styles/prism';
import type { InvestigationScript } from '../../types';

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
    created_at: script?.created_at || new Date().toISOString(),
    updated_at: script?.updated_at || new Date().toISOString(),
    author: script?.author || 'user',
    enabled: script?.enabled ?? true
  });

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

  if (!isOpen) return null;

  return (
    <div className="modal-overlay" style={{
      position: 'fixed',
      top: 0,
      left: 0,
      right: 0,
      bottom: 0,
      background: 'rgba(0, 0, 0, 0.8)',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      zIndex: 2000,
      backdropFilter: 'blur(5px)'
    }}>
      <div className="modal-content" style={{
        background: 'rgba(0, 0, 0, 0.95)',
        border: '1px solid var(--color-accent)',
        borderRadius: 'var(--border-radius-lg)',
        width: '90vw',
        height: '90vh',
        maxWidth: '1200px',
        overflow: 'hidden',
        display: 'flex',
        flexDirection: 'column',
        boxShadow: 'var(--shadow-bright-glow)'
      }}>
        
        {/* Modal Header */}
        <div className="modal-header" style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          padding: 'var(--spacing-lg)',
          borderBottom: '1px solid var(--color-accent)',
          background: 'rgba(0, 255, 0, 0.05)'
        }}>
          <div>
            <h3 style={{ 
              margin: '0 0 var(--spacing-xs) 0',
              color: 'var(--color-text-bright)',
              fontSize: 'var(--font-size-xl)'
            }}>
              {script?.name || 'New Investigation Script'}
            </h3>
            <p style={{
              margin: 0,
              color: 'var(--color-text-dim)',
              fontSize: 'var(--font-size-sm)'
            }}>
              {script?.description || 'Enter script description...'}
            </p>
          </div>
          
          <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--spacing-sm)' }}>
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
                onClick={handleExecute}
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
                onClick={handleSave}
                disabled={isSaving}
                title="Save Script"
              >
                {isSaving ? <Loader size={16} className="animate-spin" /> : <Save size={16} />}
                {isSaving ? 'SAVING...' : 'SAVE'}
              </button>
            )}
            
            <button 
              className="modal-close"
              onClick={onClose}
              style={{
                background: 'none',
                border: 'none',
                color: 'var(--color-text)',
                fontSize: 'var(--font-size-xl)',
                cursor: 'pointer',
                padding: 0,
                width: '32px',
                height: '32px',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center'
              }}
            >
              <X size={20} />
            </button>
          </div>
        </div>

        {/* Modal Body */}
        <div className="modal-body" style={{
          flex: 1,
          display: 'flex',
          flexDirection: 'column',
          overflow: 'hidden'
        }}>
          
          {/* Script Metadata (when editing) */}
          {(currentMode === 'edit' || mode === 'create') && (
            <div className="script-metadata" style={{
              padding: 'var(--spacing-md)',
              borderBottom: '1px solid rgba(0, 255, 0, 0.2)',
              background: 'rgba(0, 0, 0, 0.3)'
            }}>
              <div style={{
                display: 'grid',
                gridTemplateColumns: 'repeat(auto-fit, minmax(250px, 1fr))',
                gap: 'var(--spacing-md)'
              }}>
                <div>
                  <label style={{ 
                    display: 'block',
                    color: 'var(--color-text-dim)', 
                    fontSize: 'var(--font-size-sm)',
                    marginBottom: 'var(--spacing-xs)'
                  }}>
                    Script ID:
                  </label>
                  <input 
                    type="text"
                    value={scriptData.id}
                    onChange={(e) => setScriptData({...scriptData, id: e.target.value})}
                    placeholder="script-name"
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
                    Name:
                  </label>
                  <input 
                    type="text"
                    value={scriptData.name}
                    onChange={(e) => setScriptData({...scriptData, name: e.target.value})}
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
                    Category:
                  </label>
                  <select 
                    value={scriptData.category}
                    onChange={(e) => setScriptData({...scriptData, category: e.target.value})}
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
              </div>
              
              <div style={{ marginTop: 'var(--spacing-md)' }}>
                <label style={{ 
                  display: 'block',
                  color: 'var(--color-text-dim)', 
                  fontSize: 'var(--font-size-sm)',
                  marginBottom: 'var(--spacing-xs)'
                }}>
                  Description:
                </label>
                <textarea 
                  value={scriptData.description}
                  onChange={(e) => setScriptData({...scriptData, description: e.target.value})}
                  placeholder="Brief description of what this script investigates"
                  rows={2}
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
              background: 'rgba(0, 255, 0, 0.05)',
              borderBottom: '1px solid rgba(0, 255, 0, 0.2)',
              fontSize: 'var(--font-size-sm)',
              color: 'var(--color-text-bright)'
            }}>
              <span>Script Code</span>
              <div style={{ color: 'var(--color-text-dim)' }}>
                {currentMode === 'view' ? 'Read Only' : 'Editable'} | {scriptContent.length} chars
              </div>
            </div>
            
            <div style={{ flex: 1, overflow: 'auto' }}>
              {currentMode === 'view' ? (
                <SyntaxHighlighter
                  language="bash"
                  style={{
                    ...tomorrow,
                    'pre[class*="language-"]': {
                      ...tomorrow['pre[class*="language-"]'],
                      background: 'rgba(0, 0, 0, 0.8)',
                      margin: 0,
                      padding: 'var(--spacing-md)',
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
                  {scriptContent || '# No script content available'}
                </SyntaxHighlighter>
              ) : (
                <textarea
                  value={scriptContent}
                  onChange={(e) => setScriptContent(e.target.value)}
                  placeholder="#!/bin/bash&#10;# Your investigation script here..."
                  style={{
                    width: '100%',
                    height: '100%',
                    padding: 'var(--spacing-md)',
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
      </div>
    </div>
  );
};