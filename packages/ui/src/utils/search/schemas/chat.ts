import { ChatSortBy, FormSchema, endpointsChat } from "@local/shared";
import { toParams } from "./base.js";
import { searchFormLayout } from "./common.js";

export function chatSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchChat"),
        containers: [], //TODO
        elements: [], //TODO
    };
}

export function chatSearchParams() {
    return toParams(chatSearchSchema(), endpointsChat, ChatSortBy, ChatSortBy.DateUpdatedDesc);
}
