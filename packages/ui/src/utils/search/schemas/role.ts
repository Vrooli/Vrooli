import { RoleSortBy } from "@shared/consts";
import { roleFindMany } from "api/generated/endpoints/role";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const roleSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchRoles', lng),
    containers: [], //TODO
    fields: [], //TODO
})

export const roleSearchParams = (lng: string) => toParams(roleSearchSchema(lng), roleFindMany, RoleSortBy, RoleSortBy.DateCreatedDesc)