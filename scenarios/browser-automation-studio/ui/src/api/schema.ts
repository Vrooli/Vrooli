import { createClient } from '@connectrpc/connect';
import {
  SchemaService,
} from '@vrooli/proto-types/browser-automation-studio/v1/schema/schema_pb';

import { transport } from './client';

export const schemaClient = createClient(SchemaService, transport);
