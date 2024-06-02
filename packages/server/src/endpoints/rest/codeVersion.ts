import { codeVersion_create, codeVersion_findMany, codeVersion_findOne, codeVersion_update } from "../generated";
import { CodeVersionEndpoints } from "../logic/codeVersion";
import { setupRoutes } from "./base";

export const CodeVersionRest = setupRoutes({
    "/codeVersion/:id": {
        get: [CodeVersionEndpoints.Query.codeVersion, codeVersion_findOne],
        put: [CodeVersionEndpoints.Mutation.codeVersionUpdate, codeVersion_update],
    },
    "/codeVersions": {
        get: [CodeVersionEndpoints.Query.codeVersions, codeVersion_findMany],
    },
    "/codeVersion": {
        post: [CodeVersionEndpoints.Mutation.codeVersionCreate, codeVersion_create],
    },
});
