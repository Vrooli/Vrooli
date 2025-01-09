import { endpointsMemberInvite, FormSchema, MemberInviteSortBy } from "@local/shared";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

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
