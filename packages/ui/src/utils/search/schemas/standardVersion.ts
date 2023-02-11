import { StandardVersionSortBy } from "@shared/consts";
import { standardVersionFindMany } from "api/generated/endpoints/standardVersion";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const standardVersionSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchStandardVersions', lng),
    containers: [], //TODO
    fields: [], //TODO
})

export const standardVersionSearchParams = (lng: string) => toParams(standardVersionSearchSchema(lng), standardVersionFindMany, StandardVersionSortBy, StandardVersionSortBy.DateCreatedDesc);