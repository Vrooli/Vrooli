import { createClient } from '@connectrpc/connect';
import { createScenarioConnectTransport } from '@vrooli/api-base';
import {
  FeedbackService,
  FeedbackStatus,
  FeedbackType,
  type FeedbackRecord as GeneratedFeedbackRecord,
} from '@vrooli/proto-types/landing-page-business-suite/feedback_pb';
import { CONNECT_API_BASE } from './common';

const feedbackClient = createClient(FeedbackService, createScenarioConnectTransport({ baseUrl: CONNECT_API_BASE }));

export interface FeedbackRequest {
  id: number;
  type: 'refund' | 'bug' | 'feature' | 'general';
  email: string;
  subject: string;
  message: string;
  order_id?: string | null;
  status: 'pending' | 'in_progress' | 'resolved' | 'rejected';
  created_at: string;
  updated_at: string;
}

export interface CreateFeedbackInput {
  type: string;
  email: string;
  subject: string;
  message: string;
  orderId?: string;
}

function typeFromProto(value: FeedbackType): FeedbackRequest['type'] {
  switch (value) {
    case FeedbackType.REFUND: return 'refund';
    case FeedbackType.BUG: return 'bug';
    case FeedbackType.FEATURE: return 'feature';
    default: return 'general';
  }
}

function statusFromProto(value: FeedbackStatus): FeedbackRequest['status'] {
  switch (value) {
    case FeedbackStatus.IN_PROGRESS: return 'in_progress';
    case FeedbackStatus.RESOLVED: return 'resolved';
    case FeedbackStatus.REJECTED: return 'rejected';
    default: return 'pending';
  }
}

function statusToProto(value?: string): FeedbackStatus | undefined {
  switch (value) {
    case undefined: return undefined;
    case 'pending': return FeedbackStatus.PENDING;
    case 'in_progress': return FeedbackStatus.IN_PROGRESS;
    case 'resolved': return FeedbackStatus.RESOLVED;
    case 'rejected': return FeedbackStatus.REJECTED;
    default: throw new Error(`Invalid feedback status: ${value}`);
  }
}

function timestampToISO(value: GeneratedFeedbackRecord['createdAt']): string {
  if (!value) return '';
  return new Date(Number(value.seconds) * 1000 + value.nanos / 1_000_000).toISOString().replace('.000Z', 'Z');
}

function feedbackFromProto(value: GeneratedFeedbackRecord): FeedbackRequest {
  return {
    id: Number(value.id), type: typeFromProto(value.type), email: value.email, subject: value.subject,
    message: value.message, ...(value.orderId !== undefined ? { order_id: value.orderId } : {}),
    status: statusFromProto(value.status), created_at: timestampToISO(value.createdAt), updated_at: timestampToISO(value.updatedAt),
  };
}

export async function createFeedback(input: CreateFeedbackInput, signal?: AbortSignal): Promise<{ success: boolean; id: number }> {
  const response = await feedbackClient.createFeedback(
    { type: input.type, email: input.email, subject: input.subject, message: input.message, ...(input.orderId !== undefined ? { orderId: input.orderId } : {}) },
    { signal },
  );
  return { success: response.success, id: Number(response.id) };
}

export async function fetchFeedbackList(status?: string): Promise<FeedbackRequest[]> {
  const response = await feedbackClient.listFeedback({ status: statusToProto(status) });
  return response.feedback.map(feedbackFromProto);
}

export async function fetchFeedbackById(id: number): Promise<FeedbackRequest> {
  const response = await feedbackClient.getFeedback({ id: BigInt(id) });
  if (!response.feedback) throw new Error('Feedback response did not include a record');
  return feedbackFromProto(response.feedback);
}

export async function updateFeedbackStatus(id: number, status: string): Promise<FeedbackRequest> {
  const protoStatus = statusToProto(status);
  if (protoStatus === undefined) throw new Error('Feedback status is required');
  const response = await feedbackClient.updateFeedbackStatus({ id: BigInt(id), status: protoStatus });
  if (!response.feedback) throw new Error('Feedback response did not include a record');
  return feedbackFromProto(response.feedback);
}

export async function deleteFeedback(id: number): Promise<{ success: boolean; id: number }> {
  const response = await feedbackClient.deleteFeedback({ id: BigInt(id) });
  return { success: response.deleted, id: Number(response.id) };
}

export async function deleteFeedbackBulk(ids: number[]): Promise<{ success: boolean; deleted: number }> {
  const response = await feedbackClient.deleteFeedbackBulk({ ids: ids.map((id) => BigInt(id)) });
  return { success: true, deleted: Number(response.deleted) };
}
