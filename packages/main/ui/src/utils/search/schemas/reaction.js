import { ReactionSortBy } from "@local/consts";
import { reactionFindMany } from "../../../api/generated/endpoints/reaction_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";
export const reactionSearchSchema = () => ({
    formLayout: searchFormLayout("SearchReaction"),
    containers: [],
    fields: [],
});
export const reactionSearchParams = () => toParams(reactionSearchSchema(), reactionFindMany, ReactionSortBy, ReactionSortBy.DateUpdatedDesc);
//# sourceMappingURL=reaction.js.map