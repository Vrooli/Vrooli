import { ProjectSortBy } from "@local/consts";
import { projectFindMany } from "../../../api/generated/endpoints/project_findMany";
import { toParams } from "./base";
import { bookmarksContainer, bookmarksFields, hasCompleteVersionContainer, hasCompleteVersionFields, languagesVersionContainer, languagesVersionFields, searchFormLayout, tagsContainer, tagsFields, votesContainer, votesFields } from "./common";
export const projectSearchSchema = () => ({
    formLayout: searchFormLayout("SearchProject"),
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
export const projectSearchParams = () => toParams(projectSearchSchema(), projectFindMany, ProjectSortBy, ProjectSortBy.ScoreDesc);
//# sourceMappingURL=project.js.map