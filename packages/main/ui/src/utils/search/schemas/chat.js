import { ChatSortBy } from "@local/consts";
import { chatFindMany } from "../../../api/generated/endpoints/chat_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";
export const chatSearchSchema = () => ({
    formLayout: searchFormLayout("SearchChat"),
    containers: [],
    fields: [],
});
export const chatSearchParams = () => toParams(chatSearchSchema(), chatFindMany, ChatSortBy, ChatSortBy.DateUpdatedDesc);
//# sourceMappingURL=chat.js.map