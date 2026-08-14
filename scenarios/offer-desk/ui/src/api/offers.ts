import { createClient } from "@connectrpc/connect";
import { create } from "@bufbuild/protobuf";
import { BoardService, CatalogService, GatesService, ListEdgesRequestSchema, ListNodesRequestSchema, ProjectionRequestSchema, EvaluateRequestSchema } from "@vrooli/proto-types/offer-desk/v1/offers/offers_pb";
import { transport } from "./client";

export const boardClient = createClient(BoardService, transport);
export const catalogClient = createClient(CatalogService, transport);
export const gatesClient = createClient(GatesService, transport);
export function fetchBoard() { return boardClient.getBoard(create(ProjectionRequestSchema)); }
export function fetchNodes() { return catalogClient.listNodes(create(ListNodesRequestSchema)); }
export function fetchEdges() { return catalogClient.listEdges(create(ListEdgesRequestSchema)); }
export function evaluateTriggers(dryRun = true) { return gatesClient.evaluate(create(EvaluateRequestSchema, { dryRun })); }
