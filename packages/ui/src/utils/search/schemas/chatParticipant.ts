import { ChatParticipantSortBy, type FormSchema, endpointsChatParticipant } from "@local/shared";
import { toParams } from "./base.js";
import { searchFormLayout } from "./common.js";

export function chatParticipantSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchChatParticipant"),
        containers: [], //TODO
        elements: [], //TODO
    };
}

export function chatParticipantSearchParams() {
    return toParams(chatParticipantSearchSchema(), endpointsChatParticipant, ChatParticipantSortBy, ChatParticipantSortBy.DateUpdatedDesc);
}
