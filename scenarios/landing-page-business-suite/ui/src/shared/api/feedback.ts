import { apiCall } from './common';

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
  order_id?: string;
}

export function createFeedback(input: CreateFeedbackInput) {
  return apiCall<{ success: boolean; id: number }>('/feedback', {
    method: 'POST',
    body: JSON.stringify(input),
  });
}

export function fetchFeedbackList(status?: string) {
  const url = status ? `/admin/feedback?status=${status}` : '/admin/feedback';
  return apiCall<FeedbackRequest[]>(url);
}

export function fetchFeedbackById(id: number) {
  return apiCall<FeedbackRequest>(`/admin/feedback/${id}`);
}

export function updateFeedbackStatus(id: number, status: string) {
  return apiCall<FeedbackRequest>(`/admin/feedback/${id}/status`, {
    method: 'PATCH',
    body: JSON.stringify({ status }),
  });
}

export function deleteFeedback(id: number) {
  return apiCall<{ success: boolean; id: number }>(`/admin/feedback/${id}`, {
    method: 'DELETE',
  });
}

export function deleteFeedbackBulk(ids: number[]) {
  return apiCall<{ success: boolean; deleted: number }>('/admin/feedback/bulk-delete', {
    method: 'POST',
    body: JSON.stringify({ ids }),
  });
}
