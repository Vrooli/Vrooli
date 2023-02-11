import { ReputationHistorySortBy } from "@shared/consts";
import { reputationHistoryFindMany } from "api/generated/endpoints/reputationHistory";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const reputationHistorySearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchReputationHistory', lng),
    containers: [], //TODO
    fields: [], //TODO
})

export const reputationHistorySearchParams = (lng: string) => toParams(reputationHistorySearchSchema(lng), reputationHistoryFindMany, ReputationHistorySortBy, ReputationHistorySortBy.DateCreatedDesc)