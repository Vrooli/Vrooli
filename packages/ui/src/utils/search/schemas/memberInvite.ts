import { MemberInviteSortBy } from "@shared/consts";
import { memberInviteFindMany } from "api/generated/endpoints/memberInvite";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const memberInviteSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout('SearchMemberInvite'),
    containers: [], //TODO
    fields: [], //TODO
})

export const memberInviteSearchParams = () => toParams(memberInviteSearchSchema(), memberInviteFindMany, MemberInviteSortBy, MemberInviteSortBy.DateCreatedDesc);