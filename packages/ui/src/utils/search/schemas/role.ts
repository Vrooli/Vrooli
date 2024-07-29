import { endpointGetRole, endpointGetRoles, FormSchema, RoleSortBy } from "@local/shared";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const roleSearchSchema = (): FormSchema => ({
    layout: searchFormLayout("SearchRole"),
    containers: [], //TODO
    elements: [], //TODO
});

export const roleSearchParams = () => toParams(roleSearchSchema(), endpointGetRoles, endpointGetRole, RoleSortBy, RoleSortBy.DateCreatedDesc);
