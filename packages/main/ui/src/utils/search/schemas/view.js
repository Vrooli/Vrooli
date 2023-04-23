import { ViewSortBy } from "@local/consts";
import { viewFindMany } from "../../../api/generated/endpoints/view_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";
export const viewSearchSchema = () => ({
    formLayout: searchFormLayout("SearchView"),
    containers: [],
    fields: [],
});
export const viewSearchParams = () => toParams(viewSearchSchema(), viewFindMany, ViewSortBy, ViewSortBy.LastViewedDesc);
//# sourceMappingURL=view.js.map