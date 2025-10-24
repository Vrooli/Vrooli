import type { ChangeEvent, ReactNode } from 'react';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  AlertCircle,
  AppWindow,
  Archive,
  Brain,
  CalendarClock,
  ChevronDown,
  ChevronRight,
  ExternalLink,
  FileCode,
  FileDown,
  FileText,
  Loader2,
  Hash,
  Image as ImageIcon,
  KanbanSquare,
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
  const statusLabel = toTitleCase(issue.status.replace(/-/g, ' '));
  const assigneeLabel = issue.assignee || 'Unassigned';
  const description = issue.description?.trim();
  const notes = issue.notes?.trim();
  const timelineEntries = useMemo<IssueTimelineEntry[]>(() => {
    const entries: Array<IssueTimelineEntry | null> = [
      issue.createdAt
        ? {
            label: 'Created',
            value: formatDateTime(issue.createdAt),
            hint: formatRelativeTime(issue.createdAt),
            timestamp: parseTimestamp(issue.createdAt),
          }
        : null,
      issue.updatedAt
        ? {
            label: 'Updated',
            value: formatDateTime(issue.updatedAt),
            hint: formatRelativeTime(issue.updatedAt),
            timestamp: parseTimestamp(issue.updatedAt),
          }
        : null,
      issue.resolvedAt
        ? {
            label: 'Resolved',
            value: formatDateTime(issue.resolvedAt),
            hint: formatRelativeTime(issue.resolvedAt),
            timestamp: parseTimestamp(issue.resolvedAt),
          }
        : null,
    ];

    return entries.filter(isTimelineEntry).sort((a, b) => (a.timestamp ?? 0) - (b.timestamp ?? 0));
  }, [issue.createdAt, issue.updatedAt, issue.resolvedAt]);

  const conversationUrl = useMemo(() => buildAgentConversationUrl(apiBaseUrl, issue.id), [apiBaseUrl, issue.id]);
  const [conversation, setConversation] = useState<AgentConversationPayloadResponse | null>(null);
  const [conversationLoading, setConversationLoading] = useState(false);
  const [conversationError, setConversationError] = useState<string | null>(null);
  const [conversationExpanded, setConversationExpanded] = useState(false);
  const [collapsedSections, setCollapsedSections] = useState<Record<string, boolean>>({});
  const [archivePending, setArchivePending] = useState(false);
  const [deletePending, setDeletePending] = useState(false);
  const [timelineOpen, setTimelineOpen] = useState(false);
  const fetchAbortRef = useRef<AbortController | null>(null);

  const providerLabel = conversation?.provider ?? issue.investigation?.agent_id ?? null;

  const hasAgentTranscript = Boolean(
    issue.metadata?.extra?.agent_transcript_path || issue.metadata?.extra?.agent_last_message_path,
  );

  const followUpAvailable =
    (issue.status === 'completed' || issue.status === 'failed') && typeof onFollowUp === 'function';
  const followUpLoading = followUpLoadingId === issue.id;
  const shouldShowTranscriptView = conversationExpanded && hasAgentTranscript;

  useEffect(() => {
    if (fetchAbortRef.current) {
      fetchAbortRef.current.abort();
      fetchAbortRef.current = null;
    }
    setConversation(null);
    setConversationError(null);
    setConversationExpanded(false);
    setConversationLoading(false);
    setCollapsedSections({});
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

  const isSectionCollapsed = (key: string) => Boolean(collapsedSections[key]);

  const handleToggleSection = (key: string) => {
    setCollapsedSections((prev) => ({
      ...prev,
      [key]: !prev[key],
    }));
  };

  const buildSectionIds = (key: string) => {
    const normalizedKey = key.toLowerCase().replace(/[^a-z0-9_-]+/g, '-');
    return {
      headingId: `issue-${issue.id}-${normalizedKey}-heading`,
      contentId: `issue-${issue.id}-${normalizedKey}-content`,
    };
  };

  const handleToggleConversation = () => {
    const next = !conversationExpanded;
    setConversationExpanded(next);
    if (next && !conversation && !conversationLoading) {
      void fetchConversation();
    }
  };

  const handleToggleInvestigation = () => {
    handleToggleSection('investigation');
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

  const renderTranscriptView = () => {
    const { headingId, contentId } = buildSectionIds('agent-transcript');

    if (!hasAgentTranscript) {
      return (
        <div className="issue-transcript-view" id={contentId}>
          <div className="agent-transcript-empty">Transcript not captured for this run.</div>
        </div>
      );
    }

    const header = (
      <div className="issue-section-heading issue-transcript-heading">
        <div className="issue-section-title">
          <h3 id={headingId}>Agent Transcript</h3>
          {providerLabel && <span className="issue-section-meta">Backend: {providerLabel}</span>}
        </div>
      </div>
    );

    if (conversationError) {
      return (
        <div className="issue-transcript-view" id={contentId} role="region" aria-labelledby={headingId}>
          {header}
          <div className="agent-transcript-error" role="alert">
            {conversationError}
          </div>
        </div>
      );
    }

    return (
      <div className="issue-transcript-view" id={contentId} role="region" aria-labelledby={headingId}>
        {header}
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
        {renderConversationContent()}
      </div>
    );
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
    <>
      <Modal onClose={onClose} labelledBy="issue-details-title" panelClassName="modal-panel--issue-details">
        <div className="issue-details-container">
        <div className="issue-details-scroll">
          <header className="issue-details-header">
            <div className="issue-header-bar">
              <p className="modal-eyebrow">
                <Hash size={14} />
                {issue.id}
              </p>
              <div className="issue-header-controls">
                {issue.status === 'open' && typeof onEdit === 'function' && (
                  <button
                    type="button"
                    className="icon-button icon-button--ghost"
                    onClick={() => onEdit(issue)}
                    aria-label="Edit issue"
                    title="Edit issue"
                  >
                    <Pencil size={18} />
                  </button>
                )}
                {typeof onArchive === 'function' && (
                  <button
                    type="button"
                    className="icon-button icon-button--ghost"
                    onClick={handleArchive}
                    aria-label={archivePending ? 'Archiving issue…' : issue.status === 'archived' ? 'Issue archived' : 'Archive issue'}
                    title={issue.status === 'archived' ? 'Issue archived' : 'Archive issue'}
                    disabled={archivePending || issue.status === 'archived'}
                  >
                    {archivePending ? <Loader2 size={18} className="button-spinner" /> : <Archive size={18} />}
                  </button>
                )}
                {typeof onDelete === 'function' && (
                  <button
                    type="button"
                    className="icon-button icon-button--danger-ghost"
                    onClick={handleDelete}
                    aria-label={deletePending ? 'Deleting issue…' : 'Delete issue'}
                    title="Delete issue"
                    disabled={deletePending}
                  >
                    {deletePending ? <Loader2 size={18} className="button-spinner" /> : <Trash2 size={18} />}
                  </button>
                )}
                <button
                  className="icon-button icon-button--ghost"
                  type="button"
                  aria-label="Close issue details"
                  onClick={onClose}
                  title="Close"
                >
                  <X size={18} />
                </button>
              </div>
            </div>
            <div className="issue-header-main">
              <div className="issue-header-title-row">
                <h2 id="issue-details-title" className="modal-title">
                  {issue.title}
                </h2>
              </div>
              <div className="issue-header-meta">
                <IssueHeaderBadge
                  label="Status"
                  value={statusLabel}
                  icon={<KanbanSquare size={14} />}
                  showLabel={false}
                />
                <IssueHeaderBadge
                  label="Priority"
                  value={issue.priority}
                  icon={<Tag size={14} />}
                  showLabel={false}
                />
                <IssueHeaderBadge
                  label="Assignee"
                  value={assigneeLabel}
                  icon={<User size={14} />}
                  showLabel={false}
                />
                <IssueHeaderBadge
                  label="App"
                  value={issue.app}
                  icon={<AppWindow size={14} />}
                  showLabel={false}
                />
              </div>
            </div>
          </header>

          <div className={`issue-details-content${shouldShowTranscriptView ? ' is-transcript' : ''}`}>
            {shouldShowTranscriptView ? (
              renderTranscriptView()
            ) : (
              <>
            {onStatusChange && (
              <div className="form-field form-field-full">
                <label htmlFor="issue-status-selector">
                  <span>Status</span>
                  <div className="select-wrapper select-wrapper--with-icon">
                    <span className="select-leading-icon" aria-hidden="true">
                      <KanbanSquare size={16} />
                    </span>
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

            {timelineEntries.length > 0 && (
              <TimelineSummary entries={timelineEntries} onOpen={() => setTimelineOpen(true)} />
            )}

            {description && (() => {
              const { headingId, contentId } = buildSectionIds('description');
              const collapsed = isSectionCollapsed('description');

              return (
                <section
                  className={`issue-detail-section${collapsed ? ' is-collapsed' : ''}`}
                  aria-labelledby={headingId}
                >
                  <div className="issue-section-heading">
                    <div className="issue-section-title">
                      <h3 id={headingId}>Description</h3>
                    </div>
                    <button
                      type="button"
                      className="issue-section-toggle"
                      onClick={() => handleToggleSection('description')}
                      aria-expanded={!collapsed}
                      aria-controls={contentId}
                      aria-label={collapsed ? 'Expand description' : 'Collapse description'}
                      title={collapsed ? 'Expand description' : 'Collapse description'}
                    >
                      <ChevronDown
                        size={16}
                        className={`issue-section-toggle-icon${!collapsed ? ' is-open' : ''}`}
                      />
                    </button>
                  </div>
                  <div id={contentId} className="issue-section-content" hidden={collapsed}>
                    <MarkdownView content={description} />
                  </div>
                </section>
              );
            })()}

            {notes && (() => {
              const { headingId, contentId } = buildSectionIds('notes');
              const collapsed = isSectionCollapsed('notes');

              return (
                <section
                  className={`issue-detail-section${collapsed ? ' is-collapsed' : ''}`}
                  aria-labelledby={headingId}
                >
                  <div className="issue-section-heading">
                    <div className="issue-section-title">
                      <h3 id={headingId}>Notes</h3>
                    </div>
                    <button
                      type="button"
                      className="issue-section-toggle"
                      onClick={() => handleToggleSection('notes')}
                      aria-expanded={!collapsed}
                      aria-controls={contentId}
                      aria-label={collapsed ? 'Expand notes' : 'Collapse notes'}
                      title={collapsed ? 'Expand notes' : 'Collapse notes'}
                    >
                      <ChevronDown
                        size={16}
                        className={`issue-section-toggle-icon${!collapsed ? ' is-open' : ''}`}
                      />
                    </button>
                  </div>
                  <div id={contentId} className="issue-section-content" hidden={collapsed}>
                    <MarkdownView content={notes} />
                  </div>
                </section>
              );
            })()}

            {issue.attachments.length > 0 && (() => {
              const { headingId, contentId } = buildSectionIds('attachments');
              const collapsed = isSectionCollapsed('attachments');

              return (
                <section
                  className={`issue-detail-section${collapsed ? ' is-collapsed' : ''}`}
                  aria-labelledby={headingId}
                >
                  <div className="issue-section-heading">
                    <div className="issue-section-title">
                      <h3 id={headingId}>Attachments</h3>
                    </div>
                    <button
                      type="button"
                      className="issue-section-toggle"
                      onClick={() => handleToggleSection('attachments')}
                      aria-expanded={!collapsed}
                      aria-controls={contentId}
                      aria-label={collapsed ? 'Expand attachments' : 'Collapse attachments'}
                      title={collapsed ? 'Expand attachments' : 'Collapse attachments'}
                    >
                      <ChevronDown
                        size={16}
                        className={`issue-section-toggle-icon${!collapsed ? ' is-open' : ''}`}
                      />
                    </button>
                  </div>
                  <div id={contentId} className="issue-section-content" hidden={collapsed}>
                    <div className="issue-attachments-grid">
                      {issue.attachments.map((attachment) => (
                        <AttachmentPreview key={attachment.path} attachment={attachment} />
                      ))}
                    </div>
                  </div>
                </section>
              );
            })()}

            {issue.tags.length > 0 && (() => {
              const { headingId, contentId } = buildSectionIds('tags');
              const collapsed = isSectionCollapsed('tags');

              return (
                <section
                  className={`issue-detail-section${collapsed ? ' is-collapsed' : ''}`}
                  aria-labelledby={headingId}
                >
                  <div className="issue-section-heading">
                    <div className="issue-section-title">
                      <h3 id={headingId}>Tags</h3>
                    </div>
                    <button
                      type="button"
                      className="issue-section-toggle"
                      onClick={() => handleToggleSection('tags')}
                      aria-expanded={!collapsed}
                      aria-controls={contentId}
                      aria-label={collapsed ? 'Expand tags' : 'Collapse tags'}
                      title={collapsed ? 'Expand tags' : 'Collapse tags'}
                    >
                      <ChevronDown
                        size={16}
                        className={`issue-section-toggle-icon${!collapsed ? ' is-open' : ''}`}
                      />
                    </button>
                  </div>
                  <div id={contentId} className="issue-section-content" hidden={collapsed}>
                    <div className="issue-detail-tags">
                      <Tag size={14} />
                      {issue.tags.map((tag) => (
                        <span key={tag} className="issue-detail-tag">
                          {tag}
                        </span>
                      ))}
                    </div>
                  </div>
                </section>
              );
            })()}

                {issue.investigation?.report && (() => {
                  const { headingId, contentId } = buildSectionIds('investigation');
                  const collapsed = isSectionCollapsed('investigation');

                  return (
                    <section
                      className={`issue-detail-section${collapsed ? ' is-collapsed' : ''}`}
                      aria-labelledby={headingId}
                    >
                      <div className="issue-section-heading">
                        <div className="issue-section-title">
                          <h3 id={headingId}>Investigation Report</h3>
                        </div>
                        <button
                          type="button"
                          className="issue-section-toggle"
                          onClick={handleToggleInvestigation}
                          aria-expanded={!collapsed}
                          aria-controls={contentId}
                          aria-label={collapsed ? 'Expand investigation report' : 'Collapse investigation report'}
                          title={collapsed ? 'Expand investigation report' : 'Collapse investigation report'}
                        >
                          <ChevronDown
                            size={16}
                            className={`issue-section-toggle-icon${!collapsed ? ' is-open' : ''}`}
                          />
                        </button>
                      </div>
                      <div id={contentId} className="issue-section-content" hidden={collapsed}>
                        <div className="investigation-report">
                          {issue.investigation.agent_id && (
                            <div className="investigation-meta">
                              <Brain size={14} />
                              <span>Agent: {issue.investigation.agent_id}</span>
                            </div>
                          )}
                          <div className="investigation-report-body">
                            <MarkdownView content={issue.investigation.report} />
                          </div>
                        </div>
                      </div>
                    </section>
                  );
                })()}

                {issue.metadata?.extra?.agent_last_error && (() => {
                  const { headingId, contentId } = buildSectionIds('agent-error');
                  const collapsed = isSectionCollapsed('agent-error');

                  return (
                    <section
                      className={`issue-detail-section${collapsed ? ' is-collapsed' : ''}`}
                      aria-labelledby={headingId}
                    >
                      <div className="issue-section-heading">
                        <div className="issue-section-title">
                          <h3 id={headingId}>Agent Execution Error</h3>
                        </div>
                        <button
                          type="button"
                          className="issue-section-toggle"
                          onClick={() => handleToggleSection('agent-error')}
                          aria-expanded={!collapsed}
                          aria-controls={contentId}
                          aria-label={collapsed ? 'Expand agent error details' : 'Collapse agent error details'}
                          title={collapsed ? 'Expand agent error details' : 'Collapse agent error details'}
                        >
                          <ChevronDown
                            size={16}
                            className={`issue-section-toggle-icon${!collapsed ? ' is-open' : ''}`}
                          />
                        </button>
                      </div>
                      <div id={contentId} className="issue-section-content" hidden={collapsed}>
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
                      </div>
                    </section>
                  );
                })()}

                {(issue.reporterName || issue.reporterEmail) && (() => {
                  const { headingId, contentId } = buildSectionIds('reporter');
                  const collapsed = isSectionCollapsed('reporter');

                  return (
                    <section
                      className={`issue-detail-section${collapsed ? ' is-collapsed' : ''}`}
                      aria-labelledby={headingId}
                    >
                      <div className="issue-section-heading">
                        <div className="issue-section-title">
                          <h3 id={headingId}>Reporter</h3>
                        </div>
                        <button
                          type="button"
                          className="issue-section-toggle"
                          onClick={() => handleToggleSection('reporter')}
                          aria-expanded={!collapsed}
                          aria-controls={contentId}
                          aria-label={collapsed ? 'Expand reporter details' : 'Collapse reporter details'}
                          title={collapsed ? 'Expand reporter details' : 'Collapse reporter details'}
                        >
                          <ChevronDown
                            size={16}
                            className={`issue-section-toggle-icon${!collapsed ? ' is-open' : ''}`}
                          />
                        </button>
                      </div>
                      <div id={contentId} className="issue-section-content" hidden={collapsed}>
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
                      </div>
                    </section>
                  );
                })()}
              </>
            )}
          </div>
        </div>
        {(followUpAvailable || hasAgentTranscript) && (
          <footer className="issue-details-footer">
            <div className="issue-footer-actions">
              {hasAgentTranscript && (() => {
                const { contentId } = buildSectionIds('agent-transcript');
                return (
                  <button
                    type="button"
                    className="agent-transcript-toggle"
                    onClick={handleToggleConversation}
                    aria-label={conversationExpanded ? 'Hide transcript' : 'View transcript'}
                    aria-controls={contentId}
                    aria-expanded={conversationExpanded}
                  >
                    {conversationExpanded ? 'Hide Transcript' : 'View Transcript'}
                    {conversationLoading && <Loader2 size={14} className="agent-transcript-spinner" />}
                  </button>
                );
              })()}
              {followUpAvailable && (
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
              )}
            </div>
          </footer>
        )}
        </div>
      </Modal>
      {timelineOpen && (
        <TimelineDialog entries={timelineEntries} onClose={() => setTimelineOpen(false)} />
      )}
    </>
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
  const storageName = attachment.name?.trim() || attachment.path.split(/[\\/]+/).pop() || 'attachment';
  const displayName = attachment.name?.trim() || storageName;
  const dotIndex = displayName.lastIndexOf('.');
  const baseName = dotIndex > 0 ? displayName.slice(0, dotIndex) : displayName;
  const extension = dotIndex > 0 ? displayName.slice(dotIndex) : '';
  const sizeLabel = formatFileSize(attachment.size);
  const categoryLabel = attachment.category
    ? toTitleCase(attachment.category.replace(/[-_]+/g, ' '))
    : null;
  const detailItems = [
    categoryLabel
      ? { key: 'category', text: categoryLabel, className: 'attachment-card-detail attachment-card-detail--category' }
      : null,
    sizeLabel ? { key: 'size', text: sizeLabel, className: 'attachment-card-detail' } : null,
  ].filter(Boolean) as Array<{ key: string; text: string; className: string }>;
  const isImage = kind === 'image';

  const renderIcon = () => {
    if (kind === 'image') {
      return <ImageIcon size={16} />;
    }
    if (kind === 'json') {
      return <FileCode size={16} />;
    }
    if (kind === 'text') {
      return <FileText size={16} />;
    }
    return <Paperclip size={16} />;
  };

  return (
    <article className={`attachment-card attachment-card--${kind}`}>
      <header className="attachment-card-header">
        <div className="attachment-card-heading">
          <span className={`attachment-icon attachment-icon--${kind}`} aria-hidden="true">
            {renderIcon()}
          </span>
          <div className="attachment-card-meta">
            <p className="attachment-card-name" title={displayName} aria-label={displayName}>
              <span className="attachment-card-name-base">{baseName}</span>
              {extension && <span className="attachment-card-name-ext">{extension}</span>}
            </p>
            <p className="attachment-card-details">
              {detailItems.map((item) => (
                <span key={item.key} className={item.className}>
                  {item.text}
                </span>
              ))}
            </p>
          </div>
        </div>
      </header>

      {isImage && (
        <div className="attachment-preview">
          <img
            className="attachment-preview-image"
            src={attachment.url}
            alt={displayName}
            loading="lazy"
            decoding="async"
          />
        </div>
      )}

      <div className="attachment-actions">
        <a
          className="attachment-button"
          href={attachment.url}
          target="_blank"
          rel="noopener noreferrer"
          aria-label={`Open ${displayName} in a new tab`}
        >
          <ExternalLink size={14} />
        </a>
        <a
          className="attachment-button"
          href={attachment.url}
          download={storageName}
          aria-label={`Download ${displayName}`}
        >
          <FileDown size={14} />
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

interface IssueHeaderBadgeProps {
  label: string;
  value: string;
  icon?: ReactNode;
  showLabel?: boolean;
}

function IssueHeaderBadge({ label, value, icon, showLabel = true }: IssueHeaderBadgeProps) {
  const accessibleLabel = `${label}: ${value}`;

  return (
    <span
      className="issue-header-badge"
      {...(!showLabel ? { 'aria-label': accessibleLabel, title: accessibleLabel } : {})}
    >
      {icon && (
        <span className="issue-header-badge-icon" aria-hidden="true">
          {icon}
        </span>
      )}
      {showLabel && <span className="issue-header-badge-label">{label}</span>}
      <span className="issue-header-badge-value">{value}</span>
    </span>
  );
}

interface IssueTimelineEntry {
  label: string;
  value: string;
  hint?: string;
  timestamp?: number;
}

function isTimelineEntry(entry: IssueTimelineEntry | null): entry is IssueTimelineEntry {
  return Boolean(entry && entry.value && entry.value !== '—');
}

function parseTimestamp(value?: string | null): number | undefined {
  if (!value) {
    return undefined;
  }
  const parsed = Date.parse(value);
  return Number.isNaN(parsed) ? undefined : parsed;
}

function TimelineSummary({ entries, onOpen }: { entries: IssueTimelineEntry[]; onOpen: () => void }) {
  if (!entries.length) {
    return null;
  }

  const latest = entries[entries.length - 1];
  const summaryText = latest.hint ?? latest.value;
  const accessibleLabel = `Open timeline. Latest event: ${latest.label}, ${summaryText}.`;

  return (
    <button
      type="button"
      className="timeline-summary"
      onClick={onOpen}
      aria-label={accessibleLabel}
      title={latest.value}
    >
      <span className="timeline-summary-icon" aria-hidden="true">
        <CalendarClock size={16} />
      </span>
      <span className="timeline-summary-content">
        <span className="timeline-summary-label">{latest.label}</span>
        <span className="timeline-summary-subtext">{summaryText}</span>
      </span>
      <span className="timeline-summary-chevron" aria-hidden="true">
        <ChevronRight size={16} />
      </span>
    </button>
  );
}

interface TimelineDialogProps {
  entries: IssueTimelineEntry[];
  onClose: () => void;
}

function TimelineDialog({ entries, onClose }: TimelineDialogProps) {
  const closeButtonRef = useRef<HTMLButtonElement | null>(null);

  useEffect(() => {
    closeButtonRef.current?.focus();
  }, []);

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.stopPropagation();
        event.preventDefault();
        onClose();
      }
    };

    const options: AddEventListenerOptions = { capture: true };
    window.addEventListener('keydown', handleKeyDown, options);
    return () => {
      window.removeEventListener('keydown', handleKeyDown, options);
    };
  }, [onClose]);

  if (!entries.length) {
    return null;
  }

  const sorted = [...entries].sort((a, b) => (b.timestamp ?? 0) - (a.timestamp ?? 0));

  return (
    <div className="timeline-dialog-overlay" role="presentation" onClick={onClose}>
      <div
        className="timeline-dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby="timeline-dialog-title"
        onClick={(event) => event.stopPropagation()}
      >
        <div className="timeline-dialog-header">
          <div className="timeline-dialog-title" id="timeline-dialog-title">
            <CalendarClock size={16} />
            <span>Timeline</span>
          </div>
          <button
            type="button"
            className="icon-button icon-button--ghost"
            onClick={onClose}
            aria-label="Close timeline"
            ref={closeButtonRef}
          >
            <X size={16} />
          </button>
        </div>
        <ul className="timeline-dialog-list">
          {sorted.map((entry) => (
            <li key={`${entry.label}-${entry.value}`} className="timeline-dialog-item">
              <div className="timeline-dialog-item-header">
                <span className="timeline-dialog-item-label">{entry.label}</span>
                {entry.hint && <span className="timeline-dialog-item-hint">{entry.hint}</span>}
              </div>
              <span className="timeline-dialog-item-value">{entry.value}</span>
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
}
