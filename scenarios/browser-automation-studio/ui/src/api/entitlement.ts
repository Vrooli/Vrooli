import { createClient } from '@connectrpc/connect';
import { EntitlementService } from '@vrooli/proto-types/browser-automation-studio/v1/entitlement/entitlement_pb';

import { transport } from './client';

export const entitlementClient = createClient(EntitlementService, transport);
