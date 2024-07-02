import { endpointGetReputationHistories, endpointGetReputationHistory, ReputationHistorySortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const reputationHistorySearchSchema = (): FormSchema => ({
    layout: searchFormLayout("SearchReputationHistory"),
    containers: [], //TODO
    elements: [], //TODO
});

export const reputationHistorySearchParams = () => toParams(reputationHistorySearchSchema(), endpointGetReputationHistories, endpointGetReputationHistory, ReputationHistorySortBy, ReputationHistorySortBy.DateCreatedDesc);
