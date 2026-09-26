import { createClient } from "@connectrpc/connect";
import {
  GoldenService,
  type Golden,
  type ListGoldensResponse,
} from "@vrooli/proto-types/development-toolchain-validator/v1/golden/golden_pb";

import { transport } from "./client";

export const goldenClient = createClient(GoldenService, transport);

export type { Golden, ListGoldensResponse };
