import { createClient, type Client } from "@connectrpc/connect";
import {
  HolderService,
  MinterService,
} from "@vrooli/proto-types/token-economy/v1/access/access_pb";
import { EarningService } from "@vrooli/proto-types/token-economy/v1/earning/earning_pb";

import { transport } from "./client";

/**
 * Generated-contract clients for the two audience boundaries.
 *
 * The operator console may use `minterClient` and `earningClient`. Holder
 * screens use `holderClient` exclusively so authority RPCs are absent from
 * their client surface rather than hidden with a runtime role check.
 */
export const minterClient: Client<typeof MinterService> = createClient(MinterService, transport);
export const earningClient: Client<typeof EarningService> = createClient(EarningService, transport);
export const holderClient: Client<typeof HolderService> = createClient(HolderService, transport);

export function nextIdempotencyKey(prefix: string) {
  return `${prefix}-${crypto.randomUUID()}`;
}
