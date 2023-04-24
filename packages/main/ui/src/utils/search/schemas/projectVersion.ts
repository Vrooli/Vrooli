import { ProjectVersionSortBy } from ":local/consts";
import { projectVersionFindMany } from "../../../api/generated/endpoints/projectVersion_findMany";
import { FormSchema } from "../../../forms/types";
import { toParams } from "./base";
import { bookmarksRootContainer, bookmarksRootFields, isCompleteWithRootContainer, isCompleteWithRootFields, isLatestContainer, isLatestFields, languagesContainer, languagesFields, searchFormLayout, tagsRootContainer, tagsRootFields, votesRootContainer, votesRootFields } from "./common";

export const projectVersionSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchProjectVersion"),
    containers: [
        isCompleteWithRootContainer,
        isLatestContainer,
        votesRootContainer(),
        bookmarksRootContainer(),
        languagesContainer(),
        tagsRootContainer(),
    ],
    fields: [
        ...isCompleteWithRootFields(),
        ...isLatestFields(),
        ...votesRootFields(),
        ...bookmarksRootFields(),
        ...languagesFields(),
        ...tagsRootFields(),
    ],
});

export const projectVersionSearchParams = () => toParams(projectVersionSearchSchema(), projectVersionFindMany, ProjectVersionSortBy, ProjectVersionSortBy.DateCreatedDesc);
