import { ChatInviteSortBy, endpointGetChatInvites } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const chatInviteSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchChatInvite"),
    containers: [], //TODO
    fields: [], //TODO
});

export const chatInviteSearchParams = () => toParams(chatInviteSearchSchema(), endpointGetChatInvites, ChatInviteSortBy, ChatInviteSortBy.DateUpdatedDesc);
