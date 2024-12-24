import { Node, NodeCreateInput, NodeUpdateInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { ApiEndpoint, CreateOneResult, UpdateOneResult } from "../../types";

export type EndpointsNode = {
    Mutation: {
        nodeCreate: ApiEndpoint<NodeCreateInput, CreateOneResult<Node>>;
        nodeUpdate: ApiEndpoint<NodeUpdateInput, UpdateOneResult<Node>>;
    }
}

const objectType = "Node";
export const NodeEndpoints: EndpointsNode = {
    Mutation: {
        nodeCreate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 2000, req });
            return createOneHelper({ info, input, objectType, req });
        },
        nodeUpdate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 2000, req });
            return updateOneHelper({ info, input, objectType, req });
        },
    },
};
