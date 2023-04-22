import { ApiVersionSortBy } from "@shared/consts";
import { FormSchema } from "../../../forms/types";
import { apiVersionFindMany } from "../../../api/generated/endpoints/apiVersion_findMany";
import { toParams } from "./base";
import { bookmarksRootContainer, bookmarksRootFields, isCompleteWithRootContainer, isCompleteWithRootFields, isLatestContainer, isLatestFields, languagesContainer, languagesFields, searchFormLayout, tagsRootContainer, tagsRootFields, votesRootContainer, votesRootFields } from "./common";

export const apiVersionSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchApiVersion"),
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

export const apiVersionSearchParams = () => toParams(apiVersionSearchSchema(), apiVersionFindMany, ApiVersionSortBy, ApiVersionSortBy.DateCreatedDesc);
