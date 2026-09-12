import { createClient, type Client } from "@connectrpc/connect";
import { DomainsService } from "@vrooli/proto-types/architecture-cartographer/v1/domains/domains_pb";

import { transport } from "./client";

export const domainsClient: Client<typeof DomainsService> = createClient(DomainsService, transport);
