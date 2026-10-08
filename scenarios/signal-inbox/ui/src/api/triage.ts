import { createClient } from "@connectrpc/connect";
import { TriageService } from "../../../../../packages/proto/gen/typescript/signal-inbox/v1/triage/triage_pb";
import { transport } from "./client";

export const triageClient = createClient(TriageService, transport);
