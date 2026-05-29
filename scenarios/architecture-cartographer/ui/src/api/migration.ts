import { createClient, type Client } from "@connectrpc/connect";
import { MigrationService } from "@vrooli/proto-types/architecture-cartographer/v1/migration/migration_pb";

import { transport } from "./client";

export const migrationClient: Client<typeof MigrationService> = createClient(MigrationService, transport);
