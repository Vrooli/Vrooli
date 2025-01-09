import { ChatParticipantSortBy, FormSchema, endpointsChatParticipant } from "@local/shared";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

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
