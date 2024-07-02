import { endpointGetResourceList, endpointGetResourceLists, ResourceListSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const resourceListSearchSchema = (): FormSchema => ({
    layout: searchFormLayout("SearchResourceList"),
    containers: [], //TODO
    elements: [], //TODO
});

export const resourceListSearchParams = () => toParams(resourceListSearchSchema(), endpointGetResourceLists, endpointGetResourceList, ResourceListSortBy, ResourceListSortBy.DateCreatedDesc);
