import { endpointGetPosts, PostSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const postSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchPost"),
    containers: [], //TODO
    fields: [], //TODO
});

export const postSearchParams = () => toParams(postSearchSchema(), endpointGetPosts, PostSortBy, PostSortBy.DateCreatedDesc);
