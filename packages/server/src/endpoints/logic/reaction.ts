import { ReactInput, Reaction, ReactionSearchInput, Success } from "@local/shared";
import { readManyHelper } from "../../actions/reads";
import { RequestService } from "../../auth/request";
import { ReactionModel } from "../../models/base/reaction";
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
        reactions: async (_, { input }, { req }, info) => {
            const userData = RequestService.assertRequestFrom(req, { isUser: true });
            await RequestService.get().rateLimit({ maxUser: 2000, req });
            return readManyHelper({ info, input, objectType, req, additionalQueries: { userId: userData.id } });
        },
    },
    Mutation: {
        /**
         * Adds or removes a reaction on an object. A user can only have one reaction per object, meaning 
         * the previous reaction is overruled
         */
        react: async (_, { input }, { req }) => {
            const userData = RequestService.assertRequestFrom(req, { isUser: true });
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            const success = await ReactionModel.react(userData, input);
            return { __typename: "Success", success };
        },
    },
};
