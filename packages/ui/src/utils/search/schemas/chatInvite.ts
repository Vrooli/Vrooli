import { ChatInviteSortBy, endpointGetChatInvite, endpointGetChatInvites } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const chatInviteSearchSchema = (): FormSchema => ({
    layout: searchFormLayout("SearchChatInvite"),
    containers: [], //TODO
    elements: [], //TODO
});

export const chatInviteSearchParams = () => toParams(chatInviteSearchSchema(), endpointGetChatInvites, endpointGetChatInvite, ChatInviteSortBy, ChatInviteSortBy.DateUpdatedDesc);
