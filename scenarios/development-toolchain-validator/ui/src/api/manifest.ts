import { createClient } from "@connectrpc/connect";
import {
  ManifestService,
  ConvergenceTarget,
  type Manifest,
  type ContentRule,
  type ListManifestsResponse,
  type GetManifestResponse,
  type UpsertManifestResponse,
  type ClearStaleResponse,
} from "@vrooli/proto-types/development-toolchain-validator/v1/manifest/manifest_pb";

import { transport } from "./client";

export const manifestClient = createClient(ManifestService, transport);

export { ConvergenceTarget };
export type {
  Manifest,
  ContentRule,
  ListManifestsResponse,
  GetManifestResponse,
  UpsertManifestResponse,
  ClearStaleResponse,
};
