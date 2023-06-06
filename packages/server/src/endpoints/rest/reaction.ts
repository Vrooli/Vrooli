import { reaction_findMany, reaction_react } from "../generated";
import { ReactionEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const ReactionRest = setupRoutes({
    "/reactions": {
        get: [ReactionEndpoints.Query.reactions, reaction_findMany],
    },
    "/react": {
        post: [ReactionEndpoints.Mutation.react, reaction_react],
    },
});
