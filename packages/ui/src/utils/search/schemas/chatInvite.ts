import { ChatInviteSortBy, FormSchema, endpointsChatInvite } from "@local/shared";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export function chatInviteSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchChatInvite"),
        containers: [], //TODO
        elements: [], //TODO
    };
}

export function chatInviteSearchParams() {
    return toParams(chatInviteSearchSchema(), endpointsChatInvite, ChatInviteSortBy, ChatInviteSortBy.DateUpdatedDesc);
}
