import { endpointGetMemberInvite, endpointGetMemberInvites, MemberInviteSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const memberInviteSearchSchema = (): FormSchema => ({
    layout: searchFormLayout("SearchMemberInvite"),
    containers: [], //TODO
    elements: [], //TODO
});

export const memberInviteSearchParams = () => toParams(memberInviteSearchSchema(), endpointGetMemberInvites, endpointGetMemberInvite, MemberInviteSortBy, MemberInviteSortBy.DateCreatedDesc);
