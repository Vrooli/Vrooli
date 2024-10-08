import { ChatSortBy, FormSchema, endpointGetChat, endpointGetChats } from "@local/shared";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const chatSearchSchema = (): FormSchema => ({
    layout: searchFormLayout("SearchChat"),
    containers: [], //TODO
    elements: [], //TODO
});

export const chatSearchParams = () => toParams(chatSearchSchema(), endpointGetChats, endpointGetChat, ChatSortBy, ChatSortBy.DateUpdatedDesc);
