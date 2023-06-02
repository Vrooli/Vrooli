import { projectOrOrganization_findMany, projectOrRoutine_findMany, runProjectOrRunRoutine_findMany } from "@local/shared";
import { UnionsEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const UnionsRest = setupRoutes({
    "/unions/projectOrRoutines": {
        get: [UnionsEndpoints.Query.projectOrRoutines, projectOrRoutine_findMany],
    },
    "/unions/projectOrOrganizations": {
        get: [UnionsEndpoints.Query.projectOrOrganizations, projectOrOrganization_findMany],
    },
    "/unions/runProjectOrRunRoutines": {
        get: [UnionsEndpoints.Query.runProjectOrRunRoutines, runProjectOrRunRoutine_findMany],
    },
});
