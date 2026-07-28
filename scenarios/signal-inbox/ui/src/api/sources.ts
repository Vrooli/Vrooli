import { createClient } from "@connectrpc/connect";

import { SourcesService, type AdapterState } from "../../../../../packages/proto/gen/typescript/signal-inbox/v1/sources/sources_pb";
import { transport } from "./client";

export const sourcesClient = createClient(SourcesService, transport);
export type { AdapterState };
