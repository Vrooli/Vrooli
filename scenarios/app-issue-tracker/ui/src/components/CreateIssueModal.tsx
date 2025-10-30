import { useCallback, useRef, useState } from 'react';
import type {
  ChangeEvent,
  DragEvent,
  FormEvent,
  KeyboardEvent as ReactKeyboardEvent,
} from 'react';
import {
  CircleAlert,
  KanbanSquare,
  Loader2,
  Paperclip,
  UploadCloud,
  X,
} from 'lucide-react';
import type { Issue, IssueStatus, Priority } from '../types/issue';
import { Modal } from './Modal';
import { getFallbackStatuses } from '../utils/issues';
import { toTitleCase } from '../utils/string';
import { formatFileSize } from '../utils/files';
import type {
  CreateIssueInitialFields,
  LockedAttachmentDraft,
  CreateIssueInput,
  UpdateIssueInput,
} from '../types/issueCreation';

interface AttachmentDraft {
  id: string;
  file: File;
  name: string;
  size: number;
  contentType: string;
  content: string | null;
  encoding: 'base64';
  status: 'pending' | 'ready' | 'error';
  errorMessage?: string;
}

type SharedModalProps = {
  onClose: () => void;
  initialData?: CreateIssueInitialFields | null;
  lockedAttachments?: LockedAttachmentDraft[] | null;
  followUpInfo?: { id: string; title: string } | null;
  validStatuses?: IssueStatus[] | null;
};

type CreateModeProps = SharedModalProps & {
  mode?: 'create';
  onSubmit: (input: CreateIssueInput) => Promise<void>;
};

type EditModeProps = SharedModalProps & {
  mode: 'edit';
  onSubmit: (input: UpdateIssueInput) => Promise<void>;
  existingIssue: Issue;
};

export type CreateIssueModalProps = CreateModeProps | EditModeProps;

function buildInitialFieldsFromIssue(issue: Issue): CreateIssueInitialFields {
  return {
    title: issue.rawTitle ?? issue.title,
    description: issue.description ?? '',
    priority: issue.priority,
    status: issue.status,
    appId: issue.app,
    tags: issue.tags,
    reporterName: issue.reporterName,
    reporterEmail: issue.reporterEmail,
  };
}

function normalizeTagsInput(value: string): string[] {
  return value
    .split(',')
    .map((tag) => tag.trim())
    .filter(Boolean);
}

export function CreateIssueModal(props: CreateIssueModalProps) {
  const {
    onClose,
    lockedAttachments: lockedAttachmentsProp,
    followUpInfo,
    validStatuses,
  } = props;

  const isEditMode = props.mode === 'edit';
  const initialData =
    props.initialData ?? (isEditMode ? buildInitialFieldsFromIssue(props.existingIssue) : null);

  const lockedAttachments = lockedAttachmentsProp ?? [];
  const statusOptions = (validStatuses && validStatuses.length > 0
    ? validStatuses
    : getFallbackStatuses()) as IssueStatus[];
  const [title, setTitle] = useState(initialData?.title ?? '');
  const [description, setDescription] = useState(initialData?.description ?? '');
  const [priority, setPriority] = useState<Priority>(initialData?.priority ?? 'Medium');
  const [status, setStatus] = useState<IssueStatus>(initialData?.status ?? 'open');
  const [appId, setAppId] = useState(initialData?.appId ?? 'web-ui');
  const [tags, setTags] = useState(initialData?.tags?.join(', ') ?? '');
  const [reporterName, setReporterName] = useState(initialData?.reporterName ?? '');
  const [reporterEmail, setReporterEmail] = useState(initialData?.reporterEmail ?? '');
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [attachments, setAttachments] = useState<AttachmentDraft[]>([]);
  const [isDragActive, setIsDragActive] = useState(false);
  const fileInputRef = useRef<HTMLInputElement | null>(null);

  const handleFileSelection = useCallback(
    (fileSource: FileList | File[]) => {
      const fileArray = Array.from(fileSource instanceof FileList ? Array.from(fileSource) : fileSource);
      if (fileArray.length === 0) {
        return;
      }
      fileArray.forEach((file) => {
        const duplicate = attachments.some(
          (item) =>
            item.file.name === file.name &&
            item.file.size === file.size &&
            item.file.lastModified === file.lastModified,
        );
        if (duplicate) {
          return;
        }

        const id = `${file.name}-${file.size}-${file.lastModified}-${Math.random().toString(16).slice(2)}`;
        const draft: AttachmentDraft = {
          id,
          file,
          name: file.name,
          size: file.size,
          contentType: file.type || 'application/octet-stream',
          content: null,
          encoding: 'base64',
          status: 'pending',
        };

        setAttachments((prev) => [...prev, draft]);

        const reader = new FileReader();
        reader.onload = () => {
          const result = reader.result;
          if (typeof result !== 'string') {
            setAttachments((prev) =>
              prev.map((item) =>
                item.id === id
                  ? { ...item, status: 'error', errorMessage: 'Unsupported attachment format.' }
                  : item,
              ),
            );
            return;
          }

          const base64 = result.includes(',') ? result.split(',')[1] ?? '' : result;
          setAttachments((prev) =>
            prev.map((item) =>
              item.id === id
                ? { ...item, content: base64, status: 'ready', errorMessage: undefined }
                : item,
            ),
          );
        };
        reader.onerror = () => {
          const message = reader.error?.message ?? 'Failed to read file.';
          setAttachments((prev) =>
            prev.map((item) =>
              item.id === id ? { ...item, status: 'error', errorMessage: message } : item,
            ),
          );
        };
        reader.readAsDataURL(file);
      });
    },
    [attachments],
  );

  const handleFileInputChange = (event: ChangeEvent<HTMLInputElement>) => {
    if (event.target.files && event.target.files.length > 0) {
      handleFileSelection(event.target.files);
      event.target.value = '';
    }
  };

  const handleDropzoneClick = () => {
    fileInputRef.current?.click();
  };

  const handleDropzoneKeyDown = (event: ReactKeyboardEvent<HTMLDivElement>) => {
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault();
      fileInputRef.current?.click();
    }
  };

  const handleDragEnter = (event: DragEvent<HTMLDivElement>) => {
    event.preventDefault();
    setIsDragActive(true);
  };

  const handleDragOver = (event: DragEvent<HTMLDivElement>) => {
    event.preventDefault();
    event.dataTransfer.effectAllowed = 'copy';
    event.dataTransfer.dropEffect = 'copy';
    setIsDragActive(true);
  };

  const handleDragLeave = (event: DragEvent<HTMLDivElement>) => {
    const nextTarget = event.relatedTarget as Node | null;
    if (nextTarget && event.currentTarget.contains(nextTarget)) {
      return;
    }
    setIsDragActive(false);
  };

  const handleDrop = (event: DragEvent<HTMLDivElement>) => {
    event.preventDefault();
    setIsDragActive(false);
    const fileList = event.dataTransfer?.files;
    if (fileList && fileList.length > 0) {
      handleFileSelection(fileList);
    }
  };

  const handleRemoveAttachment = (id: string) => {
    setAttachments((prev) => prev.filter((attachment) => attachment.id !== id));
  };

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!title.trim()) {
      setErrorMessage('Title is required.');
      return;
    }

    if (attachments.some((attachment) => attachment.status === 'pending')) {
      setErrorMessage('Please wait for attachments to finish processing.');
      return;
    }

    const userPreparedAttachments = attachments
      .filter((attachment) => attachment.status === 'ready' && attachment.content)
      .map((attachment) => ({
        name: attachment.name,
        content: attachment.content as string,
        encoding: attachment.encoding,
        contentType: attachment.contentType,
        category: 'attachment',
      }));

    const preparedAttachments = [
      ...lockedAttachments.map((item) => item.payload),
      ...userPreparedAttachments,
    ];

    setSubmitting(true);
    setErrorMessage(null);

    try {
      if (props.mode === 'edit') {
        const issue = props.existingIssue;
        const baseline = buildInitialFieldsFromIssue(issue);
        const trimmedTitle = title.trim();
        const trimmedDescription = description.trim();
        const trimmedReporterName = reporterName.trim();
        const trimmedReporterEmail = reporterEmail.trim();
        const normalizedTags = normalizeTagsInput(tags);
        const baselineTags = baseline.tags ?? [];
        const baselineTagsSorted = [...baselineTags].sort();
        const nextTagsSorted = [...normalizedTags].sort();
        const tagsChanged =
          baselineTagsSorted.length !== nextTagsSorted.length ||
          baselineTagsSorted.some((value, index) => value !== nextTagsSorted[index]);

        const updatePayload: UpdateIssueInput = {
          issueId: issue.id,
        };

        if (trimmedTitle !== (baseline.title ?? '')) {
          updatePayload.title = trimmedTitle;
        }
        if (trimmedDescription !== (baseline.description ?? '')) {
          updatePayload.description = trimmedDescription;
        }
        if (priority !== baseline.priority) {
          updatePayload.priority = priority;
        }
        if (status !== baseline.status) {
          updatePayload.status = status;
        }
        if (appId.trim() !== (baseline.appId ?? '').trim()) {
          updatePayload.appId = appId.trim();
        }
        if (tagsChanged) {
          updatePayload.tags = normalizedTags;
        }
        if (trimmedReporterName !== (baseline.reporterName ?? '')) {
          updatePayload.reporterName = trimmedReporterName;
        }
        if (trimmedReporterEmail !== (baseline.reporterEmail ?? '')) {
          updatePayload.reporterEmail = trimmedReporterEmail;
        }
        if (preparedAttachments.length > 0) {
          updatePayload.attachments = preparedAttachments;
        }

        const payloadKeys = Object.keys(updatePayload).filter((key) => key !== 'issueId');
        const hasChanges = payloadKeys.length > 0;

        if (!hasChanges) {
          setErrorMessage('No changes detected. Update a field or add an attachment.');
          return;
        }

        await props.onSubmit(updatePayload);
      } else {
        await props.onSubmit({
          title,
          description,
          priority,
          status,
          appId,
          tags: normalizeTagsInput(tags),
          reporterName: reporterName.trim() || undefined,
          reporterEmail: reporterEmail.trim() || undefined,
          attachments: preparedAttachments,
        });
      }
      onClose();
    } catch (error) {
      const fallbackMessage = props.mode === 'edit' ? 'Failed to update issue.' : 'Failed to create issue.';
      setErrorMessage(error instanceof Error ? error.message : fallbackMessage);
    } finally {
      setSubmitting(false);
    }
  };

  const totalAttachmentCount = lockedAttachments.length + attachments.length;
  const attachmentCountLabel = totalAttachmentCount === 1 ? 'file' : 'files';
  const hasErroredAttachments = attachments.some((attachment) => attachment.status === 'error');
  const modalEyebrow = isEditMode
    ? `Issue ${props.existingIssue.id}`
    : followUpInfo
      ? 'Follow-up'
      : 'New Issue';
  const modalTitle = isEditMode
    ? 'Update Issue'
    : followUpInfo
      ? 'Create Follow-up Issue'
      : 'Create Issue';
  const submitLabel = isEditMode ? (submitting ? 'Updating…' : 'Update Issue') : submitting ? 'Creating…' : 'Create Issue';

  return (
    <Modal onClose={onClose} labelledBy="create-issue-title">
      <form className="modal-form" onSubmit={handleSubmit}>
        <div className="modal-header">
          <div>
            <p className="modal-eyebrow">{modalEyebrow}</p>
            <h2 id="create-issue-title" className="modal-title">
              {modalTitle}
            </h2>
          </div>
          <button className="modal-close" type="button" aria-label="Close create issue" onClick={onClose}>
            <X size={18} />
          </button>
        </div>

        <div className="modal-body">
          {followUpInfo && (
            <div className="follow-up-banner">
              <KanbanSquare size={16} />
              <span>
                Follow-up for {followUpInfo.id}
                {followUpInfo.title ? ` — ${followUpInfo.title}` : ''}
              </span>
            </div>
          )}
          <div className="form-grid">
            <label className="form-field">
              <span>Title</span>
              <input
                type="text"
                value={title}
                onChange={(event) => setTitle(event.target.value)}
                placeholder="Summarize the issue"
                required
              />
            </label>
            <label className="form-field">
              <span>Application</span>
              <input
                type="text"
                value={appId}
                onChange={(event) => setAppId(event.target.value)}
                placeholder="e.g. web-ui"
              />
            </label>
            <label className="form-field">
              <span>Priority</span>
              <select value={priority} onChange={(event) => setPriority(event.target.value as Priority)}>
                {(['Critical', 'High', 'Medium', 'Low'] as Priority[]).map((option) => (
                  <option key={option} value={option}>
                    {option}
                  </option>
                ))}
              </select>
            </label>
            <label className="form-field">
              <span>Status</span>
              <select value={status} onChange={(event) => setStatus(event.target.value as IssueStatus)}>
                {statusOptions.map((option) => (
                  <option key={option} value={option}>
                    {toTitleCase(option.replace(/-/g, ' '))}
                  </option>
                ))}
              </select>
            </label>
            <label className="form-field">
              <span>Reporter name</span>
              <input
                type="text"
                value={reporterName}
                onChange={(event) => setReporterName(event.target.value)}
                placeholder="Optional"
              />
            </label>
            <label className="form-field">
              <span>Reporter email</span>
              <input
                type="email"
                value={reporterEmail}
                onChange={(event) => setReporterEmail(event.target.value)}
                placeholder="Optional"
              />
            </label>
            <label className="form-field form-field-full">
              <span>Tags</span>
              <input
                type="text"
                value={tags}
                onChange={(event) => setTags(event.target.value)}
                placeholder="Comma separated (e.g. backend, regression)"
              />
            </label>
            <label className="form-field form-field-full">
              <span>Description</span>
              <textarea
                value={description}
                onChange={(event) => setDescription(event.target.value)}
                placeholder="Describe the issue, error messages, reproduction steps…"
                rows={5}
              />
            </label>
          </div>

          <div className="form-section attachments-section">
            <div className="form-section-header">
            <span className="form-section-title">Attachments</span>
            {totalAttachmentCount > 0 && (
              <span className="form-section-meta">
                {totalAttachmentCount} {attachmentCountLabel}
              </span>
            )}
          </div>
            {lockedAttachments.length > 0 && (
              <div className="locked-attachments-info">
                <p>
                  These files from {followUpInfo ? `issue ${followUpInfo.id}` : 'the previous issue'} will be attached automatically.
                </p>
                <ul className="locked-attachment-list">
                  {lockedAttachments.map((item) => {
                    const categoryLabel = item.category
                      ? toTitleCase(item.category.replace(/[-_]+/g, ' '))
                      : null;
                    const detailParts = [categoryLabel, item.description, item.sizeLabel].filter(Boolean);
                    return (
                      <li key={item.id} className="locked-attachment-item">
                        <span className="locked-attachment-icon">
                          <Paperclip size={14} />
                        </span>
                        <div className="locked-attachment-meta">
                          <span className="locked-attachment-name">{item.name}</span>
                          {detailParts.length > 0 && (
                            <span className="locked-attachment-details">{detailParts.join(' • ')}</span>
                          )}
                        </div>
                      </li>
                    );
                  })}
                </ul>
              </div>
            )}
            <p className="form-section-description">
              Drag and drop logs, screenshots, or other helpful context.
            </p>
            <div
              className={`attachment-dropzone${isDragActive ? ' is-active' : ''}`}
              onDragEnter={handleDragEnter}
              onDragOver={handleDragOver}
              onDragLeave={handleDragLeave}
              onDrop={handleDrop}
              onClick={handleDropzoneClick}
              role="button"
              tabIndex={0}
              onKeyDown={handleDropzoneKeyDown}
            >
              <UploadCloud size={20} />
              <div className="attachment-dropzone-copy">
                <strong>Drag & drop files</strong>
                <span>or click to upload</span>
              </div>
            </div>
            <input
              ref={fileInputRef}
              type="file"
              multiple
              onChange={handleFileInputChange}
              className="hidden-input"
            />

            {attachments.length > 0 && (
              <ul className="attachment-pending-list">
                {attachments.map((attachment) => {
                  const sizeLabel = formatFileSize(attachment.size);
                  return (
                    <li
                      key={attachment.id}
                      className={`attachment-pending-item attachment-pending-item--${attachment.status}`}
                    >
                      <div className="attachment-pending-details">
                        <span className="attachment-pending-icon">
                          <Paperclip size={14} />
                        </span>
                        <div className="attachment-pending-meta">
                          <span className="attachment-file-name">{attachment.name}</span>
                          <span className="attachment-file-subtext">{sizeLabel ?? 'Unknown size'}</span>
                          {attachment.status === 'error' && attachment.errorMessage && (
                            <span className="attachment-file-error">{attachment.errorMessage}</span>
                          )}
                        </div>
                      </div>
                      <div className="attachment-pending-actions">
                        {attachment.status === 'pending' && (
                          <span className="attachment-status-chip attachment-status-chip--pending">
                            <Loader2 size={14} className="attachment-spinner" />
                            Processing…
                          </span>
                        )}
                        {attachment.status === 'error' && (
                          <span className="attachment-status-chip attachment-status-chip--error">
                            <CircleAlert size={14} />
                            Failed
                          </span>
                        )}
                        <button
                          type="button"
                          className="link-button attachment-remove"
                          onClick={() => handleRemoveAttachment(attachment.id)}
                          aria-label={`Remove ${attachment.name}`}
                        >
                          Remove
                        </button>
                      </div>
                    </li>
                  );
                })}
              </ul>
            )}

            {hasErroredAttachments && (
              <p className="attachment-warning">
                Files marked as failed will be skipped. Remove them and try again.
              </p>
            )}
          </div>
        </div>

        <div className="modal-footer">
          {errorMessage && <span className="form-error">{errorMessage}</span>}
          <div className="modal-actions">
            <button className="ghost-button" type="button" onClick={onClose} disabled={submitting}>
              Cancel
            </button>
            <button className="primary-button" type="submit" disabled={submitting}>
              {submitLabel}
            </button>
          </div>
        </div>
      </form>
    </Modal>
  );
}
