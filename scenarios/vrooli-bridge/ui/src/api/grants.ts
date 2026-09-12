import { createClient } from "@connectrpc/connect";
import {
  CredentialGrantService,
  CreateGrantRequestSchema,
  ListGrantsRequestSchema,
  RevokeGrantRequestSchema,
  RotateAddressRequestSchema,
  type CredentialGrant,
  type RotationResponse,
} from "@vrooli/proto-types/vrooli-bridge/v1/credentialgrant/credentialgrant_pb";

import { transport } from "./client";

export const grantsClient = createClient(CredentialGrantService, transport);
export { CreateGrantRequestSchema, ListGrantsRequestSchema, RevokeGrantRequestSchema, RotateAddressRequestSchema };
export type { CredentialGrant, RotationResponse };
