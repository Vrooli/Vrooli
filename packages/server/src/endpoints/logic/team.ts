import { FindByIdOrHandleInput, Team, TeamCreateInput, TeamSearchInput, TeamUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { ApiEndpoint, CreateOneResult, FindManyResult, FindOneResult, UpdateOneResult } from "../../types";

export type EndpointsTeam = {
    Query: {
        team: ApiEndpoint<FindByIdOrHandleInput, FindOneResult<Team>>;
        teams: ApiEndpoint<TeamSearchInput, FindManyResult<Team>>;
    },
    Mutation: {
        teamCreate: ApiEndpoint<TeamCreateInput, CreateOneResult<Team>>;
        teamUpdate: ApiEndpoint<TeamUpdateInput, UpdateOneResult<Team>>;
    }
}

const objectType = "Team";
export const TeamEndpoints: EndpointsTeam = {
    Query: {
        team: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        teams: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
    Mutation: {
        teamCreate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 100, req });
            return createOneHelper({ info, input, objectType, req });
        },
        teamUpdate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 250, req });
            return updateOneHelper({ info, input, objectType, req });
        },
    },
};
