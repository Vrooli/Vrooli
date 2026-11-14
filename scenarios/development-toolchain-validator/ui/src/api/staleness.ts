import { createClient } from "@connectrpc/connect";
import {
  StalenessService,
  StaleKind,
  type StaleEntry,
  type ListStaleResponse,
} from "@vrooli/proto-types/development-toolchain-validator/v1/staleness/staleness_pb";

import { transport } from "./client";

export const stalenessClient = createClient(StalenessService, transport);

export { StaleKind };
export type { StaleEntry, ListStaleResponse };
