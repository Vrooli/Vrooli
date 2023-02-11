import { StarSortBy } from "@shared/consts";
import { starFindMany } from "api/generated/endpoints/star";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const starSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchStars', lng),
    containers: [], //TODO
    fields: [], //TODO
})

export const starSearchParams = (lng: string) => toParams(starSearchSchema(lng), starFindMany, StarSortBy, StarSortBy.DateUpdatedDesc);