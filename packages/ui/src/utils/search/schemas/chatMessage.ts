import { ChatMessageSortBy, endpointsChatMessage, type FormSchema } from "@vrooli/shared";
import { toParams } from "./base.js";
import { searchFormLayout } from "./common.js";

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
