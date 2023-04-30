import { PostSortBy } from "@local/shared";
import { postFindMany } from "api/generated/endpoints/post_findMany";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const postSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchPost"),
    containers: [], //TODO
    fields: [], //TODO
});

export const postSearchParams = () => toParams(postSearchSchema(), postFindMany, PostSortBy, PostSortBy.DateCreatedDesc);
