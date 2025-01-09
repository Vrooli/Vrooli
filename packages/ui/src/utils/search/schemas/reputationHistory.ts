import { endpointsReputationHistory, FormSchema, ReputationHistorySortBy } from "@local/shared";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

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
