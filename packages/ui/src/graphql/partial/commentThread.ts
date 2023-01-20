import { CommentThread } from "@shared/consts";
import { relPartial } from "graphql/utils";
import { GqlPartial } from "types";
import { commentPartial } from "./comment";

export const commentThreadVersionPartial: GqlPartial<CommentThread> = {
    __typename: 'CommentThread',
    common: () => ({
        id: true,
        childThreads: {
            childThreads: {
                comment: () => relPartial(commentPartial, 'list'),
                endCursor: true,
                totalInThread: true,
            },
            comment: () => relPartial(commentPartial, 'list'),
            endCursor: true,
            totalInThread: true,
        },
        comment: () => relPartial(commentPartial, 'list'),
        endCursor: true,
        totalInThread: true,
    }),
}