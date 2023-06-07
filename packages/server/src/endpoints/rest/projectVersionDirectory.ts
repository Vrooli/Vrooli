import { projectVersionDirectory_findMany } from "../generated";
import { ProjectVersionDirectoryEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const ProjectVersionDirectoryRest = setupRoutes({
    "/projectVersionDirectories": {
        get: [ProjectVersionDirectoryEndpoints.Query.projectVersionDirectories, projectVersionDirectory_findMany],
    },
});