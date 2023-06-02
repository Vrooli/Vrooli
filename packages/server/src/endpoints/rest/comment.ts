import { comment_create, comment_findMany, comment_findOne, comment_update } from "@local/shared";
import { CommentEndpoints } from "../logic";
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
