import { Node, NodeCreateInput, NodeUpdateInput } from "@local/shared";
import { createHelper } from "../../actions/creates";
import { updateHelper } from "../../actions/updates";
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
            return createHelper({ info, input, objectType, prisma, req });
        },
        nodeUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 2000, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    },
};
