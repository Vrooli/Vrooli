import { FindByIdOrHandleInput, Team, TeamCreateInput, TeamSearchInput, TeamUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { rateLimit } from "../../middleware/rateLimit";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsTeam = {
    Query: {
        team: GQLEndpoint<FindByIdOrHandleInput, FindOneResult<Team>>;
        teams: GQLEndpoint<TeamSearchInput, FindManyResult<Team>>;
    },
    Mutation: {
        teamCreate: GQLEndpoint<TeamCreateInput, CreateOneResult<Team>>;
        teamUpdate: GQLEndpoint<TeamUpdateInput, UpdateOneResult<Team>>;
    }
}

const objectType = "Team";
export const TeamEndpoints: EndpointsTeam = {
    Query: {
        team: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        teams: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
    Mutation: {
        teamCreate: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 100, req });
            return createOneHelper({ info, input, objectType, req });
        },
        teamUpdate: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 250, req });
            return updateOneHelper({ info, input, objectType, req });
        },
    },
};
