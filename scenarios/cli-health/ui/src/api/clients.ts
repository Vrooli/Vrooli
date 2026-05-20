import { createClient } from "@connectrpc/connect";
import { ReindexService } from "@vrooli/proto-types/cli-health/v1/reindex/reindex_pb";
import { SearchService } from "@vrooli/proto-types/cli-health/v1/search/search_pb";
import { ValidationService } from "@vrooli/proto-types/cli-health/v1/validation/validation_pb";

import { transport } from "./client";

export const searchClient = createClient(SearchService, transport);
export const validationClient = createClient(ValidationService, transport);
export const reindexClient = createClient(ReindexService, transport);
