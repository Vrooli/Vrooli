import { PostSortBy } from "@shared/consts";
import { postFindMany } from "api/generated/endpoints/post";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const postSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchPosts', lng),
    containers: [], //TODO
    fields: [], //TODO
})

export const postSearchParams = (lng: string) => toParams(postSearchSchema(lng), postFindMany, PostSortBy, PostSortBy.DateCreatedDesc);