import { ResourceSortBy } from "@local/consts";
import { resourceFindMany } from "../../../api/generated/endpoints/resource_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";
export const resourceSearchSchema = () => ({
    formLayout: searchFormLayout("SearchResource"),
    containers: [],
    fields: [],
});
export const resourceSearchParams = () => toParams(resourceSearchSchema(), resourceFindMany, ResourceSortBy, ResourceSortBy.DateCreatedDesc);
//# sourceMappingURL=resource.js.map