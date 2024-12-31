import { endpointsApiVersion } from "@local/shared";
import { apiVersion_create, apiVersion_findMany, apiVersion_findOne, apiVersion_update } from "../generated";
import { ApiVersionEndpoints } from "../logic/apiVersion";
import { setupRoutes } from "./base";

export const ApiVersionRest = setupRoutes([
    [endpointsApiVersion.findOne, ApiVersionEndpoints.Query.apiVersion, apiVersion_findOne],
    [endpointsApiVersion.findMany, ApiVersionEndpoints.Query.apiVersions, apiVersion_findMany],
    [endpointsApiVersion.createOne, ApiVersionEndpoints.Mutation.apiVersionCreate, apiVersion_create],
    [endpointsApiVersion.updateOne, ApiVersionEndpoints.Mutation.apiVersionUpdate, apiVersion_update],
]);
