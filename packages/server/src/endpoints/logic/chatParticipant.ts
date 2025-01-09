import { ChatParticipant, ChatParticipantSearchInput, ChatParticipantSearchResult, ChatParticipantUpdateInput, FindByIdInput } from "@local/shared";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { ApiEndpoint } from "../../types";

export type EndpointsChatParticipant = {
    findOne: ApiEndpoint<FindByIdInput, ChatParticipant>;
    findMany: ApiEndpoint<ChatParticipantSearchInput, ChatParticipantSearchResult>;
    updateOne: ApiEndpoint<ChatParticipantUpdateInput, ChatParticipant>;
}

const objectType = "ChatParticipant";
export const chatParticipant: EndpointsChatParticipant = {
    findOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readOneHelper({ info, input, objectType, req });
    },
    findMany: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req });
    },
    updateOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 250, req });
        return updateOneHelper({ info, input, objectType, req });
    },
};
