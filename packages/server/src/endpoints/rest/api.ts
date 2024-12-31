import { endpointsApi } from "@local/shared";
import { api_create, api_findMany, api_findOne, api_update } from "../generated";
import { ApiEndpoints } from "../logic/api";
import { setupRoutes } from "./base";

export const ApiRest = setupRoutes([
    [endpointsApi.findOne, ApiEndpoints.Query.api, api_findOne],
    [endpointsApi.findMany, ApiEndpoints.Query.apis, api_findMany],
    [endpointsApi.createOne, ApiEndpoints.Mutation.apiCreate, api_create],
    [endpointsApi.updateOne, ApiEndpoints.Mutation.apiUpdate, api_update],
]);
