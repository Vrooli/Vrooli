import { ChatInviteSortBy } from "@local/consts";
import { chatInviteFindMany } from "../../../api/generated/endpoints/chatInvite_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";
export const chatInviteSearchSchema = () => ({
    formLayout: searchFormLayout("SearchChatInvite"),
    containers: [],
    fields: [],
});
export const chatInviteSearchParams = () => toParams(chatInviteSearchSchema(), chatInviteFindMany, ChatInviteSortBy, ChatInviteSortBy.DateUpdatedDesc);
//# sourceMappingURL=chatInvite.js.map