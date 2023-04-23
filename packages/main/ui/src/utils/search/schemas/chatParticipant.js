import { ChatParticipantSortBy } from "@local/consts";
import { chatParticipantFindMany } from "../../../api/generated/endpoints/chatParticipant_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";
export const chatParticipantSearchSchema = () => ({
    formLayout: searchFormLayout("SearchChatParticipant"),
    containers: [],
    fields: [],
});
export const chatParticipantSearchParams = () => toParams(chatParticipantSearchSchema(), chatParticipantFindMany, ChatParticipantSortBy, ChatParticipantSortBy.DateUpdatedDesc);
//# sourceMappingURL=chatParticipant.js.map