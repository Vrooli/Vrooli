import { gql } from 'apollo-server-express';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { FindByIdInput, Tag, TagCreateInput, TagUpdateInput, TagSearchInput, TagSortBy } from './types';
import { rateLimit } from '../middleware';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';

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

    type TagSearchResult {
        pageInfo: PageInfo!
        edges: [TagEdge!]!
    }

    type TagEdge {
        cursor: String!
        node: Tag!
    }

    extend type Query {
        tag(input: FindByIdInput!): Tag
        tags(input: TagSearchInput!): TagSearchResult!
    }

    extend type Mutation {
        tagCreate(input: TagCreateInput!): Tag!
        tagUpdate(input: TagUpdateInput!): Tag!
    }
`

const objectType = 'Tag';
export const resolvers: {
    TagSortBy: typeof TagSortBy;
    Query: {
        tag: GQLEndpoint<FindByIdInput, FindOneResult<Tag>>;
        tags: GQLEndpoint<TagSearchInput, FindManyResult<Tag>>;
    },
    Mutation: {
        tagCreate: GQLEndpoint<TagCreateInput, CreateOneResult<Tag>>;
        tagUpdate: GQLEndpoint<TagUpdateInput, UpdateOneResult<Tag>>;
    }
} = {
    TagSortBy,
    Query: {
        tag: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req })
        },
        tags: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req })
        },
    },
    Mutation: {
        tagCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 500, req });
            return createHelper({ info, input, objectType, prisma, req })
        },
        tagUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 500, req });
            return updateHelper({ info, input, objectType, prisma, req })
        },
    }
}