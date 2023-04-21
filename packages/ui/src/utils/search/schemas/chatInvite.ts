import { ChatInviteSortBy } from "@shared/consts";
import { FormSchema } from "forms/types";
import { chatInviteFindMany } from "../../api/generated/endpoints/chatInvite_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const chatInviteSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout('SearchChatInvite'),
    containers: [], //TODO
    fields: [], //TODO
})

export const chatInviteSearchParams = () => toParams(chatInviteSearchSchema(), chatInviteFindMany, ChatInviteSortBy, ChatInviteSortBy.DateUpdatedDesc)