import { gql } from 'apollo-server-express';
import { CODE } from '@local/shared';
import { CustomError } from '../error';
import { PrismaSelect } from '@paljs/plugins';
import { TagModel } from '../models';

export const typeDef = gql`

    input TagInput {
        id: ID
    }

    type Tag {
        id: ID!
    }

    input TagsQueryInput {
        first: Int
        skip: Int
    }

    input TagVoteInput {
        id: ID!
        isUpvote: Boolean!
        objectType: String!
        objectId: ID!
    }

    extend type Query {
        tag(input: FindByIdInput!): Tag
        tags(input: TagsQueryInput!): [Tag!]!
        tagsCount: Int!
    }

    extend type Mutation {
        addTag(input: ResourceInput!): Resource!
        updateTag(input: ResourceInput!): Resource!
        deleteTags(input: DeleteManyInput!): Count!
        reportTag(input: ReportInput!): Boolean!
        voteTag(input: TagVoteInput!): Boolean!
    }
`

export const resolvers = {
    Query: {
        tag: async (_parent: undefined, { input }: any, context: any, info: any) => {
            return new CustomError(CODE.NotImplemented);
        },
        tags: async (_parent: undefined, { input }: any, context: any, info: any) => {
            return new CustomError(CODE.NotImplemented);
        },
        tagsCount: async (_parent: undefined, _args: any, context: any, info: any) => {
            return new CustomError(CODE.NotImplemented);
        },
    },
    Mutation: {
        /**
         * Add a new tag. Must be unique.
         * @returns Tag object if successful
         */
        addTag: async (_parent: undefined, { input }: any, context: any, info: any) => {
            // Must be logged in
            if (!context.req.isLoggedIn) return new CustomError(CODE.Unauthorized);
            // Create object
            return await new TagModel(context.prisma).create(input, info)
        },
        /**
         * Update tags you've created
         * @returns 
         */
        updateTag: async (_parent: undefined, { input }: any, context: any, info: any) => {
            // Must be logged in
            if (!context.req.isLoggedIn) return new CustomError(CODE.Unauthorized);
            // Update object
            return await new TagModel(context.prisma).update(input, info);
        },
        /**
         * Delete tags you've created. Other tags must go through a reporting system
         * @returns 
         */
        deleteTags: async (_parent: undefined, { input }: any, context: any, _info: any) => {
            // Must be logged in
            if (!context.req.isLoggedIn) return new CustomError(CODE.Unauthorized);
            // Delete objects
            return await new TagModel(context.prisma).deleteMany(input.ids);
        },
        /**
         * Reports a tag. After enough reports, the tag will be deleted.
         * Objects associated with the tag will not be deleted.
         * @returns True if report was successfully recorded
         */
         reportTag: async (_parent: undefined, { input }: any, context: any, _info: any) => {
            // Must be logged in
            if (!context.req.isLoggedIn) return new CustomError(CODE.Unauthorized);
            throw new CustomError(CODE.NotImplemented);
        },
        voteTag: async (_parent: undefined, { input }: any, context: any, _info: any) => {
            return new CustomError(CODE.NotImplemented);
        }
    }
}