import { code_create, code_findMany, code_findOne, code_update } from "../generated";
import { CodeEndpoints } from "../logic/code";
import { setupRoutes } from "./base";

export const CodeRest = setupRoutes({
    "/code/:id": {
        get: [CodeEndpoints.Query.code, code_findOne],
        put: [CodeEndpoints.Mutation.codeUpdate, code_update],
    },
    "/codes": {
        get: [CodeEndpoints.Query.codes, code_findMany],
    },
    "/code": {
        post: [CodeEndpoints.Mutation.codeCreate, code_create],
    },
});
