import { gql } from 'apollo-server-express';
import { CODE, TAG_SORT_BY } from '@local/shared';
import { CustomError } from '../error';
import { TagModel } from '../models';
import { IWrap, RecursivePartial } from '../types';
import { Count, DeleteManyInput, FindByIdInput, ReportInput, Success, Tag, TagInput, TagSearchInput, TagVoteInput } from './types';
import { Context } from '../context';
import { GraphQLResolveInfo } from 'graphql';

export const typeDef = gql`
    enum TagSortBy {
        AlphabeticalAsc
        AlphabeticalDesc
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        StarsAsc
        StarsDesc
    }

    input TagInput {
        id: ID
    }

    type Tag {
        id: ID!
        tag: String!
        description: String
        created_at: Date!
        updated_at: Date!
        starredBy: [User!]!
    }

    input TagVoteInput {
        id: ID!
        isUpvote: Boolean!
        objectType: String!
        objectId: ID!
    }

    input TagSearchInput {
        userId: Int
        ids: [ID!]
        sortBy: TagSortBy
        searchString: String
        after: String
        take: Int
    }

    # Return type for search result
    type TagSearchResult {
        pageInfo: PageInfo!
        edges: [TagEdge!]!
    }

    # Return type for search result edge
    type TagEdge {
        cursor: String!
        node: Tag!
    }

    extend type Query {
        tag(input: FindByIdInput!): Tag
        tags(input: TagSearchInput!): TagSearchResult!
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
    TagSortBy: TAG_SORT_BY,
    Query: {
        tag: async (_parent: undefined, { input }: IWrap<FindByIdInput>, { prisma }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Tag> | null> => {
            // Query database
            const dbModel = await TagModel(prisma).findById(input, info);
            // Format data
            return dbModel ? TagModel().toGraphQL(dbModel) : null;
        },
        tags: async (_parent: undefined, { input }: IWrap<TagSearchInput>, context: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Tag>[]> => {
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
            // Create object
            const dbModel = await TagModel(prisma).create(input, info);
            // Format object to GraphQL type
            return TagModel().toGraphQL(dbModel);
        },
        /**
         * Update tags you've created
         * @returns 
         */
        tagUpdate: async (_parent: undefined, { input }: IWrap<TagInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Tag>> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            // TODO add more restrictions
            // Update object
            const dbModel = await TagModel(prisma).update(input, info);
            // Format to GraphQL type
            return TagModel().toGraphQL(dbModel);
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