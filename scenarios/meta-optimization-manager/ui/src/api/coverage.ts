import { createClient, type Client } from "@connectrpc/connect";
import { CoverageService } from "@vrooli/proto-types/meta-optimization-manager/v1/coverage/coverage_pb";

import { transport } from "./client";

/**
 * Typed Connect client for the CoverageService (the readiness scoreboard). The
 * UI calls these methods with plain message-init objects; connect-es validates
 * against the generated schema.
 */
export const coverageClient: Client<typeof CoverageService> = createClient(CoverageService, transport);
