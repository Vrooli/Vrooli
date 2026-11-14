import { createClient } from '@connectrpc/connect';
import { AdminResetService } from '@vrooli/proto-types/landing-page-react-vite/v1/admin_pb';
import type { ResetDemoDataResponse } from '@vrooli/proto-types/landing-page-react-vite/v1/admin_pb';

import { transport } from './client';

const resetClient = createClient(AdminResetService, transport);

/** Resets the demo dataset to its seeded state (admin). */
export function resetDemoData(): Promise<ResetDemoDataResponse> {
  return resetClient.resetDemoData({});
}

export type { ResetDemoDataResponse };
