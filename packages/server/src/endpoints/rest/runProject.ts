import { runProject_cancel, runProject_complete, runProject_create, runProject_deleteAll, runProject_findMany, runProject_findOne, runProject_update } from "@local/shared";
import { RunProjectEndpoints } from "../logic";
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
    "/runProject/:id/complete": {
        put: [RunProjectEndpoints.Mutation.runProjectComplete, runProject_complete],
    },
    "/runProject/:id/cancel": {
        put: [RunProjectEndpoints.Mutation.runProjectCancel, runProject_cancel],
    },
});
