import { ChatMessageSortBy } from "@shared/consts";
import { FormSchema } from "forms/types";
import { chatMessageFindMany } from "../../../api/generated/endpoints/chatMessage_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const chatMessageSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchChatMessage"),
    containers: [], //TODO
    fields: [], //TODO
})

export const chatMessageSearchParams = () => toParams(chatMessageSearchSchema(), chatMessageFindMany, ChatMessageSortBy, ChatMessageSortBy.DateUpdatedDesc)