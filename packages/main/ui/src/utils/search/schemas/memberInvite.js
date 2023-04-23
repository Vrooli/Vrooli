import { MemberInviteSortBy } from "@local/consts";
import { memberInviteFindMany } from "../../../api/generated/endpoints/memberInvite_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";
export const memberInviteSearchSchema = () => ({
    formLayout: searchFormLayout("SearchMemberInvite"),
    containers: [],
    fields: [],
});
export const memberInviteSearchParams = () => toParams(memberInviteSearchSchema(), memberInviteFindMany, MemberInviteSortBy, MemberInviteSortBy.DateCreatedDesc);
//# sourceMappingURL=memberInvite.js.map