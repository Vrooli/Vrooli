import { endpointGetReactions, ReactionSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const reactionSearchSchema = (): FormSchema => ({
    layout: searchFormLayout("SearchReaction"),
    containers: [], //TODO
    elements: [], //TODO
});

export const reactionSearchParams = () => toParams(reactionSearchSchema(), endpointGetReactions, undefined, ReactionSortBy, ReactionSortBy.DateUpdatedDesc);
