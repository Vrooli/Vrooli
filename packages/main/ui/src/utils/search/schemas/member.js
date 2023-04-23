import { MemberSortBy } from "@local/consts";
import { memberFindMany } from "../../../api/generated/endpoints/member_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";
export const memberSearchSchema = () => ({
    formLayout: searchFormLayout("SearchMember"),
    containers: [],
    fields: [],
});
export const memberSearchParams = () => toParams(memberSearchSchema(), memberFindMany, MemberSortBy, MemberSortBy.DateCreatedDesc);
//# sourceMappingURL=member.js.map