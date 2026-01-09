import type { IssueStatus, Priority, Target } from './issue';

export interface CreateIssueAttachmentPayload {
  name: string;
  content: string;
  encoding: 'base64' | 'plain';
  contentType: string;
  category?: string;
}

export interface LockedAttachmentDraft {
  id: string;
  payload: CreateIssueAttachmentPayload;
  name: string;
  description?: string;
  sizeLabel?: string | null;
  category?: string;
}

export interface CreateIssueInitialFields {
  title?: string;
  description?: string;
  priority?: Priority;
  status?: IssueStatus;
  targets?: Target[];
  tags?: string[];
  reporterName?: string;
  reporterEmail?: string;
}

export interface CreateIssuePrefill {
  key: string;
  initial: CreateIssueInitialFields;
  lockedAttachments: LockedAttachmentDraft[];
  followUpOf?: { id: string; title: string };
}

export interface CreateIssueInput {
  title: string;
  description: string;
  priority: Priority;
  status: IssueStatus;
  targets: Target[];
  tags: string[];
  reporterName?: string;
  reporterEmail?: string;
  attachments: CreateIssueAttachmentPayload[];
}

export interface UpdateIssueInput {
  issueId: string;
  title?: string;
  description?: string;
  priority?: Priority;
  status?: IssueStatus;
  targets?: Target[];
  tags?: string[];
  reporterName?: string;
  reporterEmail?: string;
  attachments?: CreateIssueAttachmentPayload[];
  manual_review?: {
    marked_as_failed?: boolean;
    failure_reason?: string;
    reviewed_by?: string;
    reviewed_at?: string;
    review_notes?: string;
    original_status?: string;
  };
}

export type PriorityFilterValue = Priority | 'all';
