import { ChatParticipantSortBy, endpointGetChatParticipant, endpointGetChatParticipants } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const chatParticipantSearchSchema = (): FormSchema => ({
    layout: searchFormLayout("SearchChatParticipant"),
    containers: [], //TODO
    elements: [], //TODO
});

export const chatParticipantSearchParams = () => toParams(chatParticipantSearchSchema(), endpointGetChatParticipants, endpointGetChatParticipant, ChatParticipantSortBy, ChatParticipantSortBy.DateUpdatedDesc);
