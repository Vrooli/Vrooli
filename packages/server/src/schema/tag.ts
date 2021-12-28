import { gql } from 'apollo-server-express';
import { CODE } from '@local/shared';
import { CustomError } from '../error';
import { TagModel } from '../models';
import { IWrap, RecursivePartial } from '../types';
import { Count, DeleteManyInput, FindByIdInput, ReportInput, Success, Tag, TagInput, TagsQueryInput, TagVoteInput } from './types';
import { Context } from '../context';
import { GraphQLResolveInfo } from 'graphql';

export const typeDef = gql`

    input TagInput {
        id: ID
    }

    type Tag {
        id: ID!
        tag: String!
        description: String
        created_at: Date!
        updated_at: Date!
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
        tagsCount: Count!
    }

    extend type Mutation {
        tagAdd(input: TagInput!): Tag!
        tagUpdate(input: TagInput!): Tag!
        tagDeleteMany(input: DeleteManyInput!): Count!
        tagReport(input: ReportInput!): Success!
        tagVote(input: TagVoteInput!): Success!
    }
`

export const resolvers = {
    Query: {
        tag: async (_parent: undefined, { input }: IWrap<FindByIdInput>, { prisma }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Tag> | null> => {
            return await TagModel(prisma).findById(input, info);
        },
        tags: async (_parent: undefined, { input }: IWrap<TagsQueryInput>, context: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Tag>[]> => {
            throw new CustomError(CODE.NotImplemented);
        },
        tagsCount: async (_parent: undefined, _args: undefined, context: Context, info: GraphQLResolveInfo): Promise<Count> => {
            throw new CustomError(CODE.NotImplemented);
        },
    },
    Mutation: {
        /**
         * Add a new tag. Must be unique.
         * @returns Tag object if successful
         */
        tagAdd: async (_parent: undefined, { input }: IWrap<TagInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Tag>> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            // TODO add more restrictions
            return await TagModel(prisma).create(input, info)
        },
        /**
         * Update tags you've created
         * @returns 
         */
        tagUpdate: async (_parent: undefined, { input }: IWrap<TagInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Tag>> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            // TODO add more restrictions
            return await TagModel(prisma).update(input, info);
        },
        /**
         * Delete tags you've created. Other tags must go through a reporting system
         * @returns 
         */
        tagDeleteMany: async (_parent: undefined, { input }: IWrap<DeleteManyInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<Count> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            // TODO add more restrictions
            return await TagModel(prisma).deleteMany(input);
        },
        /**
         * Reports a tag. After enough reports, the tag will be deleted.
         * Objects associated with the tag will not be deleted.
         * @returns True if report was successfully recorded
         */
        tagReport: async (_parent: undefined, { input }: IWrap<ReportInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            const success = await TagModel(prisma).report(input);
            return { success };
        },
        tagVote: async (_parent: undefined, { input }: IWrap<TagVoteInput>, context: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            throw new CustomError(CODE.NotImplemented);
        }
    }
}