import { TagSortBy } from "@shared/consts";
import { tagFindMany } from "api/generated/endpoints/tag";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const tagSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchTag', lng),
    containers: [], //TODO
    fields: [], //TODO
})

export const tagSearchParams = (lng: string) => toParams(tagSearchSchema(lng), tagFindMany, TagSortBy, TagSortBy.StarsDesc);