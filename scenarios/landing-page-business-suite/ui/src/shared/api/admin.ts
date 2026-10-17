import { createClient } from '@connectrpc/connect';
import { createScenarioConnectTransport } from '@vrooli/api-base';
import { AdminResetService } from '@vrooli/proto-types/landing-page-business-suite/v1/admin_pb';
import { CONNECT_API_BASE } from './common';

export interface ResetDemoDataResponse {
  reset: boolean;
  timestamp: string;
}

const adminResetClient = createClient(AdminResetService, createScenarioConnectTransport({ baseUrl: CONNECT_API_BASE }));

export async function resetDemoData(): Promise<ResetDemoDataResponse> {
  const response = await adminResetClient.resetDemoData({});
  return { reset: response.reset, timestamp: response.timestamp };
}
