import { commentFields, threadFields } from 'graphql/fragment';
import { toGql } from 'graphql/utils';

export const commentEndpoint = {
    findOne: toGql('query', 'comment', 'FindByIdInput', [commentFields], `...commentFields`),
    findMany: toGql('query', 'comments', 'CommentSearchInput', [threadFields], `
        endCursor
        totalThreads
        threads {
            ...threadFields
        }
    `),
    create: toGql('mutation', 'commentCreate', 'CommentCreateInput', [commentFields], `...commentFields`),
    update: toGql('mutation', 'commentUpdate', 'CommentUpdateInput', [commentFields], `...commentFields`)
}