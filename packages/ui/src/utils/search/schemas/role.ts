import { RoleSortBy } from "@local/shared";
import { roleFindMany } from "api/generated/endpoints/role_findMany";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const roleSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout('SearchRole'),
    containers: [], //TODO
    fields: [], //TODO
})

export const roleSearchParams = () => toParams(roleSearchSchema(), roleFindMany, RoleSortBy, RoleSortBy.DateCreatedDesc)