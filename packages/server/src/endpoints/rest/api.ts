import { api_create, api_findMany, api_findOne, api_update } from "@local/shared";
import { Router } from "express";
import { ApiEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const ApiRest: any = {
    "/api/:id": {
        get: [ApiEndpoints.Query.api, api_findOne],
        put: [ApiEndpoints.Mutation.apiUpdate, api_update],
    },
    "/apis": {
        get: [ApiEndpoints.Query.apis, api_findMany],
    },
    "/api": {
        post: [ApiEndpoints.Mutation.apiCreate, api_create],
    },
};

const router = Router();
setupRoutes(router, ApiRest);
export default router;
