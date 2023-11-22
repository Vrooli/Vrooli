import { post_create, post_findMany, post_findOne, post_update } from "../generated";
import { PostEndpoints } from "../logic/post";
import { setupRoutes } from "./base";

export const PostRest = setupRoutes({
    "/post/:id": {
        get: [PostEndpoints.Query.post, post_findOne],
        put: [PostEndpoints.Mutation.postUpdate, post_update],
    },
    "/posts": {
        get: [PostEndpoints.Query.posts, post_findMany],
    },
    "/post": {
        post: [PostEndpoints.Mutation.postCreate, post_create],
    },
});
