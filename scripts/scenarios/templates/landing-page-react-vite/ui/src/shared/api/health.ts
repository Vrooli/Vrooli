import { apiCall } from './common';

export interface HealthResponse {
  status: string;
  service: string;
  timestamp: string;
}

export function fetchHealth() {
  return apiCall<HealthResponse>('/health');
}
