import { ChatMessageSortBy, endpointGetChatMessage, endpointGetChatMessages } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const chatMessageSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchChatMessage"),
    containers: [], //TODO
    fields: [], //TODO
});

export const chatMessageSearchParams = () => toParams(chatMessageSearchSchema(), endpointGetChatMessages, endpointGetChatMessage, ChatMessageSortBy, ChatMessageSortBy.DateCreatedDesc);
