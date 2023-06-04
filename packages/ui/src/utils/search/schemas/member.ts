import { endpointGetMembers, MemberSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const memberSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchMember"),
    containers: [], //TODO
    fields: [], //TODO
});

export const memberSearchParams = () => toParams(memberSearchSchema(), endpointGetMembers, MemberSortBy, MemberSortBy.DateCreatedDesc);
