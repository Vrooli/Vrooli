import { createClient } from "@connectrpc/connect";
import {
  PermissionRegistryService,
  type ListPermissionsResponse,
  type ReadPermissionResponse,
  type WritePermissionResponse,
} from "@vrooli/proto-types/vrooli-onboarding/v1/permissions/permissions_pb";
import { onboardingTransport } from "./base";

// All owner-specific permission adapters share one typed onboarding transport.
// Domain contracts stay in their own modules; this file owns only the common
// Connect client binding.
type PermissionTargetInput = {
  machineId?: string;
  nodeId?: string;
  enrollmentGeneration?: bigint;
};

interface PermissionRegistryPort {
  listPermissions(request: { owner: string; target?: PermissionTargetInput }): Promise<ListPermissionsResponse>;
  readPermission(request: { permissionId: string; owner: string; target?: PermissionTargetInput }): Promise<ReadPermissionResponse>;
  writePermission(request: {
    permissionId: string;
    owner: string;
    expectedRevision: string;
    idempotencyKey: string;
    valueJson: Uint8Array;
    target?: PermissionTargetInput;
  }): Promise<WritePermissionResponse>;
}

// Connect's inferred permission RPC replies widen to Message<string>. Keep the
// proto-shaped response contract at this transport boundary so owner adapters
// can read fields without repeating casts.
export const permissionRegistryClient = createClient(
  PermissionRegistryService,
  onboardingTransport(),
) as unknown as PermissionRegistryPort;
