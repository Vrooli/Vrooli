import { VoteSortBy } from "@shared/consts";
import { voteFindMany } from "api/generated/endpoints/vote";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const voteSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchVote', lng),
    containers: [], //TODO
    fields: [], //TODO
})

export const voteSearchParams = (lng: string) => toParams(voteSearchSchema(lng), voteFindMany, VoteSortBy, VoteSortBy.DateUpdatedDesc)