import { ApiSortBy } from "@local/consts";
import { apiFindMany } from "../../../api/generated/endpoints/api_findMany";
import { toParams } from "./base";
import { bookmarksContainer, bookmarksFields, hasCompleteVersionContainer, hasCompleteVersionFields, languagesVersionContainer, languagesVersionFields, searchFormLayout, tagsContainer, tagsFields, votesContainer, votesFields } from "./common";
export const apiSearchSchema = () => ({
    formLayout: searchFormLayout("SearchApi"),
    containers: [
        hasCompleteVersionContainer,
        votesContainer(),
        bookmarksContainer(),
        languagesVersionContainer(),
        tagsContainer(),
    ],
    fields: [
        ...hasCompleteVersionFields(),
        ...votesFields(),
        ...bookmarksFields(),
        ...languagesVersionFields(),
        ...tagsFields(),
    ],
});
export const apiSearchParams = () => toParams(apiSearchSchema(), apiFindMany, ApiSortBy, ApiSortBy.ScoreDesc);
//# sourceMappingURL=api.js.map