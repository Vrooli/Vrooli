import { createClient } from '@connectrpc/connect';
import { ExportsService } from '@vrooli/proto-types/browser-automation-studio/v1/exports/exports_pb';

import { transport } from './client';

export const exportsClient = createClient(ExportsService, transport);
