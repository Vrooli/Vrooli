import { endpointGetRole, endpointGetRoles, RoleSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const roleSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchRole"),
    containers: [], //TODO
    fields: [], //TODO
});

export const roleSearchParams = () => toParams(roleSearchSchema(), endpointGetRoles, endpointGetRole, RoleSortBy, RoleSortBy.DateCreatedDesc);
