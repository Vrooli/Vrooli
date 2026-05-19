import { createClient } from '@connectrpc/connect';
import { WorkflowsService } from '@vrooli/proto-types/browser-automation-studio/v1/api/service_pb';

import { transport } from './client';

export const workflowsClient = createClient(WorkflowsService, transport);
