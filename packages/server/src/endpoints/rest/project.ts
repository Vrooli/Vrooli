import { project_create, project_findMany, project_findOne, project_update } from "../generated";
import { ProjectEndpoints } from "../logic/project";
import { setupRoutes } from "./base";

export const ProjectRest = setupRoutes({
    "/project/:id": {
        get: [ProjectEndpoints.Query.project, project_findOne],
        put: [ProjectEndpoints.Mutation.projectUpdate, project_update],
    },
    "/projects": {
        get: [ProjectEndpoints.Query.projects, project_findMany],
    },
    "/project": {
        post: [ProjectEndpoints.Mutation.projectCreate, project_create],
    },
});
