import { StandardSortBy } from "@local/consts";
import { standardFindMany } from "../../../api/generated/endpoints/standard_findMany";
import { FormSchema } from "../../../forms/types";
import { toParams } from "./base";
import { bookmarksContainer, bookmarksFields, hasCompleteVersionContainer, hasCompleteVersionFields, languagesVersionContainer, languagesVersionFields, searchFormLayout, tagsContainer, tagsFields, votesContainer, votesFields } from "./common";

export const standardSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchStandard"),
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

export const standardSearchParams = () => toParams(standardSearchSchema(), standardFindMany, StandardSortBy, StandardSortBy.ScoreDesc);