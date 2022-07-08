import { gql } from 'apollo-server-express';
import { countHelper, createHelper, deleteManyHelper, readManyHelper, readOneHelper, TagModel, updateHelper } from '../models';
import { IWrap, RecursivePartial } from '../types';
import { Count, DeleteManyInput, FindByIdInput, Tag, TagCountInput, TagCreateInput, TagUpdateInput, TagSearchInput, TagSearchResult, TagSortBy } from './types';
import { Context } from '../context';
import { GraphQLResolveInfo } from 'graphql';
import { rateLimit } from '../rateLimit';

export const typeDef = gql`
    enum TagSortBy {
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        StarsAsc
        StarsDesc
    }

    input TagCreateInput {
        id: ID!
        anonymous: Boolean
        tag: String!
        translationsCreate: [TagTranslationCreateInput!]
    }
    input TagUpdateInput {
        id: ID!
        anonymous: Boolean
        tag: String
        translationsDelete: [ID!]
        translationsCreate: [TagTranslationCreateInput!]
        translationsUpdate: [TagTranslationUpdateInput!]
    }

    # User's hidden topics
    input TagHiddenCreateInput {
        id: ID!
        isBlur: Boolean
        tagCreate: TagCreateInput
        tagConnect: ID
    }

    input TagHiddenUpdateInput {
        id: ID
        isBlur: Boolean
    }

    type Tag {
        id: ID!
        tag: String!
        created_at: Date!
        updated_at: Date!
        stars: Int!
        isStarred: Boolean!
        isOwn: Boolean!
        starredBy: [User!]!
        translations: [TagTranslation!]!
    }

    input TagTranslationCreateInput {
        id: ID!
        language: String!
        description: String
    }
    input TagTranslationUpdateInput {
        id: ID!
        language: String
        description: String
    }
    type TagTranslation {
        id: ID!
        language: String!
        description: String
    }

    # Wraps tag with hidden/blurred option
    type TagHidden {
        id: ID!
        isBlur: Boolean!
        tag: Tag!
    }

    input TagSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        hidden: Boolean
        ids: [ID!]
        languages: [String!]
        minStars: Int
        myTags: Boolean
        searchString: String
        sortBy: TagSortBy
        take: Int
        updatedTimeFrame: TimeFrame
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
        tag: async (_parent: undefined, { input }: IWrap<FindByIdInput>, context: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Tag> | null> => {
            await rateLimit({ context, info, max: 1000 });
            return readOneHelper(context.req.userId, input, info, TagModel(context.prisma));
        },
        tags: async (_parent: undefined, { input }: IWrap<TagSearchInput>, context: Context, info: GraphQLResolveInfo): Promise<TagSearchResult> => {
            await rateLimit({ context, info, max: 1000 });
            return readManyHelper(context.req.userId, input, info, TagModel(context.prisma));
        },
        tagsCount: async (_parent: undefined, { input }: IWrap<TagCountInput>, context: Context, info: GraphQLResolveInfo): Promise<number> => {
            await rateLimit({ context, info, max: 1000 });
            return countHelper(input, TagModel(context.prisma));
        },
    },
    Mutation: {
        /**
         * Create a new tag. Must be unique.
         * @returns Tag object if successful
         */
        tagCreate: async (_parent: undefined, { input }: IWrap<TagCreateInput>, context: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Tag>> => {
            await rateLimit({ context, info, max: 500, byAccount: true });
            return createHelper(context.req.userId, input, info, TagModel(context.prisma));
        },
        /**
         * Update tags you've created
         * @returns 
         */
        tagUpdate: async (_parent: undefined, { input }: IWrap<TagUpdateInput>, context: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Tag>> => {
            await rateLimit({ context, info, max: 500, byAccount: true });
            return updateHelper(context.req.userId, input, info, TagModel(context.prisma));
        },
        /**
         * Delete tags you've created. Other tags must go through a reporting system
         * @returns 
         */
        tagDeleteMany: async (_parent: undefined, { input }: IWrap<DeleteManyInput>, context: Context, info: GraphQLResolveInfo): Promise<Count> => {
            await rateLimit({ context, info, max: 250, byAccount: true });
            return deleteManyHelper(context.req.userId, input, TagModel(context.prisma));
        },
    }
}