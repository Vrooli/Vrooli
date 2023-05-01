import { TagSortBy } from "@local/shared";
import { tagFindMany } from "api/generated/endpoints/tag_findMany";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const tagSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchTag"),
    containers: [], //TODO
    fields: [], //TODO
});

export const tagSearchParams = () => toParams(tagSearchSchema(), tagFindMany, TagSortBy, TagSortBy.BookmarksDesc);
