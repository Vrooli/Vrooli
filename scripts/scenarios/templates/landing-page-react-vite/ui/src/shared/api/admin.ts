import { apiCall } from './common';

export interface ResetDemoDataResponse {
  reset: boolean;
  timestamp: string;
}

export function resetDemoData() {
  return apiCall<ResetDemoDataResponse>('/admin/reset-demo-data', {
    method: 'POST',
  });
}
