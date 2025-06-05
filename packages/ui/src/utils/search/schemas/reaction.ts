import { ReactionSortBy, endpointsReaction, type FormSchema } from "@vrooli/shared";
import { toParams } from "./base.js";
import { searchFormLayout } from "./common.js";

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
