import { endpointGetRole, endpointGetRoles, RoleSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const roleSearchSchema = (): FormSchema => ({
    layout: searchFormLayout("SearchRole"),
    containers: [], //TODO
    elements: [], //TODO
});

export const roleSearchParams = () => toParams(roleSearchSchema(), endpointGetRoles, endpointGetRole, RoleSortBy, RoleSortBy.DateCreatedDesc);
