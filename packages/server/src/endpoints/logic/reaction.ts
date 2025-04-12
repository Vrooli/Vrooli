import { ReactInput, ReactionSearchInput, ReactionSearchResult, Success } from "@local/shared";
import { readManyHelper } from "../../actions/reads.js";
import { RequestService } from "../../auth/request.js";
import { ReactionModel } from "../../models/base/reaction.js";
import { ApiEndpoint } from "../../types.js";

export type EndpointsReaction = {
    findMany: ApiEndpoint<ReactionSearchInput, ReactionSearchResult>;
    createOne: ApiEndpoint<ReactInput, Success>;
}

const objectType = "Reaction";
export const reaction: EndpointsReaction = {
    findMany: async ({ input }, { req }, info) => {
        const userData = RequestService.assertRequestFrom(req, { isUser: true });
        await RequestService.get().rateLimit({ maxUser: 2000, req });
        return readManyHelper({ info, input, objectType, req, additionalQueries: { userId: userData.id } });
    },
    /**
     * Adds or removes a reaction on an object. A user can only have one reaction per object, meaning 
     * the previous reaction is overruled
     */
    createOne: async ({ input }, { req }) => {
        const userData = RequestService.assertRequestFrom(req, { isUser: true });
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        const success = await ReactionModel.react(userData, input);
        return { __typename: "Success", success };
    },
};
