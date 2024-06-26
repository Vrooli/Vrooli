import { tag_create, tag_findMany, tag_findOne, tag_update } from "../generated";
import { TagEndpoints } from "../logic/tag";
import { setupRoutes } from "./base";

export const TagRest = setupRoutes({
    "/tag/:id": {
        get: [TagEndpoints.Query.tag, tag_findOne],
        put: [TagEndpoints.Mutation.tagUpdate, tag_update],
    },
    "/tags": {
        get: [TagEndpoints.Query.tags, tag_findMany],
    },
    "/tag": {
        post: [TagEndpoints.Mutation.tagCreate, tag_create],
    },
});
