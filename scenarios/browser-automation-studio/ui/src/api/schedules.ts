import { createClient } from '@connectrpc/connect';
import { SchedulesService } from '@vrooli/proto-types/browser-automation-studio/v1/schedules/schedules_pb';

import { transport } from './client';

export const schedulesClient = createClient(SchedulesService, transport);
