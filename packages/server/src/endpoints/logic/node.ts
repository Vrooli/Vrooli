import { Node, NodeCreateInput, NodeUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { updateOneHelper } from "../../actions/updates";
import { rateLimit } from "../../middleware/rateLimit";
import { CreateOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsNode = {
    Mutation: {
        nodeCreate: GQLEndpoint<NodeCreateInput, CreateOneResult<Node>>;
        nodeUpdate: GQLEndpoint<NodeUpdateInput, UpdateOneResult<Node>>;
    }
}

const objectType = "Node";
export const NodeEndpoints: EndpointsNode = {
    Mutation: {
        nodeCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 2000, req });
            return createOneHelper({ info, input, objectType, prisma, req });
        },
        nodeUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 2000, req });
            return updateOneHelper({ info, input, objectType, prisma, req });
        },
    },
};
