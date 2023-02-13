import { MemberInviteSortBy } from "@shared/consts";
import { memberInviteFindMany } from "api/generated/endpoints/memberInvite";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const memberInviteSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchMemberInvite', lng),
    containers: [], //TODO
    fields: [], //TODO
})

export const memberInviteSearchParams = (lng: string) => toParams(memberInviteSearchSchema(lng), memberInviteFindMany, MemberInviteSortBy, MemberInviteSortBy.DateCreatedDesc);