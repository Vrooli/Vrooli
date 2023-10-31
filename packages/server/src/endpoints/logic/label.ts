import { FindByIdInput, Label, LabelCreateInput, LabelSearchInput, LabelUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { rateLimit } from "../../middleware/rateLimit";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsLabel = {
    Query: {
        label: GQLEndpoint<FindByIdInput, FindOneResult<Label>>;
        labels: GQLEndpoint<LabelSearchInput, FindManyResult<Label>>;
    },
    Mutation: {
        labelCreate: GQLEndpoint<LabelCreateInput, CreateOneResult<Label>>;
        labelUpdate: GQLEndpoint<LabelUpdateInput, UpdateOneResult<Label>>;
    }
}

const objectType = "Label";
export const LabelEndpoints: EndpointsLabel = {
    Query: {
        label: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        labels: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        labelCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 100, req });
            return createOneHelper({ info, input, objectType, prisma, req });
        },
        labelUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 250, req });
            return updateOneHelper({ info, input, objectType, prisma, req });
        },
    },
};
