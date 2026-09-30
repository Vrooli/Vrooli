import { createClient } from '@connectrpc/connect';
import { ReplayConfigService } from '@vrooli/proto-types/browser-automation-studio/v1/replay_config/replay_config_pb';

import { transport } from './client';

export const replayConfigClient = createClient(ReplayConfigService, transport);
