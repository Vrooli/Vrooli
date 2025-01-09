import { endpointsReaction, FormSchema, ReactionSortBy } from "@local/shared";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export function reactionSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchReaction"),
        containers: [], //TODO
        elements: [], //TODO
    };
}

export function reactionSearchParams() {
    return toParams(reactionSearchSchema(), { findMany: endpointsReaction.findMany }, ReactionSortBy, ReactionSortBy.DateUpdatedDesc);
}
