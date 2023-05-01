import { ReactionSortBy } from "@local/shared";
import { reactionFindMany } from "api/generated/endpoints/reaction_findMany";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const reactionSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchReaction"),
    containers: [], //TODO
    fields: [], //TODO
});

export const reactionSearchParams = () => toParams(reactionSearchSchema(), reactionFindMany, ReactionSortBy, ReactionSortBy.DateUpdatedDesc);
