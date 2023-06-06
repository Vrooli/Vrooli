import { ChatParticipantSortBy, endpointGetChatParticipants } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const chatParticipantSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchChatParticipant"),
    containers: [], //TODO
    fields: [], //TODO
});

export const chatParticipantSearchParams = () => toParams(chatParticipantSearchSchema(), endpointGetChatParticipants, ChatParticipantSortBy, ChatParticipantSortBy.DateUpdatedDesc);
