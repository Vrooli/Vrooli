import { endpointsProjectVersionDirectory } from "@local/shared";
import { projectVersionDirectory_create, projectVersionDirectory_findMany, projectVersionDirectory_findOne, projectVersionDirectory_update } from "../generated";
import { ProjectVersionDirectoryEndpoints } from "../logic/projectVersionDirectory";
import { setupRoutes } from "./base";

export const ProjectVersionDirectoryRest = setupRoutes([
    [endpointsProjectVersionDirectory.findOne, ProjectVersionDirectoryEndpoints.Query.projectVersionDirectory, projectVersionDirectory_findOne],
    [endpointsProjectVersionDirectory.findMany, ProjectVersionDirectoryEndpoints.Query.projectVersionDirectories, projectVersionDirectory_findMany],
    [endpointsProjectVersionDirectory.createOne, ProjectVersionDirectoryEndpoints.Mutation.projectVersionDirectoryCreate, projectVersionDirectory_create],
    [endpointsProjectVersionDirectory.updateOne, ProjectVersionDirectoryEndpoints.Mutation.projectVersionDirectoryUpdate, projectVersionDirectory_update],
]);
