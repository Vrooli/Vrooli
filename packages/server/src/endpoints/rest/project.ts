import { endpointsProject } from "@local/shared";
import { project_create, project_findMany, project_findOne, project_update } from "../generated";
import { ProjectEndpoints } from "../logic/project";
import { setupRoutes } from "./base";

export const ProjectRest = setupRoutes([
    [endpointsProject.findOne, ProjectEndpoints.Query.project, project_findOne],
    [endpointsProject.findMany, ProjectEndpoints.Query.projects, project_findMany],
    [endpointsProject.createOne, ProjectEndpoints.Mutation.projectCreate, project_create],
    [endpointsProject.updateOne, ProjectEndpoints.Mutation.projectUpdate, project_update],
]);
