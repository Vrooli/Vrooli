import type { ChangeEvent, ReactNode } from 'react';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  AlertCircle,
  Archive,
  Brain,
  CalendarClock,
  ChevronDown,
  ExternalLink,
  FileCode,
  FileDown,
  FileText,
  Hash,
  Image as ImageIcon,
  KanbanSquare,
  Loader2,
  Mail,
  Pencil,
  Paperclip,
  Tag,
  Trash2,
  User,
  X,
} from 'lucide-react';
import type { Issue, IssueAttachment, IssueStatus } from '../data/sampleData';
import { formatDistanceToNow as formatRelativeDistance } from '../utils/date';
import { formatFileSize } from '../utils/files';
import { toTitleCase } from '../utils/string';
import { getFallbackStatuses } from '../utils/issues';
import { Modal } from './Modal';

const MAX_ATTACHMENT_PREVIEW_CHARS = 8000;

interface AgentConversationEntryPayload {
  kind: string;
  id?: string;
  type?: string;
  role?: string;
  text?: string;
  data?: Record<string, unknown> | null;
  raw?: Record<string, unknown> | null;
}

interface AgentConversationPayloadResponse {
  issue_id: string;
  available: boolean;
  provider?: string;
  prompt?: string;
  metadata?: Record<string, unknown> | null;
  entries?: AgentConversationEntryPayload[];
  last_message?: string;
  transcript_timestamp?: string;
}

interface ApiConversationResponse {
  success: boolean;
  message?: string;
  data?: {
    conversation: AgentConversationPayloadResponse;
  };
}

export interface IssueDetailsModalProps {
  issue: Issue;
  apiBaseUrl: string;
  onClose: () => void;
  onStatusChange?: (issueId: string, newStatus: IssueStatus) => void | Promise<void>;
  onEdit?: (issue: Issue) => void;
  onArchive?: (issue: Issue) => void | Promise<void>;
  onDelete?: (issue: Issue) => void | Promise<void>;
  onFollowUp?: (issue: Issue) => void;
  followUpLoadingId?: string | null;
  validStatuses?: IssueStatus[];
}

export function IssueDetailsModal({
  issue,
  apiBaseUrl,
  onClose,
  onStatusChange,
  onEdit,
  onArchive,
  onDelete,
  onFollowUp,
  followUpLoadingId,
  validStatuses = getFallbackStatuses(),
}: IssueDetailsModalProps) {
  const createdText = formatDateTime(issue.createdAt);
  const updatedText = formatDateTime(issue.updatedAt);
  const resolvedText = formatDateTime(issue.resolvedAt);
  const createdHint = formatRelativeTime(issue.createdAt);
  const updatedHint = formatRelativeTime(issue.updatedAt);
  const resolvedHint = formatRelativeTime(issue.resolvedAt);
  const description = issue.description?.trim();
  const notes = issue.notes?.trim();

  const conversationUrl = useMemo(() => buildAgentConversationUrl(apiBaseUrl, issue.id), [apiBaseUrl, issue.id]);
  const [conversation, setConversation] = useState<AgentConversationPayloadResponse | null>(null);
  const [conversationLoading, setConversationLoading] = useState(false);
  const [conversationError, setConversationError] = useState<string | null>(null);
  const [conversationExpanded, setConversationExpanded] = useState(false);
  const [investigationExpanded, setInvestigationExpanded] = useState(true);
  const [archivePending, setArchivePending] = useState(false);
  const [deletePending, setDeletePending] = useState(false);
  const fetchAbortRef = useRef<AbortController | null>(null);

  const providerLabel = conversation?.provider ?? issue.investigation?.agent_id ?? null;

  const hasAgentTranscript = Boolean(
    issue.metadata?.extra?.agent_transcript_path || issue.metadata?.extra?.agent_last_message_path,
  );

  const followUpAvailable =
    (issue.status === 'completed' || issue.status === 'failed') && typeof onFollowUp === 'function';
  const followUpLoading = followUpLoadingId === issue.id;

  useEffect(() => {
    if (fetchAbortRef.current) {
      fetchAbortRef.current.abort();
      fetchAbortRef.current = null;
    }
    setConversation(null);
    setConversationError(null);
    setConversationExpanded(false);
    setConversationLoading(false);
    setInvestigationExpanded(true);
    setArchivePending(false);
    setDeletePending(false);
  }, [issue.id]);

  const fetchConversation = useCallback(async () => {
    if (fetchAbortRef.current) {
      fetchAbortRef.current.abort();
    }

    const controller = new AbortController();
    fetchAbortRef.current = controller;

    setConversationLoading(true);
    setConversationError(null);

    try {
      const response = await fetch(conversationUrl, { signal: controller.signal });
      if (!response.ok) {
        throw new Error(`Request failed with status ${response.status}`);
      }
      const payload: ApiConversationResponse = await response.json();
      if (!payload.success) {
        throw new Error(payload.message || 'Transcript request failed');
      }
      setConversation(payload.data?.conversation ?? null);
    } catch (error) {
      if (error instanceof DOMException && error.name === 'AbortError') {
        return;
      }
      console.error('[IssueTracker] Failed to fetch agent conversation', error);
      setConversationError(error instanceof Error ? error.message : 'Failed to load transcript');
    } finally {
      if (fetchAbortRef.current === controller) {
        fetchAbortRef.current = null;
      }
      setConversationLoading(false);
    }
  }, [conversationUrl]);

  useEffect(() => {
    return () => {
      if (fetchAbortRef.current) {
        fetchAbortRef.current.abort();
        fetchAbortRef.current = null;
      }
    };
  }, []);

  const handleToggleConversation = () => {
    const next = !conversationExpanded;
    setConversationExpanded(next);
    if (next && !conversation && !conversationLoading) {
      void fetchConversation();
    }
  };

  const handleToggleInvestigation = () => {
    setInvestigationExpanded((state) => !state);
  };

  const handleStatusChange = async (event: ChangeEvent<HTMLSelectElement>) => {
    const newStatus = event.target.value as IssueStatus;
    if (newStatus !== issue.status && onStatusChange) {
      await onStatusChange(issue.id, newStatus);
    }
  };

  const renderConversationContent = () => {
    if (conversationLoading) {
      return (
        <div className="agent-transcript-loading">
          <Loader2 size={16} className="agent-transcript-spinner" />
          <span>Loading transcript…</span>
        </div>
      );
    }

    if (!conversation) {
      return <div className="agent-transcript-empty">Transcript not loaded yet.</div>;
    }

    if (conversation.available === false) {
      return <div className="agent-transcript-empty">Transcript not available for this run.</div>;
    }

    return <AgentConversationPanel conversation={conversation} />;
  };

  const handleArchive = async () => {
    if (!onArchive || issue.status === 'archived' || archivePending) {
      return;
    }
    console.info('[IssueTracker] Modal archive tap', issue.id);
    let shouldArchive = true;
    const bypassConfirm = typeof window !== 'undefined' && typeof window.matchMedia === 'function' && window.matchMedia('(pointer: coarse)').matches;
    if (!bypassConfirm && typeof window !== 'undefined' && typeof window.confirm === 'function') {
      shouldArchive = window.confirm(
        `Archive ${issue.id}? The issue will move to the Archived column.`,
      );
    }
    if (!shouldArchive) {
      return;
    }

    try {
      setArchivePending(true);
      await onArchive(issue);
    } finally {
      setArchivePending(false);
    }
  };

  const handleDelete = async () => {
    if (!onDelete || deletePending) {
      return;
    }

    console.info('[IssueTracker] Modal delete tap', issue.id);
    let shouldDelete = true;
    const bypassConfirm = typeof window !== 'undefined' && typeof window.matchMedia === 'function' && window.matchMedia('(pointer: coarse)').matches;
    if (!bypassConfirm && typeof window !== 'undefined' && typeof window.confirm === 'function') {
      shouldDelete = window.confirm(
        `Delete ${issue.id}${issue.title ? ` — ${issue.title}` : ''}? This cannot be undone.`,
      );
    }
    if (!shouldDelete) {
      return;
    }

    try {
      setDeletePending(true);
      await onDelete(issue);
      onClose();
    } finally {
      setDeletePending(false);
    }
  };

  return (
    <Modal onClose={onClose} labelledBy="issue-details-title" panelClassName="modal-panel--issue-details">
      <div className="issue-details-container">
        <div className="issue-details-topbar">
          <button className="modal-close" type="button" aria-label="Close issue details" onClick={onClose}>
            <X size={18} />
          </button>
        </div>

        <div className="issue-details-scroll">
          <header className="issue-details-header">
            <p className="modal-eyebrow">
              <Hash size={14} />
              {issue.id}
            </p>
            <h2 id="issue-details-title" className="modal-title">
              {issue.title}
            </h2>
            {(typeof onArchive === 'function' || typeof onDelete === 'function' ||
              (issue.status === 'open' && typeof onEdit === 'function')) && (
              <div className="issue-details-actions">
                {typeof onArchive === 'function' && (
                  <button
                    type="button"
                    className="ghost-button"
                    onClick={handleArchive}
                    aria-label={issue.status === 'archived' ? 'Issue already archived' : 'Archive issue'}
                    disabled={archivePending || issue.status === 'archived'}
                  >
                    {archivePending ? (
                      <Loader2 size={16} className="button-spinner" />
                    ) : (
                      <Archive size={16} />
                    )}
                    <span>{issue.status === 'archived' ? 'Archived' : 'Archive'}</span>
                  </button>
                )}
                {typeof onDelete === 'function' && (
                  <button
                    type="button"
                    className="ghost-button ghost-button--danger"
                    onClick={handleDelete}
                    aria-label="Delete issue"
                    disabled={deletePending}
                  >
                    {deletePending ? (
                      <Loader2 size={16} className="button-spinner" />
                    ) : (
                      <Trash2 size={16} />
                    )}
                    <span>{deletePending ? 'Deleting…' : 'Delete'}</span>
                  </button>
                )}
                {issue.status === 'open' && typeof onEdit === 'function' && (
                  <button
                    type="button"
                    className="ghost-button"
                    onClick={() => onEdit(issue)}
                    aria-label="Edit issue"
                  >
                    <Pencil size={16} />
                    <span>Edit</span>
                  </button>
                )}
              </div>
            )}
          </header>

          <div className="issue-details-content">
            {onStatusChange && (
              <div className="form-field form-field-full">
                <label htmlFor="issue-status-selector">
                  <span>Status (Accessibility Feature)</span>
                  <div className="select-wrapper">
                    <select id="issue-status-selector" value={issue.status} onChange={handleStatusChange}>
                      {validStatuses.map((status) => (
                        <option key={status} value={status}>
                          {toTitleCase(status.replace(/-/g, ' '))}
                        </option>
                      ))}
                    </select>
                    <ChevronDown size={16} />
                  </div>
                </label>
              </div>
            )}

            <div className="issue-detail-grid">
              <IssueMetaTile label="Status" value={toTitleCase(issue.status.replace(/-/g, ' '))} />
              <IssueMetaTile label="Priority" value={issue.priority} />
              <IssueMetaTile label="Assignee" value={issue.assignee || 'Unassigned'} />
              <IssueMetaTile label="App" value={issue.app} />
              <IssueMetaTile
                label="Created"
                value={createdText}
                hint={createdHint}
                icon={<CalendarClock size={14} />}
              />
              {issue.updatedAt && (
                <IssueMetaTile
                  label="Updated"
                  value={updatedText}
                  hint={updatedHint}
                  icon={<CalendarClock size={14} />}
                />
              )}
              {issue.resolvedAt && (
                <IssueMetaTile
                  label="Resolved"
                  value={resolvedText}
                  hint={resolvedHint}
                  icon={<CalendarClock size={14} />}
                />
              )}
            </div>

            {description && (
              <section className="issue-detail-section">
                <h3>Description</h3>
                <MarkdownView content={description} />
              </section>
            )}

            {notes && (
              <section className="issue-detail-section">
                <h3>Notes</h3>
                <MarkdownView content={notes} />
              </section>
            )}

            {issue.attachments.length > 0 && (
              <section className="issue-detail-section">
                <h3>Attachments</h3>
                <div className="issue-attachments">
                  {issue.attachments.map((attachment) => (
                    <AttachmentPreview key={attachment.path} attachment={attachment} />
                  ))}
                </div>
              </section>
            )}

            {issue.tags.length > 0 && (
              <section className="issue-detail-section">
                <h3>Tags</h3>
                <div className="issue-detail-tags">
                  <Tag size={14} />
                  {issue.tags.map((tag) => (
                    <span key={tag} className="issue-detail-tag">
                      {tag}
                    </span>
                  ))}
                </div>
              </section>
            )}

            {issue.investigation?.report && (
              <section className="issue-detail-section">
                <div className="issue-section-heading">
                  <h3>Investigation Report</h3>
                  <button
                    type="button"
                    className="issue-section-toggle"
                    onClick={handleToggleInvestigation}
                    aria-expanded={investigationExpanded}
                  >
                    <ChevronDown
                      size={16}
                      className={`issue-section-toggle-icon${investigationExpanded ? ' is-open' : ''}`}
                    />
                    <span>{investigationExpanded ? 'Hide report' : 'Show report'}</span>
                  </button>
                </div>
                <div className={`investigation-report${investigationExpanded ? ' is-expanded' : ' is-collapsed'}`}>
                  {issue.investigation.agent_id && (
                    <div className="investigation-meta">
                      <Brain size={14} />
                      <span>Agent: {issue.investigation.agent_id}</span>
                    </div>
                  )}
                  {investigationExpanded ? (
                    <div className="investigation-report-body">
                      <MarkdownView content={issue.investigation.report} />
                    </div>
                  ) : (
                    <p className="investigation-report-placeholder">
                      Report hidden. Expand to review the findings.
                    </p>
                  )}
                </div>
              </section>
            )}

            {followUpAvailable && (
              <div className="issue-followup-actions">
                <button
                  type="button"
                  className="primary-button follow-up-button"
                  onClick={() => onFollowUp?.(issue)}
                  disabled={followUpLoading}
                  aria-label={followUpLoading ? 'Preparing follow-up issue' : 'Create follow-up issue'}
                >
                  {followUpLoading ? <Loader2 size={16} className="button-spinner" /> : <KanbanSquare size={16} />}
                  <span>{followUpLoading ? 'Preparing…' : 'Follow Up'}</span>
                </button>
              </div>
            )}

            {hasAgentTranscript && (
              <section className="issue-detail-section">
                <div className="issue-section-heading">
                  <h3>Agent Transcript</h3>
                  {providerLabel && <span className="issue-section-meta">Backend: {providerLabel}</span>}
                </div>
                {conversation?.last_message && (
                  <p className="agent-transcript-last-message">
                    <strong>Last message:</strong> {conversation.last_message}
                  </p>
                )}
                {conversation?.transcript_timestamp && (
                  <p className="agent-transcript-timestamp">
                    Captured: {formatDateTime(conversation.transcript_timestamp)}
                  </p>
                )}
                <div className="agent-transcript-actions">
                  <button type="button" className="agent-transcript-toggle" onClick={handleToggleConversation}>
                    {conversationExpanded ? 'Hide Transcript' : 'View Transcript'}
                    {conversationLoading && <Loader2 size={14} className="agent-transcript-spinner" />}
                  </button>
                  {!conversationExpanded && conversation && conversation.available === false && (
                    <span className="agent-transcript-hint">Transcript not captured for this run.</span>
                  )}
                </div>
                {conversationError && (
                  <div className="agent-transcript-error" role="alert">
                    {conversationError}
                  </div>
                )}
            {conversationExpanded && renderConversationContent()}
              </section>
            )}

            {issue.metadata?.extra?.agent_last_error && (
              <section className="issue-detail-section">
                <h3>Agent Execution Error</h3>
                <div className="issue-error-details">
                  <div className="issue-error-header">
                    <AlertCircle size={16} />
                    <span className="issue-error-status">
                      Status: {issue.metadata?.extra?.agent_last_status || 'failed'}
                    </span>
                    {issue.metadata?.extra?.agent_failure_time && (
                      <span className="issue-error-time">
                        Failed at: {formatDateTime(issue.metadata.extra.agent_failure_time)}
                      </span>
                    )}
                  </div>
                  <pre className="issue-error-content">{issue.metadata.extra.agent_last_error}</pre>
                </div>
              </section>
            )}

            {(issue.reporterName || issue.reporterEmail) && (
              <section className="issue-detail-section">
                <h3>Reporter</h3>
                <div className="issue-detail-inline">
                  {issue.reporterName && (
                    <span>
                      <User size={14} />
                      {issue.reporterName}
                    </span>
                  )}
                  {issue.reporterEmail && (
                    <a href={`mailto:${issue.reporterEmail}`}>
                      <Mail size={14} />
                      {issue.reporterEmail}
                    </a>
                  )}
                </div>
              </section>
            )}
          </div>
        </div>
      </div>
    </Modal>
  );
}

interface MarkdownViewProps {
  content: string;
}

function MarkdownView({ content }: MarkdownViewProps) {
  const rendered = useMemo(() => renderMarkdownToHtml(content), [content]);
  return <div className="markdown-body" dangerouslySetInnerHTML={{ __html: rendered }} />;
}

interface AgentConversationPanelProps {
  conversation: AgentConversationPayloadResponse;
}

function AgentConversationPanel({ conversation }: AgentConversationPanelProps) {
  const entries = Array.isArray(conversation.entries) ? conversation.entries : [];

  if (entries.length === 0 && !conversation.prompt) {
    return <div className="agent-transcript-empty">Transcript captured but no events were recorded.</div>;
  }

  return (
    <div className="agent-conversation" role="log" aria-live="polite">
      {conversation.prompt && (
        <details className="agent-conversation-prompt">
          <summary>Initial prompt</summary>
          <pre>{conversation.prompt}</pre>
        </details>
      )}
      <ol className="agent-conversation-list">
        {entries.map((entry, index) => {
          const key = `${entry.id ?? index}-${entry.type ?? entry.kind}-${index}`;
          const detailsData = entry.data && Object.keys(entry.data).length > 0 ? entry.data : null;
          const rawData = !detailsData && entry.raw && Object.keys(entry.raw).length > 0 ? entry.raw : null;

          return (
            <li key={key} className="agent-conversation-item">
              <div className="agent-conversation-entry">
                <div className="agent-conversation-entry-meta">
                  <span className={`agent-conversation-chip agent-conversation-chip--${entry.kind}`}>
                    {entry.kind}
                  </span>
                  {entry.type && <span className="agent-conversation-type">{entry.type}</span>}
                  {entry.role && <span className="agent-conversation-role">{entry.role}</span>}
                </div>
                {entry.text && (
                  <div className="agent-conversation-text">
                    <MarkdownView content={entry.text} />
                  </div>
                )}
                {detailsData && (
                  <details className="agent-conversation-data">
                    <summary>Details</summary>
                    <pre>{JSON.stringify(detailsData, null, 2)}</pre>
                  </details>
                )}
                {rawData && (
                  <details className="agent-conversation-data">
                    <summary>Raw event</summary>
                    <pre>{JSON.stringify(rawData, null, 2)}</pre>
                  </details>
                )}
              </div>
            </li>
          );
        })}
      </ol>
    </div>
  );
}

interface AttachmentPreviewProps {
  attachment: IssueAttachment;
}

function AttachmentPreview({ attachment }: AttachmentPreviewProps) {
  const kind = classifyAttachment(attachment);
  const canPreviewText = kind === 'text' || kind === 'json';
  const [expanded, setExpanded] = useState(kind === 'image');
  const [content, setContent] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const previewRequested = useRef(false);
  const isCollapsible = canPreviewText;

  useEffect(() => {
    setExpanded(kind === 'image');
    setContent(null);
    setErrorMessage(null);
    setLoading(false);
    previewRequested.current = false;
  }, [attachment.path, kind]);

  useEffect(() => {
    if (!canPreviewText || !expanded) {
      return;
    }
    if (content !== null || previewRequested.current) {
      return;
    }

    previewRequested.current = true;
    const controller = new AbortController();
    const { signal } = controller;

    setLoading(true);
    setErrorMessage(null);

    fetch(attachment.url, { signal })
      .then((response) => {
        if (!response.ok) {
          throw new Error(`Preview request failed with status ${response.status}`);
        }
        return response.text();
      })
      .then((data) => {
        if (signal.aborted) {
          return;
        }
        let previewText = data;
        if (kind === 'json') {
          try {
            previewText = JSON.stringify(JSON.parse(data), null, 2);
          } catch (parseError) {
            // Keep raw response if parsing fails
          }
        }
        if (previewText.length > MAX_ATTACHMENT_PREVIEW_CHARS) {
          previewText = `${previewText.slice(0, MAX_ATTACHMENT_PREVIEW_CHARS)}\n…`;
        }
        setContent(previewText);
      })
      .catch((error) => {
        if (signal.aborted) {
          return;
        }
        console.error('[IssueTracker] Failed to fetch attachment preview', error);
        setErrorMessage('Failed to load preview');
      })
      .finally(() => {
        if (signal.aborted) {
          return;
        }
        setLoading(false);
      });

    return () => {
      controller.abort();
    };
  }, [attachment.url, canPreviewText, content, expanded, kind]);

  const storageName = attachment.name || attachment.path.split(/[\\/]+/).pop() || 'attachment';
  const sizeLabel = formatFileSize(attachment.size);

  const toggle = () => {
    if (!isCollapsible) {
      return;
    }
    setExpanded((state) => !state);
  };

  return (
    <article className={`attachment-card attachment-card--${kind}`}>
      <header className="attachment-header">
        <div className="attachment-heading">
          <Paperclip size={14} />
          <div>
            <p className="attachment-title">{attachment.name || storageName}</p>
            <p className="attachment-subtitle">
              {kind === 'image' && <ImageIcon size={12} />}
              {kind === 'json' && <FileCode size={12} />}
              {kind === 'text' && <FileText size={12} />}
              <span>
                {sizeLabel ?? 'Unknown size'}
                {attachment.category && ` · ${toTitleCase(attachment.category.replace(/[-_]+/g, ' '))}`}
              </span>
            </p>
          </div>
        </div>
        {isCollapsible && (
          <button type="button" className="attachment-toggle" onClick={toggle} aria-expanded={expanded}>
            {expanded ? 'Hide preview' : 'Show preview'}
          </button>
        )}
      </header>

      {(expanded || !isCollapsible) && (
        <div className="attachment-preview">
          {kind === 'image' && (
            <img className="attachment-preview-image" src={attachment.url} alt={attachment.name || storageName} />
          )}

          {canPreviewText && expanded && (
            <>
              {loading && (
                <div className="attachment-preview-placeholder">
                  <Loader2 size={16} className="attachment-spinner" />
                  <span>Loading preview…</span>
                </div>
              )}
              {!loading && errorMessage && (
                <div className="attachment-preview-error">{errorMessage}</div>
              )}
              {!loading && !errorMessage && content !== null && (
                <pre className="attachment-preview-text">{content}</pre>
              )}
              {!loading && !errorMessage && content === null && (
                <div className="attachment-preview-placeholder">No preview available.</div>
              )}
            </>
          )}
        </div>
      )}

      <div className="attachment-actions">
        <a className="attachment-button" href={attachment.url} target="_blank" rel="noopener noreferrer">
          <ExternalLink size={14} />
          <span>Open</span>
        </a>
        <a className="attachment-button" href={attachment.url} download={storageName}>
          <FileDown size={14} />
          <span>Download</span>
        </a>
      </div>
    </article>
  );
}

function buildAgentConversationUrl(apiBaseUrl: string, issueId: string): string {
  const normalizedBase = apiBaseUrl.endsWith('/') ? apiBaseUrl.slice(0, -1) : apiBaseUrl;
  const safeIssueId = encodeURIComponent(issueId);
  return `${normalizedBase}/issues/${safeIssueId}/agent/conversation`;
}

function formatDateTime(value?: string | null): string {
  if (!value) {
    return '—';
  }
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return value;
  }
  return parsed.toLocaleString();
}

function formatRelativeTime(value?: string | null): string | undefined {
  if (!value) {
    return undefined;
  }
  return formatRelativeDistance(value);
}

function classifyAttachment(attachment: IssueAttachment): 'image' | 'text' | 'json' | 'other' {
  const mime = attachment.type?.toLowerCase() ?? '';
  if (mime.startsWith('image/')) {
    return 'image';
  }
  if (mime === 'application/json' || mime.endsWith('+json')) {
    return 'json';
  }
  if (mime.startsWith('text/')) {
    return 'text';
  }

  const lowerPath = attachment.path.toLowerCase();
  if (lowerPath.endsWith('.json')) {
    return 'json';
  }
  if (['.log', '.txt', '.md', '.markdown', '.yaml', '.yml', '.har'].some((ext) => lowerPath.endsWith(ext))) {
    return 'text';
  }
  return 'other';
}

function escapeHtml(value: string): string {
  return value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

function renderInlineMarkdown(source: string): string {
  const codeSegments: string[] = [];
  const codePlaceholderPrefix = '\x00CODESEG\x00';

  let output = escapeHtml(source);

  output = output.replace(/`([^`]+)`/g, (_match, code: string) => {
    const placeholder = `${codePlaceholderPrefix}${codeSegments.length}\x00`;
    codeSegments.push(`<code>${escapeHtml(code)}</code>`);
    return placeholder;
  });

  output = output.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>');
  output = output.replace(/__([^_]+)__/g, '<strong>$1</strong>');
  output = output.replace(/\*([^*]+)\*/g, '<em>$1</em>');
  output = output.replace(/_([^_]+)_/g, '<em>$1</em>');
  output = output.replace(/~~([^~]+)~~/g, '<del>$1</del>');
  output = output.replace(
    /\[([^\]]+)]\(((?:https?:\/\/)[^\s)]+)\)/g,
    '<a href="$2" target="_blank" rel="noopener noreferrer">$1</a>',
  );

  output = output.replace(/\x00CODESEG\x00(\d+)\x00/g, (_match, index: string) => {
    const idx = Number.parseInt(index, 10);
    return codeSegments[idx] ?? '';
  });

  return output;
}

function renderMarkdownToHtml(markdown: string): string {
  const normalized = markdown.replace(/\r\n/g, '\n');
  const lines = normalized.split('\n');
  const blocks: string[] = [];
  let listType: 'ul' | 'ol' | null = null;
  let inCodeBlock = false;
  let codeLanguage = '';
  let codeBuffer: string[] = [];
  let paragraphLines: string[] = [];
  let blockquoteLines: string[] = [];

  const flushParagraph = () => {
    if (paragraphLines.length === 0) {
      return;
    }
    const paragraph = paragraphLines.map((line) => renderInlineMarkdown(line)).join('<br />');
    blocks.push(`<p>${paragraph}</p>`);
    paragraphLines = [];
  };

  const flushList = () => {
    if (!listType) {
      return;
    }
    blocks.push(listType === 'ul' ? '</ul>' : '</ol>');
    listType = null;
  };

  const flushBlockquote = () => {
    if (blockquoteLines.length === 0) {
      return;
    }
    const quote = blockquoteLines.map((line) => renderInlineMarkdown(line)).join('<br />');
    blocks.push(`<blockquote>${quote}</blockquote>`);
    blockquoteLines = [];
  };

  const flushCodeBlock = () => {
    if (!inCodeBlock) {
      return;
    }
    const languageClass = codeLanguage ? ` class="language-${escapeHtml(codeLanguage)}"` : '';
    blocks.push(`<pre><code${languageClass}>${escapeHtml(codeBuffer.join('\n'))}</code></pre>`);
    codeBuffer = [];
    codeLanguage = '';
    inCodeBlock = false;
  };

  for (const line of lines) {
    const trimmed = line.trimEnd();

    if (inCodeBlock) {
      if (trimmed.startsWith('```')) {
        flushCodeBlock();
        continue;
      }
      codeBuffer.push(line);
      continue;
    }

    if (trimmed.startsWith('```')) {
      flushParagraph();
      flushList();
      flushBlockquote();
      inCodeBlock = true;
      codeLanguage = trimmed.slice(3).trim();
      continue;
    }

    if (trimmed === '') {
      flushParagraph();
      flushList();
      flushBlockquote();
      continue;
    }

    if (/^(?:-{3,}|\*{3,}|_{3,})$/.test(trimmed)) {
      flushParagraph();
      flushList();
      flushBlockquote();
      blocks.push('<hr />');
      continue;
    }

    const headingMatch = trimmed.match(/^(#{1,6})\s+(.*)$/);
    if (headingMatch) {
      flushParagraph();
      flushList();
      flushBlockquote();
      const level = headingMatch[1].length;
      const headingText = headingMatch[2];
      blocks.push(`<h${level}>${renderInlineMarkdown(headingText)}</h${level}>`);
      continue;
    }

    if (trimmed.startsWith('>')) {
      flushParagraph();
      flushList();
      blockquoteLines.push(trimmed.replace(/^>\s?/, ''));
      continue;
    }

    const orderedMatch = trimmed.match(/^(\d+)\.\s+(.*)$/);
    const unorderedMatch = trimmed.match(/^[-*+]\s+(.*)$/);
    if (orderedMatch) {
      flushParagraph();
      flushBlockquote();
      if (listType !== 'ol') {
        flushList();
        listType = 'ol';
        blocks.push('<ol>');
      }
      blocks.push(`<li>${renderInlineMarkdown(orderedMatch[2])}</li>`);
      continue;
    }
    if (unorderedMatch) {
      flushParagraph();
      flushBlockquote();
      if (listType !== 'ul') {
        flushList();
        listType = 'ul';
        blocks.push('<ul>');
      }
      blocks.push(`<li>${renderInlineMarkdown(unorderedMatch[1])}</li>`);
      continue;
    }

    flushList();
    flushBlockquote();
    paragraphLines.push(trimmed);
  }

  flushCodeBlock();
  flushParagraph();
  flushList();
  flushBlockquote();

  if (blocks.length === 0) {
    return `<p>${renderInlineMarkdown(normalized)}</p>`;
  }

  return blocks.join('');
}

interface IssueMetaTileProps {
  label: string;
  value: string;
  hint?: string;
  icon?: ReactNode;
}

function IssueMetaTile({ label, value, hint, icon }: IssueMetaTileProps) {
  return (
    <div className="issue-detail-tile">
      <span className="issue-detail-label">{label}</span>
      <span className="issue-detail-value">
        {icon && <span className="issue-detail-value-icon">{icon}</span>}
        {value}
      </span>
      {hint && <span className="issue-detail-hint">{hint}</span>}
    </div>
  );
}
