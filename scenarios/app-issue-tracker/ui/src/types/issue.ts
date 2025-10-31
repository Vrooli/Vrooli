// Consolidated Issue type definition - single source of truth

export type IssueStatus =
  | "open"
  | "active"
  | "completed"
  | "failed"
  | "archived"
  | (string & {});

export type Priority = "Critical" | "High" | "Medium" | "Low";

export interface Target {
  type: "scenario" | "resource";
  id: string;
  name?: string; // Optional display name
}

export interface IssueAttachment {
  name: string;
  type?: string;
  path: string;
  size?: number;
  url: string;
  category?: string;
  description?: string;
}

export interface ManualReview {
  marked_as_failed: boolean;
  failure_reason?: string;
  reviewed_by?: string;
  reviewed_at?: string;
  review_notes?: string;
  original_status?: string;
}

// Note: When manual_review is defined, marked_as_failed must be present

export interface Issue {
  id: string;
  rawTitle?: string;
  title: string;
  summary: string;
  description?: string;
  assignee: string;
  priority: Priority;
  createdAt: string;
  status: IssueStatus;
  targets: Target[];
  tags: string[];
  attachments: IssueAttachment[];
  resolvedAt?: string | null;
  updatedAt?: string | null;
  reporterName?: string;
  reporterEmail?: string;
  notes?: string;
  manual_review?: ManualReview;
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
      agent_transcript_path?: string;
      agent_last_message_path?: string;
      rate_limit_until?: string;
      rate_limit_agent?: string;
      [key: string]: string | undefined;
    };
  };
  investigation?: {
    agent_id?: string;
    report?: string;
    confidence_score?: number;
    started_at?: string;
    completed_at?: string;
  };
  reporter?: {
    name?: string;
    email?: string;
    timestamp?: string;
  };
}

// Predefined failure reasons for manual review
export const MANUAL_FAILURE_REASONS = [
  "incomplete_work",
  "wrong_approach",
  "needs_followup",
  "misunderstood_requirements",
  "partial_completion",
  "introduced_new_issues",
  "other",
] as const;

export type ManualFailureReason = typeof MANUAL_FAILURE_REASONS[number];

export const MANUAL_FAILURE_REASON_LABELS: Record<ManualFailureReason, string> = {
  incomplete_work: "Incomplete Work",
  wrong_approach: "Wrong Approach",
  needs_followup: "Needs Follow-up",
  misunderstood_requirements: "Misunderstood Requirements",
  partial_completion: "Partial Completion",
  introduced_new_issues: "Introduced New Issues",
  other: "Other (specify in notes)",
};
