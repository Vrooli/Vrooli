import { standard_create, standard_findMany, standard_findOne, standard_update } from "../generated";
import { StandardEndpoints } from "../logic/standard";
import { setupRoutes } from "./base";

export const StandardRest = setupRoutes({
    "/standard/:id": {
        get: [StandardEndpoints.Query.standard, standard_findOne],
        put: [StandardEndpoints.Mutation.standardUpdate, standard_update],
    },
    "/standards": {
        get: [StandardEndpoints.Query.standards, standard_findMany],
    },
    "/standard": {
        post: [StandardEndpoints.Mutation.standardCreate, standard_create],
    },
});
