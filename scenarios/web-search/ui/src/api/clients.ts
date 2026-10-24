import { createClient } from "@connectrpc/connect";
import { FindingsService } from "@vrooli/proto-types/web-search/v1/findings/findings_pb";
import { LiveSearchService } from "@vrooli/proto-types/web-search/v1/livesearch/livesearch_pb";
import { ResearchService } from "@vrooli/proto-types/web-search/v1/research/research_pb";

import { transport } from "./client";

/**
 * Connect-RPC clients for the web-search domain services. Built once at
 * module load over the shared `transport` (createScenarioConnectTransport).
 * Components call `findingsClient.listFindings({...})` etc. with plain request
 * init objects — the generated service descriptors type them.
 */
export const findingsClient = createClient(FindingsService, transport);
export const liveSearchClient = createClient(LiveSearchService, transport);
export const researchClient = createClient(ResearchService, transport);
