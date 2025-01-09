import { ChatMessageSortBy, FormSchema, endpointsChatMessage } from "@local/shared";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export function chatMessageSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchChatMessage"),
        containers: [], //TODO
        elements: [], //TODO
    };
}

export function chatMessageSearchParams() {
    return toParams(chatMessageSearchSchema(), endpointsChatMessage, ChatMessageSortBy, ChatMessageSortBy.DateCreatedDesc);
}
