import { MemberSortBy } from "@local/shared";
import { memberFindMany } from "api/generated/endpoints/member_findMany";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const memberSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout('SearchMember'),
    containers: [], //TODO
    fields: [], //TODO
})

export const memberSearchParams = () => toParams(memberSearchSchema(), memberFindMany, MemberSortBy, MemberSortBy.DateCreatedDesc);