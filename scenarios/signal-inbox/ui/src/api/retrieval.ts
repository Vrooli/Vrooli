import { createClient } from "@connectrpc/connect";
import { RetrievalService, type RetrievedSignal } from "../../../../../packages/proto/gen/typescript/signal-inbox/v1/retrieval/retrieval_pb";
import { transport } from "./client";

export const retrievalClient = createClient(RetrievalService, transport);
export type { RetrievedSignal };
