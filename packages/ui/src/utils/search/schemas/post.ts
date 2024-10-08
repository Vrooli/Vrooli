import { endpointGetPost, endpointGetPosts, FormSchema, PostSortBy } from "@local/shared";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const postSearchSchema = (): FormSchema => ({
    layout: searchFormLayout("SearchPost"),
    containers: [], //TODO
    elements: [], //TODO
});

export const postSearchParams = () => toParams(postSearchSchema(), endpointGetPosts, endpointGetPost, PostSortBy, PostSortBy.DateCreatedDesc);
