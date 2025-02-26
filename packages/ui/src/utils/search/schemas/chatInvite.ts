import { ChatInviteSortBy, FormSchema, endpointsChatInvite } from "@local/shared";
import { toParams } from "./base.js";
import { searchFormLayout } from "./common.js";

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
