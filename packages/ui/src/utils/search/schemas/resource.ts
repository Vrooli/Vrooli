import { endpointGetResource, endpointGetResources, ResourceSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const resourceSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchResource"),
    containers: [], //TODO
    fields: [], //TODO
});

export const resourceSearchParams = () => toParams(resourceSearchSchema(), endpointGetResources, endpointGetResource, ResourceSortBy, ResourceSortBy.DateCreatedDesc);
