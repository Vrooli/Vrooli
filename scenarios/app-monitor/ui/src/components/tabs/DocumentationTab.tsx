import { useState, useCallback } from 'react';
import type { AppDocumentsList, AppDocument } from '@/types';
import { FileText, Folder, Loader2, ChevronRight, X } from 'lucide-react';
import { appService } from '@/services/api';
import './DocumentationTab.css';

interface DocumentationTabProps {
  appId: string;
  documents: AppDocumentsList | null | undefined;
  loading?: boolean;
}

export default function DocumentationTab({ appId, documents, loading }: DocumentationTabProps) {
  const [selectedDoc, setSelectedDoc] = useState<AppDocument | null>(null);
  const [loadingDoc, setLoadingDoc] = useState(false);
  const [docError, setDocError] = useState<string | null>(null);

  const handleDocumentClick = useCallback(async (docPath: string) => {
    setLoadingDoc(true);
    setDocError(null);

    try {
      const doc = await appService.getAppDocument(appId, docPath, true);
      if (doc) {
        setSelectedDoc(doc);
      } else {
        setDocError('Failed to load document');
      }
    } catch (error) {
      setDocError(error instanceof Error ? error.message : 'Unknown error');
    } finally {
      setLoadingDoc(false);
    }
  }, [appId]);

  const handleCloseViewer = useCallback(() => {
    setSelectedDoc(null);
    setDocError(null);
  }, []);

  if (loading) {
    return (
      <div className="documentation-tab">
        <div className="documentation-tab__loading">
          <Loader2 size={32} className="documentation-tab__loading-icon" />
          <p>Loading documentation...</p>
        </div>
      </div>
    );
  }

  if (!documents || documents.total === 0) {
    return (
      <div className="documentation-tab">
        <div className="documentation-tab__empty">
          <FileText size={32} />
          <p>No documentation available</p>
          <p className="documentation-tab__empty-hint">
            Add README.md, PRD.md, or create a docs/ folder
          </p>
        </div>
      </div>
    );
  }

  const hasRootDocs = documents.root_docs.length > 0;
  const hasDocsDocs = documents.docs_docs.length > 0;

  // Document viewer overlay
  if (selectedDoc) {
    return (
      <div className="documentation-tab documentation-tab--viewer">
        <div className="doc-viewer">
          <div className="doc-viewer__header">
            <div className="doc-viewer__title">
              <FileText size={18} />
              <span>{selectedDoc.name}</span>
            </div>
            <button
              type="button"
              className="doc-viewer__close"
              onClick={handleCloseViewer}
              aria-label="Close document viewer"
            >
              <X size={20} />
            </button>
          </div>
          <div className="doc-viewer__content">
            {selectedDoc.is_markdown && selectedDoc.rendered_html ? (
              <div
                className="doc-viewer__markdown"
                dangerouslySetInnerHTML={{ __html: selectedDoc.rendered_html }}
              />
            ) : (
              <pre className="doc-viewer__raw">{selectedDoc.content}</pre>
            )}
          </div>
        </div>
      </div>
    );
  }

  // Loading state for document
  if (loadingDoc) {
    return (
      <div className="documentation-tab">
        <div className="documentation-tab__loading">
          <Loader2 size={32} className="documentation-tab__loading-icon" />
          <p>Loading document...</p>
        </div>
      </div>
    );
  }

  // Error state
  if (docError) {
    return (
      <div className="documentation-tab">
        <div className="documentation-tab__error">
          <p>Error loading document: {docError}</p>
          <button
            type="button"
            className="documentation-tab__error-back"
            onClick={handleCloseViewer}
          >
            Back to list
          </button>
        </div>
      </div>
    );
  }

  // Document list view
  return (
    <div className="documentation-tab">
      <div className="documentation-header">
        <div className="documentation-header__title">
          <FileText size={20} />
          <span>Available Documentation</span>
        </div>
        <div className="documentation-header__count">
          {documents.total} document{documents.total !== 1 ? 's' : ''}
        </div>
      </div>

      {/* Root Documents */}
      {hasRootDocs && (
        <section className="documentation-section">
          <h3 className="documentation-section__title">
            <Folder size={16} />
            <span>Scenario Root ({documents.root_docs.length})</span>
          </h3>
          <div className="documentation-list">
            {documents.root_docs.map((doc) => (
              <button
                key={doc.path}
                type="button"
                className="documentation-item"
                onClick={() => handleDocumentClick(doc.path)}
              >
                <div className="documentation-item__icon">
                  <FileText size={18} />
                </div>
                <div className="documentation-item__details">
                  <span className="documentation-item__name">{doc.name}</span>
                  <span className="documentation-item__meta">
                    {formatFileSize(doc.size)}
                    {doc.is_markdown && <span className="documentation-item__badge">Markdown</span>}
                  </span>
                </div>
                <ChevronRight size={16} className="documentation-item__arrow" />
              </button>
            ))}
          </div>
        </section>
      )}

      {/* Docs Folder */}
      {hasDocsDocs && (
        <section className="documentation-section">
          <h3 className="documentation-section__title">
            <Folder size={16} />
            <span>docs/ Folder ({documents.docs_docs.length})</span>
          </h3>
          <div className="documentation-list">
            {documents.docs_docs.map((doc) => (
              <button
                key={doc.path}
                type="button"
                className="documentation-item"
                onClick={() => handleDocumentClick(doc.path)}
              >
                <div className="documentation-item__icon">
                  <FileText size={18} />
                </div>
                <div className="documentation-item__details">
                  <span className="documentation-item__name">{doc.name}</span>
                  <span className="documentation-item__path">{doc.path}</span>
                  <span className="documentation-item__meta">
                    {formatFileSize(doc.size)}
                    {doc.is_markdown && <span className="documentation-item__badge">Markdown</span>}
                  </span>
                </div>
                <ChevronRight size={16} className="documentation-item__arrow" />
              </button>
            ))}
          </div>
        </section>
      )}
    </div>
  );
}

// Helper function to format file sizes
function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}
