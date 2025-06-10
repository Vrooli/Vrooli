import { type ReactInput, type ReactionSearchInput, type ReactionSearchResult, type Success } from "@vrooli/shared";
import { readManyHelper } from "../../actions/reads.js";
import { RequestService } from "../../auth/request.js";
import { ReactionModel } from "../../models/base/reaction.js";
import { type ApiEndpoint } from "../../types.js";
import { createStandardCrudEndpoints, PermissionPresets, RateLimitPresets } from "../helpers/endpointFactory.js";

export type EndpointsReaction = {
    findMany: ApiEndpoint<ReactionSearchInput, ReactionSearchResult>;
    createOne: ApiEndpoint<ReactInput, Success>;
}

export const reaction: EndpointsReaction = createStandardCrudEndpoints({
    objectType: "Reaction",
    endpoints: {
        findMany: {
            rateLimit: RateLimitPresets.VERY_HIGH,
            permissions: PermissionPresets.READ_PUBLIC,
            customImplementation: async ({ input, req, info }) => {
                const userData = RequestService.assertRequestFrom(req, { isUser: true });
                return readManyHelper({ info, input, objectType: "Reaction", req, additionalQueries: { userId: userData.id } });
            },
        },
    },
    customEndpoints: {
        /**
         * Adds or removes a reaction on an object. A user can only have one reaction per object, meaning 
         * the previous reaction is overruled
         */
        createOne: async (wrapped, { req }) => {
            const input = wrapped?.input;
            const userData = RequestService.assertRequestFrom(req, { isUser: true });
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            RequestService.assertRequestFrom(req, { hasWritePrivatePermissions: true });
            const success = await ReactionModel.react(userData, input);
            return { __typename: "Success" as const, success };
        },
    },
});
