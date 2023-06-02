import { ReactInput, Reaction, ReactionSearchInput, Success } from "@local/shared";
import { readManyHelper } from "../../actions";
import { assertRequestFrom } from "../../auth";
import { rateLimit } from "../../middleware";
import { ReactionModel } from "../../models/base";
import { FindManyResult, GQLEndpoint } from "../../types";

export type EndpointsReaction = {
    Query: {
        reactions: GQLEndpoint<ReactionSearchInput, FindManyResult<Reaction>>;
    },
    Mutation: {
        react: GQLEndpoint<ReactInput, Success>;
    }
}

const objectType = "Reaction";
export const ReactionEndpoints: EndpointsReaction = {
    Query: {
        reactions: async (_, { input }, { prisma, req }, info) => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ info, maxUser: 2000, req });
            return readManyHelper({ info, input, objectType, prisma, req, additionalQueries: { userId: userData.id } });
        },
    },
    Mutation: {
        /**
         * Adds or removes a reaction on an object. A user can only have one reaction per object, meaning 
         * the previous reaction is overruled
         */
        react: async (_, { input }, { prisma, req }, info) => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ info, maxUser: 1000, req });
            const success = await ReactionModel.react(prisma, userData, input);
            return { __typename: "Success", success };
        },
    },
};
