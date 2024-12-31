import { endpointsComment } from "@local/shared";
import { comment_create, comment_findMany, comment_findOne, comment_update } from "../generated";
import { CommentEndpoints } from "../logic/comment";
import { setupRoutes } from "./base";

export const CommentRest = setupRoutes([
    [endpointsComment.findOne, CommentEndpoints.Query.comment, comment_findOne],
    [endpointsComment.findMany, CommentEndpoints.Query.comments, comment_findMany],
    [endpointsComment.createOne, CommentEndpoints.Mutation.commentCreate, comment_create],
    [endpointsComment.updateOne, CommentEndpoints.Mutation.commentUpdate, comment_update],
]);
