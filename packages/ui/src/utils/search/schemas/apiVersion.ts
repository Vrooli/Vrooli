import { ApiVersionSortBy } from "@shared/consts";
import { apiVersionFindMany } from "api/generated/endpoints/apiVersion_findMany";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { isCompleteWithRootContainer, isCompleteWithRootFields, votesRootContainer, votesRootFields, bookmarksRootContainer, bookmarksRootFields, languagesContainer, languagesFields, tagsRootContainer, tagsRootFields, searchFormLayout, isLatestContainer, isLatestFields } from "./common";

export const apiVersionSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout('SearchApiVersion'),
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
    ]
})

export const apiVersionSearchParams = () => toParams(apiVersionSearchSchema(), apiVersionFindMany, ApiVersionSortBy, ApiVersionSortBy.DateCreatedDesc);