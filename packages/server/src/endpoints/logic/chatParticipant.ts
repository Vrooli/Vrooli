import { ChatParticipant, ChatParticipantSearchInput, ChatParticipantUpdateInput, FindByIdInput } from "@local/shared";
import { readManyHelper, readOneHelper, updateHelper } from "../../actions";
import { rateLimit } from "../../middleware";
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
        chatParticipant: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        chatParticipants: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        chatParticipantUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    },
};
