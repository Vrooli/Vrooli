import { ChatParticipantSortBy } from "@local/shared";
import { chatParticipantFindMany } from "api/generated/endpoints/chatParticipant_findMany";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const chatParticipantSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout('SearchChatParticipant'),
    containers: [], //TODO
    fields: [], //TODO
})

export const chatParticipantSearchParams = () => toParams(chatParticipantSearchSchema(), chatParticipantFindMany, ChatParticipantSortBy, ChatParticipantSortBy.DateUpdatedDesc)