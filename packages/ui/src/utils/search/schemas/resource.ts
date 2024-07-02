import { endpointGetResource, endpointGetResources, ResourceSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const resourceSearchSchema = (): FormSchema => ({
    layout: searchFormLayout("SearchResource"),
    containers: [], //TODO
    elements: [], //TODO
});

export const resourceSearchParams = () => toParams(resourceSearchSchema(), endpointGetResources, endpointGetResource, ResourceSortBy, ResourceSortBy.DateCreatedDesc);
