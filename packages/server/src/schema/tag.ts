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
        anonymous: Boolean
        tag: String!
        translationsCreate: [TagTranslationCreateInput!]
    }
    input TagUpdateInput {
        anonymous: Boolean
        tag: String!
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
        id: ID!
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
        excludeIds: [ID!]
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
        tag: async (_parent: undefined, { input }: IWrap<FindByIdInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Tag> | null> => {
            await rateLimit({ info, max: 1000, req });
            return readOneHelper({ info, input, model: TagModel, prisma, userId: req.userId })
        },
        tags: async (_parent: undefined, { input }: IWrap<TagSearchInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<TagSearchResult> => {
            await rateLimit({ info, max: 1000, req });
            return readManyHelper({ info, input, model: TagModel, prisma, userId: req.userId })
        },
        tagsCount: async (_parent: undefined, { input }: IWrap<TagCountInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<number> => {
            await rateLimit({ info, max: 1000, req });
            return countHelper({ input, model: TagModel, prisma })
        },
    },
    Mutation: {
        /**
         * Create a new tag. Must be unique.
         * @returns Tag object if successful
         */
        tagCreate: async (_parent: undefined, { input }: IWrap<TagCreateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Tag>> => {
            await rateLimit({ info, max: 500, byAccountOrKey: true, req });
            return createHelper({ info, input, model: TagModel, prisma, userId: req.userId })
        },
        /**
         * Update tags you've created
         */
        tagUpdate: async (_parent: undefined, { input }: IWrap<TagUpdateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Tag>> => {
            await rateLimit({ info, max: 500, byAccountOrKey: true, req });
            return updateHelper({ info, input, model: TagModel, prisma, userId: req.userId })
        },
        /**
         * Delete tags you've created. Other tags must go through a reporting system
         */
        tagDeleteMany: async (_parent: undefined, { input }: IWrap<DeleteManyInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<Count> => {
            await rateLimit({ info, max: 250, byAccountOrKey: true, req });
            return deleteManyHelper({ input, model: TagModel, prisma, userId: req.userId })
        },
    }
}