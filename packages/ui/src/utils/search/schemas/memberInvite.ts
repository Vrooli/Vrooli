import { MemberInviteSortBy } from "@shared/consts";
import { FormSchema } from "forms/types";
import { memberInviteFindMany } from "../../../api/generated/endpoints/memberInvite_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const memberInviteSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchMemberInvite"),
    containers: [], //TODO
    fields: [], //TODO
})

export const memberInviteSearchParams = () => toParams(memberInviteSearchSchema(), memberInviteFindMany, MemberInviteSortBy, MemberInviteSortBy.DateCreatedDesc);