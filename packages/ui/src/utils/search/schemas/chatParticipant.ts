import { ChatParticipantSortBy } from "@shared/consts";
import { FormSchema } from "../../../forms/types";
import { chatParticipantFindMany } from "../../../api/generated/endpoints/chatParticipant_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const chatParticipantSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchChatParticipant"),
    containers: [], //TODO
    fields: [], //TODO
})

export const chatParticipantSearchParams = () => toParams(chatParticipantSearchSchema(), chatParticipantFindMany, ChatParticipantSortBy, ChatParticipantSortBy.DateUpdatedDesc)