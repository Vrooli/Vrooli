import { FindByIdOrHandleInput, Organization, OrganizationCreateInput, OrganizationSearchInput, OrganizationUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { rateLimit } from "../../middleware/rateLimit";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsOrganization = {
    Query: {
        organization: GQLEndpoint<FindByIdOrHandleInput, FindOneResult<Organization>>;
        organizations: GQLEndpoint<OrganizationSearchInput, FindManyResult<Organization>>;
    },
    Mutation: {
        organizationCreate: GQLEndpoint<OrganizationCreateInput, CreateOneResult<Organization>>;
        organizationUpdate: GQLEndpoint<OrganizationUpdateInput, UpdateOneResult<Organization>>;
    }
}

const objectType = "Organization";
export const OrganizationEndpoints: EndpointsOrganization = {
    Query: {
        organization: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        organizations: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
    },
    Mutation: {
        organizationCreate: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 100, req });
            return createOneHelper({ info, input, objectType, req });
        },
        organizationUpdate: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 250, req });
            return updateOneHelper({ info, input, objectType, req });
        },
    },
};
