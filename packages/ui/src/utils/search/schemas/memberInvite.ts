import { endpointsMemberInvite, type FormSchema, MemberInviteSortBy } from "@local/shared";
import { toParams } from "./base.js";
import { searchFormLayout } from "./common.js";

export function memberInviteSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchMemberInvite"),
        containers: [], //TODO
        elements: [], //TODO
    };
}

export function memberInviteSearchParams() {
    return toParams(memberInviteSearchSchema(), endpointsMemberInvite, MemberInviteSortBy, MemberInviteSortBy.DateCreatedDesc);
}
