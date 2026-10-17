import { createClient } from '@connectrpc/connect';
import {
  ScenariosService,
  type Scenario,
  type ListScenariosResponse,
  type GetScenarioPortResponse,
} from '@vrooli/proto-types/browser-automation-studio/v1/scenarios/scenarios_pb';

import { transport } from './client';

export const scenariosClient = createClient(ScenariosService, transport);

export type { Scenario, ListScenariosResponse, GetScenarioPortResponse };
