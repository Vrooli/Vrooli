import { createClient } from '@connectrpc/connect';
import {
  ToolsService,
  type ListToolsResponse,
  type GetToolResponse,
  type ExecuteToolRequest,
  type ExecuteToolResponse,
} from '@vrooli/proto-types/browser-automation-studio/v1/tools/tools_pb';

import { transport } from './client';

export const toolsClient = createClient(ToolsService, transport);

export type {
  ListToolsResponse,
  GetToolResponse,
  ExecuteToolRequest,
  ExecuteToolResponse,
};
