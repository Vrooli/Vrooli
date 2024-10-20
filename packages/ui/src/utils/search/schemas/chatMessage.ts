import { ChatMessageSortBy, FormSchema, endpointGetChatMessage, endpointGetChatMessages } from "@local/shared";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const chatMessageSearchSchema = (): FormSchema => ({
    layout: searchFormLayout("SearchChatMessage"),
    containers: [], //TODO
    elements: [], //TODO
});

export const chatMessageSearchParams = () => toParams(chatMessageSearchSchema(), endpointGetChatMessages, endpointGetChatMessage, ChatMessageSortBy, ChatMessageSortBy.DateCreatedDesc);
