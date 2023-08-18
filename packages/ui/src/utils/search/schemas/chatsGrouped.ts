import { ChatsGroupedSortBy, endpointGetChatsGrouped } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const chatsGroupedSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchChat"),
    containers: [], //TODO
    fields: [], //TODO
});

export const chatsGroupedSearchParams = () => toParams(chatsGroupedSearchSchema(), endpointGetChatsGrouped, ChatsGroupedSortBy, ChatsGroupedSortBy.DateUpdatedDesc);
