import { createClient } from "@connectrpc/connect";
import { ReindexService } from "@vrooli/proto-types/cli-health/v1/reindex/reindex_pb";
import { SearchService } from "@vrooli/proto-types/cli-health/v1/search/search_pb";
import { ScenarioValidationService } from "@vrooli/proto-types/scenario-validation/v1/validation_pb";

import { transport } from "./client";

export const searchClient = createClient(SearchService, transport);
export const validationClient = createClient(ScenarioValidationService, transport);
export const reindexClient = createClient(ReindexService, transport);
