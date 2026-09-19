import { createClient } from "@connectrpc/connect";
import { IdentityService } from "@vrooli/proto-types/vrooli-bridge/v1/identity/identity_pb";

import { transport } from "./client";

/**
 * Typed client for the control plane's own IdentityService — the SAME-ORIGIN
 * owner sign-in / registration facade. The browser never calls
 * scenario-authenticator directly (no cross-origin calls); it calls this bridge
 * RPC, and the bridge forwards to scenario-authenticator (resolved by name via
 * api-core/discovery) and relays back the issued owner JWT. Both RPCs are
 * unauthenticated (they precede the caller holding a token), so they ride the
 * shared transport without a credential.
 */
export const identityClient = createClient(IdentityService, transport);
