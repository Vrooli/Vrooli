import { ReputationHistorySortBy, endpointsReputationHistory, type FormSchema } from "@vrooli/shared";
import { toParams } from "./base.js";
import { searchFormLayout } from "./common.js";

export function reputationHistorySearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchReputationHistory"),
        containers: [], //TODO
        elements: [], //TODO
    };
}

export function reputationHistorySearchParams() {
    return toParams(reputationHistorySearchSchema(), endpointsReputationHistory, ReputationHistorySortBy, ReputationHistorySortBy.DateCreatedDesc);
}
