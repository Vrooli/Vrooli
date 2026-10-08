import { createClient } from '@connectrpc/connect';
import { ExecutionsService } from '@vrooli/proto-types/browser-automation-studio/v1/api/service_pb';

import { transport } from './client';

export const executionsClient = createClient(ExecutionsService, transport);
