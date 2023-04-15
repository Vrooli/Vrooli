import { ChatSortBy } from "@shared/consts";
import { chatFindMany } from "api/generated/endpoints/chat_findMany";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const chatSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout('SearchChat'),
    containers: [], //TODO
    fields: [], //TODO
})

export const chatSearchParams = () => toParams(chatSearchSchema(), chatFindMany, ChatSortBy, ChatSortBy.DateUpdatedDesc)