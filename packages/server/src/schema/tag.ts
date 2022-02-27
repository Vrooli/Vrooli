import { gql } from 'apollo-server-express';
import { TagSortBy } from '@local/shared';
import { countHelper, createHelper, deleteManyHelper, readManyHelper, readOneHelper, TagModel, tagSearcher, updateHelper } from '../models';
import { IWrap, RecursivePartial } from '../types';
import { Count, DeleteManyInput, FindByIdInput, Tag, TagCountInput, TagCreateInput, TagUpdateInput, TagSearchInput, TagSearchResult } from './types';
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

    input TagCreateInput {
        anonymous: Boolean
        description: String
        tag: String!
    }
    input TagUpdateInput {
        id: ID!
        anonymous: Boolean
        description: String
        tag: String
    }

    type Tag {
        id: ID!
        tag: String!
        description: String
        created_at: Date!
        updated_at: Date!
        stars: Int!
        isStarred: Boolean!
        isOwn: Boolean!
        starredBy: [User!]!
    }

    input TagSearchInput {
        myTags: Boolean
        hidden: Boolean
        ids: [ID!]
        sortBy: TagSortBy
        searchString: String
        createdTimeFrame: TimeFrame
        updatedTimeFrame: TimeFrame
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

    # Input for count
    input TagCountInput {
        createdTimeFrame: TimeFrame
        updatedTimeFrame: TimeFrame
    }

    extend type Query {
        tag(input: FindByIdInput!): Tag
        tags(input: TagSearchInput!): TagSearchResult!
        tagsCount(input: TagCountInput!): Int!
    }

    extend type Mutation {
        tagCreate(input: TagCreateInput!): Tag!
        tagUpdate(input: TagUpdateInput!): Tag!
        tagDeleteMany(input: DeleteManyInput!): Count!
    }
`

export const resolvers = {
    TagSortBy: TagSortBy,
    Query: {
        tag: async (_parent: undefined, { input }: IWrap<FindByIdInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Tag> | null> => {
            return readOneHelper(req.userId, input, info, prisma);
        },
        tags: async (_parent: undefined, { input }: IWrap<TagSearchInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<TagSearchResult> => {
            return readManyHelper(req.userId, input, info, prisma, tagSearcher());
        },
        tagsCount: async (_parent: undefined, { input }: IWrap<TagCountInput>, { prisma }: Context, _info: GraphQLResolveInfo): Promise<number> => {
            return countHelper(input, 'Tag', prisma);
        },
    },
    Mutation: {
        /**
         * Create a new tag. Must be unique.
         * @returns Tag object if successful
         */
        tagCreate: async (_parent: undefined, { input }: IWrap<TagCreateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Tag>> => {
            return createHelper(req.userId, input, info, TagModel(prisma).cud, prisma);
        },
        /**
         * Update tags you've created
         * @returns 
         */
        tagUpdate: async (_parent: undefined, { input }: IWrap<TagUpdateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Tag>> => {
            return updateHelper(req.userId, input, info, TagModel(prisma).cud, prisma);
        },
        /**
         * Delete tags you've created. Other tags must go through a reporting system
         * @returns 
         */
        tagDeleteMany: async (_parent: undefined, { input }: IWrap<DeleteManyInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<Count> => {
            return deleteManyHelper(req.userId, input, TagModel(prisma).cud);
        },
    }
}