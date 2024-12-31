import { endpointsPost } from "@local/shared";
import { post_create, post_findMany, post_findOne, post_update } from "../generated";
import { PostEndpoints } from "../logic/post";
import { setupRoutes } from "./base";

export const PostRest = setupRoutes([
    [endpointsPost.findOne, PostEndpoints.Query.post, post_findOne],
    [endpointsPost.findMany, PostEndpoints.Query.posts, post_findMany],
    [endpointsPost.createOne, PostEndpoints.Mutation.postCreate, post_create],
    [endpointsPost.updateOne, PostEndpoints.Mutation.postUpdate, post_update],
]);
