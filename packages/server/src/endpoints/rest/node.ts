import { node_create, node_update } from "../generated";
import { NodeEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const NodeRest = setupRoutes({
    "/node": {
        post: [NodeEndpoints.Mutation.nodeCreate, node_create],
    },
    "/node/:id": {
        put: [NodeEndpoints.Mutation.nodeUpdate, node_update],
    },
});
