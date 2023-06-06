import { endpointGetMemberInvites, MemberInviteSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const memberInviteSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchMemberInvite"),
    containers: [], //TODO
    fields: [], //TODO
});

export const memberInviteSearchParams = () => toParams(memberInviteSearchSchema(), endpointGetMemberInvites, MemberInviteSortBy, MemberInviteSortBy.DateCreatedDesc);
