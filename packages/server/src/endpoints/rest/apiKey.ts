import { apiKey_create, apiKey_deleteOne, apiKey_update, apiKey_validate } from "@local/shared";
import { ApiKeyEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const ApiKeyRest = setupRoutes({
    "/apiKey": {
        post: [ApiKeyEndpoints.Mutation.apiKeyCreate, apiKey_create],
    },
    "/apiKey/:id": {
        put: [ApiKeyEndpoints.Mutation.apiKeyUpdate, apiKey_update],
        delete: [ApiKeyEndpoints.Mutation.apiKeyDeleteOne, apiKey_deleteOne],
    },
    "/apiKey/validate": {
        post: [ApiKeyEndpoints.Mutation.apiKeyValidate, apiKey_validate],
    },
});
