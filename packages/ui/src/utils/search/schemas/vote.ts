import { VoteSortBy } from "@shared/consts";
import { voteFindMany } from "api/generated/endpoints/vote";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const voteSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout('SearchVote'),
    containers: [], //TODO
    fields: [], //TODO
})

export const voteSearchParams = () => toParams(voteSearchSchema(), voteFindMany, VoteSortBy, VoteSortBy.DateUpdatedDesc)