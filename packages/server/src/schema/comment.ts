import { gql } from 'apollo-server-express';
import { CODE } from '@local/shared';
import { CustomError } from '../error';
import { PrismaSelect } from '@paljs/plugins';

export const typeDef = gql`
    input CommentInput {
        id: ID
        text: String
        objectType: String
        objectId: ID
    }

    type Comment {
        id: ID!
        text: String
        createdAt: Date!
        updatedAt: Date!
        userId: ID
        user: User
        organizationId: ID
        organization: Organization
        stars: Int
        vote: Int
    }

    input VoteInput {
        id: ID!
        isUpvote: Boolean!
    }

    extend type Mutation {
        addComment(input: CommentInput!): Email!
        updateComment(input: CommentInput!): Email!
        deleteComment(input: DeleteOneInput!): Boolean!
        reportComment(input: ReportInput!): Boolean!
        voteComment(input: VoteInput!): Boolean!
    }
`

export const resolvers = {
    Mutation: {
        addComment: async (_parent: undefined, { input }: any, context: any, info: any) => {
            return new CustomError(CODE.NotImplemented);
        },
        updateComment: async (_parent: undefined, { input }: any, context: any, info: any) => {
            return new CustomError(CODE.NotImplemented);
        },
        deleteComment: async (_parent: undefined, { input }: any, context: any, _info: any) => {
            return new CustomError(CODE.NotImplemented);
        },
        /**
         * Reports a comment. After enough reports, the comment will be deleted.
         * @returns True if report was successfully recorded
         */
         reportComment: async (_parent: undefined, { input }: any, context: any, _info: any) => {
            // Must be logged in
            if (!context.req.isLoggedIn) return new CustomError(CODE.Unauthorized);
            throw new CustomError(CODE.NotImplemented);
        },
        voteComment: async (_parent: undefined, { input }: any, context: any, _info: any) => {
            return new CustomError(CODE.NotImplemented);
        }
    }
}