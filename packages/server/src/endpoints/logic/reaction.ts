import { ReactInput, Reaction, ReactionSearchInput, Success } from "@local/shared";
import { readManyHelper } from "../../actions/reads";
import { RequestService } from "../../auth/request";
import { ReactionModel } from "../../models/base/reaction";
import { ApiEndpoint, FindManyResult } from "../../types";

export type EndpointsReaction = {
    findMany: ApiEndpoint<ReactionSearchInput, FindManyResult<Reaction>>;
    createOne: ApiEndpoint<ReactInput, Success>;
}

const objectType = "Reaction";
export const reaction: EndpointsReaction = {
    findMany: async (_, { input }, { req }, info) => {
        const userData = RequestService.assertRequestFrom(req, { isUser: true });
        await RequestService.get().rateLimit({ maxUser: 2000, req });
        return readManyHelper({ info, input, objectType, req, additionalQueries: { userId: userData.id } });
    },
    /**
     * Adds or removes a reaction on an object. A user can only have one reaction per object, meaning 
     * the previous reaction is overruled
     */
    createOne: async (_, { input }, { req }) => {
        const userData = RequestService.assertRequestFrom(req, { isUser: true });
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        const success = await ReactionModel.react(userData, input);
        return { __typename: "Success", success };
    },
};
