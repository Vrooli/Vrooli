import { MemberSortBy } from "@shared/consts";
import { memberFindMany } from "api/generated/endpoints/member";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const memberSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchMembers', lng),
    containers: [], //TODO
    fields: [], //TODO
})

export const memberSearchParams = (lng: string) => toParams(memberSearchSchema(lng), memberFindMany, MemberSortBy, MemberSortBy.DateCreatedDesc);