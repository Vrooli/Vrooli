import { gql } from 'apollo-server-express';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { FindByIdInput, Post, PostSearchInput, PostCreateInput, PostUpdateInput, PostSortBy } from './types';
import { rateLimit } from '../middleware';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';

export const typeDef = gql`
    enum PostSortBy {
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        StarsAsc
        StarsDesc
    }

    input PostCreateInput {
        anonymous: Boolean
        tag: String!
        translationsCreate: [PostTranslationCreateInput!]
    }
    input PostUpdateInput {
        anonymous: Boolean
        tag: String!
        translationsDelete: [ID!]
        translationsCreate: [PostTranslationCreateInput!]
        translationsUpdate: [PostTranslationUpdateInput!]
    }

    # User's hidden topics
    input PostHiddenCreateInput {
        id: ID!
        isBlur: Boolean
        tagCreate: PostCreateInput
        tagConnect: ID
    }

    input PostHiddenUpdateInput {
        id: ID!
        isBlur: Boolean
    }

    type Post {
        id: ID!
        tag: String!
        created_at: Date!
        updated_at: Date!
        stars: Int!
        isStarred: Boolean!
        isOwn: Boolean!
        starredBy: [User!]!
        translations: [PostTranslation!]!
    }

    input PostTranslationCreateInput {
        id: ID!
        language: String!
        description: String
    }
    input PostTranslationUpdateInput {
        id: ID!
        language: String
        description: String
    }
    type PostTranslation {
        id: ID!
        language: String!
        description: String
    }

    # Wraps tag with hidden/blurred option
    type PostHidden {
        id: ID!
        isBlur: Boolean!
        tag: Post!
    }

    input PostSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        excludeIds: [ID!]
        hidden: Boolean
        ids: [ID!]
        languages: [String!]
        minStars: Int
        myPosts: Boolean
        searchString: String
        sortBy: PostSortBy
        take: Int
        updatedTimeFrame: TimeFrame
    }

    type PostSearchResult {
        pageInfo: PageInfo!
        edges: [PostEdge!]!
    }

    type PostEdge {
        cursor: String!
        node: Post!
    }

    extend type Query {
        post(input: FindByIdInput!): Post
        posts(input: PostSearchInput!): PostSearchResult!
    }

    extend type Mutation {
        postCreate(input: PostCreateInput!): Post!
        postUpdate(input: PostUpdateInput!): Post!
    }
`

const objectType = 'Post';
export const resolvers: {
    PostSortBy: typeof PostSortBy;
    Query: {
        post: GQLEndpoint<FindByIdInput, FindOneResult<Post>>;
        posts: GQLEndpoint<PostSearchInput, FindManyResult<Post>>;
    },
    Mutation: {
        postCreate: GQLEndpoint<PostCreateInput, CreateOneResult<Post>>;
        postUpdate: GQLEndpoint<PostUpdateInput, UpdateOneResult<Post>>;
    }
} = {
    PostSortBy: PostSortBy,
    Query: {
        post: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req })
        },
        posts: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req })
        },
    },
    Mutation: {
        postCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 500, req });
            return createHelper({ info, input, objectType, prisma, req })
        },
        postUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 500, req });
            return updateHelper({ info, input, objectType, prisma, req })
        },
    }
}