import { projectVersion_create, projectVersion_findMany, projectVersion_findOne, projectVersion_update } from "../generated";
import { ProjectVersionEndpoints } from "../logic/projectVersion";
import { setupRoutes } from "./base";

export const ProjectVersionRest = setupRoutes({
    "/projectVersion/:id": {
        get: [ProjectVersionEndpoints.Query.projectVersion, projectVersion_findOne],
        put: [ProjectVersionEndpoints.Mutation.projectVersionUpdate, projectVersion_update],
    },
    "/projectVersions": {
        get: [ProjectVersionEndpoints.Query.projectVersions, projectVersion_findMany],
    },
    "/projectVersionContents": {
        get: [ProjectVersionEndpoints.Query.projectVersionContents, projectVersion_findMany], //TODO selection not right
    },
    "/projectVersion": {
        post: [ProjectVersionEndpoints.Mutation.projectVersionCreate, projectVersion_create],
    },
});
