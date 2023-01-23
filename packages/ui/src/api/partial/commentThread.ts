import { CommentThread } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "types";

export const commentThreadPartial: GqlPartial<CommentThread> = {
    __typename: 'CommentThread',
    common: {
        childThreads: {
            childThreads: {
                comment: () => relPartial(require('./comment').commentPartial, 'list'),
                endCursor: true,
                totalInThread: true,
            },
            comment: () => relPartial(require('./comment').commentPartial, 'list'),
            endCursor: true,
            totalInThread: true,
        },
        comment: () => relPartial(require('./comment').commentPartial, 'list'),
        endCursor: true,
        totalInThread: true,
    },
}