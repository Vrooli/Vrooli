import { endpointsMember, FormSchema, MemberSortBy } from "@local/shared";
import { toParams } from "./base.js";
import { searchFormLayout } from "./common.js";

export function memberSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchMember"),
        containers: [], //TODO
        elements: [], //TODO
    };
}

export function memberSearchParams() {
    return toParams(memberSearchSchema(), endpointsMember, MemberSortBy, MemberSortBy.DateCreatedDesc);
}
