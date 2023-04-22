import { StandardVersionSortBy } from "@shared/consts";
import { FormSchema } from "../../../forms/types";
import { standardVersionFindMany } from "../../../api/generated/endpoints/standardVersion_findMany";
import { toParams } from "./base";
import { bookmarksRootContainer, bookmarksRootFields, isCompleteWithRootContainer, isCompleteWithRootFields, isLatestContainer, isLatestFields, languagesContainer, languagesFields, searchFormLayout, tagsRootContainer, tagsRootFields, votesRootContainer, votesRootFields } from "./common";

export const standardVersionSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchStandardVersion"),
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

export const standardVersionSearchParams = () => toParams(standardVersionSearchSchema(), standardVersionFindMany, StandardVersionSortBy, StandardVersionSortBy.DateCreatedDesc);
