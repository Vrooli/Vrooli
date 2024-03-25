import { FindByIdInput, Role, RoleCreateInput, RoleSearchInput, RoleUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { rateLimit } from "../../middleware/rateLimit";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsRole = {
    Query: {
        role: GQLEndpoint<FindByIdInput, FindOneResult<Role>>;
        roles: GQLEndpoint<RoleSearchInput, FindManyResult<Role>>;
    },
    Mutation: {
        roleCreate: GQLEndpoint<RoleCreateInput, CreateOneResult<Role>>;
        roleUpdate: GQLEndpoint<RoleUpdateInput, UpdateOneResult<Role>>;
    }
}

const objectType = "Role";
export const RoleEndpoints: EndpointsRole = {
    Query: {
        role: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        roles: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
    Mutation: {
        roleCreate: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 100, req });
            return createOneHelper({ info, input, objectType, req });
        },
        roleUpdate: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 250, req });
            return updateOneHelper({ info, input, objectType, req });
        },
    },
};
