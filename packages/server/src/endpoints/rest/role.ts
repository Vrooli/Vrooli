import { role_create, role_findMany, role_findOne, role_update } from "../generated";
import { RoleEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const RoleRest = setupRoutes({
    "/role/:id": {
        get: [RoleEndpoints.Query.role, role_findOne],
        put: [RoleEndpoints.Mutation.roleUpdate, role_update],
    },
    "/roles": {
        get: [RoleEndpoints.Query.roles, role_findMany],
    },
    "/role": {
        post: [RoleEndpoints.Mutation.roleCreate, role_create],
    },
});
