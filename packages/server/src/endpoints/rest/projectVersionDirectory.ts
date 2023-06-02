import { projectVersionDirectory_findMany } from "@local/shared";
import { ProjectVersionDirectoryEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const ProjectVersionDirectoryRest = setupRoutes({
    "/projectVersionDirectories": {
        get: [ProjectVersionDirectoryEndpoints.Query.projectVersionDirectories, projectVersionDirectory_findMany],
    },
});
