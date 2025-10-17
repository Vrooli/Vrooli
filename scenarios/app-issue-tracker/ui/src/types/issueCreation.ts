import type { IssueStatus, Priority } from '../data/sampleData';

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
  appId?: string;
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
  appId: string;
  tags: string[];
  reporterName?: string;
  reporterEmail?: string;
  attachments: CreateIssueAttachmentPayload[];
}

export type PriorityFilterValue = Priority | 'all';
