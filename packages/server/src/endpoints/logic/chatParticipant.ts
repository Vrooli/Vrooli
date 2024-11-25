import { ChatParticipant, ChatParticipantSearchInput, ChatParticipantUpdateInput, FindByIdInput } from "@local/shared";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsChatParticipant = {
    Query: {
        chatParticipant: GQLEndpoint<FindByIdInput, FindOneResult<ChatParticipant>>;
        chatParticipants: GQLEndpoint<ChatParticipantSearchInput, FindManyResult<ChatParticipant>>;
    },
    Mutation: {
        chatParticipantUpdate: GQLEndpoint<ChatParticipantUpdateInput, UpdateOneResult<ChatParticipant>>;
    }
}

const objectType = "ChatParticipant";
export const ChatParticipantEndpoints: EndpointsChatParticipant = {
    Query: {
        chatParticipant: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        chatParticipants: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
    Mutation: {
        chatParticipantUpdate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 250, req });
            return updateOneHelper({ info, input, objectType, req });
        },
    },
};
