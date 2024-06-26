import { comment_create, comment_findMany, comment_findOne, comment_update } from "../generated";
import { CommentEndpoints } from "../logic/comment";
import { setupRoutes } from "./base";

export const CommentRest = setupRoutes({
    "/comment/:id": {
        get: [CommentEndpoints.Query.comment, comment_findOne],
        put: [CommentEndpoints.Mutation.commentUpdate, comment_update],
    },
    "/comments": {
        get: [CommentEndpoints.Query.comments, comment_findMany],
    },
    "/comment": {
        post: [CommentEndpoints.Mutation.commentCreate, comment_create],
    },
});
