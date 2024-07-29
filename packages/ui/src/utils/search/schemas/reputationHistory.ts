import { endpointGetReputationHistories, endpointGetReputationHistory, FormSchema, ReputationHistorySortBy } from "@local/shared";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const reputationHistorySearchSchema = (): FormSchema => ({
    layout: searchFormLayout("SearchReputationHistory"),
    containers: [], //TODO
    elements: [], //TODO
});

export const reputationHistorySearchParams = () => toParams(reputationHistorySearchSchema(), endpointGetReputationHistories, endpointGetReputationHistory, ReputationHistorySortBy, ReputationHistorySortBy.DateCreatedDesc);
