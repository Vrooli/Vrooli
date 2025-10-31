import type { Issue } from './issue';

export type EventType =
  | 'issue.created'
  | 'issue.updated'
  | 'issue.status_changed'
  | 'issue.deleted'
  | 'agent.started'
  | 'agent.completed'
  | 'agent.failed'
  | 'processor.state_changed'
  | 'processor.error'
  | 'rate_limit.changed';

export interface BaseEvent<T = unknown> {
  type: EventType;
  timestamp: string;
  data: T;
}

// API Issue type for websocket events (may differ from UI Issue type)
export interface ApiIssue {
  id: string;
  title: string;
  description?: string;
  notes?: string;
  priority?: string;
  app_id?: string; // Deprecated, kept for backward compatibility
  targets?: Array<{ type: string; id: string; name?: string }>;
  status?: string;
  metadata?: {
    created_at?: string;
    updated_at?: string;
    resolved_at?: string;
    tags?: string[];
    labels?: Record<string, string>;
    extra?: {
      agent_last_error?: string;
      agent_last_status?: string;
      agent_failure_time?: string;
      rate_limit_until?: string;
      rate_limit_agent?: string;
      agent_transcript_path?: string;
      agent_last_message_path?: string;
      [key: string]: string | undefined;
    };
  };
  reporter?: {
    name?: string;
    email?: string;
    timestamp?: string;
  };
  investigation?: {
    agent_id?: string;
    report?: string;
    confidence_score?: number;
    started_at?: string;
    completed_at?: string;
  };
  attachments?: Array<{
    name?: string;
    type?: string;
    path?: string;
    size?: number;
    category?: string;
    description?: string;
  }>;
  manual_review?: {
    marked_as_failed: boolean;
    failure_reason?: string;
    reviewed_by?: string;
    reviewed_at?: string;
    review_notes?: string;
    original_status?: string;
  };
}

export interface IssueEventData {
  issue: ApiIssue;
}

export interface IssueStatusChangedData {
  issue_id: string;
  old_status: string;
  new_status: string;
}

export interface IssueDeletedData {
  issue_id: string;
}

export interface AgentStartedData {
  issue_id: string;
  agent_id: string;
  start_time: string;
}

export interface AgentCompletedData {
  issue_id: string;
  agent_id: string;
  success: boolean;
  end_time: string;
  new_status?: string;
  scenario_restart?: string; // "success", "failed:<reason>", or undefined if not attempted
}

export interface ProcessorStateData {
  active: boolean;
  concurrent_slots: number;
  refresh_interval: number;
  max_issues: number;
  issues_processed: number;
  issues_remaining?: number | string;
  max_issues_disabled: boolean;
}

export interface RateLimitData {
  rate_limited: boolean;
  rate_limited_count: number;
  reset_time?: string;
  seconds_until_reset: number;
  rate_limit_agent?: string;
}

export interface ProcessorErrorData {
  message: string;
  details?: Record<string, unknown>;
}

export type IssueCreatedEvent = BaseEvent<IssueEventData>;
export type IssueUpdatedEvent = BaseEvent<IssueEventData>;
export type IssueStatusChangedEvent = BaseEvent<IssueStatusChangedData>;
export type IssueDeletedEvent = BaseEvent<IssueDeletedData>;
export type AgentStartedEvent = BaseEvent<AgentStartedData>;
export type AgentCompletedEvent = BaseEvent<AgentCompletedData>;
export type AgentFailedEvent = BaseEvent<AgentCompletedData>;
export type ProcessorStateChangedEvent = BaseEvent<ProcessorStateData>;
export type ProcessorErrorEvent = BaseEvent<ProcessorErrorData>;
export type RateLimitChangedEvent = BaseEvent<RateLimitData>;

export type WebSocketEvent =
  | IssueCreatedEvent
  | IssueUpdatedEvent
  | IssueStatusChangedEvent
  | IssueDeletedEvent
  | AgentStartedEvent
  | AgentCompletedEvent
  | AgentFailedEvent
  | ProcessorStateChangedEvent
  | ProcessorErrorEvent
  | RateLimitChangedEvent;
