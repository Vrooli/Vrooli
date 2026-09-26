import { createClient } from '@connectrpc/connect';
import { UXMetricsService } from '@vrooli/proto-types/browser-automation-studio/v1/uxmetrics/uxmetrics_pb';

import { transport } from './client';

export const uxmetricsClient = createClient(UXMetricsService, transport);
