import { ResourceListSortBy } from "@local/consts";
import { resourceListFindMany } from "../../../api/generated/endpoints/resourceList_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";
export const resourceListSearchSchema = () => ({
    formLayout: searchFormLayout("SearchResourceList"),
    containers: [],
    fields: [],
});
export const resourceListSearchParams = () => toParams(resourceListSearchSchema(), resourceListFindMany, ResourceListSortBy, ResourceListSortBy.DateCreatedDesc);
//# sourceMappingURL=resourceList.js.map