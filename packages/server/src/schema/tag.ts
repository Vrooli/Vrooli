import { gql } from 'apollo-server-express';
import { CODE } from '@local/shared';
import { CustomError } from '../error';
import { PrismaSelect } from '@paljs/plugins';
import { TagModel } from '../models';
import { IWrap } from 'types';
import { DeleteManyInput, FindByIdInput, ReportInput, Tag, TagInput, TagsQueryInput, TagVoteInput } from './types';

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
        addTag(input: TagInput!): Tag!
        updateTag(input: TagInput!): Tag!
        deleteTags(input: DeleteManyInput!): Count!
        reportTag(input: ReportInput!): Boolean!
        voteTag(input: TagVoteInput!): Boolean!
    }
`

export const resolvers = {
    Query: {
        tag: async (_parent: undefined, { input }: IWrap<FindByIdInput>, context: any, info: any): Promise<Tag> => {
            throw new CustomError(CODE.NotImplemented);
        },
        tags: async (_parent: undefined, { input }: IWrap<TagsQueryInput>, context: any, info: any): Promise<Tag[]> => {
            throw new CustomError(CODE.NotImplemented);
        },
        tagsCount: async (_parent: undefined, _args: undefined, context: any, info: any): Promise<number> => {
            throw new CustomError(CODE.NotImplemented);
        },
    },
    Mutation: {
        /**
         * Add a new tag. Must be unique.
         * @returns Tag object if successful
         */
        addTag: async (_parent: undefined, { input }: IWrap<TagInput>, context: any, info: any): Promise<Tag> => {
            // Must be logged in
            if (!context.req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            // Create object
            return await new TagModel(context.prisma).create(input, info)
        },
        /**
         * Update tags you've created
         * @returns 
         */
        updateTag: async (_parent: undefined, { input }: IWrap<TagInput>, context: any, info: any): Promise<Tag> => {
            // Must be logged in
            if (!context.req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            // Update object
            return await new TagModel(context.prisma).update(input, info);
        },
        /**
         * Delete tags you've created. Other tags must go through a reporting system
         * @returns 
         */
        deleteTags: async (_parent: undefined, { input }: IWrap<DeleteManyInput>, context: any, _info: any): Promise<number> => {
            // Must be logged in
            if (!context.req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            // Delete objects
            return await new TagModel(context.prisma).deleteMany(input.ids);
        },
        /**
         * Reports a tag. After enough reports, the tag will be deleted.
         * Objects associated with the tag will not be deleted.
         * @returns True if report was successfully recorded
         */
         reportTag: async (_parent: undefined, { input }: IWrap<ReportInput>, context: any, _info: any): Promise<boolean> => {
            // Must be logged in
            if (!context.req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            throw new CustomError(CODE.NotImplemented);
        },
        voteTag: async (_parent: undefined, { input }: IWrap<TagVoteInput>, context: any, _info: any): Promise<boolean> => {
            throw new CustomError(CODE.NotImplemented);
        }
    }
}