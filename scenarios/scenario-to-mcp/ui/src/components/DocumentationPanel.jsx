import React, { useMemo, useCallback } from 'react';
import { X, BookOpen, FileText, RefreshCw } from 'lucide-react';
import { renderMarkdown } from '../utils/markdown';
import './DocumentationPanel.css';

const DocumentationPanel = ({
  open,
  onClose,
  docs,
  loadingDocs,
  docsError,
  selectedDoc,
  selectedDocId,
  onSelectDoc,
  content,
  loadingContent,
  contentError,
  onRefreshDocs
}) => {
  const groupedDocs = useMemo(() => {
    if (!Array.isArray(docs) || docs.length === 0) {
      return [];
    }

    const map = new Map();
    docs.forEach(doc => {
      const key = doc?.category || 'Documentation';
      if (!map.has(key)) {
        map.set(key, []);
      }
      map.get(key).push(doc);
    });

    return Array.from(map.entries());
  }, [docs]);

  const formattedUpdated = useMemo(() => {
    if (!selectedDoc || !selectedDoc.lastModified) {
      return null;
    }

    try {
      return new Date(selectedDoc.lastModified).toLocaleString();
    } catch (error) {
      return selectedDoc.lastModified;
    }
  }, [selectedDoc]);

  const handleOverlayClick = () => {
    if (onClose) {
      onClose();
    }
  };

  const stopPropagation = (event) => {
    event.stopPropagation();
  };

  const handleDocClick = (doc) => {
    if (onSelectDoc) {
      onSelectDoc(doc);
    }
  };

  const docHtml = useMemo(() => {
    if (!content || contentError) {
      return '';
    }
    return renderMarkdown(content);
  }, [content, contentError]);

  const handleContentClick = useCallback((event) => {
    if (!selectedDoc || !Array.isArray(docs) || docs.length === 0) {
      return;
    }

    const targetNode = event.target;
    if (!targetNode) {
      return;
    }

    let element = null;
    if (typeof targetNode.closest === 'function') {
      element = targetNode.closest('a');
    } else if (targetNode.parentElement && typeof targetNode.parentElement.closest === 'function') {
      element = targetNode.parentElement.closest('a');
    }
    if (!element) {
      return;
    }

    const href = element.getAttribute('href') || '';
    if (!href || href.startsWith('#') || /^(https?:|mailto:|tel:)/i.test(href)) {
      return;
    }

    event.preventDefault();
    event.stopPropagation();

    let resolvedPath = href;
    try {
      const base = new URL(selectedDoc.relativePath, 'https://scenario.local/');
      resolvedPath = new URL(href, base).pathname;
    } catch (error) {
      // Ignore parse errors and use fallback resolution
    }

    resolvedPath = decodeURIComponent(resolvedPath)
      .replace(/^\/+/, '')
      .replace(/^\.?\//, '');

    const match = docs.find((doc) => {
      if (!doc?.relativePath) {
        return false;
      }
      const normalized = doc.relativePath.replace(/^\/+/, '');
      if (normalized === resolvedPath || `${normalized}.md` === resolvedPath) {
        return true;
      }
      if (normalized.endsWith('.md') && normalized.slice(0, -3) === resolvedPath) {
        return true;
      }
      return false;
    });

    if (match && onSelectDoc) {
      onSelectDoc(match);
    }
  }, [docs, onSelectDoc, selectedDoc]);

  if (!open) {
    return null;
  }

  const refreshHandler = onRefreshDocs || (() => {});
  const sizeLabel = selectedDoc?.size
    ? `${Math.max(1, Math.round(selectedDoc.size / 1024))} KB`
    : null;

  return (
    <div className="docs-overlay" onClick={handleOverlayClick}>
      <div className="docs-panel" onClick={stopPropagation}>
        <div className="docs-header">
          <div className="docs-header-title">
            <BookOpen className="docs-header-icon" />
            <div>
              <h2>Scenario Documentation</h2>
              <p>Browse MCP specifications and scenario guides.</p>
            </div>
          </div>
          <div className="docs-header-actions">
            <button
              className="docs-refresh"
              onClick={refreshHandler}
              disabled={loadingDocs}
              type="button"
            >
              <RefreshCw className={loadingDocs ? 'spinning' : ''} size={16} />
              Refresh
            </button>
            <button className="docs-close" onClick={onClose} type="button">
              <X size={18} />
            </button>
          </div>
        </div>
        <div className="docs-body">
          <aside className="docs-sidebar">
            {loadingDocs ? (
              <div className="docs-placeholder">Loading documentation...</div>
            ) : docsError ? (
              <div className="docs-error">{docsError}</div>
            ) : groupedDocs.length === 0 ? (
              <div className="docs-placeholder">No documentation available yet.</div>
            ) : (
              groupedDocs.map(([category, items]) => (
                <div className="docs-category" key={category}>
                  <div className="docs-category-title">{category}</div>
                  <ul className="docs-list">
                    {items.map(doc => (
                      <li key={doc.id}>
                        <button
                          type="button"
                          className={`docs-item ${selectedDocId === doc.id ? 'active' : ''}`}
                          onClick={() => handleDocClick(doc)}
                        >
                          <FileText size={14} />
                          <div className="docs-item-details">
                            <span className="docs-item-title">{doc.title}</span>
                            {doc.summary && (
                              <span className="docs-item-summary">{doc.summary}</span>
                            )}
                          </div>
                        </button>
                      </li>
                    ))}
                  </ul>
                </div>
              ))
            )}
          </aside>
          <section className="docs-content">
            {selectedDoc ? (
              <>
                <div className="docs-content-header">
                  <div>
                    <h3>{selectedDoc.title}</h3>
                    <div className="docs-meta">
                      {selectedDoc.relativePath && (
                        <span className="docs-meta-item">Path: {selectedDoc.relativePath}</span>
                      )}
                      {formattedUpdated && (
                        <span className="docs-meta-item">Updated: {formattedUpdated}</span>
                      )}
                      {sizeLabel && (
                        <span className="docs-meta-item">{sizeLabel}</span>
                      )}
                    </div>
                  </div>
                </div>
                <div className="docs-content-body" onClick={handleContentClick}>
                  {loadingContent ? (
                    <div className="docs-placeholder">Loading document...</div>
                  ) : contentError ? (
                    <div className="docs-error">{contentError}</div>
                  ) : (
                    <div
                      className="docs-markdown"
                      dangerouslySetInnerHTML={{ __html: docHtml }}
                    />
                  )}
                </div>
              </>
            ) : (
              <div className="docs-placeholder">Select a document to view its contents.</div>
            )}
          </section>
        </div>
      </div>
    </div>
  );
};

export default DocumentationPanel;
