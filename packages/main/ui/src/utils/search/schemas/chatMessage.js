import { ChatMessageSortBy } from "@local/consts";
import { chatMessageFindMany } from "../../../api/generated/endpoints/chatMessage_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";
export const chatMessageSearchSchema = () => ({
    formLayout: searchFormLayout("SearchChatMessage"),
    containers: [],
    fields: [],
});
export const chatMessageSearchParams = () => toParams(chatMessageSearchSchema(), chatMessageFindMany, ChatMessageSortBy, ChatMessageSortBy.DateUpdatedDesc);
//# sourceMappingURL=chatMessage.js.map