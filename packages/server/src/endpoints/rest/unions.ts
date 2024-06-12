import { projectOrRoutine_findMany, projectOrTeam_findMany, runProjectOrRunRoutine_findMany } from "../generated";
import { UnionsEndpoints } from "../logic/unions";
import { setupRoutes } from "./base";

export const UnionsRest = setupRoutes({
    "/unions/projectOrRoutines": {
        get: [UnionsEndpoints.Query.projectOrRoutines, projectOrRoutine_findMany],
    },
    "/unions/projectOrTeams": {
        get: [UnionsEndpoints.Query.projectOrTeams, projectOrTeam_findMany],
    },
    "/unions/runProjectOrRunRoutines": {
        get: [UnionsEndpoints.Query.runProjectOrRunRoutines, runProjectOrRunRoutine_findMany],
    },
});
