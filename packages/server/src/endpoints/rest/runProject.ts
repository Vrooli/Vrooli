import { runProject_create, runProject_deleteAll, runProject_findMany, runProject_findOne, runProject_update } from "../generated";
import { RunProjectEndpoints } from "../logic/runProject";
import { setupRoutes } from "./base";

export const RunProjectRest = setupRoutes({
    "/runProject/:id": {
        get: [RunProjectEndpoints.Query.runProject, runProject_findOne],
        put: [RunProjectEndpoints.Mutation.runProjectUpdate, runProject_update],
    },
    "/runProjects": {
        get: [RunProjectEndpoints.Query.runProjects, runProject_findMany],
    },
    "/runProject": {
        post: [RunProjectEndpoints.Mutation.runProjectCreate, runProject_create],
    },
    "/runProject/deleteAll": {
        delete: [RunProjectEndpoints.Mutation.runProjectDeleteAll, runProject_deleteAll],
    },
});
