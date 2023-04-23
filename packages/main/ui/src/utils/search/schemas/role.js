import { RoleSortBy } from "@local/consts";
import { roleFindMany } from "../../../api/generated/endpoints/role_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";
export const roleSearchSchema = () => ({
    formLayout: searchFormLayout("SearchRole"),
    containers: [],
    fields: [],
});
export const roleSearchParams = () => toParams(roleSearchSchema(), roleFindMany, RoleSortBy, RoleSortBy.DateCreatedDesc);
//# sourceMappingURL=role.js.map