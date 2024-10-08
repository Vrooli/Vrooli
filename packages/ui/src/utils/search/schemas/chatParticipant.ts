import { ChatParticipantSortBy, FormSchema, endpointGetChatParticipant, endpointGetChatParticipants } from "@local/shared";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const chatParticipantSearchSchema = (): FormSchema => ({
    layout: searchFormLayout("SearchChatParticipant"),
    containers: [], //TODO
    elements: [], //TODO
});

export const chatParticipantSearchParams = () => toParams(chatParticipantSearchSchema(), endpointGetChatParticipants, endpointGetChatParticipant, ChatParticipantSortBy, ChatParticipantSortBy.DateUpdatedDesc);
