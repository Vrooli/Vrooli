import { endpointsApiKey } from "@local/shared";
import { apiKey_create, apiKey_deleteOne, apiKey_update, apiKey_validate } from "../generated";
import { ApiKeyEndpoints } from "../logic/apiKey";
import { setupRoutes } from "./base";

export const ApiKeyRest = setupRoutes([
    [endpointsApiKey.createOne, ApiKeyEndpoints.Mutation.apiKeyCreate, apiKey_create],
    [endpointsApiKey.updateOne, ApiKeyEndpoints.Mutation.apiKeyUpdate, apiKey_update],
    [endpointsApiKey.deleteOne, ApiKeyEndpoints.Mutation.apiKeyDeleteOne, apiKey_deleteOne],
    [endpointsApiKey.validateOne, ApiKeyEndpoints.Mutation.apiKeyValidate, apiKey_validate],
]);
