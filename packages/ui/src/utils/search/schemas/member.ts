import { endpointGetMember, endpointGetMembers, MemberSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const memberSearchSchema = (): FormSchema => ({
    layout: searchFormLayout("SearchMember"),
    containers: [], //TODO
    elements: [], //TODO
});

export const memberSearchParams = () => toParams(memberSearchSchema(), endpointGetMembers, endpointGetMember, MemberSortBy, MemberSortBy.DateCreatedDesc);
