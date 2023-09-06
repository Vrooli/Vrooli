import { endpointGetReputationHistories, endpointGetReputationHistory, ReputationHistorySortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const reputationHistorySearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchReputationHistory"),
    containers: [], //TODO
    fields: [], //TODO
});

export const reputationHistorySearchParams = () => toParams(reputationHistorySearchSchema(), endpointGetReputationHistories, endpointGetReputationHistory, ReputationHistorySortBy, ReputationHistorySortBy.DateCreatedDesc);
