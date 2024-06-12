import { projectVersionDirectory_create, projectVersionDirectory_findMany, projectVersionDirectory_findOne, projectVersionDirectory_update } from "../generated";
import { ProjectVersionDirectoryEndpoints } from "../logic/projectVersionDirectory";
import { setupRoutes } from "./base";

export const ProjectVersionDirectoryRest = setupRoutes({
    "/projectVersionDirectory/:id": {
        get: [ProjectVersionDirectoryEndpoints.Query.projectVersionDirectory, projectVersionDirectory_findOne],
        put: [ProjectVersionDirectoryEndpoints.Mutation.projectVersionDirectoryUpdate, projectVersionDirectory_update],
    },
    "/projectVersionDirectories": {
        get: [ProjectVersionDirectoryEndpoints.Query.projectVersionDirectories, projectVersionDirectory_findMany],
    },
    "/projectVersionDirectory": {
        post: [ProjectVersionDirectoryEndpoints.Mutation.projectVersionDirectoryCreate, projectVersionDirectory_create],
    },
});
