import { endpointsRole, FormSchema, RoleSortBy } from "@local/shared";
import { toParams } from "./base.js";
import { searchFormLayout } from "./common.js";

export function roleSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchRole"),
        containers: [], //TODO
        elements: [], //TODO
    };
}

export function roleSearchParams() {
    return toParams(roleSearchSchema(), endpointsRole, RoleSortBy, RoleSortBy.DateCreatedDesc);
}
