import { ReputationHistorySortBy } from "@shared/consts";
import { reputationHistoryFindMany } from "api/generated/endpoints/reputationHistory_findMany";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const reputationHistorySearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout('SearchReputationHistory'),
    containers: [], //TODO
    fields: [], //TODO
})

export const reputationHistorySearchParams = () => toParams(reputationHistorySearchSchema(), reputationHistoryFindMany, ReputationHistorySortBy, ReputationHistorySortBy.DateCreatedDesc)