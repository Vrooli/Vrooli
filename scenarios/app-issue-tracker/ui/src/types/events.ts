export type EventType =
  | 'issue.created'
  | 'issue.updated'
  | 'issue.status_changed'
  | 'issue.deleted'
  | 'agent.started'
  | 'agent.completed'
  | 'agent.failed'
  | 'processor.state_changed'
  | 'rate_limit.changed';

export interface BaseEvent<T = unknown> {
  type: EventType;
  timestamp: string;
  data: T;
}

export interface Issue {
  id: string;
  title: string;
  description?: string;
  notes?: string;
  priority?: string;
  app_id?: string;
  status?: string;
  metadata?: {
    created_at?: string;
    updated_at?: string;
    resolved_at?: string;
    tags?: string[];
    labels?: Record<string, string>;
  };
  reporter?: {
    name?: string;
    email?: string;
    timestamp?: string;
  };
  attachments?: Array<{
    name?: string;
    type?: string;
    path?: string;
    size?: number;
  }>;
}

export interface IssueEventData {
  issue: Issue;
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
}

export interface ProcessorStateData {
  active: boolean;
  concurrent_slots: number;
  refresh_interval: number;
  max_issues: number;
  issues_processed: number;
  issues_remaining?: number;
}

export interface RateLimitData {
  rate_limited: boolean;
  rate_limited_count: number;
  reset_time?: string;
  seconds_until_reset: number;
  rate_limit_agent?: string;
}

export type IssueCreatedEvent = BaseEvent<IssueEventData>;
export type IssueUpdatedEvent = BaseEvent<IssueEventData>;
export type IssueStatusChangedEvent = BaseEvent<IssueStatusChangedData>;
export type IssueDeletedEvent = BaseEvent<IssueDeletedData>;
export type AgentStartedEvent = BaseEvent<AgentStartedData>;
export type AgentCompletedEvent = BaseEvent<AgentCompletedData>;
export type AgentFailedEvent = BaseEvent<AgentCompletedData>;
export type ProcessorStateChangedEvent = BaseEvent<ProcessorStateData>;
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
  | RateLimitChangedEvent;
