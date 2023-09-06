import { ChatSortBy, endpointGetChat, endpointGetChats } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const chatSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchChat"),
    containers: [], //TODO
    fields: [], //TODO
});

export const chatSearchParams = () => toParams(chatSearchSchema(), endpointGetChats, endpointGetChat, ChatSortBy, ChatSortBy.DateUpdatedDesc);
