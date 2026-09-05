import { createClient } from "@connectrpc/connect";
import { IdentityService } from "@vrooli/proto-types/device-sync-hub/v1/identity/identity_pb";

import { transport } from "./client";

/**
 * Typed client for the hub's own IdentityService — the SAME-ORIGIN owner
 * sign-in / registration facade. The browser never calls scenario-authenticator
 * directly (no cross-origin calls); it calls this hub RPC, and the hub forwards
 * to scenario-authenticator (resolved by name via api-core/discovery) and
 * relays back the issued owner JWT. Both RPCs are unauthenticated (they precede
 * the caller holding a token), so they ride the shared transport without a
 * credential.
 */
export const identityClient = createClient(IdentityService, transport);
